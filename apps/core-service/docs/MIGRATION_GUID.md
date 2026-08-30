# Database Migrations — As Implemented

## Who this doc is for

You're comfortable with Go and SQL but haven't necessarily used
[golang-migrate](https://github.com/golang-migrate/migrate) before. Section 0 is a short
primer on the one non-obvious piece of Postgres behavior this project leans on — skip it
if you already know how Postgres handles multi-statement queries.

This doc is verified against the current source (`cmd/migration/main.go`, `db/migrations/*.sql`,
`Makefile`), not the intended design. It also corrects a mechanism claim from an earlier
version of this doc — read it before assuming `make migrate-down` does what its name
suggests, because it doesn't (Section 2).

## Section 0 — Background primer: how a multi-statement file becomes "atomic"

`migrate` is used here as an **external CLI binary**, not a vendored Go dependency — it
doesn't appear in `go.mod`. `cmd/migration/main.go` shells out to it with `exec.Command`.

The atomicity this project relies on doesn't come from golang-migrate wrapping your file
in `BEGIN`/`COMMIT` for you. Looking at the driver it shells out to
([`database/postgres/postgres.go`](https://github.com/golang-migrate/migrate/blob/master/database/postgres/postgres.go)),
`Run()` reads the entire migration file into one string and executes it as a single,
argument-less `ExecContext` call — no transaction handling in that function or in
`runStatement`.

What actually makes the file atomic is Postgres itself: `ExecContext` with no query
arguments makes `lib/pq` send the whole file as one **simple-query** protocol message.
When a simple-query message contains several `;`-separated statements, Postgres
implicitly wraps all of them in one transaction — *unless* the message itself contains
explicit `BEGIN`/`COMMIT`, in which case those explicit commands take over. Either way,
the net effect is the same: all statements in one file commit or roll back together.

**Gotchas that surprise newcomers:**

1. The explicit `BEGIN; … COMMIT;` you'll see in this repo's `*.up.sql` files isn't
   required for atomicity — Postgres already gives you that implicitly for a
   multi-statement message. It was added deliberately anyway (commit `52f01b7`,
   "Update migration file to run on transaction") for explicitness. This is why
   Section 3 flags an inconsistency: the matching `*.down.sql` files were never updated
   to match and still rely on the implicit behavior.
2. Because this is Postgres's own multi-statement rule and not something golang-migrate
   configures, statements that categorically cannot run inside *any* transaction
   (`CREATE INDEX CONCURRENTLY`, etc.) will fail no matter what you do — see Section 3.
3. `migrate`'s CLI has a real `down` command that runs `*.down.sql` files. This project's
   `down` subcommand does **not** call it — see Section 2.

## Section 1 — Architecture at a Glance

The composition root is [`cmd/migration/main.go`](../cmd/migration/main.go). It does not
implement migration logic itself — it dispatches CLI args to config loading and then
shells out to the external `migrate` binary.

| Concern | Owner | Analogy |
|---|---|---|
| CLI entrypoint & dispatch | `cmd/migration/main.go` (`main`, lines 60-91) | A switchboard operator — routes the call, does no work itself |
| DB connection string | `internal/config.LoadConfig`, shared with `cmd/server` | A phone book both the migration tool and the server look up |
| Migration execution engine | External `migrate` CLI binary — not a Go dependency | A contractor hired to do the actual work |
| Migration SQL & transaction boundaries | `db/migrations/*.sql` | The work orders handed to the contractor |
| Version bookkeeping | `schema_migrations` table, managed entirely by `migrate` | The contractor's own logbook — this project never reads or writes it directly |

It's split this way so the project never has to reimplement version tracking or SQL
execution — `migrate` already does both. The cost of that thinness shows up in Section 2:
because `main.go` is just a dispatcher, its `down` subcommand ended up wired to the wrong
underlying `migrate` command and nothing in the type system catches that mismatch.

## Section 2 — `cmd/migration/main.go`: the CLI wrapper

**The problem this solves:** give the project a `go run`/`make` interface for
`migrate create` / `up` / `down` that pulls `DB_SOURCE` from this project's own config
loader, and make failures visible to scripts/CI (a failed run must not exit `0`).

**How each subcommand is actually implemented:**

| Subcommand | What it actually runs | Matches expectation? |
|---|---|---|
| `create <name>` | `migrate create -ext sql -dir db/migrations -seq <name>` ([main.go:14](../cmd/migration/main.go#L14)) | Yes — scaffolds a blank `up`/`down` SQL pair |
| `up` | `migrate -path db/migrations -database <DB_SOURCE> up` ([main.go:27](../cmd/migration/main.go#L27)) | Yes — runs all pending `*.up.sql` files in order |
| `down` | `migrate -path db/migrations -database <DB_SOURCE> force 1` ([main.go:44](../cmd/migration/main.go#L44)) | **No** — see callout below |

### Worth flagging: `make migrate-down` does not run any `*.down.sql` file

`downMigration` doesn't call `migrate ... down` at all. It calls `migrate ... force 1`,
which only rewrites the `schema_migrations` bookkeeping row to `(version=1, dirty=false)`
— `force` runs **zero SQL**. So `make migrate-down`:

- Does not drop the `users` table, does not remove `user_id` columns, does not undo
  anything `000002`-`000004` added.
- Will silently desync your recorded version from your actual schema if the schema was
  ever really at a version other than 1.

Git history shows this wasn't an accident that got fixed later — an earlier, abandoned
line in the same function even tried `down force 1` (not valid `migrate` syntax) before
settling on the current `force 1` shortcut. If you actually need to undo a migration,
apply the relevant `*.down.sql` file yourself (e.g. `psql "$DB_SOURCE" -f
db/migrations/000004_authorization_by_user_tabel.down.sql`) or call the `migrate` binary
directly with its real `down` command — this project's CLI wrapper doesn't expose it.

**Other rough edges in this file:**

- The failure-path comments on [main.go:32-34](../cmd/migration/main.go#L32-L34) and
  [main.go:49-51](../cmd/migration/main.go#L49-L51) point readers to
  `db/migrations/README.md` for more detail — that file doesn't exist in this repo. This
  doc (`docs/MIGRATION_GUID.md`) is the real reference; the comments are stale.
- `force 1` hardcodes the target version to `1`. There's no flag or arg to force to any
  other version through this wrapper — you have to call `migrate ... force <version>`
  directly, which is also what Section 4's recovery steps tell you to do.

## Section 3 — Migration files & the transaction boundary (`db/migrations/*.sql`)

**The problem this solves:** a mid-file failure (the second `ALTER TABLE` in a file
errors, say) must not leave the first `ALTER TABLE` in that same file committed.

**How it's actually implemented:** as explained in Section 0, every file is executed as
one multi-statement message, so Postgres either commits or rolls back the whole file as
a unit. Concretely: if `000004_authorization_by_user_tabel.up.sql`'s second `DO $$ ... $$`
block failed, the earlier `ADD COLUMN user_id` in the same file would roll back too.

| File | `up.sql` wraps in explicit `BEGIN`/`COMMIT`? | `down.sql` wraps in explicit `BEGIN`/`COMMIT`? |
|---|---|---|
| `000001_init-transaction-feature` | Yes | No |
| `000002_add-balance-constraints` | Yes | No |
| `000003_add-users-table` | Yes | No |
| `000004_authorization_by_user_tabel` | Yes | No |

**Worth flagging:** commit `52f01b7` added explicit `BEGIN;`/`COMMIT;` to all four
`up.sql` files but never touched the matching `down.sql` files, which still rely on
Postgres's *implicit* multi-statement wrapping (Section 0). The end result is identical
today because every statement used is transaction-safe either way — but it means the
`up`/`down` pairs are no longer written to the same convention, and a `down.sql` file
that someday needs a transaction-unsafe statement won't get the same explicit-boundary
treatment its `up.sql` sibling would.

### Statements that must NOT go in a migration file

Because the whole file runs inside one transaction (implicit or explicit), avoid
statements Postgres refuses to run transactionally at all:

- `CREATE INDEX CONCURRENTLY` / `DROP INDEX CONCURRENTLY`
- `ALTER TYPE ... ADD VALUE` combined with using that new value in the same file
- `VACUUM`, `CREATE DATABASE`, `CREATE TABLESPACE`

If one of these is ever needed, put it alone in its own migration file — wrapping it in
`BEGIN`/`COMMIT` won't help, since Postgres rejects these regardless of transaction mode.

## Section 4 — Failure handling & recovery

**What this does *not* cover:** each migration *file* is its own atomic unit, not the
whole `up`/`down` run. If you have migrations `0005`, `0006`, `0007` and `0007` fails,
`0005` and `0006` stay committed — only `0007`'s own changes roll back. `migrate` records
the schema as "dirty" at version 7 (in the `schema_migrations` table's `dirty` column) and
refuses to run further migrations until it's fixed.

`upMigration`/`downMigration` both call `os.Exit(1)` on failure
([main.go:37](../cmd/migration/main.go#L37), [main.go:54](../cmd/migration/main.go#L54)),
so a failed `make migrate-up` is visible to any script or CI step calling it — the
database itself was already left in a safe, non-half-applied state by Postgres; this exit
code just stops tooling from treating that as success.

**If a migration fails partway through a multi-file `up` run:**

`migrate` reports the schema as dirty at the failed version. Fix the root cause in the
offending file, then either:

- Re-run `make migrate-up` after manually reverting whatever *did* commit from earlier
  files in that same run, or
- Run `migrate -path db/migrations -database "$DB_SOURCE" force <version>` directly
  (bypassing `cmd/migration/main.go`, since its `down` subcommand only ever forces to `1`
  — see Section 2) once the DB state is confirmed correct by hand, then re-run.

`force` does **not** run any SQL — it only rewrites `migrate`'s internal version
bookkeeping row, so only use it after verifying the actual schema yourself.

## Cross-feature coupling

`cmd/migration` and `cmd/server` both read `DB_SOURCE`/`DB_DRIVER` through the same
`internal/config.LoadConfig`. There's no runtime coupling beyond that — migrations run as
a fully separate process before/independent of the server — but a bad `app.env` value
breaks both the same way, so config-loading failures diagnosed on one side apply to the
other too.

## Data Flow Summary

**`make migrate-up` happy path:**

1. `Makefile` → `go run ./cmd/migration/main.go up`.
2. `main()` loads config, calls `upMigration(cfg.DBSource)` ([main.go:77-82](../cmd/migration/main.go#L77-L82)).
3. `upMigration` shells out to `migrate -path db/migrations -database <DB_SOURCE> up`.
4. `migrate` reads `schema_migrations.version`, then for each pending `*.up.sql` file in
   order: sends it as one multi-statement message, Postgres commits or rolls back that
   file atomically, `migrate` advances the recorded version on success.
5. Exit code `0`; all files applied.

**Failure path:** step 4 fails on some file N → that file's own changes roll back, files
before N stay committed, `schema_migrations` is marked dirty at N, `main.go` prints the
error and calls `os.Exit(1)` → recover per Section 4.

**`make migrate-down`:** skips the file-by-file logic above entirely — it's a single
`force 1` call that only rewrites the bookkeeping row (Section 2).

## Final Reference Table

| Command | What it actually runs | Purpose | Notes |
|---|---|---|---|
| `make migrate-create name=<x>` | `migrate create -ext sql -dir db/migrations -seq <x>` | Scaffold a new blank `up`/`down` SQL pair | |
| `make migrate-up` | `migrate -path db/migrations -database $DB_SOURCE up` | Apply all pending `*.up.sql` files in order | Exits `1` on failure |
| `make migrate-down` | `migrate -path db/migrations -database $DB_SOURCE force 1` | **Not a rollback** — only rewrites `schema_migrations` to `(version=1, dirty=false)`; runs no SQL | See Section 2 |
