# Database Migrations — As Implemented

## Who this doc is for

You're comfortable with Go and SQL but haven't necessarily used
[golang-migrate](https://github.com/golang-migrate/migrate) before. Section 0 is a short
primer on the one non-obvious piece of Postgres behavior this project leans on — skip it
if you already know how Postgres handles multi-statement queries.

This doc is verified against the current source (`cmd/migration/main.go`,
`internal/db/migrations/*.sql`, `Makefile`, `.github/workflows/core-service-ci.yml`), not the
intended design. **It replaces an earlier version of this file that incorrectly claimed
`make migrate-down` only forces the version pointer without running any SQL — that claim no
longer matches the code** (and, per `git log`, matches an abandoned experiment that was never
actually shipped). Read Section 2 before assuming anything about what a subcommand does.

## Section 0 — Background primer: how a multi-statement file becomes "atomic"

`migrate` is used here as an **external CLI binary**, not a vendored Go dependency — it
doesn't appear in `go.mod`. `cmd/migration/main.go` shells out to it.

The atomicity this project relies on doesn't come from golang-migrate wrapping your file in
`BEGIN`/`COMMIT` for you. Looking at the driver it shells out to
([`database/postgres/postgres.go`](https://github.com/golang-migrate/migrate/blob/master/database/postgres/postgres.go)),
`Run()` reads the entire migration file into one string and executes it as a single,
argument-less `ExecContext` call — no transaction handling in that function or in
`runStatement`.

What actually makes the file atomic is Postgres itself: `ExecContext` with no query
arguments makes `lib/pq` send the whole file as one **simple-query** protocol message. When a
simple-query message contains several `;`-separated statements, Postgres implicitly wraps all
of them in one transaction — *unless* the message itself contains explicit `BEGIN`/`COMMIT`,
in which case those explicit commands take over. Either way, the net effect is the same: all
statements in one file commit or roll back together.

**Gotchas that surprise newcomers:**

1. The explicit `BEGIN; … COMMIT;` you'll see in most of this repo's `*.sql` files isn't
   required for atomicity — Postgres already gives you that implicitly for a multi-statement
   message. It was added deliberately anyway (commit `52f01b7`, "Update migration file to run
   on transaction") for explicitness. This is why Section 3 flags an inconsistency: the
   earliest migrations (`000001`-`000004`) only ever got this treatment on their `up.sql`
   half — their `down.sql` files still rely on the implicit behavior.
2. Because this is Postgres's own multi-statement rule and not something golang-migrate
   configures, statements that categorically cannot run inside *any* transaction
   (`CREATE INDEX CONCURRENTLY`, etc.) will fail no matter what you do — see Section 3.
3. `migrate`'s CLI has a real `down` command that runs the previous `*.down.sql` file(s), and
   this project's `down` subcommand really does call it — one step at a time. See Section 2 if
   you came from the earlier (incorrect) version of this doc expecting otherwise.

## Section 1 — Architecture at a Glance

The composition root is [`cmd/migration/main.go`](../cmd/migration/main.go) (`main`,
[main.go:94-158](../cmd/migration/main.go#L94-L158)). It does not implement migration logic
itself — it dispatches CLI args to config loading and then shells out to the external
`migrate` binary.

| Concern | Owner | Analogy |
|---|---|---|
| CLI entrypoint & dispatch (5 subcommands: `create`, `up`, `down`, `force`, `goto`) | `cmd/migration/main.go` (`main`, [main.go:103-156](../cmd/migration/main.go#L103-L156)) | A switchboard operator — routes the call, does no work itself |
| DB connection string | `internal/config.LoadConfig`, shared with `cmd/server` | A phone book both the migration tool and the server look up |
| Migration execution engine | External `migrate` CLI binary — not a Go dependency | A contractor hired to do the actual work |
| Migration SQL & transaction boundaries | `internal/db/migrations/*.sql` (10 pairs, `000001`-`000010`) | The work orders handed to the contractor |
| Version bookkeeping | `schema_migrations` table, managed entirely by `migrate` | The contractor's own logbook — this project never reads or writes it directly |
| Human-facing recovery instructions | [`internal/db/migrations/README.md`](../internal/db/migrations/README.md) | The contractor's own quick-reference card |

It's split this way so the project never has to reimplement version tracking or SQL
execution — `migrate` already does both. The cost of that thinness shows up in Section 2:
because `main.go` is a thin dispatcher, two of its five subcommands shell out through `bash -c`
string interpolation while the other three call `migrate` directly with argv — an
inconsistency worth knowing about before you add a sixth subcommand.

## Section 2 — `cmd/migration/main.go`: the CLI wrapper

**The problem this solves:** give the project a `go run`/`make` interface for `migrate`'s
`create` / `up` / `down` / `force` / `goto` subcommands that pulls `DB_SOURCE` from this
project's own config loader, and make failures visible to scripts/CI (a failed run must not
exit `0`).

**How each subcommand is actually implemented:**

| Subcommand | What it actually runs | Invocation style | Matches expectation? |
|---|---|---|---|
| `create <name>` | `migrate create -ext sql -dir internal/db/migrations -seq <name>` ([main.go:15](../cmd/migration/main.go#L15)) | `bash -c "<interpolated string>"` | Yes — scaffolds a blank `up`/`down` SQL pair |
| `up` | `migrate -path internal/db/migrations -database <DB_SOURCE> up` ([main.go:28](../cmd/migration/main.go#L28)) | `bash -c "<interpolated string>"` | Yes — runs all pending `*.up.sql` files in order |
| `down` | `migrate -path internal/db/migrations -database <DB_SOURCE> down 1` ([main.go:45-50](../cmd/migration/main.go#L45-L50)) | direct argv (no shell) | Yes — rolls back exactly one step |
| `force <version>` | `migrate -path internal/db/migrations -database <DB_SOURCE> force <version>` ([main.go:67](../cmd/migration/main.go#L67)) | direct argv (no shell) | Yes — rewrites bookkeeping only, runs no SQL |
| `goto <version>` | `migrate -path internal/db/migrations -database <DB_SOURCE> goto <version>` ([main.go:81](../cmd/migration/main.go#L81)) | direct argv (no shell) | Yes — migrates directly to the given version |

### Worth flagging: `create` and `up` build a shell string, the rest don't

`createMigrationFile` and `upMigration` both hand a **string** built with `fmt.Sprintf` to
`exec.Command("bash", "-c", ...)` ([main.go:15](../cmd/migration/main.go#L15),
[main.go:28](../cmd/migration/main.go#L28)) — the shell re-parses that string, so any
shell-metacharacter in `migrationName` (from `make migrate-create name=...`) would be
interpreted by `bash`, not passed through literally. `downMigration`, `forceMigration`, and
`migrateGotoversion` instead call `exec.Command("migrate", arg1, arg2, ...)` directly with a
real argv slice — no shell involved, no interpolation risk.

This isn't exploitable by anyone but whoever is already running the command on their own
machine (there's no remote/multi-tenant use of this CLI), so it's a latent inconsistency
rather than a live vulnerability — but it means `create`/`up` and `down`/`force`/`goto` don't
follow the same pattern, and a future subcommand copy-pasted from the wrong sibling will
silently inherit whichever style it copied from.

**Other rough edges in this file:**

- The failure-path comment on [main.go:33-35](../cmd/migration/main.go#L33-L35) points readers
  to `internal/db/migrations/README.md` for more detail. That file now exists (it didn't for a
  while) — see the link in Section 1.
- `forceMigration` and `migrateGotoversion` both require an explicit numeric `<version>` arg
  ([main.go:124-137](../cmd/migration/main.go#L124-L137),
  [main.go:140-153](../cmd/migration/main.go#L140-L153)) parsed with `strconv.Atoi` — a
  non-numeric arg is caught with a clean message rather than crashing, but there's no
  validation that the version number actually corresponds to a real migration file; `migrate`
  itself is what rejects an out-of-range version.

## Section 3 — Migration files & the transaction boundary (`internal/db/migrations/*.sql`)

**The problem this solves:** a mid-file failure (the second `ALTER TABLE` in a file errors,
say) must not leave the first `ALTER TABLE` in that same file committed.

**How it's actually implemented:** as explained in Section 0, every file is executed as one
multi-statement message, so Postgres either commits or rolls back the whole file as a unit.

| File | `up.sql` wraps in explicit `BEGIN`/`COMMIT`? | `down.sql` wraps in explicit `BEGIN`/`COMMIT`? |
|---|---|---|
| `000001_init-transaction-feature` | Yes | No |
| `000002_add-balance-constraints` | Yes | No |
| `000003_add-users-table` | Yes | No |
| `000004_authorization_by_user_tabel` | Yes | No |
| `000005_update-account` | Yes | Yes |
| `000006_account_lifecycle` | Yes | Yes |
| `000007_account_number_unique` | Yes | Yes |
| `000008_transfer_and_entry_metadata` | Yes | Yes |
| `000009_account_view_table` | Yes | Yes |
| `000010_create-account_entries_table_view` | Yes | Yes |

**Worth flagging:** commit `52f01b7` added explicit `BEGIN;`/`COMMIT;` to the `up.sql` files
of `000001`-`000004` but never touched their matching `down.sql` files. Every migration from
`000005` onward was written with both halves wrapped consistently — so this is a
fully-inherited inconsistency from the project's earliest four migrations, not an ongoing
pattern. The net effect on `000001`-`000004`'s `down.sql` files is identical today because
every statement they use is transaction-safe either way (relying on Postgres's *implicit*
wrapping from Section 0) — but a future edit to one of those four `down.sql` files that adds a
transaction-unsafe statement won't get the same explicit-boundary protection its `up.sql`
sibling has.

### Statements that must NOT go in a migration file

Because the whole file runs inside one transaction (implicit or explicit), avoid statements
Postgres refuses to run transactionally at all:

- `CREATE INDEX CONCURRENTLY` / `DROP INDEX CONCURRENTLY`
- `ALTER TYPE ... ADD VALUE` combined with using that new value in the same file
- `VACUUM`, `CREATE DATABASE`, `CREATE TABLESPACE`

If one of these is ever needed, put it alone in its own migration file — wrapping it in
`BEGIN`/`COMMIT` won't help, since Postgres rejects these regardless of transaction mode.

## Section 4 — Failure handling & recovery

**What this does *not* cover:** each migration *file* is its own atomic unit, not the whole
`up`/`down` run. If migrations `0008`, `0009`, `0010` are pending and `0010` fails, `0008` and
`0009` stay committed — only `0010`'s own changes roll back. `migrate` records the schema as
**dirty** at version 10 (the `dirty` column in the `schema_migrations` table) and refuses to
run *any* further command — `up`, `down`, or `goto` — until it's resolved.

`upMigration`/`downMigration`/`forceMigration`/`migrateGotoversion` all call `os.Exit(1)` on
failure ([main.go:38](../cmd/migration/main.go#L38),
[main.go:60](../cmd/migration/main.go#L60), [main.go:74](../cmd/migration/main.go#L74),
[main.go:88](../cmd/migration/main.go#L88)), so a failed `make migrate-*` call is visible to
any script or CI step calling it — the database itself was already left in a safe,
non-half-applied state by Postgres; this exit code just stops tooling from treating that as
success.

**A concrete worked example (observed on a real dev database during this project):**

```
$ go run ./cmd/migration/main.go down
Running migration down...
error: Dirty database version 10. Fix and force version.

Migration down failed.
Error: exit status 1
```

```sql
SELECT * FROM schema_migrations;
--  version | dirty
-- ---------+-------
--       10 | t
```

This message appears regardless of which command you actually ran (`down` here, but `up` or
`goto` would produce the identical dirty-check error) — it's a pre-flight guard, not a
description of what just failed. The fix:

1. **Check reality first.** Inspect the DB directly (`\dv`, `\d <table>`, etc.) to see whether
   version 10's changes are actually present. In this case `account_entries_view` (created by
   `000010`'s `up.sql`) already existed — the migration had, in fact, fully succeeded; only the
   bookkeeping row was left dirty.
2. **Tell `migrate` the true state with `force`** — it runs **zero SQL**, it only rewrites the
   `schema_migrations` row:
   - Changes present → `make migrate-force version=10` (clears the dirty flag, keeps the
     version).
   - Changes absent → `make migrate-force version=9` (rewinds the pointer to the last
     known-good version).
3. Retry the original command.

## Cross-feature coupling

- `cmd/migration` and `cmd/server` both read `DB_SOURCE`/`DB_DRIVER` through the same
  `internal/config.LoadConfig`. There's no runtime coupling beyond that — migrations run as a
  fully separate process before/independent of the server — but a bad `app.env` value breaks
  both the same way, so config-loading failures diagnosed on one side apply to the other too.
- **CI bypasses `cmd/migration/main.go` entirely.**
  [`.github/workflows/core-service-ci.yml`](../../../.github/workflows/core-service-ci.yml)
  installs `migrate` v4.17.1 directly from GitHub releases and runs
  `migrate -path internal/db/migrations -database "<hardcoded-local-uri>" up` as a raw shell
  step — it never calls `go run ./cmd/migration/main.go up`, and it only ever runs `up` (never
  `down`/`force`/`goto`). Worth knowing before assuming a local `make migrate-*` run and CI
  behave identically: they use *different `migrate` binary versions* (CI pins `v4.17.1`; a
  local dev machine may have a newer one, e.g. `v4.19.1`), so CLI output/behavior differences
  between those versions would surface differently in CI vs. locally.

## Data Flow Summary

**`make migrate-up` happy path:**

1. `Makefile` → `go run ./cmd/migration/main.go up`.
2. `main()` loads config, calls `upMigration(cfg.DBSource)` ([main.go:112-116](../cmd/migration/main.go#L112-L116)).
3. `upMigration` shells out (via `bash -c`) to
   `migrate -path internal/db/migrations -database <DB_SOURCE> up`.
4. `migrate` reads `schema_migrations.version`, then for each pending `*.up.sql` file in
   order: sends it as one multi-statement message, Postgres commits or rolls back that file
   atomically, `migrate` advances the recorded version on success.
5. Exit code `0`; all files applied.

**Failure path:** step 4 fails on some file N → that file's own changes roll back, files
before N stay committed, `schema_migrations` is marked dirty at N, `main.go` prints the error
and calls `os.Exit(1)` → recover per Section 4.

**`make migrate-down` happy path:**

1. `Makefile` → `go run ./cmd/migration/main.go down`.
2. `main()` loads config, calls `downMigration(cfg.DBSource)` ([main.go:117-122](../cmd/migration/main.go#L117-L122)).
3. `downMigration` calls `migrate -path internal/db/migrations -database <DB_SOURCE> down 1`
   directly (argv, no shell).
4. `migrate` runs the single most recent `*.down.sql` file and decrements the recorded
   version by one on success.
5. Exit code `0`; exactly one migration rolled back. (Call it again, or use `goto`, to roll
   back further — see Section 2.)

## Final Reference Table

| Command | What it actually runs | Purpose | Notes |
|---|---|---|---|
| `make migrate-create name=<x>` | `migrate create -ext sql -dir internal/db/migrations -seq <x>` | Scaffold a new blank `up`/`down` SQL pair | Runs via `bash -c` string interpolation |
| `make migrate-up` | `migrate -path internal/db/migrations -database $DB_SOURCE up` | Apply all pending `*.up.sql` files in order | Exits `1` on failure; also run directly (different binary version, `v4.17.1`) in CI |
| `make migrate-down` | `migrate -path internal/db/migrations -database $DB_SOURCE down 1` | Roll back exactly the single most recent migration | Not a full rollback — call repeatedly or use `goto` for more than one step |
| `make migrate-force version=<n>` | `migrate -path internal/db/migrations -database $DB_SOURCE force <n>` | Rewrite the `schema_migrations` bookkeeping row without running any SQL | Use only after verifying real schema state yourself — see Section 4 |
| `make migrate-goto version=<n>` | `migrate -path internal/db/migrations -database $DB_SOURCE goto <n>` | Migrate directly (up or down) to a specific version | Runs real `*.up.sql`/`*.down.sql` files as needed |
