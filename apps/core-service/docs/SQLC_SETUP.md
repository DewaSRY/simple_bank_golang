# sqlc Database Access Layer — As Implemented

## Who this doc is for

Anyone touching `db/`, `internal/api/`, or adding a new query — whether you've used sqlc before or never heard of it. It assumes you're comfortable with Go and basic SQL but not necessarily with sqlc's code-generation model. This doc is verified line-by-line against the current source, not against the original design intent — where the code does something surprising (a stale doc reference, a migration `down` that doesn't run `.down.sql`, a params field whose name undersells what it does), that's called out explicitly rather than smoothed over. Read the callouts before you copy a pattern from an existing query.

---

## Section 0 — Background Primer: why sqlc, not the alternatives

| Approach | You write | Codegen writes | Typical failure mode |
|---|---|---|---|
| Full ORM (GORM, ent) | Struct tags + builder calls | Query construction + scanning | Builder fights you the moment a query gets non-trivial (row locking, custom joins) |
| Raw `database/sql` | SQL string + struct + `rows.Scan(&a, &b, &c)` | Nothing | Scan-order/SELECT-list mismatch is a silent runtime bug, not a compile error |
| sqlc | SQL only, in `.sql` files | Struct + params + method per query | None of the above — but you're locked to one SQL dialect |

