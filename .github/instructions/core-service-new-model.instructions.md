---
description: "Use when adding a new database entity in apps/core-service: a new table/column, a new SQL query, or new multi-step business logic on Storer. Trigger phrases: 'add a new table', 'create a new model', 'add a new entity', 'new migration', 'add a column', 'new query', 'add a Storer method', 'add a *Tx function'. Covers the migration -> query -> sqlc -> mock -> store workflow. Companion to core-service-new-module.instructions.md (HTTP layer) and core-service.instructions.md (general rules)."
---

# Creating a new "model" (data layer) — core-service

A "model" here isn't an ORM class — there is no ORM. It's the chain:
**migration (schema) → SQL query (`internal/db/query/*.sql`) → generated
`sqlc` code → optional `store` business logic (`*Tx`)**. Follow every step in
order; skipping the mock/build step is the most common way this goes wrong
silently.

Read `apps/core-service/AGENTS.md` if it's not already in context — this file
assumes its "Important Constraints" section (money-as-string, lock ordering,
no hand-editing generated files).

## Decision: do you need a migration, a query, or both?

| Situation | Do this |
|---|---|
| Brand-new table, or new/changed column on an existing table | Migration first (below), then query |
| Existing table, need a new way to read/write it (no schema change) | Just a query — skip to step 2 |
| The new query needs to run alongside other writes atomically, or has a business rule beyond "run this SQL" (e.g. "insufficient funds", "must differ") | Also add a `*Tx` function in `internal/db/store/` (step 4) |
| Single query, no atomicity/business-rule need | Handler calls the generated `Storer` method directly — no `*Tx` needed |

## 1. Migration (schema change only)

```bash
make migrate-create name=add_something   # apps/core-service — never hand-number a pair
```

Creates a sequential `NNNNNN_add_something.up.sql` / `.down.sql` pair in
`internal/db/migrations/`. Write plain SQL, wrapped in `BEGIN;`/`COMMIT;`
(match the existing files):

```sql
-- 0000XX_add_something.up.sql
BEGIN;

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS some_field VARCHAR(255);

COMMIT;
```

The `.down.sql` must symmetrically undo it (`DROP COLUMN`, `DROP TABLE`, etc.).
`sqlc.yaml` points its `schema:` at this directory — migrations are the
source of truth for sqlc's generated Go types, not a separate model file.

Apply and verify locally before moving on:

```bash
make db-up            # starts Postgres, applies pending migrations
make migrate-down      # then re-run to confirm the down migration is symmetric
make db-up             # bring schema back to head
```

New tables need real constraints, not app-level trust — e.g. the existing
`accounts` table enforces non-negative balance with a DB check constraint
(migration `000002_add-balance-constraints`); a `pq.Error` with code
`check_violation` is how the store layer later detects a business-rule
violation (see `errors.go` mapping pattern in step 4).

## 2. Query — write the SQL

Add to the relevant existing file in `internal/db/query/*.sql` (one file
roughly per table/feature — `account.sql`, `entries.sql`, `transfers.sql`,
`user.sql`), or create a new file for a genuinely new table. Every query is a
`-- name: X :one|:many|:exec` comment immediately above plain SQL:

```sql
-- name: GetSomethingById :one
SELECT *
FROM somethings
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateSomething :one
INSERT INTO somethings (
    field_a, field_b
) VALUES (
    $1, $2
)
RETURNING *;
```

Prefer `SELECT *` / `RETURNING *` and `sqlc.embed(...)` to reuse an existing
generated row type for a joined shape, rather than hand-writing a mapper —
there is no `db/mapper` package (see `docs/SQLC_ROW_MAPPING.md`). For a
paginated list query, always add a paired `*Count` query (see
`GetAccountsCount`-style queries) — handlers need it for `core.Meta.Total`.

**Money columns**: always the SQL type backing a `string` in Go (`sqlc` maps
`numeric`/`decimal` columns to `string` here). Never introduce a column that
would generate a `float64` field.

## 3. Regenerate — always both, always in this order

```bash
make sqlc            # regenerates internal/db/sqlc/*.sql.go from the query + schema
make generate-mock    # regenerates internal/db/mock/querier.go from the now-changed Storer/Querier interface
go build ./...        # catches every call site the signature change broke
```

Never hand-edit anything under `internal/db/sqlc/` or `internal/db/mock/` —
both carry a `// Code generated ... DO NOT EDIT.` header. If `make sqlc`
produces an unexpected diff, the query/schema is wrong, not the generator.

## 4. Business logic — only if you need a `*Tx`

Skip this section entirely if step 2's generated method is all a handler will
ever need to call directly.

Add a new `store_<name>_tx.go` in `internal/db/store/` following the shape
every existing one uses (`store_account_create_tx.go`,
`store_account_deposit_tx.go`, `store_transaction.go`):

```go
// internal/db/store/store_something_tx.go
package store

import (
	"context"

	sqlc "github.com/DewaSRY/core-service/internal/db/sqlc"
)

type SomethingTxParams struct {
	// ...fields the business logic needs
}

// SomethingTx opens a transaction; execTx handles commit/rollback.
func (store *_store) SomethingTx(ctx context.Context, arg SomethingTxParams) (sqlc.Something, error) {
	var result sqlc.Something

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = somethingTx(ctx, q, arg)
		return err
	})

	return result, err
}

// somethingTx is parameterized on the sqlc.Querier interface (never the
// concrete *sqlc.Queries) so it's unit-testable against a mock.
func somethingTx(ctx context.Context, q sqlc.Querier, arg SomethingTxParams) (sqlc.Something, error) {
	// business rules go here (validate, then call q.SomeGeneratedMethod(...))
	return q.SomeGeneratedMethod(ctx, sqlc.SomeGeneratedParams{ /* ... */ })
}
```

Then add the public method to the `Storer` interface in `store.go`:

```go
type Storer interface {
	db.Querier
	// ...existing methods...
	SomethingTx(ctx context.Context, arg SomethingTxParams) (db.Something, error)
}
```

**If the transaction touches more than one account row**, lock every account
with `GetAccountByIdForUpdate` in **ascending-ID order**, regardless of
logical from/to roles — copy the pattern in `transferTx`/`deleteAccountTx`,
never reorder or lock in request order. This is a correctness property
(deadlock prevention under concurrent load), not a style choice.

**New failure mode the caller must distinguish?** Add a sentinel to
`internal/db/store/errors.go`:

```go
var ErrSomethingInvalid = errors.New("something is invalid because ...")
```

Plain `errors.New`, matched later with `errors.Is` in the calling domain's
`apperror.go` (see `core-service-new-module.instructions.md`) — don't invent
a custom error type.

## 5. Verify

```bash
go test ./internal/db/store/...     # if you added/changed a *Tx function
go test ./internal/db/sqlc/...      # make db-up first — integration tests against real Postgres
go build ./...
```

Handler tests mock the *entire* `Storer` call, so they will **not** catch a
regression inside a `*Tx` function (wrong lock order, wrong call sequence) —
don't rely on them as coverage for step 4's logic.

## Hard rules (see `core-service.instructions.md` for the full list)

- Money stays `string` end-to-end — parse with `decimal.NewFromString` only
  at the point of arithmetic/comparison.
- Never hand-edit `internal/db/sqlc/*.sql.go` or `internal/db/mock/querier.go`.
- `make sqlc` then `make generate-mock`, always in that order.
- Multi-account transactions lock in ascending-ID order, no exceptions.
