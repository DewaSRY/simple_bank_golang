# sqlc Row-to-Response Mapping — As Implemented

## Who this doc is for

Anyone who's just been bitten by sqlc's default behavior: add or rename a query, run `sqlc generate`, and discover you now need to hand-write a brand-new mapper function because sqlc minted a brand-new Go struct for it — even when the query selects exactly the same columns as three other queries already do. You don't need prior sqlc experience beyond what [SQLC_SETUP.md](SQLC_SETUP.md) covers, but this doc **supersedes** two specific claims in that file: its "Gotcha #1" (`db/mapper` exists because of per-query `Row` structs) and its whole "Section 5 — `db/mapper`" no longer describe this codebase — `internal/db/mapper` was deleted as part of the change this doc describes. Everything below is verified against the source in this repo as of `7ffd894` (2026-09-14), file:line cited throughout. It's not a design doc — one genuine bug was found and fixed along the way, and it's called out explicitly (§4) rather than folded silently into "before/after."

---

## 0. Background primer — why sqlc mints a new struct per query

sqlc's default rule is simple and, once you know it, unsurprising: **a query's result type is a struct matching exactly the columns that query's `SELECT`/`RETURNING` list names, in that order.** Two queries against the same table almost never get the same Go type, because almost no two hand-written column lists are byte-for-byte identical — miss `updated_at` in one and include it in another, and sqlc treats them as unrelated shapes.

| Query shape | What sqlc generates | Consequence |
|---|---|---|
| `SELECT id, balance, ... FROM accounts` (explicit columns) | A **new** `<QueryName>Row` struct, every time | A hand-written mapper is needed anywhere the caller wants the canonical model type |
| `SELECT *` / `RETURNING *` from **one** unaliased table, no joins | Reuses that table's **existing model struct** — no new type | Caller gets the canonical type directly, no mapper needed |
| `SELECT sqlc.embed(v) FROM some_view v ...` | A wrapper struct with **one field**, the view/table's model type nested inside | Every query using `sqlc.embed()` on the same view produces a Row wrapping the *same* nested type — one mapper, reused forever |
| A genuinely multi-table, ad hoc shape (a `JOIN` producing columns no single table has) | Its own one-off struct, unavoidably | Correctly needs its own one-off mapper — this is the *legitimate* case, not the pain point |

The pain this codebase had wasn't "sqlc generates structs" — that's normal — it was that most of its queries fell into the **first** row of that table when they could have fallen into the second or third. `ListMeAccountsByUserId`, `ListAccountsSearchByUserNumber`, and `GetAccountViewById` all read `account_user_details_view` via an explicit, identical column list, so `internal/api/account_response.go` carried three nearly-identical hand-written mapper functions (`toAccountuserResponse`, `toAccountuserResponseFromSearch`, `toPublicAccountResponse`) that only existed because sqlc couldn't tell the three queries produced the same shape.

---

## 1. Architecture at a glance

There's no single composition root for this — the fix is a pattern applied query-by-query in [db/query/*.sql](../internal/db/query), then consumed the same way at every call site. The shape is:

```
db/query/*.sql                          <- rewritten to SELECT *, RETURNING *, or sqlc.embed(v)
        │  sqlc generate (make sqlc)
        ▼
db/sqlc/*.sql.go                        <- Row types now wrap or ARE the shared model
        │
        ▼
internal/api/account_response.go        <- one mapper per shared model, not per query
internal/db/store/*.go                  <- direct assignment, no mapper package at all
```