This service talks to Postgres through [sqlc](https://sqlc.dev). sqlc reads the migrations in [db/migrations](../db/migrations) and the query files in [db/query](../db/query) and generates typed Go: one struct per row shape, one params struct per query's inputs, and one method per query. You write SQL; sqlc writes the boilerplate that would otherwise be hand-maintained in parallel with it.

**Why not a full ORM.** This codebase's transfer logic needs `SELECT ... FOR UPDATE` with a specific row-locking order (`GetAccountByIdForUpdate` in [db/query/account.sql](../db/query/account.sql), used from [db/store/store_transaction.go](../db/store/store_transaction.go)) to avoid deadlocking concurrent transfers between the same two accounts. That's exactly the kind of query an ORM's builder makes awkward to express and easy to get subtly wrong. With sqlc you just write the SQL.

**Why not raw `database/sql`.** Every query becomes: write the SQL as a string, write a struct, hand-write `rows.Scan(...)` in the same order as the `SELECT` list, and hope nobody reorders one without the other. sqlc removes the hand-written half of that pair — the SQL stays the single source of truth, the Go side is generated from it, and a change to one regenerates the other.

**The trade-off you're accepting.** A query written for Postgres's `FOR UPDATE` and `sqlc.arg()` syntax doesn't portably move to MySQL or SQLite. That's fine here — this project has already committed to Postgres (`github.com/lib/pq`, migrations under [db/migrations](../db/migrations)) — but it's the reason sqlc would be the wrong tool for a project that genuinely needs to be database-agnostic.

**Gotchas that surprise newcomers** (each connects forward to a section below):

1. A query that doesn't `SELECT *` or match a table exactly gets its **own** `<QueryName>Row` struct instead of reusing the table's model type — this is why [db/mapper](../db/mapper) exists (Section 5).
2. Generated params struct fields are ordered by **first appearance in the SQL text**, not by your intuition of argument order — this bites the first time you read a generated call site (Section 3).
3. A method signature on `Store` looks identical whether you're inside a transaction or not — because the same generated `Queries` type is constructed against either a `*sql.DB` or a `*sql.Tx` (Section 6).

---

## Section 1 — Architecture at a Glance

The composition root is **[sqlc.yaml](../sqlc.yaml)** plus the `sqlc generate` invocation it drives (`make sqlc`) — it doesn't implement any query logic itself, it only wires "these `.sql` files, read against this schema, produce Go in this package." Business logic (locking order, currency checks, error sentinels) lives entirely downstream, in hand-written Go that depends on what sqlc generated.

```
db/query/*.sql          <- hand-written SQL, source of truth
db/migrations/*.sql     <- schema, also source of truth
        │
        │  sqlc generate  (make sqlc)
        ▼
db/sqlc/*.sql.go        <- generated: Queries methods, Params/Row structs
db/sqlc/querier.go       <- generated: Querier interface (every query as a method)
db/sqlc/models.go        <- generated: one struct per table
        │
        │  db/mapper translates Row -> model where they diverge
        ▼
db/store/*.go            <- hand-written: Store, execTx, transferTx (business logic)
        │
        │  internal/api depends on the Storer interface (db.Querier + TransferTx)
        ▼
internal/api/*_router.go <- handlers call server.store.<Method>(ctx, params)
```

| Concern | Owner (file/package) | Analogy |
|---|---|---|
| Schema source of truth | [db/migrations/*.sql](../db/migrations) | The database's own change log |
| Query source of truth | [db/query/*.sql](../db/query) | Hand-written SQL, never generated |
| Codegen config | [sqlc.yaml](../sqlc.yaml) | Build config for the generation step |
| Generated query methods | [db/sqlc/*.sql.go](../db/sqlc) | Compiler-generated glue code |
| Generated interface seam | [db/sqlc/querier.go](../db/sqlc/querier.go) | An abstract base class with no logic |
| Generated table models | [db/sqlc/models.go](../db/sqlc/models.go) | Plain data records |
| Row → model translation | [db/mapper/account.go](../db/mapper/account.go) | An adapter plug between two sockets |
| Transaction composition | [db/store/store.go](../db/store/store.go) | A unit-of-work wrapper |
| Multi-step business logic | [db/store/store_transaction.go](../db/store/store_transaction.go) | The actual "transfer money" recipe |
| Mock for the seam | [db/mock/querier.go](../db/mock/querier.go) (generated) | A stunt double for the database |
| API entry points | [internal/api/*_router.go](../internal/api) | The front desk that only knows the interface |

One paragraph on why it's split this way: the `Querier` interface is what makes `transferTx` testable without a database — a mock can stand in for it in a unit test (Section 8), while a real `*sqlc.Queries` (built against either a `*sql.DB` or a `*sql.Tx`) stands in for it in production and in DB-backed integration tests. That split is also the source of one of this doc's more important callouts: the two kinds of tests that use a real database (`db/sqlc` vs `db/store`) clean up in genuinely different ways, and assuming they behave the same will surprise you (Section 8).

---

## Section 2 — Schema & Migrations

**The problem this piece solves.** Postgres needs a schema before sqlc can validate any query against it, and that schema needs to evolve over time without hand-editing a live database.

**How it's actually implemented.** Migrations live under [db/migrations/](../db/migrations) as four numbered `.up.sql` / `.down.sql` pairs, run through the golang-migrate CLI wrapped by [cmd/migration/main.go](../cmd/migration/main.go):

| Migration | What it does |
|---|---|
| `000001_init-transaction-feature` | Creates `accounts` (`balance NUMERIC(20,2)`, `currency VARCHAR(3)`), `transfers`, `entries`; FKs from entries/transfers to accounts; indexes on `accounts(owner)`, `entries(account_id)`, `transfers(from_account_id)`, `transfers(to_account_id)` |
| `000002_add-balance-constraints` | Idempotently adds `CHECK (balance >= 0)` on accounts, `CHECK (amount > 0)` on transfers |
| `000003_add-users-table` | Creates `users` (unique `username`/`email`, `hashed_password`), index on `email` |
| `000004_authorization_by_user_tabel` | Adds nullable `user_id` to `accounts` and `entries`, each with an idempotent FK to `users(id)` |

sqlc reads this same directory as its `schema:` source ([sqlc.yaml:7](../sqlc.yaml)), so a migration and the queries that depend on it should land in the same PR — sqlc generates against whatever schema is on disk, not what's actually applied to any database.

Makefile targets: `make migrate-create name=...`, `make migrate-up`, `make migrate-down` ([Makefile:7-14](../Makefile)).

**Rough edge — `migrate-down` doesn't run `.down.sql`.** [cmd/migration/main.go:44](../cmd/migration/main.go) runs `migrate ... force 1`, which forcibly resets the `schema_migrations` version, not `migrate down`, which would actually execute the most recent migration's `.down.sql`. If you're expecting `make migrate-down` to undo migration N's schema changes, it won't — it only clears migrate's bookkeeping.

**Rough edge — dead doc reference.** Both `upMigration` and `downMigration` in `cmd/migration/main.go` (comments at lines 32-33 and 49-50) point readers to a `db/migrations/README.md` that does not exist in the repo.

---

## Section 3 — Query Files (`db/query/*.sql`)

**The problem this piece solves.** SQL needs to live somewhere sqlc can read it, one query per intended Go method, with enough annotation for sqlc to know what shape to generate.

**How it's actually implemented.** One file per table under [db/query/](../db/query). Every query is a `-- name: MethodName :one|:many|:exec` comment directly above a SQL statement — that annotation, not the SQL body, is the contract deciding whether the generated method returns a single struct, a slice, or nothing.

| File | Query | Annotation | Notable detail |
|---|---|---|---|
| [account.sql](../db/query/account.sql) | `CreateAccount` | `:one` | `RETURNING id, owner, balance, currency, user_id, created_at` |
| | `GetAccountById` | `:one` | Plain SELECT by id |
| | `GetAccountByIdForUpdate` | `:one` | Adds `FOR UPDATE` — pessimistic row lock used by `transferTx` |
| | `IncrementAccountBalance` | `:one` | `SET balance = balance + $2` — see rough edge below |
| | `CheckIsAccountWithIdExist` | `:one` | `SELECT EXISTS(...)` |
| | `ListAccountsByUserId` | `:many` | `sqlc.arg(user_id)`, `sqlc.arg(limit_count)`, `sqlc.arg(offset_count)` |
| | `CountAccountsByUserId` | `:one` | Plain `$1`, no `sqlc.arg` |
| [entries.sql](../db/query/entries.sql) | `CreateEntries` | `:one` | `RETURNING id, account_id, type, amount, created_at` |
| | `ListEntriesByAccount` | `:many` | `sqlc.arg(...)` for account/limit/offset |
| | `CountEntriesByAccount` | `:one` | Plain `$1` |
| [transfers.sql](../db/query/transfers.sql) | `CreateTransfer` | `:one` | `RETURNING id, from_account_id, to_account_id, amount, created_at` |
| | `GetTransferById` | `:one` | Plain SELECT by id |
| | `ListTransfersByOwner` | `:many` | Two-way `JOIN accounts` (`fa`/`ta` aliases) matching either the from- or to-owner; `sqlc.arg(owner)` |
| | `CountTransfersByOwner` | `:one` | Same join pattern, plain `$1` |
| [user.sql](../db/query/user.sql) | `CreateUser` | `:one` | `RETURNING id, username, email, created_at` — deliberately excludes `hashed_password` |
| | `GetUserByEmail` | `:one` | Includes `hashed_password` (used for login) |
| | `GetUserById` | `:one` | Excludes `hashed_password` (used for profile reads) |
| | `CheckIsUsernameExist` | `:one` | `SELECT EXISTS(...)` |

**`sqlc.arg(name)`.** Plain `$1`-style positional params work fine for one or two arguments, but any query with three or more (the `List*` queries above) uses `sqlc.arg(...)` instead — this names the generated params struct fields (`LimitCount`, `OffsetCount`, ...) rather than leaving them ordinal, which stays readable as a query grows.

**Rough edge — generated params field order isn't argument order.** `ListAccountsByUserIdParams` ends up as `{UserID, OffsetCount, LimitCount}` ([db/sqlc/account.sql.go:190-194](../db/sqlc/account.sql.go)) even though the SQL names `user_id`, `limit_count`, `offset_count` in that order — sqlc assigns struct field order by each arg's first *positional* occurrence in the rendered SQL (`LIMIT $3 OFFSET $2`), not by the order you wrote `sqlc.arg(...)` calls. It's easy to pass the wrong field to the wrong meaning if you're skimming instead of reading the generated struct.

**Rough edge — `IncrementAccountBalanceParams.Balance` is a delta, not a new balance.** The query is `balance = balance + $2` ([db/query/account.sql:27](../db/query/account.sql)), so `Balance` in the params struct means "amount to add," signed. It's only safe because every caller passes a signed string — see `"-" + arg.Amount` at [store_transaction.go:88](../db/store/store_transaction.go). The field name alone would mislead a new caller into passing an absolute value.

---

## Section 4 — Generated Code (`db/sqlc/*.sql.go`, `querier.go`, `models.go`)

**The problem this piece solves.** Someone has to turn the SQL above into typed Go without a human re-deriving struct shapes by hand every time a query changes.

**How it's actually implemented.** `sqlc generate` (wrapped by `make sqlc`) reads [sqlc.yaml](../sqlc.yaml):

```yaml
version: "1"
packages:
  - name: "db"
    path: "./db/sqlc"
    queries: "./db/query"
    schema: "./db/migrations"
    engine: "postgresql"
    emit_json_tags: true          # generated structs get `json:"..."` tags for free
    emit_prepared_queries: true   # queries are prepared statements, not ad-hoc SQL each call
    emit_interface: true          # generates the Querier interface — required for mocking
    emit_exact_table_names: false # `accounts` table -> `Account` struct, not `Accounts`
    emit_empty_slices: true       # a :many query with 0 rows returns []T{}, not nil
```

Every file it writes is stamped `// Code generated by sqlc. DO NOT EDIT.` — never hand-edit under [db/sqlc/](../db/sqlc); if the generated code is wrong, the query or the schema is wrong.

**`Querier` is the seam the whole codebase is built around.** `emit_interface: true` generates the full interface at [db/sqlc/querier.go:12-31](../db/sqlc/querier.go) — one method per query, e.g.:

```go
type Querier interface {
	CreateAccount(ctx context.Context, arg CreateAccountParams) (CreateAccountRow, error)
	GetAccountByIdForUpdate(ctx context.Context, id int64) (GetAccountByIdForUpdateRow, error)
	IncrementAccountBalance(ctx context.Context, arg IncrementAccountBalanceParams) (IncrementAccountBalanceRow, error)
	ListAccountsByUserId(ctx context.Context, arg ListAccountsByUserIdParams) ([]ListAccountsByUserIdRow, error)
	// ...
}
var _ Querier = (*Queries)(nil)
```

Business logic that needs to be tested without Postgres — `transferTx` in [db/store/store_transaction.go](../db/store/store_transaction.go) — is written against this interface, not against the concrete `*Queries` type. That's what lets [go.uber.org/mock](GOMOCK_TESTING.md) generate a stand-in for it (Section 8).

**Row types vs. model types.** [db/sqlc/models.go](../db/sqlc/models.go) has exactly four structs, one per table: `Account`, `Entry`, `Transfer`, `User` — each carrying the `COMMENT ON COLUMN` text from the migrations as a Go doc comment (e.g. `models.go:16`, from the `CHECK (balance >= 0)` comment in migration 000002). A query that doesn't select every column of its table gets its own `<QueryName>Row` struct instead of reusing the model — e.g. `CreateAccountRow` and `GetAccountByIdForUpdateRow` both exist separately from `Account` even though they're nearly identical, because sqlc generates a struct matching exactly the columns the query selected, per query.

**Money is `string`, never `float64` or a Go numeric type.** `Account.Balance`, `Entry.Amount`, `Transfer.Amount`, and every `*Params.Balance`/`*Params.Amount` field are Go `string`. sqlc maps Postgres `numeric` this way because `numeric` doesn't round-trip losslessly through `float64`. The codebase parses that string into a [`shopspring/decimal`](https://github.com/shopspring/decimal) value only at the point it needs arithmetic — `decimal.NewFromString` in `transferTx` ([store_transaction.go:34,69](../db/store/store_transaction.go)) and in the transfer request handler ([internal/api/transactions_router.go:18,68](../internal/api/transactions_router.go)) — never inside generated code. Decimal math is the caller's job.

**`emit_empty_slices: true`.** A `:many` query with zero matching rows returns an initialized empty slice (`items := []ListAccountsByUserIdRow{}`, [account.sql.go:211](../db/sqlc/account.sql.go)), not `nil`. Worth knowing if you're about to write `if result == nil` to check "no rows" — that check will never trigger.

---

## Section 5 — `db/mapper`: Bridging Row Types Back to Models

**The problem this piece solves.** `transferTx` needs to hand back `sqlc.Account` values (the type `TransferTxResult` is declared with), but `IncrementAccountBalance` returns `IncrementAccountBalanceRow`, a distinct type, because its `RETURNING` list doesn't include every column `Account` has.

**How it's actually implemented.** [db/mapper/account.go](../db/mapper/account.go) is the entire package — one function:

```go
func UpdateBalanceAccountToAccount(updateAccount sqlc.IncrementAccountBalanceRow) sqlc.Account {
	return sqlc.Account{
		ID:        updateAccount.ID,
		Owner:     updateAccount.Owner,
		Balance:   updateAccount.Balance,
		Currency:  updateAccount.Currency,
		CreatedAt: updateAccount.CreatedAt,
	}
}
```

Used at [store_transaction.go:109](../db/store/store_transaction.go) and `:131` to populate `TransferTxResult.FromAccount` / `.ToAccount`. Note this mapper doesn't set `UpdatedAt` — the returned `Account.UpdatedAt` is left zero-valued, since `IncrementAccountBalanceRow` doesn't carry it forward from the query.

This is the pattern to follow if a future query returns a `*Row` type that the rest of the codebase needs as the canonical model: add a mapper function here rather than widening the query's `RETURNING` list just to make the types match, or reaching for the `Row` type in places that expect `Account`.

---

## Section 6 — Transaction Composition (`db/store/store.go`)

**The problem this piece solves.** Some operations (transferring money) need several SQL statements to succeed or fail together. sqlc alone gives you one method per query with no notion of "run these atomically."

**How it's actually implemented.** [db/store/store.go](../db/store/store.go), in full:

```go
type Store struct {
	*sqlc.Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, Queries: sqlc.New(db)}
}

func (store *Store) execTx(ctx context.Context, fn func(sqlc.Querier) error) error {
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

`Store` embeds `*sqlc.Queries` (line 13), so every `Querier` method is promoted onto `Store` for free — `server.store.GetAccountById(...)` works with no wrapper method needed. `execTx` opens a `*sql.Tx` and constructs a **second, fresh** `sqlc.New(tx)` against it, then runs the caller's closure against that — because sqlc generates `New(db DBTX) *Queries` where `DBTX` is satisfied by both `*sql.DB` and `*sql.Tx`, the exact same generated type works identically inside and outside a transaction, with no separate "transactional query" code path to maintain.

**Rough edge.** If the caller's function fails and `tx.Rollback()` also fails, both errors are combined into one via `fmt.Errorf`. If rollback succeeds, only the original error is returned — rollback success is silent, which is standard practice but means you won't see any log line confirming the rollback happened.

---

## Section 7 — Business Logic: `transferTx` (`db/store/store_transaction.go`)

**The problem this piece solves.** Moving money between two accounts needs: validation, a debit entry, a credit entry, two balance updates, all atomic, and — because two people can transfer between the same pair of accounts concurrently in either direction — a locking strategy that can't deadlock.

**How it's actually implemented**, in order:

1. **Entry point.** `TransferTx` ([store_transaction.go:13-23](../db/store/store_transaction.go)) wraps `transferTx` in `store.execTx`.
2. **The interface, not the struct.** `transferTx(ctx, q sqlc.Querier, arg)` takes `sqlc.Querier`, explicitly commented (lines 25-26) as what makes this function testable with a gomock `MockQuerier` instead of a real database.
3. **Same-account check** (30-32): `arg.FromAccountID == arg.ToAccountID` → `ErrSameAccount`.
4. **Amount validation** (34-40): `decimal.NewFromString(arg.Amount)`, reject if not positive → `ErrInvalidAmount`.
5. **Lock ordering — deadlock avoidance** (42-63). The comment in the source states the reasoning directly:

   > Accounts are locked in a fixed ascending-ID order (rather than from-then-to) so two concurrent transfers between the same pair of accounts always acquire their row locks in the same order and can't deadlock against each other.

   ```go
   firstID, secondID := arg.FromAccountID, arg.ToAccountID
   if firstID > secondID {
       firstID, secondID = secondID, firstID
   }
   first, _ := q.GetAccountByIdForUpdate(ctx, firstID)
   second, _ := q.GetAccountByIdForUpdate(ctx, secondID)
   // re-derive which of first/second is actually fromAccount/toAccount
   ```

   The end-to-end consequence: transfer A→B and transfer B→A running concurrently both lock the lower account ID first, so neither can hold a lock the other is waiting on — no deadlock, regardless of transfer direction.
6. **Currency mismatch** (65-67): `fromAccount.Currency != toAccount.Currency` → `ErrCurrencyMismatch`.
7. **Insufficient funds** (69-75): parse `fromAccount.Balance` as decimal, compare to the transfer amount → `ErrInsufficientFunds`.
8. **Ordered writes** (78-131): `CreateTransfer` → debit `CreateEntries` (amount negated, `"-" + arg.Amount`, type `ENTRY_TYPE_SEND`) → debit `IncrementAccountBalance` (mapped back to `sqlc.Account` via [db/mapper](../db/mapper)) → credit `CreateEntries` (type `ENTRY_TYPE_RECEIVED`) → credit `IncrementAccountBalance` (mapped). Every step returns on its own error, so nothing after a failing step executes — this exact property is what the gomock-based unit tests assert via `.Times(0)` on the calls that shouldn't happen.

Sentinel errors, [db/store/errors.go](../db/store/errors.go), in full:

| Error | Meaning |
|---|---|
| `ErrSameAccount` | `from_account_id` and `to_account_id` are identical |
| `ErrCurrencyMismatch` | The two accounts don't share a currency |
| `ErrInvalidAmount` | Transfer amount is not greater than zero |
| `ErrInsufficientFunds` | The from-account's balance is less than the transfer amount |

**Rough edge — an unused entry type.** `constant.ENTRY_TYPE_DEPOSIT` is defined ([domain/constant/entries.go:6](../domain/constant/entries.go)) but never referenced anywhere else in the codebase — only `ENTRY_TYPE_SEND` and `ENTRY_TYPE_RECEIVED` are actually used, both exclusively inside `transferTx`. Reads as scaffolding for a deposit feature that was never wired up.

---

## Section 8 — Testing Without (and With) a Real Database

**The problem this piece solves.** `transferTx`'s branches (same-account, insufficient funds, currency mismatch, lock ordering) need to be tested without depending on Postgres being up, while still having a separate suite that proves the real SQL actually works.

**How it's actually implemented.**

**Mocking.** `make generate-mock` ([Makefile:16-17](../Makefile)) runs:
```
go run go.uber.org/mock/mockgen@latest -package mockdb -destination db/mock/querier.go github.com/DewaSRY/core-service/db/sqlc Querier
```
regenerating [db/mock/querier.go](../db/mock/querier.go) — a `MockQuerier` with one mock method + one `EXPECT()` recorder method per `Querier` method. There's no `//go:generate` directive anywhere in source; regeneration is Makefile-driven only, so it's easy to forget after changing `Querier` — see [GOMOCK_TESTING.md](GOMOCK_TESTING.md) for the mechanics of writing tests against the mock (`gomock.NewController`, `gomock.Any()`, `gomock.InOrder`), which this doc doesn't repeat.

`MockQuerier` is used in exactly three places:

| File | What it tests |
|---|---|
| [db/store/store_transaction_test.go](../db/store/store_transaction_test.go) | `transferTx` directly — no `Store`, no DB |
| [internal/api/auth_router_test.go](../internal/api/auth_router_test.go) | Auth handlers, via a `mockStorer` that embeds `*mockdb.MockQuerier` and stubs `TransferTx` to satisfy the `Storer` interface |
| [internal/api/profile_router_test.go](../internal/api/profile_router_test.go) | Profile handlers, same `mockStorer` pattern |

**Rough edge — the two DB-backed test suites clean up differently; don't assume they behave the same.** Both connect to the same local Postgres instance, but:

- [db/sqlc/main_test.go](../db/sqlc/main_test.go): `createTestQueries(t)` begins a transaction and registers `t.Cleanup(func(){ tx.Rollback() })` — every test in package `db` runs inside a transaction that's always rolled back, so nothing it does ever persists.
- [db/store/main_test.go](../db/store/main_test.go): `newTestStore(t)` returns `NewStore(testDB)` directly against the raw `*sql.DB`, with **no** wrapping transaction. It can't roll back, because `TransferTx` itself opens and commits real transactions internally via `execTx` — a wrapping outer transaction would conflict with that. Integration tests here ([store_transaction_integration_test.go](../db/store/store_transaction_integration_test.go)) instead clean up with explicit `DELETE FROM entries/transfers/accounts WHERE id IN (...)` in `t.Cleanup`, and use goroutines (`TestTransferTxConcurrent`, `TestTransferTxConcurrentReverse`) to actually exercise row locking under real concurrency.

If you're writing a new test under `db/store` expecting automatic rollback the way `db/sqlc` tests get it, it won't happen — you own cleanup.

**Rough edge — a dead reference inside a real test file.** A comment at [store_transaction_integration_test.go:15-18](../db/store/store_transaction_integration_test.go) points to `NEED_TO_IMPOROVE.md` #1/#3 (typo preserved) — no such file exists anywhere in the repo.

**No drift detection.** Nothing in CI enforces that `make sqlc` and `make generate-mock` were both run after a `Querier` change — acknowledged directly in [GOMOCK_TESTING.md:33](GOMOCK_TESTING.md).

---

## Section 9 — Cross-Cutting Coupling: How Handlers Depend on This Layer

Handlers under [internal/api/](../internal/api) never depend on `*sqlc.Queries` or `*store.Store` concretely — only on `Storer`, defined at [internal/api/server.go:24-27](../internal/api/server.go):

```go
type Storer interface {
	db.Querier
	TransferTx(ctx context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error)
}
```

This is what makes `mockStorer` (Section 8) work at all — it's the composition of the generated `Querier` interface with the one hand-written multi-step method. If `Storer` ever grew a method that wasn't backed by either `Querier` or `Store`, both the real `*store.Store` and every mock built around `MockQuerier` would need updating in lockstep — worth knowing before adding a new interface method here.

---

## Section 10 — The Handler Layer (Low Detail — Logic Lives Upstream)

Handlers themselves are thin dispatch, not where sqlc concerns live. Two call styles show up side by side in the same handler, `transactionTransfer` ([internal/api/transactions_router.go:37-75](../internal/api/transactions_router.go)):

- Line 49: `server.store.GetAccountById(ctx, req.FromAccountID)` — a plain sqlc-generated query, used here for an ownership check before the transfer runs.
- Line 71: `server.store.TransferTx(ctx, arg)` — the hand-written multi-step transaction, not sqlc-generated at all.

Elsewhere, list endpoints follow a count-then-list pagination pattern — e.g. [account_router.go:137,150](../internal/api/account_router.go) and [transactions_router.go:157,167](../internal/api/transactions_router.go) call the `Count*` and `List*` sqlc methods as a pair to build a paginated response. There's no logic here worth documenting beyond "it calls the interface" — the interesting behavior is upstream in Sections 6-7.

---

## Summary: Data Flow for a Transfer Request

1. Client hits the transfer endpoint → handler decodes the request, checks ownership via `server.store.GetAccountById` (a direct sqlc call).
2. Handler calls `server.store.TransferTx(ctx, arg)`.
3. `TransferTx` calls `execTx`, which opens a `*sql.Tx` and builds a fresh `sqlc.Queries` bound to it.
4. `transferTx` runs against that `Queries` (as a `Querier`): validates same-account/amount, locks both accounts in ascending-ID order, checks currency and balance, then writes transfer + two entries + two balance updates in sequence.
5. Any error aborts the remaining steps and rolls back the transaction; success commits it.
6. The handler receives `TransferTxResult{Transfer, FromAccount, ToAccount}` and serializes it — `FromAccount`/`ToAccount` arrived via [db/mapper](../db/mapper), translating `IncrementAccountBalanceRow` back into `sqlc.Account`.

Error path: any of `ErrSameAccount`, `ErrCurrencyMismatch`, `ErrInvalidAmount`, `ErrInsufficientFunds` propagates straight back through `execTx` (triggering rollback) to the handler, which maps it to an HTTP error response.

---

## Final Reference: Every `Querier` Method

| Method | Annotation | Source file | Purpose |
|---|---|---|---|
| `CreateAccount` | `:one` | account.sql | Insert a new account |
| `GetAccountById` | `:one` | account.sql | Fetch an account, no lock |
| `GetAccountByIdForUpdate` | `:one` | account.sql | Fetch an account with `FOR UPDATE`, used only inside `transferTx` |
| `IncrementAccountBalance` | `:one` | account.sql | Add a signed delta to `balance` |
| `CheckIsAccountWithIdExist` | `:one` | account.sql | Existence check |
| `ListAccountsByUserId` | `:many` | account.sql | Paginated accounts for a user |
| `CountAccountsByUserId` | `:one` | account.sql | Total count, paired with the above |
| `CreateEntries` | `:one` | entries.sql | Insert a ledger entry (debit or credit) |
| `ListEntriesByAccount` | `:many` | entries.sql | Paginated entries for an account |
| `CountEntriesByAccount` | `:one` | entries.sql | Total count, paired with the above |
| `CreateTransfer` | `:one` | transfers.sql | Insert a transfer record |
| `GetTransferById` | `:one` | transfers.sql | Fetch a transfer by id |
| `ListTransfersByOwner` | `:many` | transfers.sql | Paginated transfers where the owner is either party |
| `CountTransfersByOwner` | `:one` | transfers.sql | Total count, paired with the above |
| `CreateUser` | `:one` | user.sql | Insert a user; deliberately doesn't return `hashed_password` |
| `GetUserByEmail` | `:one` | user.sql | Login lookup, includes `hashed_password` |
| `GetUserById` | `:one` | user.sql | Profile lookup, excludes `hashed_password` |
| `CheckIsUsernameExist` | `:one` | user.sql | Existence check |

Plus one method not on `Querier` at all: `TransferTx` (on `Storer`/`Store`), the hand-written composition of several of the above inside a transaction — see [Section 7](#section-7--business-logic-transfertx-dbstorestore_transactiongo).
