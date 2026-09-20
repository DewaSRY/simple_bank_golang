# sqlc Database Access Layer — As Implemented

## Who this doc is for

Anyone touching `internal/db/`, `internal/api/`, or adding a new query —
whether you've used sqlc before or never heard of it. It assumes you're
comfortable with Go and basic SQL but not necessarily with sqlc's
code-generation model. This doc is verified line-by-line against the current
source as of commit `13f5fc4` (2026-09-19), not against the original design
intent — where the code does something surprising, that's called out
explicitly rather than smoothed over. Read the callouts before you copy a
pattern from an existing query.

**This is a full rewrite of an earlier revision of this doc.** The previous
revision described a pre-refactor layout (`db/` instead of `internal/db/`, an
`Account.Owner` field that no longer exists, a `db/mapper` package that has
since been deleted, a `Storer` mocked directly off `sqlc.Querier`) and had
drifted badly from the code. Every file path and claim below reflects the
current layout described in [AGENTS.md](../AGENTS.md). For the deep dive on
*why* sqlc mints a new Go struct per query and how this codebase avoids
hand-written mappers, see [SQLC_ROW_MAPPING.md](SQLC_ROW_MAPPING.md) — this
doc gives the current-state summary (Section 5) and defers to that doc for
the query-by-query playbook rather than duplicating it.

---

## Section 0 — Background Primer: why sqlc, not the alternatives

| Approach              | You write                                     | Codegen writes                     | Typical failure mode                                                               |
| --------------------- | ---------------------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------ |
| Full ORM (GORM, ent)  | Struct tags + builder calls                    | Query construction + scanning       | Builder fights you the moment a query gets non-trivial (row locking, custom joins)  |
| Raw `database/sql`    | SQL string + struct + `rows.Scan(&a, &b, &c)`  | Nothing                             | Scan-order/SELECT-list mismatch is a silent runtime bug, not a compile error         |
| sqlc                  | SQL only, in `.sql` files                      | Struct + params + method per query  | None of the above — but you're locked to one SQL dialect                            |

