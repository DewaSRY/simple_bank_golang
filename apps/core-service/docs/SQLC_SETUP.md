# Why sqlc — Database Access Layer

## The decision

This service talks to Postgres through [sqlc](https://sqlc.dev), not an ORM (GORM, ent) and not a hand-rolled `database/sql` + manual `Scan()` layer. sqlc reads your migrations and your `.sql` query files and generates typed Go — structs for rows, params structs for inputs, and a method per query. You write SQL; sqlc writes the boilerplate.

## Why not the alternatives

**Why not a full ORM.** An ORM buys you a query builder and (usually) a struct-tag-driven mapping layer, at the cost of an abstraction that fights you the moment a query gets non-trivial — this codebase's transfer logic needs `SELECT ... FOR UPDATE` with a specific row-locking order (see `GetAccountByIdForUpdate` in [db/query/account.sql](../db/query/account.sql) and the comment in [db/store/store_transaction.go](../db/store/store_transaction.go)) to avoid deadlocking concurrent transfers between the same two accounts. That's exactly the kind of query an ORM's builder makes awkward to express and easy to get subtly wrong. With sqlc you just write the SQL.

**Why not raw `database/sql`.** Raw `database/sql` gives you full control but every query becomes: write the SQL as a string, write a struct, hand-write `rows.Scan(&a, &b, &c)` in the same order as the `SELECT` list, and hope nobody reorders one without the other. That mismatch is a silent runtime bug, not a compile error. sqlc removes the hand-written half of that pair — the SQL stays the single source of truth, and the Go side is generated from it.

**The trade-off you're accepting.** sqlc does not abstract away Postgres — a query written for Postgres's `FOR UPDATE` and `sqlc.arg()` syntax doesn't portably move to MySQL or SQLite. That's fine here: this project has already committed to Postgres (`github.com/lib/pq`, migrations under [db/migrations](../db/migrations)), so there's no portability to give up. If a project genuinely needs to be database-agnostic, sqlc is the wrong tool.

## How it fits this codebase

```
db/query/*.sql          <- hand-written SQL, source of truth
db/migrations/*.sql      <- schema, also source of truth
        │
        │  sqlc generate
        ▼
db/sqlc/*.sql.go         <- generated: Queries methods, Params/Row structs
db/sqlc/querier.go        <- generated: Querier interface (every query as a method)
db/sqlc/models.go         <- generated: one struct per table
```

Three points worth understanding, in order of how often you'll touch them:

1. **`Querier` is the seam the whole codebase is built around.** sqlc's `emit_interface: true` (see [sqlc.yaml](../sqlc.yaml)) generates a `Querier` interface listing every query method. [db/store/store.go](../db/store/store.go)'s `Store` embeds `*sqlc.Queries` (the concrete implementation), but business logic that needs to be tested — like `transferTx` in [db/store/store_transaction.go](../db/store/store_transaction.go) — is written against the `Querier` interface, not the concrete `*Queries` type. That's what lets [go.uber.org/mock](GOMOCK_TESTING.md) generate a mock for it and test transfer logic (same-account rejection, insufficient funds, currency mismatch, lock ordering) without a database at all.

2. **Transactions compose `Queries` against a `*sql.Tx` instead of a `*sql.DB`.** sqlc generates `New(db DBTX) *Queries` where `DBTX` is satisfied by both `*sql.DB` and `*sql.Tx`. `Store.execTx` in [db/store/store.go](../db/store/store.go) opens a `sql.Tx`, wraps it in a fresh `sqlc.New(tx)`, and runs the caller's function against that — so a single sqlc-generated type works identically inside and outside a transaction, with no separate "transactional query" code path to maintain.

3. **`:one` / `:many` / `:exec` query annotations decide the generated method shape.** Every query in [db/query/*.sql](../db/query) is written as a `-- name: X :one|:many|:exec` comment. sqlc uses that annotation to decide whether the generated method returns a single struct, a slice, or nothing — so the annotation is the contract, not the SQL body.

Some query-file details worth knowing because they'll surprise you the first time:

- **Row types vs. model types.** A query that doesn't `SELECT *` gets its own `<QueryName>Row` struct (e.g. `GetAccountByIdForUpdateRow`, `IncrementAccountBalanceRow` in [db/sqlc/account.sql.go](../db/sqlc/account.sql.go)) rather than reusing the table's `Account` model — sqlc generates a struct matching exactly the columns you selected. [db/mapper/account.go](../db/mapper/account.go) exists solely to translate one such row type back into the canonical `sqlc.Account` model where the rest of the codebase expects it.
- **`sqlc.arg(name)`.** Plain `$1`-style positional params work fine for one or two arguments, but `ListAccountsByOwner` in [db/query/account.sql](../db/query/account.sql) uses `sqlc.arg(owner)` / `sqlc.arg(limit_count)` instead — this names the generated params struct fields (`LimitCount`, `OffsetCount`) instead of leaving them as an ordinal-indexed struct, which stays readable as a query grows past two or three parameters.
- **Money is `string`, not `float64` or a numeric type.** `balance`/`amount` columns come back as Go `string` (see `CreateAccountParams.Balance`, `Transfer.Amount`). sqlc maps them this way because Postgres `numeric` doesn't round-trip losslessly through `float64`. The codebase then parses that string into a [`shopspring/decimal`](https://github.com/shopspring/decimal) value at the point it needs arithmetic (see `decimal.NewFromString` calls throughout `transferTx`) rather than anywhere in the generated code — decimal math is the caller's job, not sqlc's.

## Day-to-day integration

**Config** ([sqlc.yaml](../sqlc.yaml)):

```yaml
version: "1"
packages:
  - name: "db"
    path: "./db/sqlc"
    queries: "./db/query"
    schema: "./db/migrations"
    engine: "postgresql"
    emit_json_tags: true # generated structs get `json:"..."` tags for free
    emit_prepared_queries: true # queries are prepared statements, not ad-hoc SQL each call
    emit_interface: true # generates the Querier interface — required for mocking
    emit_exact_table_names: false # `accounts` table -> `Account` struct, not `Accounts`
```

**Adding a new query — the actual workflow:**

1. Write the SQL in the right file under [db/query/](../db/query) (`account.sql`, `entries.sql`, `transfers.sql`, `user.sql` — one file per table), with a `-- name: MethodName :one|:many|:exec` comment above it.
2. Run `make sqlc` (wraps `sqlc generate`). This regenerates everything under [db/sqlc/](../db/sqlc) — never hand-edit those files, the `// Code generated by sqlc. DO NOT EDIT.` header is not a suggestion.
3. Regenerate the mock so it picks up the new `Querier` method: `make generate-mock`.
4. Call the new method through `server.store.<MethodName>(ctx, params)` from an API handler, or through the `Querier` parameter if it's part of a multi-step transaction.

**Schema changes:** migrations live under [db/migrations/](../db/migrations) as numbered `.up.sql`/`.down.sql` pairs (`make migrate-create name=...`, `make migrate-up`, `make migrate-down` — see [cmd/migration/main.go](../cmd/migration/main.go)). sqlc reads this same directory as its `schema` source, so a migration and its dependent queries should land in the same PR — sqlc will fail to generate (or generate against a stale schema) if they drift apart.

**Testing without a database.** Because handlers and transaction logic depend on the `Querier` interface rather than `*sqlc.Queries` directly, unit tests substitute a `go.uber.org/mock`-generated `MockQuerier` and never touch Postgres. See [GOMOCK_TESTING.md](GOMOCK_TESTING.md) for how that's wired up, and [db/sqlc/main_test.go](../db/sqlc/main_test.go) / [db/store/main_test.go](../db/store/main_test.go) for the separate integration-test path that does hit a real database over a rolled-back transaction.
