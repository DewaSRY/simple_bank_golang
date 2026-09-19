# sqlc Row-to-Response Mapping — As Implemented

## Who this doc is for

Anyone who's just been bitten by sqlc's default behavior: add or rename a query, run `sqlc generate`, and discover you now need to hand-write a brand-new mapper function because sqlc minted a brand-new Go struct for it — even when the query selects exactly the same columns as another query already does. You don't need prior sqlc experience beyond what [SQLC_SETUP.md](SQLC_SETUP.md) covers. This doc **supersedes** two specific claims in that file: its "Gotcha #1" (`db/mapper` exists because of per-query `Row` structs) and its whole "Section 5 — `db/mapper`" no longer describe this codebase — `internal/db/mapper` was deleted as part of the change this doc describes, and stays deleted. Everything below is verified against the source in this repo as of `13f5fc4` (2026-09-19), file:line cited throughout.

**This is the second revision of this doc.** The first revision was written mid-refactor, when `internal/api` was still one flat package (`internal/api/account_response.go`, `internal/api/account_router.go`, ...). That refactor has since finished: handlers, response DTOs and mappers now live in per-domain packages (`internal/api/account`, `internal/api/transfer`, `internal/api/auth`) as described in [AGENTS.md](../AGENTS.md). Every file path below reflects the current, post-split layout — if you find an older copy of this doc (or a stale mention elsewhere) citing `internal/api/account_response.go` or `internal/api/account_router.go`, those files no longer exist. Two claims the first revision made under "Known gaps" are also no longer true and are corrected in §7 below: the transaction-history route is now wired up, and `/auth/profile` no longer 404s.

---

## 0. Background primer — why sqlc mints a new struct per query

sqlc's default rule is simple and, once you know it, unsurprising: **a query's result type is a struct matching exactly the columns that query's `SELECT`/`RETURNING` list names, in that order.** Two queries against the same table almost never get the same Go type, because almost no two hand-written column lists are byte-for-byte identical — miss `updated_at` in one and include it in another, and sqlc treats them as unrelated shapes.

| Query shape | What sqlc generates | Consequence |
|---|---|---|
| `SELECT id, balance, ... FROM accounts` (explicit, partial or reordered column list) | A **new** `<QueryName>Row` struct, every time | A hand-written mapper is needed anywhere the caller wants the canonical model type |
| `SELECT *` / `RETURNING *`, or an explicit list that names every column of **one** unaliased table in table order, no joins | Reuses that table's **existing model struct** — no new type | Caller gets the canonical type directly, no mapper needed |
| `SELECT sqlc.embed(v) FROM some_view v ...` | A wrapper struct with **one field**, the view/table's model type nested inside | Every query using `sqlc.embed()` on the same view produces a Row wrapping the *same* nested type — one mapper, reused forever |
| A genuinely multi-table, ad hoc shape (a `JOIN` producing columns no single table has) | Its own one-off struct, unavoidably | Correctly needs its own one-off mapper — this is the *legitimate* case, not the pain point |

The second row's "or an explicit list that names every column in table order" clause isn't hypothetical — `transfers.sql`'s `CreateTransfer`/`GetTransferById` spell out `id, from_account_id, to_account_id, amount, created_at, description` by hand rather than using `RETURNING *`/`SELECT *`, and because that list is a byte-for-byte, in-order match for the `Transfer` struct in [models.go](../internal/db/sqlc/models.go), sqlc still reuses `Transfer` directly (`internal/db/sqlc/transfers.sql.go:32`, `:57`) — no `CreateTransferRow` is generated. `SELECT *` is the easiest way to land in this bucket, but it isn't the only door.

The pain this codebase had (pre-refactor) wasn't "sqlc generates structs" — that's normal — it was that most of its account queries fell into the **first** row of that table when they could have fallen into the second or third. `ListMeAccountsByUserId`, `ListAccountsSearchByUserNumber`, and `GetAccountViewById` all read `account_user_details_view` via an explicit, identical column list, so the (now-deleted) flat `internal/api` package carried three nearly-identical hand-written mapper functions that only existed because sqlc couldn't tell the three queries produced the same shape.