| Concern | Owner (file) | Analogy |
|---|---|---|
| Deciding which queries can share a type | [db/query/account.sql](../internal/db/query/account.sql), [account-manage.sql](../internal/db/query/account-manage.sql), [account-transaction.sql](../internal/db/query/account-transaction.sql), [entries.sql](../internal/db/query/entries.sql) | Choosing which forms in a paperwork stack are actually the same form |
| Row → response translation (accounts_user_details_view) | `toAccountuserResponse` (`internal/api/account_response.go:62`) | One translator fluent in the view's shape, called by every query that speaks it |
| Row → response translation (account_entries_view) | `toAccountEntriesViewResponse` (`internal/api/account_response.go:119`) | Same idea, for the entries view |
| Repeating a single-item mapper over a slice | `mapSlice[T, R any]` (`internal/api/account_response.go:10`) | A generic "for each" that replaces one hand-written loop per response shape |
| Row → model translation inside the store layer | *(deleted — `internal/db/mapper` no longer exists)* | The adapter plug that's no longer needed because both sockets are now the same shape |

This is split the way it is because the fix operates at two independent layers that happened to have the same disease: `internal/api/account_response.go` was translating DB rows into **HTTP response DTOs**, while the now-deleted `internal/db/mapper` package was translating DB rows into **`sqlc` model types** for the store layer. Both existed only to paper over sqlc minting extra structs — fixing the query shapes at the source removed the need for either kind of translation in most cases (§2, §3), while the response-DTO mappers that remain now serve their *real* purpose: JSON shaping (`sql.NullString` → `string`, time formatting), not row-shape reconciliation.

---

## 2. Fix 1 — `SELECT *` / `RETURNING *` to reuse table models directly

**The problem this solves.** `CreateAccount`, `UpdateAccountNumber`, `UpdateAccount`, `SoftDeleteAccount`, `GetAccountById`, `GetMainAccountByUserId`, `GetAccountByIdForUpdate`, and `IncrementAccountBalance` all queried or wrote the `accounts` table with an explicit column list that happened to match every column `Account` has except `updated_at`/`deleted_at`. Each got its own `Row` struct (`CreateAccountRow`, `UpdateAccountNumberRow`, ...), and `internal/db/mapper/account.go` existed purely to convert six of those seven back into `sqlc.Account` by hand, one function per Row type, all copying the same eight fields. The same pattern existed for `entries` — `CreateEntries`'s explicit `RETURNING` list produced `CreateEntriesRow`, needing `mapper.CreateEntriesRowToEntry`.

**How it's actually implemented.** Every one of those queries now uses `SELECT *` or `RETURNING *`:

```sql
-- internal/db/query/account.sql
-- name: GetAccountById :one
SELECT *
FROM accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateAccountNumber :one
UPDATE accounts
SET number = $2, updated_at = now()
WHERE id = $1
RETURNING *;
```

