# Migrations: transaction & rollback behavior

This project runs migrations with the [golang-migrate](https://github.com/golang-migrate/migrate)
`migrate` CLI against Postgres (`make migrate-up` / `make migrate-down`, see
[cmd/migration/main.go](../../cmd/migration/main.go)).

## Each migration file already runs in its own transaction

The Postgres driver executes every `*.up.sql` / `*.down.sql` file inside a single
database transaction: `BEGIN` → run the file's statements → `COMMIT`, or
`ROLLBACK` if any statement fails. This is built into the driver, not something
we configure — Postgres supports transactional DDL, so `CREATE TABLE`,
`ALTER TABLE`, `ADD CONSTRAINT`, etc. all roll back cleanly together if a later
statement in the same file errors.

Concretely: if `000004_authorization_by_user_tabel.up.sql` fails on its second
`ALTER TABLE`, the first `ALTER TABLE` in that same file is rolled back too —
you never end up with a half-applied file.

**What this does *not* cover:** each migration *file* is its own transaction,
not the whole `up`/`down` run. If you have migrations `0005`, `0006`, `0007`
and `0007` fails, `0005` and `0006` stay committed — only `0007`'s own changes
roll back. `migrate` records the schema is now "dirty" at version 7 and refuses
to run further migrations until it's fixed (see below).

## Statements that must NOT go in a migration file

Because every file runs inside a transaction, avoid statements Postgres refuses
to run transactionally, e.g.:

- `CREATE INDEX CONCURRENTLY` / `DROP INDEX CONCURRENTLY`
- `ALTER TYPE ... ADD VALUE` combined with using that new value in the same
  transaction/file
- `VACUUM`, `CREATE DATABASE`, `CREATE TABLESPACE`

If one of these is ever needed, put it alone in its own migration file so it
doesn't block the transactional guarantee for everything else.

## Failure handling in `cmd/migration/main.go`

Previously `upMigration`/`downMigration` printed errors but still exited `0`,
so a failed `make migrate-up` looked successful to any script or CI step
calling it. Both now call `os.Exit(1)` on failure so the failure is visible to
the caller — the DB itself was already safe (rolled back by the driver), this
just makes sure tooling around it doesn't silently continue.

## If a migration fails partway through a multi-file run

`migrate` will report the schema as "dirty" at the failed version. Fix the
root cause in the offending file, then either:

- Re-run `make migrate-up` after manually reverting whatever *did* commit from
  earlier files in that same run, or
- Use `migrate force <version>` (see `downMigration` in
  `cmd/migration/main.go`) to reset the recorded version once the DB state is
  confirmed correct, then re-run.

`force` does **not** run any SQL — it only rewrites migrate's internal version
bookkeeping, so only use it after verifying the actual schema by hand.