**Not every query in this codebase has been moved out of the first bucket, and that's fine.** `user.sql`'s `CreateUser`, `GetUserByEmail`, and `GetUserById` each select a different subset of `users` columns (e.g. `GetUserByEmail` needs `hashed_password` for auth, `GetUserById` doesn't), so each still gets its own `Row` type (`CreateUserRow`, `GetUserByEmailRow`, `GetUserByIdRow` — `internal/db/sqlc/user.sql.go`). `internal/api/auth/handlers.go` reads fields off these Row types directly (`user.ID`, `user.HashedPassword`, ...) with no mapper function at all, because `auth` never needs a shared response DTO across queries the way `account`/`transfer` do — three different-shaped `Row`s each get used inline, once. This is the "legitimate first bucket" case: the columns genuinely differ per query, so a shared type wouldn't fit even if you wanted one.

---

## 1. Architecture at a glance

There's no single composition root for this — it's a pattern applied query-by-query in [db/query/*.sql](../internal/db/query), then consumed the same way at every call site that needs it. The shape is:

```
internal/db/query/*.sql                 <- written to SELECT *, RETURNING *, or sqlc.embed(v)
        │  sqlc generate (make sqlc)
        ▼
internal/db/sqlc/*.sql.go               <- Row types now wrap or ARE the shared model
        │
        ├──────────────────────────────────────────────┐
        ▼                                               ▼
internal/api/<domain>/response.go        internal/db/store/*.go
   one mapper per shared model,             direct field assignment,
   not per query                            no mapper package at all
```

| Concern | Owner (file) | Analogy |
|---|---|---|
| Deciding which queries can share a type | [db/query/account.sql](../internal/db/query/account.sql), [account-manage.sql](../internal/db/query/account-manage.sql), [account-transaction.sql](../internal/db/query/account-transaction.sql), [entries.sql](../internal/db/query/entries.sql) | Choosing which forms in a paperwork stack are actually the same form |
| Row → response translation (`account_user_details_view`) | `toAccountuserResponse` (`internal/api/account/response.go:51`) | One translator fluent in the view's shape, called by every query that speaks it |
| Row → response translation (`accounts` table) | `toAccountResponse` (`internal/api/account/response.go:19`) | Same idea, for the plain table model |
| Row → response translation (`account_entries_view`) | `toAccountEntriesViewResponse` (`internal/api/transfer/response.go:49`) | Same idea again, for the entries view, but owned by a different domain package |
| Repeating a single-item mapper over a slice | `util.MapSlice[T, R any]` (`internal/util/array.go:6`) | A generic "for each" that replaces one hand-written loop per response shape |
| Row → model translation inside the store layer | *(doesn't exist — `internal/db/mapper` was deleted)* | The adapter plug that's no longer needed because both sockets are now the same shape |

This is split the way it is because the underlying problem shows up at two independent layers that happened to have the same disease: `internal/api/<domain>/response.go` files translate DB rows into **HTTP response DTOs**, while the store layer (`internal/db/store/*.go`) needs DB rows as **`sqlc` model types** to build multi-step transaction results (`TransferTxResult`, `DepositTxResult`, `DeleteAccountTxResult` — see `internal/db/store/store_transaction.go`, `store_account_deposit_tx.go`, `store_account.go`). Both used to need a translation step purely to paper over sqlc minting extra structs; fixing the query shapes at the source removed the need for a mapper at the store layer entirely (§2, §3), while the response-DTO mappers that remain now serve their *real* purpose — JSON shaping (`sql.NullString` → `string`, time formatting) — not row-shape reconciliation. Worth noting going in: `util.MapSlice` lives in the *generic-helpers* package (`internal/util`), not alongside the response DTOs it's usually called next to — a new domain package reaches for it with an import, not a copy-paste.

---

## 2. Fix 1 — `SELECT *` / `RETURNING *` (or an exhaustive, in-order column list) to reuse table models directly

**The problem this solves.** Any query returning the `accounts`, `entries`, `transfers`, or (partially) `users` table with an explicit, non-exhaustive column list gets its own one-off `Row` struct, which then needs a hand-written function to convert it back to the canonical model whenever a caller needs that shape.

**How it's actually implemented.** Every `accounts`-table write/read in [account.sql](../internal/db/query/account.sql) and [account-transaction.sql](../internal/db/query/account-transaction.sql) now uses `SELECT *` or `RETURNING *`:

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

sqlc only skips minting a `Row` type when the `SELECT`/`RETURNING` list is `*` (or exactly enumerates every column, in table order) against a **single, unaliased** table with **no joins**. Under that condition it emits the method signature directly against the table's existing model:

```go
// internal/db/sqlc/account.sql.go / account-transaction.sql.go
func (q *Queries) GetAccountById(ctx context.Context, id int64) (Account, error)
func (q *Queries) UpdateAccountNumber(ctx context.Context, arg UpdateAccountNumberParams) (Account, error)
func (q *Queries) IncrementAccountBalance(ctx context.Context, arg IncrementAccountBalanceParams) (Account, error)
func (q *Queries) GetAccountByIdForUpdate(ctx context.Context, id int64) (Account, error)
```

Same trick for `entries` (`CreateEntries` — `internal/db/query/entries.sql` — `RETURNING *`, returns `Entry` directly), and the exhaustive-column-list variant for `transfers` (`CreateTransfer`/`GetTransferById` — `internal/db/query/transfers.sql` — spell out all six `Transfer` columns in order, still reuses `Transfer`, no `Row` type — see §0).

**Consequence at every call site: direct assignment, no mapper, anywhere in the store layer.** `internal/db/store` never converts a `Row` into a model — every `*Tx` function assigns the query result straight into its result struct:

```go
// internal/db/store/store_account_deposit_tx.go:56-72
result.Entry, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{ ... })
...
updated, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
	ID:      arg.AccountID,
	Balance: arg.Amount,
})
if err != nil {
	return result, err
}
result.Account = updated
```

`DepositTxResult.Entry` (`internal/db/store/store_account_deposit_tx.go:21`) and `TransferTxResult`'s `Transfer`/`FromAccount`/`ToAccount`/`FromEntry`/`ToEntry` fields (`internal/db/store/store_transaction.go:13-19`) are all plain `sqlc.Entry`/`sqlc.Account`/`sqlc.Transfer` — never a query-specific `Row` type. `deleteAccountTx` (`internal/db/store/store_account.go`) follows the same pattern for its sweep-then-soft-delete logic.

**`internal/db/mapper` doesn't exist, and there's nothing pulling it back into existence.** A `git log`/`ls` check today confirms no such package exists anywhere in `internal/db`. If you're tempted to add a `RowToModel` conversion function while wiring up a new query, that's the signal to widen the `SELECT` instead (see §6's checklist).

---

## 3. Fix 2 — `sqlc.embed()` to share one type across queries on the same view

**The problem this solves.** `accounts` is a real table, so "widen the `SELECT` to `*`" (§2) makes sense there. But `ListMeAccountsByUserId`, `ListAccountsSearchByUserNumber`, and `GetAccountViewById` all read `account_user_details_view` (defined in migration [000009_account_view_table.up.sql](../internal/db/migrations/000009_account_view_table.up.sql)) — and `AccountEntriesByAccountId`/`ListAccountEntriesByAccountId` both read `account_entries_view` (migration [000010_create-account_entries_table_view.up.sql](../internal/db/migrations/000010_create-account_entries_table_view.up.sql)). sqlc **already** generates one model struct per view (`AccountUserDetailsView`, `AccountEntriesView` in `internal/db/sqlc/models.go`, since a `CREATE VIEW` is schema sqlc introspects same as a table) — a query only gets to reuse that struct if it selects `sqlc.embed(alias)` rather than naming columns.

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

**One mapper per view, not per query — but the two views' mappers live in two different domain packages, because their response DTOs belong to two different domains:**

```go
// internal/api/account/response.go:51 — every query embedding account_user_details_view calls this
func toAccountuserResponse(account db.AccountUserDetailsView) accountuserResponse { ... }

// internal/api/transfer/response.go:49 — every query embedding account_entries_view calls this
func toAccountEntriesViewResponse(entry db.AccountEntriesView) accountEntriesViewResponse { ... }
```

Every call site unwraps the one embedded field before calling it — this is the only extra step compared to a query that returns the model directly:

```go
// internal/api/account/handlers.go:114-116 — listmeAccounts
responses := util.MapSlice(accounts, func(row db.ListMeAccountsByUserIdRow) accountuserResponse {
	return toAccountuserResponse(row.AccountUserDetailsView)
})
```

```go
// internal/api/account/manage.go:57 — detailAccount, a single row (not a slice), same unwrap
core.Succeed(ctx, http.StatusOK, toAccountuserResponse(account.AccountUserDetailsView), "Account retrieved successfully")
```

```go
// internal/api/transfer/transactions_handlers.go:91 — transactionTransfer reads back the entry it just wrote
core.Succeed(ctx, http.StatusOK, toAccountEntriesViewResponse(accountEntries.AccountEntriesView), "Transfer completed successfully")
```

**What this buys the next query.** A hypothetical `ListArchivedAccountsByUserId` selecting `sqlc.embed(v)` from `account_user_details_view` would produce `ListArchivedAccountsByUserIdRow{ AccountUserDetailsView AccountUserDetailsView }` — same field, same type. Its handler calls `toAccountuserResponse(row.AccountUserDetailsView)` and is done. No new mapper function, ever, for a query against a view that already has one — provided the new handler lives in (or imports from) the domain package that already owns that view's mapper; see the callout below.

**A sharp edge worth knowing before you add a domain's third view-backed query: the mapper's package is a real dependency edge, not just a file location.** `toAccountuserResponse` and its `accountuserResponse` DTO are unexported (`internal/api/account/response.go`) — a query added to `internal/api/transfer` that also embeds `account_user_details_view` cannot call `toAccountuserResponse` without either exporting it or duplicating it. This hasn't happened yet (each view's mapper today has exactly one domain package consuming it), but it's the first thing to check before assuming "one mapper per view" holds for a *cross-domain* view.

**The exception, and why it's not the same problem.** `ListRecentTransferDestinations` (`internal/db/query/account-transaction.sql`) joins `transfers`, `accounts`, and `users` into a shape no single table or view has — a `DISTINCT ON` subquery producing `(id, name, number, username, user_id, last_used_at)`. This correctly keeps its own `ListRecentTransferDestinationsRow` type and its own one-off mapper, `toPublicAccountResponseFromDestination` (`internal/api/transfer/response.go:19`). `ListAccountTransactionHistory` in `entries.sql` is the same story — a `LEFT JOIN` against `transfers`/`accounts` to resolve a counterparty, its own `Row` type, its own mapper (§7). **This is the legitimate case §0's table describes** — a truly ad hoc shape gets a truly one-off mapper, and that's correct sqlc usage, not a gap to close.

---

## 4. A bug found and fixed along the way: `deleted_at` on a view that never had it

While rewriting `ListAccountsSearchByUserNumber` and `CountAccountsSearchByUserNumber` to use `sqlc.embed()`, both queries turned out to filter `WHERE ... AND deleted_at IS NULL` against `account_user_details_view` — a view whose `SELECT` list (migration [000009_account_view_table.up.sql](../internal/db/migrations/000009_account_view_table.up.sql)) never includes `deleted_at`. Verified directly against a running Postgres container, not just inferred from the view definition:

```
$ psql ... -c "SELECT id FROM account_user_details_view WHERE ... AND deleted_at IS NULL LIMIT 1;"
ERROR:  column "deleted_at" does not exist
```

**sqlc's own `generate` step doesn't catch this** — both the pre-fix and post-fix versions of the query generate cleanly with no warning, because sqlc's static analysis for this project's config doesn't validate every `WHERE` clause against a real Postgres backend the way an actual query execution would. In other words: `sqlc generate` succeeding is not proof a query is valid SQL. This means **every call to `GET /accounts/search-by-number` was failing with a 500** before this fix — worth knowing if you assume that endpoint has always worked.

The fix removed the redundant filter rather than adding the column to the view: `account_user_details_view` already has `WHERE a.deleted_at IS NULL` baked into its own definition, so soft-deleted accounts were already excluded — the extra filter was dead-on-arrival redundancy that happened to reference a non-existent column, not a missing feature. Confirmed still fixed today: neither `ListAccountsSearchByUserNumber` nor `CountAccountsSearchByUserNumber` in [account.sql](../internal/db/query/account.sql) filters on `deleted_at`.

---

## 5. The `MapSlice` helper — one generic instead of N identical loops

**The problem this solves.** Every `[]Row -> []Response` conversion used to be its own hand-written loop function, one per response shape, whose bodies were identical except for the element types and the single-item mapper called inside.

**How it's actually implemented**, in full (`internal/util/array.go:6-12`):

```go
func MapSlice[T, R any](items []T, f func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = f(item)
	}
	return result
}
```

**Worth flagging: this lives in `internal/util`, a generic-helpers package shared across the whole codebase — not in any `internal/api/<domain>` package.** That's a deliberate move from the first revision of this refactor (which had a package-local `mapSlice` in the old flat `internal/api`); a generic slice-mapper has no reason to be per-domain, and putting it in `util` means `account`, `transfer`, and any future domain package import the same function rather than each defining their own copy.

Every list endpoint calls it directly with an inline closure (or, where the element types already match, the mapper function by name):

```go
// internal/api/transfer/handlers.go:227 — element types already match, no closure needed
responses := util.MapSlice(destinations, toPublicAccountResponseFromDestination)
```

```go
// internal/api/transfer/handlers.go:177-179 — unwrapping an embedded view field needs a closure
entries := util.MapSlice(accountEntries, func(row db.ListAccountEntriesByAccountIdRow) accountEntriesViewResponse {
	return toAccountEntriesViewResponse(row.AccountEntriesView)
})
```

This doesn't reduce the *number* of mapping decisions (each list endpoint still says exactly how to map one row), but it means a new list endpoint against an existing embedded type needs zero new looping code — just one `util.MapSlice` call.

---

## 6. Playbook: adding a new query without writing a new mapper

Given everything above, the actual answer to "I keep having to write a new mapper" is a checklist to run before accepting a query's generated `Row` type as unavoidable:

1. **Does the query select every column of exactly one table, with no joins?** Use `SELECT *` / `RETURNING *` (or spell out every column in table order, if you have a reason not to use `*`). sqlc will hand you the table's existing model type directly — no `Row`, no mapper, ever (§2).
2. **Does the query select from a view (or table) that other queries already read, and does an embeddable model for it already exist in `internal/db/sqlc/models.go`?** Alias it and use `SELECT sqlc.embed(alias)`. Write (or reuse) exactly one mapper for that embedded model type in the domain package that owns it; every query embedding the same view calls the same mapper via `row.<EmbeddedFieldName>` (§3) — but check first whether your new query lives in a *different* domain package than the existing mapper (§3's cross-domain callout), since an unexported mapper can't be called across package boundaries.
3. **Only if the shape is genuinely unique** — a join producing columns no single table/view has, or a query that intentionally selects a partial/different column subset than any existing query (like `user.sql`'s three `users` queries, §0) — accept the one-off `Row` type and write a one-off mapper, or just read fields off it inline if only one call site ever needs it. This is correct, not a workaround.
4. **For a `:many` query**, reach for `util.MapSlice(rows, yourMapper)` (§5) instead of writing a new `toListX` loop function.

---

## 7. Known gaps and corrections (out of scope for this change, called out rather than silently left)

- **`SQLC_SETUP.md` is stale beyond this doc's scope** — but it already knows it: its own header carries a "Superseded content ahead" callout pointing back at this doc, and its Gotcha #1 / Section 5 describing `internal/db/mapper` should be read as historical, not current. This doc supersedes those two specific claims only; the rest of that file's drift (paths under `db/` rather than `internal/db/`, a `Account.Owner` field that no longer exists) is out of scope here.
- **Corrected from the first revision of this doc: `ListAccountTransactionHistory` is no longer dead code.** It used to be true that `TestListAccountTransactionHistory`/`TestTransactionTransfer`/`TestDeposit` reached handler code with no route registered. As of the current router (`internal/api/transfer/transfer.go:23` — `rg.GET("/accounts/:id/transactions", h.listAccountTransactionHistory)`), the route exists and is exercised end-to-end (`internal/api/transfer/handlers.go:297`, calling `util.MapSlice(history, toTransactionHistoryItem)`). `transactionHistoryItem`/`toTransactionHistoryItem` (`internal/api/transfer/response.go:72-105`) are live code with a real caller, not the dead code an earlier pass of this doc described.
- **Corrected from the first revision: `/auth/profile` is wired up.** `internal/api/auth/auth.go`'s `RegisterAuthorizedRoutes` registers `rg.GET("/auth/profile", h.getProfile)`, and `getProfile` (`internal/api/auth/handlers.go`) calls `h.Store.GetUserById` and returns a `profileResponse`. If you're relying on an older note (in memory, in a stale doc, or in a teammate's head) that this route 404s, re-verify against the current router before trusting it.
- **`ListAccountTransactionHistory`'s `Row` type is a genuine one-off, not a missed embedding opportunity.** It `LEFT JOIN`s `entries` against `transfers`/`accounts` to resolve a counterparty name/number that no single table or view carries, so `ListAccountTransactionHistoryRow` and `toTransactionHistoryItem` are the §3-exception case, same as `ListRecentTransferDestinationsRow`.
- **The store layer still has no dedicated unit test for its `*Tx` functions.** `internal/db/store/` contains no `_test.go` file at all today (`store.go`, `store_transaction.go`, `store_account.go`, `store_account_create_tx.go`, `store_account_deposit_tx.go`, `errors.go`, `utils.go` — no test alongside them). `transferTx`'s locking order and decimal math (and `deleteAccountTx`'s sweep-then-soft-delete logic) are only exercised indirectly, through the owning domain's `*_handler_test.go`, which mocks the entire `Storer.TransferTx`/`Storer.DeleteAccountTx` call and therefore cannot catch a regression inside the `*Tx` function itself. See [AGENTS.md](../AGENTS.md)'s "Doc drift" section and [GOMOCK_TESTING.md](GOMOCK_TESTING.md) for the same observation from the test-strategy angle — this doc flags it because a future embedding/mapper change to a query a `*Tx` function calls has exactly this blind spot.

## Final reference: every query this pattern applies to

| Query | File | Row type | Shared type used |
|---|---|---|---|
| `CreateAccount` | account.sql | *(none — reuses model)* | `Account` |
| `UpdateAccountNumber` | account.sql | *(none)* | `Account` |
| `UpdateAccount` | account.sql | *(none)* | `Account` |
| `SoftDeleteAccount` | account.sql | *(none)* | `Account` |
| `GetAccountById` | account.sql | *(none)* | `Account` |
| `GetMainAccountByUserId` | account-manage.sql | *(none)* | `Account` |
| `GetAccountByIdForUpdate` | account-transaction.sql | *(none)* | `Account` |
| `IncrementAccountBalance` | account-transaction.sql | *(none)* | `Account` |
| `CreateEntries` | entries.sql | *(none)* | `Entry` |
| `CreateTransfer` | transfers.sql | *(none — exhaustive in-order column list, §0)* | `Transfer` |
| `GetTransferById` | transfers.sql | *(none)* | `Transfer` |
| `ListMeAccountsByUserId` | account-manage.sql | `ListMeAccountsByUserIdRow` | `{AccountUserDetailsView}` (embed) |
| `ListAccountsSearchByUserNumber` | account.sql | `ListAccountsSearchByUserNumberRow` | `{AccountUserDetailsView}` (embed, §4 bug fixed here) |
| `GetAccountViewById` | account-transaction.sql | `GetAccountViewByIdRow` | `{AccountUserDetailsView}` (embed) |
| `AccountEntriesByAccountId` | account-transaction.sql | `AccountEntriesByAccountIdRow` | `{AccountEntriesView}` (embed) |
| `ListAccountEntriesByAccountId` | account-transaction.sql | `ListAccountEntriesByAccountIdRow` | `{AccountEntriesView}` (embed) |
| `ListRecentTransferDestinations` | account-transaction.sql | `ListRecentTransferDestinationsRow` | own one-off shape — legitimate (§3) |
| `ListAccountTransactionHistory` | entries.sql | `ListAccountTransactionHistoryRow` | own one-off shape — legitimate (§7) |
| `CreateUser` | user.sql | `CreateUserRow` | own shape — partial `users` columns (§0) |
| `GetUserByEmail` | user.sql | `GetUserByEmailRow` | own shape — needs `hashed_password` (§0) |
| `GetUserById` | user.sql | `GetUserByIdRow` | own shape — excludes `hashed_password` (§0) |