Confirmed empirically (not just from sqlc's docs) before rewriting every call site: sqlc only skips minting a `Row` type when the `SELECT`/`RETURNING` list is `*` (or exactly enumerates every column, in table order) against a **single, unaliased** table with **no joins**. Under that condition it emits the method signature directly against the table's existing model:

```go
// internal/db/sqlc/account.sql.go
func (q *Queries) GetAccountById(ctx context.Context, id int64) (Account, error)
func (q *Queries) UpdateAccountNumber(ctx context.Context, arg UpdateAccountNumberParams) (Account, error)
func (q *Queries) IncrementAccountBalance(ctx context.Context, arg IncrementAccountBalanceParams) (Account, error)
```

Same trick for `entries` (`internal/db/query/entries.sql` — `CreateEntries` now `RETURNING *`, returns `Entry` directly).

**Consequence at every call site.** `internal/db/mapper` had exactly one caller pattern — `mapper.XyzRowToAccount(row)` right before assigning into a result struct — and every one of those calls was deleted outright, not replaced with something else:

```go
// internal/db/store/store_account_deposit_tx.go:65-72 — before had mapper.UpdateBalanceAccountToAccount(updated)
updated, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
	ID:      arg.AccountID,
	Balance: arg.Amount,
})
if err != nil {
	return result, err
}
result.Account = updated
```

`DepositTxResult.Entry` (`internal/db/store/store_account_deposit_tx.go:21`) changed type from `sqlc.CreateEntriesRow` to `sqlc.Entry` for the same reason — it's a public struct field, so this is a breaking change for anything outside this package that named the old type explicitly (nothing in this codebase did; checked by grep before merging).

**The whole `internal/db/mapper` package was deleted.** Once every one of its seven functions had zero remaining callers, the package had zero reason to exist. This is the one instance in this refactor where a whole file — not just a function — was deleted rather than rewritten.

---

## 3. Fix 2 — `sqlc.embed()` to share one type across queries on the same view

**The problem this solves.** `accounts` is a real table, so "widen the `SELECT` to `*`" (§2) makes sense there. But `ListMeAccountsByUserId`, `ListAccountsSearchByUserNumber`, and `GetAccountViewById` all read `account_user_details_view` — and `AccountEntriesByAccountId` / `ListAccountEntriesByAccountId` both read `account_entries_view`. sqlc **already** generates one model struct per view (`AccountUserDetailsView`, `AccountEntriesView` in `internal/db/sqlc/db.go`, since a `CREATE VIEW` is schema sqlc introspects same as a table) — but none of these five queries selected `*` from their view, so none of them triggered the reuse-the-model behavior from §2. Each got its own `Row` type, and `internal/api/account_response.go` carried three separate near-identical mappers for the user-details shape and two for the entries shape.

**How it's actually implemented.** Alias the view and select `sqlc.embed(alias)` instead of naming columns:

```sql
-- internal/db/query/account-manage.sql
-- name: ListMeAccountsByUserId :many
SELECT sqlc.embed(v)
FROM account_user_details_view v
WHERE v.user_id = sqlc.arg(user_id)
    AND v.name LIKE '%' || sqlc.arg(name) || '%'
ORDER BY v.created_at DESC
LIMIT  sqlc.arg(limit_count) OFFSET  sqlc.arg(offset_count);
```

This generates a Row type with **one field**, the embedded model, not a flattened struct:

```go
// internal/db/sqlc/account-manage.sql.go
type ListMeAccountsByUserIdRow struct {
	AccountUserDetailsView AccountUserDetailsView `json:"account_user_details_view"`
}
```

`ListAccountsSearchByUserNumberRow` and `GetAccountViewByIdRow` come out shaped identically — same single field, same nested type — because they embed the same view. `AccountEntriesByAccountIdRow` and `ListAccountEntriesByAccountIdRow` do the same for `AccountEntriesView`.

**One mapper per view, not per query.** `internal/api/account_response.go` now has exactly one function per embedded model:

```go
// internal/api/account_response.go:62
func toAccountuserResponse(account db.AccountUserDetailsView) accountuserResponse { ... }

// internal/api/account_response.go:119
func toAccountEntriesViewResponse(entry db.AccountEntriesView) accountEntriesViewResponse { ... }
```

Every call site unwraps the one embedded field before calling it — this is the only extra step compared to before:

```go
// internal/api/account_router.go:111-112
responses := mapSlice(accounts, func(row db.ListMeAccountsByUserIdRow) accountuserResponse {
	return toAccountuserResponse(row.AccountUserDetailsView)
})
```

```go
// internal/api/account_manage_router.go:50,55
if account.AccountUserDetailsView.UserID.Int64 != authPayload.ID { ... }
succeed(ctx, http.StatusOK, toAccountuserResponse(account.AccountUserDetailsView), "Account retrieved successfully")
```

**What this buys the next query.** A hypothetical `ListArchivedAccountsByUserId` selecting `sqlc.embed(v)` from `account_user_details_view` would produce `ListArchivedAccountsByUserIdRow{ AccountUserDetailsView AccountUserDetailsView }` — same field, same type. Its handler calls `toAccountuserResponse(row.AccountUserDetailsView)` and is done. No new mapper function, ever, for a query against a view that already has one.

**The exception, and why it's not the same problem.** `ListRecentTransferDestinations` (`internal/db/query/account-transaction.sql:1`) joins `transfers`, `accounts`, and `users` into a shape no single table or view has — a `DISTINCT ON` subquery producing `(id, name, number, username, user_id, last_used_at)`. This correctly keeps its own `ListRecentTransferDestinationsRow` type and its own one-off mapper, `toPublicAccountResponseFromDestination` (`internal/api/account_response.go:89`). The same is true of `ListAccountTransactionHistory` in `entries.sql`. **This is the legitimate case §0's table describes** — a truly ad hoc shape gets a truly one-off mapper, and that's correct sqlc usage, not a gap to close.

---

## 4. A bug found and fixed along the way: `deleted_at` on a view that never had it

While rewriting `ListAccountsSearchByUserNumber` and `CountAccountsSearchByUserNumber` to use `sqlc.embed()`, both queries turned out to filter `WHERE ... AND deleted_at IS NULL` against `account_user_details_view` — a view whose `SELECT` list (`internal/db/migrations/000009_account_view_table.up.sql`) never includes `deleted_at`. Verified directly against the running Postgres container, not just inferred from the view definition:

```
$ psql ... -c "SELECT id FROM account_user_details_view WHERE ... AND deleted_at IS NULL LIMIT 1;"
ERROR:  column "deleted_at" does not exist
```

**sqlc's own `generate` step doesn't catch this** — both the pre-fix and post-fix versions of the query generate cleanly with no warning, because sqlc's static analysis for this project's config doesn't validate every `WHERE` clause against a real Postgres backend the way an actual query execution would. In other words: `sqlc generate` succeeding is not proof a query is valid SQL. This means **every call to `GET /accounts/search-by-number` was failing with a 500** before this fix — worth knowing if anyone assumed that endpoint was working.

The fix removes the redundant filter rather than adding the column to the view: `account_user_details_view` already has `WHERE a.deleted_at IS NULL` baked into its own definition, so soft-deleted accounts were already excluded — the extra filter was dead-on-arrival redundancy that happened to reference a non-existent column, not a missing feature.

---

## 5. The `mapSlice` helper — one generic instead of N identical loops

**The problem this solves.** Before this change, every `[]Row -> []Response` conversion was its own hand-written loop function: `toListAccountuserResponse`, `toListAccountuserResponseFromSearch`, `toListAccountEntriesViewResponse`, `ListRecentTransferDestinationsToPublicAccountResponse` — four functions whose bodies were identical except for the element types and the single-item mapper called inside.

**How it's actually implemented**, in full (`internal/api/account_response.go:10-16`):

```go
func mapSlice[T, R any](items []T, f func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = f(item)
	}
	return result
}
```

Every list endpoint now calls this directly with an inline closure (or, where the element types already match, the mapper function by name):

```go
// internal/api/account_transactions_router.go:227
responses := mapSlice(destinations, toPublicAccountResponseFromDestination)
```

```go
// internal/api/account_transactions_router.go:165-166
entries := mapSlice(accountEntries, func(row db.ListAccountEntriesByAccountIdRow) accountEntriesViewResponse {
	return toAccountEntriesViewResponse(row.AccountEntriesView)
})
```

This didn't reduce the *number* of mapping decisions (each list endpoint still says exactly how to map one row), but it deleted four functions whose only content was "loop and call the other function," and it means a fifth list endpoint against an existing embedded type needs zero new looping code.

---

## 6. Playbook: adding a new query without writing a new mapper

Given everything above, the actual answer to "I keep having to write a new mapper" is a checklist to run before accepting a query's generated `Row` type as unavoidable:

1. **Does the query select every column of exactly one table, with no joins?** Use `SELECT *` / `RETURNING *`. sqlc will hand you the table's existing model type directly — no `Row`, no mapper, ever (§2).
2. **Does the query select from a view (or table) that other queries already read, and does an embeddable model for it already exist in `internal/db/sqlc/db.go`?** Alias it and use `SELECT sqlc.embed(alias)`. Write (or reuse) exactly one mapper for that embedded model type; every query embedding the same view calls the same mapper via `row.<EmbeddedFieldName>` (§3).
3. **Only if the shape is genuinely unique** — a join producing columns no single table/view has — accept the one-off `Row` type and write a one-off mapper. This is correct, not a workaround (§3's "exception" callout).
4. **For a `:many` query**, reach for `mapSlice(rows, yourMapper)` (§5) instead of writing a new `toListX` loop function.

---

## Known gaps (out of scope for this change, called out rather than silently left)

- **`SQLC_SETUP.md` is stale beyond this doc's scope.** That file's Gotcha #1 and Section 5 describe the now-deleted `internal/db/mapper` package as current; its file paths (`db/sqlc` instead of `internal/db/sqlc`) and some table columns (an `Owner` field that no longer exists) predate a larger restructuring this change didn't touch. This doc supersedes those two specific claims only — the rest of that file's drift is a separate, larger cleanup.
- **Five pre-existing test failures**, unrelated to this change and confirmed identical on `main` before it (verified by stashing this work and re-running `go test ./...` against commit `7ffd894`): `TestGetProfile` (the `/auth/profile` route 404s), `TestTransactionTransfer`, `TestDeposit`, and `TestListAccountTransactionHistory` (all reach handler code for a `ListAccountTransactionHistory`/transaction-history feature that has no route registered in `internal/api/router.go` — dead code, not wired up), and `TestListRecentTransferDestinations` (a default-limit mismatch, 10 vs. 5, unrelated to row mapping). None of this change's edits touch their code paths; none of them regressed or were fixed by this change.
- **`transactionHistoryItem`/`transactionHistoryCounterparty` and `toTransactionHistoryItem`** (`internal/api/account_response.go:142`, `internal/api/account_mappers.go`) are dead code with no handler calling them, consistent with the unwired route above.

## Final reference: every query touched by this change

| Query | File | Row type before | Row type after |
|---|---|---|---|
| `CreateAccount` | account.sql | `CreateAccountRow` | `Account` (reused) |
| `UpdateAccountNumber` | account.sql | `UpdateAccountNumberRow` | `Account` (reused) |
| `UpdateAccount` | account.sql | `UpdateAccountRow` | `Account` (reused) |
| `SoftDeleteAccount` | account.sql | `SoftDeleteAccountRow` | `Account` (reused) |
| `GetAccountById` | account.sql | `GetAccountByIdRow` | `Account` (reused) |
| `GetMainAccountByUserId` | account.sql | (explicit column list) | `Account` (reused) |
| `GetAccountByIdForUpdate` | account-transaction.sql | `GetAccountByIdForUpdateRow` | `Account` (reused) |
| `IncrementAccountBalance` | account-transaction.sql | `IncrementAccountBalanceRow` | `Account` (reused) |
| `CreateEntries` | entries.sql | `CreateEntriesRow` | `Entry` (reused) |
| `ListMeAccountsByUserId` | account-manage.sql | (explicit column list) | `{AccountUserDetailsView}` (embed) |
| `ListAccountsSearchByUserNumber` | account.sql | (explicit column list + buggy filter, §4) | `{AccountUserDetailsView}` (embed, bug fixed) |
| `GetAccountViewById` | account-transaction.sql | (explicit column list) | `{AccountUserDetailsView}` (embed) |
| `AccountEntriesByAccountId` | account-transaction.sql | (explicit column list) | `{AccountEntriesView}` (embed) |
| `ListAccountEntriesByAccountId` | account-transaction.sql | (explicit column list) | `{AccountEntriesView}` (embed) |
| `ListRecentTransferDestinations` | account-transaction.sql | own `Row` (unchanged) | own `Row` (unchanged — legitimate, §3) |
