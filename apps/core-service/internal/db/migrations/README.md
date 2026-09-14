# Database Migrations (golang-migrate)

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) through a thin
wrapper at [`cmd/migration/main.go`](../../cmd/migration/main.go), exposed via `make` targets.

## Commands

| Make target | What it runs | Purpose |
|---|---|---|
| `make migrate-create name=<name>` | `migrate create -ext sql -dir internal/db/migrations -seq <name>` | Scaffolds a new `NNNNNN_<name>.up.sql` / `.down.sql` pair |
| `make migrate-up` | `migrate -path internal/db/migrations -database $DB_SOURCE up` | Applies all pending migrations |
| `make migrate-down` | `migrate -path internal/db/migrations -database $DB_SOURCE down 1` | Rolls back the single most recent migration |
| `make migrate-force version=<n>` | `migrate ... force <n>` | Forces the version pointer without running any SQL |
| `make migrate-goto version=<n>` | `migrate ... goto <n>` | Migrates directly up/down to a specific version |

`$DB_SOURCE` comes from `app.env` / `internal/config`, e.g.:
```
DB_SOURCE=postgres://simple_bank:password@localhost:5433/simple_bank?sslmode=disable
```

## File naming

Files are sequential, not timestamped: `000010_create-account_entries_table_view.up.sql` /
`.down.sql`. `-seq` in `migrate create` is what generates the zero-padded number. Never rename
or renumber an existing pair once it has been applied anywhere (including by teammates) — the
number *is* the migration's identity in `schema_migrations`.

## How golang-migrate tracks state

It keeps exactly one row in a `schema_migrations` table:

```sql
SELECT * FROM schema_migrations;
--  version | dirty
-- ---------+-------
--       10 | f
```

- `version` — the last migration file number that was (attempted to be) applied.
- `dirty` — `true` means migrate started applying/reverting that version's SQL and cannot
  confirm it finished cleanly (statement failed, connection dropped mid-run, process killed,
  etc).

**While `dirty = true`, migrate refuses every command** (`up`, `down`, `goto`, even reruns of
the same direction) with:

```
error: Dirty database version 10. Fix and force version.
```

This is a pre-flight guard, not a description of what you just tried to do — you'll see this
exact message whether you ran `up` or `down`.

## Fixing a dirty state

1. **Check reality first** — look at the DB to see if the dirty version's SQL actually applied
   (e.g. `\dv` for a view, `\d tablename` for a table/column change). Postgres runs each
   migration file in its own transaction (assuming you wrap the file in `BEGIN; ... COMMIT;`,
   as all files in this repo do), so it's almost always all-or-nothing — but `dirty` gets set
   optimistically before migrate can confirm the commit succeeded, so it doesn't always mean
   the change is actually half-applied.
2. **Tell migrate the true state with `force`** — it does *not* run any migration SQL, it only
   rewrites the `schema_migrations` row:
   - If the dirty version's changes **are** present in the DB:
     `make migrate-force version=10` → sets `version=10, dirty=false`.
   - If the dirty version's changes are **not** present (it failed before doing anything):
     `make migrate-force version=9` → rewinds the pointer to the last known-good version.
3. Retry your original `migrate-up` / `migrate-down`.

There is no separate "mark as applied" command — `force` is used for both "clear a dirty flag"
and "manually align the version pointer to match reality" (e.g. after applying SQL by hand
outside of migrate).

## Gotchas / good habits

- **`down` only rolls back one step at a time** (`down 1` in this wrapper), not the whole
  history — call it repeatedly (or use `goto`) to go further back.
- **Every `.up.sql`/`.down.sql` should be wrapped in `BEGIN; ... COMMIT;`** so a failing
  statement rolls back cleanly instead of leaving partial DDL applied — Postgres supports
  transactional DDL, so this is cheap insurance.
- **Never edit a migration file that has already been applied** anywhere (your own DB, a
  teammate's, staging). `schema_migrations` only tracks the version number, not file content —
  migrate won't detect that the file changed, so already-applied environments will silently
  diverge from freshly-migrated ones. Write a new migration instead.
- **Always write and test the matching `.down.sql`** before considering a migration done —
  it's easy to write an up-only migration that "works" in dev and only discover the down script
  is broken/missing when someone actually needs to roll back.
- `goto <n>` will run up or down migrations as needed to reach version `n` directly, which is
  handy for jumping several steps instead of calling `down` repeatedly.
- `force` never touches your schema — it's purely bookkeeping. If you need to actually undo a
  bad migration's SQL, you still need a correct `down` script (or a manual fix) in addition to
  `force`.