This service talks to Postgres through [sqlc](https://sqlc.dev). sqlc reads
the migrations in [internal/db/migrations](../internal/db/migrations) and the
query files in [internal/db/query](../internal/db/query) and generates typed
Go: one struct per row shape, one params struct per query's inputs, and one
method per query. You write SQL; sqlc writes the boilerplate that would
otherwise be hand-maintained in parallel with it.

**Why not a full ORM.** This codebase's transfer and account-deletion logic
need `SELECT ... FOR UPDATE` with a specific row-locking order
(`GetAccountByIdForUpdate` in
[account-transaction.sql](../internal/db/query/account-transaction.sql), used
from `transferTx` and `deleteAccountTx` in
[internal/db/store](../internal/db/store)) to avoid deadlocking concurrent
transactions between the same two accounts. That's exactly the kind of query
an ORM's builder makes awkward to express and easy to get subtly wrong. With
sqlc you just write the SQL.

**Why not raw `database/sql`.** Every query becomes: write the SQL as a
string, write a struct, hand-write `rows.Scan(...)` in the same order as the
`SELECT` list, and hope nobody reorders one without the other. sqlc removes
the hand-written half of that pair — the SQL stays the single source of
truth, the Go side is generated from it, and a change to one regenerates the
other.

**The trade-off you're accepting.** A query written for Postgres's
`FOR UPDATE`, `sqlc.arg()`/`sqlc.narg()`, and `sqlc.embed()` syntax doesn't
portably move to MySQL or SQLite. That's fine here — this project has already
committed to Postgres (`github.com/lib/pq`, migrations under
[internal/db/migrations](../internal/db/migrations)) — but it's the reason
sqlc would be the wrong tool for a project that genuinely needs to be
database-agnostic.

**Gotchas that surprise newcomers** (each connects forward to a section
below):

1. **The generated mock is built from `Storer`, not from `sqlc.Querier`.** If
   you go looking for a `MockQuerier`, you won't find one — `make
   generate-mock` runs `mockgen` against
   `internal/db/store.Storer` (Section 6, Section 8), so the one generated
   mock (`internal/db/mock/querier.go`'s `MockStorer`) covers both the
   sqlc-generated query methods and the four hand-written `*Tx` methods.
2. **Generated params struct fields are ordered by first appearance in the
   *rendered* SQL, not by your intuition of argument order or the order you
   wrote `sqlc.arg(...)` calls** — this bites the first time you read a
   generated call site (Section 3).
3. **A query that doesn't `SELECT *`/`RETURNING *` (or embed a view via
   `sqlc.embed()`) gets its own one-off `<QueryName>Row` struct** instead of
   reusing a shared model type — this used to be why a `db/mapper` package
   existed; it's gone now (Section 5, and see
   [SQLC_ROW_MAPPING.md](SQLC_ROW_MAPPING.md) for the full playbook).

---

## Section 1 — Architecture at a Glance

The composition root is **[sqlc.yaml](../sqlc.yaml)** plus the `sqlc
generate` invocation it drives (`make sqlc`) — it doesn't implement any query
logic itself, it only wires "these `.sql` files, read against this schema,
produce Go in this package." Business logic (locking order, currency checks,
error sentinels) lives entirely downstream, in hand-written Go that depends
on what sqlc generated.

```
internal/db/query/*.sql       <- hand-written SQL, source of truth
internal/db/migrations/*.sql  <- schema, also source of truth
        │
        │  sqlc generate  (make sqlc)
        ▼
internal/db/sqlc/*.sql.go     <- generated: Queries methods, Params/Row structs
internal/db/sqlc/querier.go   <- generated: Querier interface (every query as a method)
internal/db/sqlc/models.go    <- generated: one struct per table + one per view
        │
        │  internal/db/store embeds *sqlc.Queries directly — no mapper package
        ▼
internal/db/store/*.go        <- hand-written: Storer, execTx, the four *Tx functions
        │
        │  internal/api/<domain> Handlers depend on store.Storer, never a concrete type
        ▼
internal/api/<domain>/*.go    <- handlers call h.Store.<Method>(ctx, params)
```

| Concern                    | Owner (file/package)                                                          | Analogy                                        |
| --------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------ |
| Schema source of truth      | [internal/db/migrations/\*.sql](../internal/db/migrations)                    | The database's own change log                   |
| Query source of truth       | [internal/db/query/\*.sql](../internal/db/query)                              | Hand-written SQL, never generated                |
| Codegen config              | [sqlc.yaml](../sqlc.yaml)                                                      | Build config for the generation step             |
| Generated query methods     | [internal/db/sqlc/\*.sql.go](../internal/db/sqlc)                             | Compiler-generated glue code                     |
| Generated interface seam    | [internal/db/sqlc/querier.go](../internal/db/sqlc/querier.go)                 | An abstract base class with no logic             |
| Generated table/view models | [internal/db/sqlc/models.go](../internal/db/sqlc/models.go)                   | Plain data records                               |
| The one seam every handler depends on | [internal/db/store/store.go](../internal/db/store/store.go) (`Storer`) | An interface that hides "database" behind "store" |
| Transaction composition     | [internal/db/store/store.go](../internal/db/store/store.go) (`execTx`)        | A unit-of-work wrapper                           |
| Multi-step business logic   | `store_transaction.go`, `store_account_create_tx.go`, `store_account_deposit_tx.go`, `store_account.go` (all in [internal/db/store](../internal/db/store)) | The actual "transfer/create/deposit/delete" recipes |
| Mock for the seam           | [internal/db/mock/querier.go](../internal/db/mock/querier.go) (generated)     | A stunt double for the whole store, not just the DB |
| API entry points            | [internal/api/account](../internal/api/account), [internal/api/transfer](../internal/api/transfer), [internal/api/auth](../internal/api/auth) | The front desk that only knows the interface |

One paragraph on why it's split this way: `Storer` (Section 6) is what makes
the `*Tx` business-logic functions testable without a database — each
`fooTx(ctx, q sqlc.Querier, arg)` function is written against the `Querier`
interface, not a concrete `*sqlc.Queries`, so a mock can stand in for it
(Section 8) — while the same generated `Queries` type, built against either a
`*sql.DB` or a `*sql.Tx`, stands in for it in production. That split is also
the source of one of this doc's more important callouts: **`internal/db/store`
has no test files of its own today** — the only coverage the `*Tx` functions
get is indirect, through each domain's handler tests, which mock the entire
`Storer` method and therefore cannot catch a regression inside the `*Tx`
function itself (Section 8).

---

## Section 2 — Schema & Migrations

**The problem this piece solves.** Postgres needs a schema before sqlc can
validate any query against it, and that schema needs to evolve over time
without hand-editing a live database.

**How it's actually implemented.** Migrations live under
[internal/db/migrations/](../internal/db/migrations) as ten numbered
`.up.sql` / `.down.sql` pairs, run through the golang-migrate CLI wrapped by
[cmd/migration/main.go](../cmd/migration/main.go):

| Migration                                          | What it does                                                                                                                                |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `000001_init-transaction-feature`                  | Creates `accounts` (`owner`, `balance NUMERIC(20,2)`, `currency VARCHAR(3)`), `transfers`, `entries`; FKs entries/transfers → accounts; indexes |
| `000002_add-balance-constraints`                   | Idempotently adds `CHECK (balance >= 0)` on accounts, `CHECK (amount > 0)` on transfers                                                       |
| `000003_add-users-table`                           | Creates `users` (unique `username`/`email`, `hashed_password`), index on `email`                                                              |
| `000004_authorization_by_user_tabel`               | Adds nullable `user_id` to `accounts` and `entries`, each with an idempotent FK to `users(id)`                                                |
| `000005_update-account`                            | Drops `accounts.owner`, adds `number`/`name`/`description` — this is the point `Account.Owner` stopped existing                               |
| `000006_account_lifecycle`                         | Adds `is_main BOOLEAN` and `deleted_at TIMESTAMPTZ`; a partial unique index enforces at most one active main account per user                 |
| `000007_account_number_unique`                     | Backfills any NULL/duplicate `number` from the row's own id, then adds a unique index on `accounts.number`                                    |
| `000008_transfer_and_entry_metadata`               | Adds `transfers.description`, `entries.description`, `entries.transfer_id` (+ FK to `transfers`)                                              |
| `000009_account_view_table`                        | Creates `account_user_details_view` (accounts LEFT JOIN users, filtered to `deleted_at IS NULL`)                                              |
| `000010_create-account_entries_table_view`         | Creates `account_entries_view` (entries LEFT JOIN accounts LEFT JOIN transfers LEFT JOIN accounts, resolving a transfer's counterparty)        |

sqlc reads this same directory as its `schema:` source
([sqlc.yaml:7](../sqlc.yaml)), so a migration and the queries that depend on
it should land in the same PR — sqlc generates against whatever schema is on
disk, not what's actually applied to any database. A `CREATE VIEW` is schema
sqlc introspects the same as a table, which is what lets `sqlc.embed()`
(Section 5) work against `account_user_details_view`/`account_entries_view`.

Makefile targets: `make migrate-create name=...`, `make migrate-up`, `make
migrate-down`, `make migrate-force version=<n>`, `make migrate-goto
version=<n>` (Makefile). `make db-up` runs `docker compose up -d postgres`
then `docker compose run --rm migrate` — the one-stop command for a fresh
local database.

**Corrected from an earlier revision of this doc: `make migrate-down` now
does what its name says.** `downMigration` in
[cmd/migration/main.go:44-64](../cmd/migration/main.go) runs `migrate ...
down 1`, which executes the most recent migration's real `.down.sql` — an
earlier revision of this doc, and an earlier version of this function,
described it as forcing the schema-version pointer instead (which is what
`migrate-force` does). If you're relying on a stale memory of that claim,
re-verify against the current function before trusting it.

**Corrected from an earlier revision: the `README.md` this file's comments
point to now exists.** `upMigration`'s comment
([cmd/migration/main.go:34](../cmd/migration/main.go)) references
`internal/db/migrations/README.md` — that file is present today (92 lines,
documenting the make targets, file naming, and `schema_migrations` state), not
a dead reference.

---

## Section 3 — Query Files (`internal/db/query/*.sql`)

**The problem this piece solves.** SQL needs to live somewhere sqlc can read
it, one query per intended Go method, with enough annotation for sqlc to know
what shape to generate.

**How it's actually implemented.** One file roughly per table/feature under
[internal/db/query/](../internal/db/query). Every query is a `-- name:
MethodName :one|:many|:exec` comment directly above a SQL statement — that
annotation, not the SQL body, is the contract deciding whether the generated
method returns a single struct, a slice, or nothing.

| File                                                                  | Query                              | Annotation | Notable detail                                                                 |
| ------------------------------------------------------------------------ | ------------------------------------ | ------------ | --------------------------------------------------------------------------------- |
| [account.sql](../internal/db/query/account.sql)                       | `CreateAccount`                    | `:one`     | `RETURNING *`                                                                  |
|                                                                        | `UpdateAccountNumber`               | `:one`     | `RETURNING *` — second half of the two-step account-number generation (Section 7 / Legacy notes) |
|                                                                        | `UpdateAccount`                     | `:one`     | `RETURNING *`, filtered to `deleted_at IS NULL`                                |
|                                                                        | `SoftDeleteAccount`                 | `:one`     | `RETURNING *` — sets `deleted_at`, not a hard delete                           |
|                                                                        | `GetAccountById`                    | `:one`     | `SELECT *`, filtered to `deleted_at IS NULL`                                   |
|                                                                        | `GetMainAccountByUserId`            | `:one`     | `SELECT *` where `is_main = true`                                              |
|                                                                        | `ListAccountsSearchByUserNumber`    | `:many`    | `sqlc.embed(v)` on `account_user_details_view`                                |
|                                                                        | `CountAccountsSearchByUserNumber`   | `:one`     | Plain `$1`, paired with the above                                              |
| [account-manage.sql](../internal/db/query/account-manage.sql)         | `ListMeAccountsByUserId`            | `:many`    | `sqlc.embed(v)` on `account_user_details_view`; `sqlc.arg(...)` x4             |
|                                                                        | `ListMeAccountsByUserIdCount`       | `:one`     | Paired with the above                                                          |
|                                                                        | `CountAccountsByUserId`             | `:one`     | Plain accounts-table count, `deleted_at IS NULL`                               |
| [account-transaction.sql](../internal/db/query/account-transaction.sql) | `ListRecentTransferDestinations`   | `:many`    | 3-way join (`transfers`/`accounts`/`users`), `DISTINCT ON` subquery — a genuinely one-off shape |
|                                                                        | `GetAccountViewById`                | `:one`     | `sqlc.embed(v)` on `account_user_details_view`                                |
|                                                                        | `GetAccountByIdForUpdate`           | `:one`     | `SELECT *` + `FOR UPDATE` — pessimistic row lock, used by `transferTx`/`deleteAccountTx`/`depositTx` |
|                                                                        | `IncrementAccountBalance`           | `:one`     | `SET balance = balance + $2`, `RETURNING *` — see rough edge below             |
|                                                                        | `CheckIsAccountWithIdExist`         | `:one`     | `SELECT EXISTS(...)`                                                          |
|                                                                        | `ListAccountEntriesByAccountId`     | `:many`    | `sqlc.embed(v)` on `account_entries_view`; `sqlc.narg(...)` for optional period/type filters |
|                                                                        | `CountAccountEntriesByAccountId`    | `:one`     | Paired with the above, same optional filters                                  |
|                                                                        | `AccountEntriesByAccountId`         | `:one`     | `sqlc.embed(v)` on `account_entries_view`, by entry id — used to re-read the entry a transfer/deposit just wrote |
| [entries.sql](../internal/db/query/entries.sql)                       | `CreateEntries`                     | `:one`     | `RETURNING *`                                                                  |
|                                                                        | `ListAccountTransactionHistory`     | `:many`    | `LEFT JOIN transfers`/`accounts` to resolve a counterparty — a genuinely one-off shape |
|                                                                        | `CountAccountTransactionHistory`    | `:one`     | Paired with the above                                                          |
| [transfers.sql](../internal/db/query/transfers.sql)                   | `CreateTransfer`                    | `:one`     | Exhaustive in-order column list (not `*`), still reuses `Transfer` (Section 5) |
|                                                                        | `GetTransferById`                   | `:one`     | Same exhaustive-column-list trick                                              |
| [user.sql](../internal/db/query/user.sql)                             | `CreateUser`                        | `:one`     | Explicit column list, deliberately excludes `hashed_password`                 |
|                                                                        | `GetUserByEmail`                    | `:one`     | Includes `hashed_password` (used for login)                                   |
|                                                                        | `GetUserById`                       | `:one`     | Excludes `hashed_password` (used for profile reads)                           |
|                                                                        | `CheckIsUsernameExist`              | `:one`     | `SELECT EXISTS(...)`                                                          |

**`sqlc.arg(name)` / `sqlc.narg(name)`.** Plain `$1`-style positional params
work fine for one or two arguments, but any query with three or more (every
`List*`/`Count*` pagination query above) uses `sqlc.arg(...)` instead — this
names the generated params struct fields rather than leaving them ordinal.
`sqlc.narg(...)` is the nullable variant, used in
`ListAccountEntriesByAccountId`/`CountAccountEntriesByAccountId` for the
optional `period_start`/`period_end`/`entry_type` filters
([account-transaction.sql:56-66](../internal/db/query/account-transaction.sql)) —
each renders as `sql.NullTime`/`sql.NullString` and the handler passes a
zero-value `sql.NullString{}` when the filter isn't in use
([internal/api/transfer/handlers.go:156](../internal/api/transfer/handlers.go)).

**Rough edge — generated params field order isn't argument order.**
`ListMeAccountsByUserIdParams` ends up as `{UserID, Name, OffsetCount,
LimitCount}`
([internal/db/sqlc/account-manage.sql.go:34-39](../internal/db/sqlc/account-manage.sql.go))
even though the SQL text names `user_id`, `name`, `limit_count`,
`offset_count` in that order
([account-manage.sql:1-7](../internal/db/query/account-manage.sql)) — sqlc
assigns struct field order by each arg's first *positional* occurrence in the
rendered SQL (`LIMIT $3 OFFSET $2`), not by the order you wrote
`sqlc.arg(...)` in the source text. It's easy to pass the wrong field to the
wrong meaning if you're skimming the generated struct instead of reading it
carefully.

**Rough edge — `IncrementAccountBalanceParams.Balance` is a delta, not a new
balance.** The query is `balance = balance + $2`
([account-transaction.sql:38-42](../internal/db/query/account-transaction.sql)),
so `Balance` in the params struct means "signed amount to add." Every caller
passes a signed string — e.g. `negativeAmount := "-" + arg.Amount` in
`transferTx` ([store_transaction.go:99](../internal/db/store/store_transaction.go))
and the identical pattern in `deleteAccountTx`
([store_account.go:89](../internal/db/store/store_account.go)). The field
name alone would mislead a new caller into passing an absolute value.

---

## Section 4 — Generated Code (`internal/db/sqlc/*.sql.go`, `querier.go`, `models.go`)

**The problem this piece solves.** Someone has to turn the SQL above into
typed Go without a human re-deriving struct shapes by hand every time a query
changes.

**How it's actually implemented.** `sqlc generate` (wrapped by `make sqlc`)
reads [sqlc.yaml](../sqlc.yaml) in full:

```yaml
version: "1"
packages:
  - name: "db"
    path: "./internal/db/sqlc"
    queries: "./internal/db/query"
    schema: "./internal/db/migrations"
    engine: "postgresql"
    emit_json_tags: true          # generated structs get `json:"..."` tags for free
    emit_prepared_queries: true   # queries are prepared statements, not ad-hoc SQL each call
    emit_interface: true          # generates the Querier interface — required for mocking
    emit_exact_table_names: false # `accounts` table -> `Account` struct, not `Accounts`
    emit_empty_slices: true       # a :many query with 0 rows returns []T{}, not nil
```

Every file it writes is stamped `// Code generated by sqlc. DO NOT EDIT.` —
never hand-edit under [internal/db/sqlc/](../internal/db/sqlc); if the
generated code is wrong, the query or the schema is wrong.

**`Querier` is the seam the whole codebase is built around.**
`emit_interface: true` generates the full interface at
[internal/db/sqlc/querier.go:12-41](../internal/db/sqlc/querier.go) — 24
methods, one per query, e.g.:

```go
type Querier interface {
	AccountEntriesByAccountId(ctx context.Context, id int64) (AccountEntriesByAccountIdRow, error)
	GetAccountByIdForUpdate(ctx context.Context, id int64) (Account, error)
	IncrementAccountBalance(ctx context.Context, arg IncrementAccountBalanceParams) (Account, error)
	ListMeAccountsByUserId(ctx context.Context, arg ListMeAccountsByUserIdParams) ([]ListMeAccountsByUserIdRow, error)
	// ... 20 more
}
var _ Querier = (*Queries)(nil)
```

Business logic that needs to be tested without Postgres — the four `*Tx`
functions in [internal/db/store](../internal/db/store) — is written against
this interface, not against the concrete `*Queries` type. `Storer`
([store.go:13-19](../internal/db/store/store.go)) embeds `db.Querier`
directly, which is what lets [go.uber.org/mock](GOMOCK_TESTING.md) generate a
stand-in for the whole thing (Section 8).

**Table models and view models live side by side, five structs total.**
[internal/db/sqlc/models.go](../internal/db/sqlc/models.go) has `Account`,
`Entry`, `Transfer`, `User` (one per table) and `AccountUserDetailsView`,
`AccountEntriesView` (one per `CREATE VIEW`) — each carrying the `COMMENT ON
COLUMN` text from the migrations as a Go doc comment (e.g.
`models.go:24-26`, from the migration `000006` column comments). A query that
doesn't select every column of exactly one table/view (with `SELECT *` /
`RETURNING *` / `sqlc.embed()`) gets its own `<QueryName>Row` struct instead —
see Section 5 for the current, mapper-free state of that split.

**Money is `string`, never `float64` or a Go numeric type.**
`Account.Balance`, `Entry.Amount`, `Transfer.Amount`, and every
`*Params.Balance`/`*Params.Amount` field are Go `string`. sqlc maps Postgres
`numeric` this way because `numeric` doesn't round-trip losslessly through
`float64`. The codebase parses that string into a
[`shopspring/decimal`](https://github.com/shopspring/decimal) value only at
the point it needs arithmetic — `decimal.NewFromString` in every `*Tx`
function ([store_transaction.go:42](../internal/db/store/store_transaction.go),
[store_account_deposit_tx.go:39](../internal/db/store/store_account_deposit_tx.go),
[store_account.go:72](../internal/db/store/store_account.go)) and in the
request handlers that accept a `decimal.Decimal` directly off JSON
([internal/api/transfer/transactions_handlers.go:19](../internal/api/transfer/transactions_handlers.go),
[internal/api/transfer/handlers.go:52](../internal/api/transfer/handlers.go))
— never inside generated code. Decimal math is the caller's job.

**`emit_empty_slices: true`.** A `:many` query with zero matching rows
returns an initialized empty slice, not `nil`. Worth knowing if you're about
to write `if result == nil` to check "no rows" — that check will never
trigger.

---

## Section 5 — Row Types: What Reuses a Model, What Doesn't

**The problem this piece solves.** sqlc's default rule is: a query's result
type is a struct matching exactly the columns its `SELECT`/`RETURNING` list
names. Two queries against the same table rarely get the same Go type unless
they're written deliberately to line up — which used to mean this codebase
carried a hand-written `db/mapper` package just to convert one query's `Row`
type back into the canonical model another part of the code needed. **That
package doesn't exist anymore** (`ls internal/db` has no `mapper` directory,
confirmed today) — it was deleted by fixing query shapes at the source
instead.

**How it's actually implemented — the three outcomes a query can land in:**

| Query shape | What sqlc generates | Example in this codebase |
|---|---|---|
| `SELECT *` / `RETURNING *` (or an explicit list that names every column of one unaliased table, in table order) | Reuses that table's existing model struct — no new type | `GetAccountById`, `IncrementAccountBalance`, `CreateEntries`, `CreateTransfer`/`GetTransferById` (exhaustive column list, still `Transfer`) |
| `SELECT sqlc.embed(v) FROM some_view v ...` | A wrapper struct with **one field**, the view's model type nested inside | `ListMeAccountsByUserId`, `ListAccountsSearchByUserNumber`, `GetAccountViewById` (all wrap `AccountUserDetailsView`); `AccountEntriesByAccountId`, `ListAccountEntriesByAccountId` (wrap `AccountEntriesView`) |
| A genuinely multi-table, ad hoc shape a join produces that no single table/view has | Its own one-off `Row` struct — correctly needs its own one-off caller-side handling | `ListRecentTransferDestinationsRow`, `ListAccountTransactionHistoryRow` |

**Consequence: `internal/db/store` never converts a `Row` into a model —
every `*Tx` function assigns the query result straight into its result
struct** (e.g. `result.Account = updated` in
[store_account_deposit_tx.go:68](../internal/db/store/store_account_deposit_tx.go)),
because `IncrementAccountBalance`, `CreateEntries`, `CreateTransfer`, etc. all
already return the canonical `sqlc.Account`/`sqlc.Entry`/`sqlc.Transfer`
types directly.

**On the response side, one mapper per view (not per query) lives in the
domain package that owns it** — `toAccountuserResponse`
([internal/api/account/response.go:51](../internal/api/account/response.go))
for every `AccountUserDetailsView` embed, `toAccountEntriesViewResponse`
([internal/api/transfer/response.go:49](../internal/api/transfer/response.go))
for every `AccountEntriesView` embed. The two genuinely one-off `Row` types
each get their own one-off mapper (`toPublicAccountResponseFromDestination`,
`toTransactionHistoryItem`, both in
[internal/api/transfer/response.go](../internal/api/transfer/response.go)) —
that's the correct outcome for a shape no shared type fits, not a gap to
close.

**This doc doesn't re-derive the full playbook.** For the query-by-query
checklist on avoiding a new mapper, the cross-domain caveat on unexported
mapper functions, and a documented bug this exact pattern caught (a
`deleted_at` filter against a view column that never existed), see
[SQLC_ROW_MAPPING.md](SQLC_ROW_MAPPING.md) — it's the authoritative, more
current doc on this specific topic; the rest of this file's scope is setup
and architecture, not row-mapping strategy.

---

## Section 6 — Transaction Composition (`internal/db/store/store.go`)

**The problem this piece solves.** Some operations (creating an account,
transferring money, depositing, deleting an account) need several SQL
statements to succeed or fail together. sqlc alone gives you one method per
query with no notion of "run these atomically," and handlers need a single
interface to depend on regardless of whether an operation is one query or
five.

**How it's actually implemented**, in full
([store.go](../internal/db/store/store.go)):

```go
type Storer interface {
	db.Querier
	TransferTx(ctx context.Context, arg db.CreateTransferParams) (TransferTxResult, error)
	CreateAccountTx(ctx context.Context, arg CreateAccountTxParams) (db.Account, error)
	DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error)
	DeleteAccountTx(ctx context.Context, arg DeleteAccountTxParams) (DeleteAccountTxResult, error)
}

type _store struct {
	*sqlc.Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Storer {
	return &_store{db: db, Queries: sqlc.New(db)}
}

func (store *_store) execTx(ctx context.Context, fn func(sqlc.Querier) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := sqlc.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %w, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}
```

`_store` embeds `*sqlc.Queries`, so every `Querier` method is promoted onto it
for free — `h.Store.GetAccountById(...)` works with no wrapper method needed.
`execTx` opens a `*sql.Tx` and constructs a **second, fresh** `sqlc.New(tx)`
against it, then runs the caller's closure against that — because sqlc
generates `New(db DBTX) *Queries` where `DBTX` is satisfied by both `*sql.DB`
and `*sql.Tx`, the exact same generated type works identically inside and
outside a transaction, with no separate "transactional query" code path to
maintain. `NewStore` returns the `Storer` interface, not `*_store` — callers
(`cmd/server/main.go`, every domain `Handler`) only ever see the interface.

**Rough edge.** If the caller's function fails and `tx.Rollback()` also
fails, both errors are combined into one via `fmt.Errorf`. If rollback
succeeds, only the original error is returned — rollback success is silent,
which is standard practice but means you won't see any log line confirming
the rollback happened.

---

## Section 7 — Business Logic: The Four `*Tx` Functions

**The problem this piece solves.** Four operations in this codebase need more
than one write to commit or roll back together: creating an account (insert,
then a second update once the id is known), transferring money (validate,
lock two accounts, write a transfer + two entries + two balance updates),
depositing (lock one account, write an entry + one balance update), and
deleting an account (sweep a balance to the main account as a real transfer,
then soft-delete). Each follows the same `FooTx(ctx, arg)` (opens `execTx`) +
`fooTx(ctx, q sqlc.Querier, arg)` (the actual logic, parameterized on the
interface) shape, all in [internal/db/store](../internal/db/store).

### `transferTx` (`store_transaction.go`)

1. **Same-account check**: `arg.FromAccountID == arg.ToAccountID` →
   `ErrSameAccount`.
2. **Amount validation**: `decimal.NewFromString(arg.Amount)`, reject if not
   positive → `ErrInvalidAmount`.
3. **Lock ordering — deadlock avoidance.** The comment in the source states
   the reasoning directly:

   > Accounts are locked in a fixed ascending-ID order (rather than
   > from-then-to) so two concurrent transfers between the same pair of
   > accounts always acquire their row locks in the same order and can't
   > deadlock against each other.

   ```go
   firstID, secondID := arg.FromAccountID, arg.ToAccountID
   if firstID > secondID {
       firstID, secondID = secondID, firstID
   }
   firstAccount, err := q.GetAccountByIdForUpdate(ctx, firstID)
   secondAccount, err := q.GetAccountByIdForUpdate(ctx, secondID)
   // re-derive which of first/second is actually fromAccount/toAccount
   ```

   The end-to-end consequence: transfer A→B and transfer B→A running
   concurrently both lock the lower account ID first, so neither can hold a
   lock the other is waiting on — no deadlock, regardless of transfer
   direction.
4. **Currency mismatch**: `fromAccount.Currency != toAccount.Currency` →
   `ErrCurrencyMismatch`.
5. **Insufficient funds**: parse `fromAccount.Balance` as decimal, compare to
   the transfer amount → `ErrInsufficientFunds`.
6. **Ordered writes**: `CreateTransfer` → debit `CreateEntries` (amount
   negated, type `ENTRY_TYPE_SEND`) → debit `IncrementAccountBalance` → credit
   `CreateEntries` (type `ENTRY_TYPE_RECEIVED`) → credit
   `IncrementAccountBalance`. Every step returns on its own error, so nothing
   after a failing step executes — this exact property is what the gomock
   handler tests assert via `.Times(0)` on calls that shouldn't happen (see
   [GOMOCK_TESTING.md](GOMOCK_TESTING.md)).

### `createAccountTx` (`store_account_create_tx.go`)

Two-step insert: `CreateAccount` (balance `"0"`, currency `"IDR"` hard-coded)
returns a row with the new auto-generated `ID`, then `UpdateAccountNumber`
derives the human-facing account number from that `ID`
(`generateAccountNumber`, [utils.go](../internal/db/store/utils.go)) and
writes it in a second statement — this can't be collapsed into a single
`RETURNING *` insert because the number depends on the row's own id. Called
both from account creation (`internal/api/account/handlers.go`) and from
registration, to create the user's one `IsMain: true` account
([internal/api/auth/handlers.go:159](../internal/api/auth/handlers.go)).

### `depositTx` (`store_account_deposit_tx.go`)

Simpler than `transferTx`: validate the amount is positive
(`ErrInvalidAmount`), lock the single account via `GetAccountByIdForUpdate`,
`CreateEntries` with type `ENTRY_TYPE_DEPOSIT`, then
`IncrementAccountBalance`. No currency/insufficient-funds checks — a deposit
only ever increases a balance, so those failure modes don't apply here.

### `deleteAccountTx` (`store_account.go`)

The "sweep-then-soft-delete" flow: `GetAccountById` the target, reject if
`IsMain` (`ErrCannotDeleteMainAccount`); `GetMainAccountByUserId` to find
where a balance would go; lock **both** the target and the main account in
the same ascending-ID order `transferTx` uses (correctness property, not
style — see AGENTS.md); if the target's balance is positive, write a real
transfer (with its own two entries and two balance updates, description
`"Account closure balance transfer"`) sweeping it to the main account; then
`SoftDeleteAccount` regardless of whether a sweep happened.

### Sentinel errors (`internal/db/store/errors.go`, in full)

| Error                          | Meaning                                                        |
| ------------------------------- | ----------------------------------------------------------------- |
| `ErrSameAccount`               | `from_account_id` and `to_account_id` are identical             |
| `ErrCurrencyMismatch`          | The two accounts don't share a currency                         |
| `ErrInvalidAmount`             | Transfer/deposit amount is not greater than zero                |
| `ErrInsufficientFunds`         | The from-account's balance is less than the transfer amount     |
| `ErrCannotDeleteMainAccount`   | The account targeted for deletion is the user's main account    |

**Corrected from an earlier revision of this doc: `ENTRY_TYPE_DEPOSIT` is not
dead code.** It's defined alongside `ENTRY_TYPE_SEND`/`ENTRY_TYPE_RECEIVED`
([internal/domain/constant/entries.go](../internal/domain/constant/entries.go))
and is used by `depositTx` — an earlier version of this codebase (and an
earlier revision of this doc) predates the deposit feature, when the constant
really was unreferenced scaffolding.

---

## Section 8 — Testing Without (and With) a Real Database

**The problem this piece solves.** The `*Tx` functions' branches (same
account, insufficient funds, currency mismatch, lock ordering) need to be
testable without depending on Postgres being up, while a separate suite
proves the real generated SQL actually works.

**How it's actually implemented.**

**Mocking.** `make generate-mock` (Makefile) runs:

```
go run go.uber.org/mock/mockgen@latest -package mockdb -destination internal/db/mock/querier.go github.com/DewaSRY/core-service/internal/db/store Storer
```

— note the target is `internal/db/store`'s `Storer`, **not**
`internal/db/sqlc`'s `Querier`. This regenerates
[internal/db/mock/querier.go](../internal/db/mock/querier.go), a
`MockStorer` with one mock method + one `EXPECT()` recorder method per
`Storer` method (both the embedded `Querier` methods and the four `*Tx`
methods). There's no `//go:generate` directive anywhere in source;
regeneration is Makefile-driven only, so it's easy to forget after changing
`Storer` — see [GOMOCK_TESTING.md](GOMOCK_TESTING.md) for the full mechanics
of writing tests against `MockStorer` (`gomock.NewController`,
`gomock.Any()`, `gomock.InOrder`, `.Times(0)`), which this doc doesn't
repeat.

`MockStorer` is used in exactly three handler test files:

| File                                                                                    | What it tests                                                        |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| [internal/api/account/account_handler_test.go](../internal/api/account/account_handler_test.go) | Account create/list/search/get/update/delete handlers                |
| [internal/api/auth/auth_handler_test.go](../internal/api/auth/auth_handler_test.go)       | Login/register/profile handlers                                      |
| [internal/api/transfer/transfer_handler_test.go](../internal/api/transfer/transfer_handler_test.go) | Deposit, transfer, entries/transactions/recent-destinations listing |

Each builds a bare `gin.Engine` with `core.ErrorHandlerMiddleware` (and
`core.AuthMiddleware` for authorized routes) wired the same way `NewServer`
wires them in production, constructs the domain's `Handler` with a
`mockdb.NewMockStorer(gomock.NewController(t))`, sets `.EXPECT()`
expectations, fires a request with `httptest`, and asserts on the recorded
response — see `newTestRouter`/`authHeaderFor`/`doAuthenticatedRequest` in
[transfer_handler_test.go:45-91](../internal/api/transfer/transfer_handler_test.go)
for the pattern every test file repeats.

**The most important rough edge in this doc: `internal/db/store` has no test
files of its own.** `ls internal/db/store/*_test.go` returns nothing — none
of `transferTx`, `createAccountTx`, `depositTx`, or `deleteAccountTx` has a
dedicated unit test against a mocked `sqlc.Querier`. The only coverage they
get is indirect, through the owning domain's `*_handler_test.go` files above,
each of which mocks the *entire* `Storer.TransferTx`/`.DepositTx`/etc. call —
so none of them can catch a regression inside a `*Tx` function itself (wrong
lock order, wrong call sequence, a currency check that got dropped). This is
called out identically from the test-strategy angle in
[GOMOCK_TESTING.md](GOMOCK_TESTING.md) and from the row-mapping angle in
[SQLC_ROW_MAPPING.md §7](SQLC_ROW_MAPPING.md#7-known-gaps-and-corrections-out-of-scope-for-this-change-called-out-rather-than-silently-left).
If you touch any `*Tx` function, don't assume an existing test will catch a
mistake — consider adding a `internal/db/store/*_test.go` against a mocked
`sqlc.Querier` first.

**Integration tests against real Postgres** live in
[internal/db/sqlc](../internal/db/sqlc): `account_test.go`, `entries_test.go`,
`transfers_test.go`. `main_test.go`'s `createTestQueries(t)` begins a
transaction against `testDB` (a fixed local connection string,
`postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable`)
and registers `t.Cleanup(func(){ tx.Rollback() })` — every test in package
`db` runs inside a transaction that's always rolled back, so nothing it does
ever persists. Run with `go test ./internal/db/sqlc/...` after `make db-up`.

**No drift detection.** Nothing in CI enforces that `make sqlc` and `make
generate-mock` were both run after a `Querier`/`Storer` change — acknowledged
directly in [GOMOCK_TESTING.md](GOMOCK_TESTING.md).

---

## Section 9 — Cross-Cutting Coupling: How Handlers Depend on This Layer

Every domain `Handler` (`account.Handler`, `auth.Handler`, `transfer.Handler`)
holds a `store.Storer` field directly — there's no separate `Storer`
interface redefined at the `internal/api` layer the way an earlier revision
of this doc described. `auth.Handler` additionally holds a `token.Maker` and
an `AccessTokenDuration`
([internal/api/auth/auth.go:16-20](../internal/api/auth/auth.go)); `account.Handler`
and `transfer.Handler` hold only `Store`
([internal/api/account/account.go:13-15](../internal/api/account/account.go),
[internal/api/transfer/transfer.go:11-13](../internal/api/transfer/transfer.go)).

This one shared interface is what makes `MockStorer` (Section 8) drop into
every domain's handler tests unmodified — a real `store.NewStore(db)` and a
`mockdb.NewMockStorer(ctrl)` are interchangeable anywhere a `Handler` is
constructed. If `Storer` ever grew a method not backed by either `Querier` or
one of the four `*Tx` functions, both the real `_store` and `MockStorer`
would need updating in lockstep — worth knowing before adding a new interface
method here.

`internal/api/router.go`'s `bindRouters` is the one place these `Handler`
structs get constructed and wired to routes
([router.go:21-47](../internal/api/router.go)) — it's the composition root
for the API layer the way `sqlc.yaml` is for codegen (Section 1), and it
never needs touching for a new endpoint on an existing domain, only for a
brand-new domain package.

---

## Section 10 — The Handler Layer (Low Detail — Logic Lives Upstream)

Handlers themselves are thin dispatch, not where sqlc concerns live. Two call
styles show up side by side in the same handler,
`transactionTransfer`
([internal/api/transfer/transactions_handlers.go:39-92](../internal/api/transfer/transactions_handlers.go)):

- A plain sqlc-generated query (`h.Store.GetAccountById`), used for an
  ownership check before the transfer runs.
- The hand-written multi-step transaction (`h.Store.TransferTx`), not
  sqlc-generated at all — followed by one more plain query
  (`h.Store.AccountEntriesByAccountId`) to re-read the entry just written, in
  the shape the response needs (Section 5's `sqlc.embed()` pattern).

Elsewhere, list endpoints follow a count-then-list pagination pattern —
`listmeAccounts`/`searchAccountByNumber`
([internal/api/account/handlers.go](../internal/api/account/handlers.go)) and
`listAccountEntriesByAccountId`/`listAccountTransactionHistory`
([internal/api/transfer/handlers.go](../internal/api/transfer/handlers.go))
each call a `List*` and matching `Count*` sqlc method as a pair, then
`util.MapSlice` ([internal/util/array.go:6](../internal/util/array.go)) to
convert `[]Row` into the response DTO slice. There's no logic here worth
documenting beyond "it calls the interface" — the interesting behavior is
upstream in Sections 6-7.

---

## Summary: Data Flow for a Transfer Request

1. Client hits `POST /api/v1/transactions/transfer` → `transactionTransfer`
   decodes the request, rejects a same-account/non-positive-amount request
   at the handler level (before ever touching the store), then checks
   ownership via `h.Store.GetAccountById` (a direct sqlc call).
2. Handler calls `h.Store.TransferTx(ctx, arg)`.
3. `TransferTx` calls `execTx`, which opens a `*sql.Tx` and builds a fresh
   `sqlc.Queries` bound to it.
4. `transferTx` runs against that `Queries` (as a `Querier`): re-validates
   same-account/amount, locks both accounts in ascending-ID order, checks
   currency and balance, then writes transfer + two entries + two balance
   updates in sequence.
5. Any error aborts the remaining steps and rolls back the transaction;
   success commits it.
6. The handler receives `TransferTxResult{Transfer, FromAccount, ToAccount,
   FromEntry, ToEntry}` — all plain `sqlc.Transfer`/`sqlc.Account`/`sqlc.Entry`
   values, no mapper needed (Section 5) — then calls
   `h.Store.AccountEntriesByAccountId(ctx, result.FromEntry.ID)` to fetch the
   entry back through `account_entries_view` and serializes that.

Error path: any of `ErrSameAccount`, `ErrCurrencyMismatch`,
`ErrInvalidAmount`, `ErrInsufficientFunds` (or a `pq.Error` `check_violation`/
`foreign_key_violation`) propagates straight back through `execTx` (triggering
rollback) to the handler, which `transferAppError`
([internal/api/transfer/apperror.go](../internal/api/transfer/apperror.go))
maps to an HTTP error response.

---

## Final Reference: Every `Querier` Method, Plus the Four `*Tx` Methods

| Method                              | Annotation | Source file              | Purpose                                                                |
| -------------------------------------- | ------------ | --------------------------- | -------------------------------------------------------------------------- |
| `CreateAccount`                     | `:one`     | account.sql               | Insert a new account (balance `"0"`, currency hard-coded `"IDR"`)      |
| `UpdateAccountNumber`               | `:one`     | account.sql               | Second half of account creation — sets the id-derived account number  |
| `UpdateAccount`                     | `:one`     | account.sql               | Update name/description                                                |
| `SoftDeleteAccount`                 | `:one`     | account.sql               | Sets `deleted_at`, not a hard delete                                    |
| `GetAccountById`                    | `:one`     | account.sql               | Fetch an account, no lock, active only                                 |
| `GetMainAccountByUserId`            | `:one`     | account.sql               | Fetch the user's `is_main = true` account                              |
| `ListAccountsSearchByUserNumber`    | `:many`    | account.sql               | Paginated account search by number, via `account_user_details_view`   |
| `CountAccountsSearchByUserNumber`   | `:one`     | account.sql               | Total count, paired with the above                                     |
| `ListMeAccountsByUserId`            | `:many`    | account-manage.sql        | Paginated accounts for a user, via `account_user_details_view`        |
| `ListMeAccountsByUserIdCount`       | `:one`     | account-manage.sql        | Total count, paired with the above                                     |
| `CountAccountsByUserId`             | `:one`     | account-manage.sql        | Plain accounts-table count for a user                                  |
| `ListRecentTransferDestinations`    | `:many`    | account-transaction.sql   | Recently-transferred-to accounts, 3-way join, one-off shape             |
| `GetAccountViewById`                | `:one`     | account-transaction.sql   | Fetch one account via `account_user_details_view`                     |
| `GetAccountByIdForUpdate`           | `:one`     | account-transaction.sql   | Fetch an account with `FOR UPDATE` — used only inside `*Tx` functions  |
| `IncrementAccountBalance`           | `:one`     | account-transaction.sql   | Add a signed delta to `balance`                                        |
| `CheckIsAccountWithIdExist`         | `:one`     | account-transaction.sql   | Existence check                                                        |
| `ListAccountEntriesByAccountId`     | `:many`    | account-transaction.sql   | Paginated entries for an account, via `account_entries_view`          |
| `CountAccountEntriesByAccountId`    | `:one`     | account-transaction.sql   | Total count, paired with the above                                     |
| `AccountEntriesByAccountId`         | `:one`     | account-transaction.sql   | Fetch one entry via `account_entries_view`, used after deposit/transfer |
| `CreateEntries`                     | `:one`     | entries.sql                | Insert a ledger entry (send/received/deposit)                          |
| `ListAccountTransactionHistory`     | `:many`    | entries.sql                | Paginated, counterparty-resolved transaction history, one-off shape    |
| `CountAccountTransactionHistory`    | `:one`     | entries.sql                | Total count, paired with the above                                     |
| `CreateTransfer`                    | `:one`     | transfers.sql               | Insert a transfer record                                               |
| `GetTransferById`                   | `:one`     | transfers.sql               | Fetch a transfer by id                                                  |
| `CreateUser`                        | `:one`     | user.sql                    | Insert a user; deliberately doesn't return `hashed_password`          |
| `GetUserByEmail`                    | `:one`     | user.sql                    | Login lookup, includes `hashed_password`                               |
| `GetUserById`                       | `:one`     | user.sql                    | Profile lookup, excludes `hashed_password`                             |
| `CheckIsUsernameExist`              | `:one`     | user.sql                    | Existence check                                                        |

Plus four methods not on `Querier` at all, hand-written on `Storer`/`_store`
(Section 7): `TransferTx`, `CreateAccountTx`, `DepositTx`, `DeleteAccountTx`.
