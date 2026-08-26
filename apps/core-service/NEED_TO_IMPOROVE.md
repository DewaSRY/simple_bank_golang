# core-service — Improvement Notes

Findings from a full-codebase pass, ordered by priority. File:line references point to current `main`.

## 🔴 Critical

### 1. Transfers overwrite balance instead of adjusting it
`db/store/store_transaction.go` (`transferTx`) and the underlying query in `db/query/account.sql`:

```sql
UPDATE accounts SET balance = $2, updated_at = now() WHERE id = $1
```

`IncrementAccountBalance` does an absolute `SET`, not `balance = balance + $2`. In `transferTx`, the "from" account is updated with `-amount` and the "to" account with `+amount` — meaning **every transfer replaces the account's balance with the transfer amount itself**, rather than debiting/crediting it.

Example: an account with $1000 that sends two $50 transfers ends up at **-$50**, not $900.

Fix: change the query to `balance = balance + $2` (add a separate signed amount param), or add an `AddAccountBalance` query used by both legs with `+amount`/`-amount`. The existing unit test (`db/store/store_transaction_test.go`) uses mocked return values and doesn't exercise real arithmetic, so it didn't catch this — add an integration/concurrency test (see Testing section) that starts from a known balance and asserts the final balance after N concurrent transfers.

### 2. No authentication or authorization
Confirmed via full-repo grep (`jwt|paseto|auth|middleware`) — zero matches. `POST /accounts`, `GET /accounts/:id`, and `POST /transactions/transfer` are all open: anyone with network access can create accounts and move money between arbitrary account IDs. Before this goes anywhere near production, add:
- An auth mechanism (PASETO or JWT are the usual choices for this tutorial lineage).
- Middleware enforcing it on account/transfer routes.
- Ownership checks (a transfer's `from_account` must belong to the authenticated caller).

## 🟠 High priority

### 3. No row locking around balance updates → race conditions
No `SELECT ... FOR UPDATE` (or equivalent) before reading/writing balances in `transferTx`, and no consistent lock ordering (e.g., always lock the lower account ID first). Once fix #1 lands, concurrent transfers between the same two accounts will still race and can deadlock or lose updates without locking.

### 4. No validation that `FromAccountID != ToAccountID`, or currency match
`transactionTransfer` (`internal/api/transactions_router.go`) accepts a transfer where source and destination are the same account, and never checks the two accounts share a currency before moving money between them.

### 5. Money handled as `float64`
`Amount float64 \`binding:"required,gt=0"\`` in the transfer request, converted via `fmt.Sprintf("%.2f", req.Amount)`. `github.com/shopspring/decimal` is already a dependency but is only used in a test helper — use it (or a string-based decimal type) end-to-end for amounts to avoid float rounding issues in a money-moving system.

### 6. Errors leak internals and collapse to 500
Every non-binding failure in the handlers returns `ctx.JSON(500, gin.H{"error": err.Error()})` — including cases that should be 404 (`GetAccountById` returning `sql.ErrNoRows`) or 400/409 (FK violations, insufficient funds). Raw driver/SQL error text is returned straight to the client, which can leak schema details. Add `errors.Is`/`errors.As` handling for `sql.ErrNoRows` and `pq.Error` codes, and map them to proper status codes instead of a blanket 500.

### 7. No CHECK constraints for non-negative balances
The migration's SQL comments claim "balance cannot be negative" and "amount cannot be negative," but there's no `CHECK (balance >= 0)` or similar in `db/migrations/000001_init-transaction-feature.up.sql`. The comment is aspirational, not enforced — add the constraint (and an application-level insufficient-funds check before the DB even tries).

## 🟡 Medium priority

### 8. No CI
No `.github/workflows` or any other CI config — `make test` only runs manually. Nothing gates merges to `main`. Worth adding a basic workflow: spin up Postgres, run migrations, `go vet`, `go test ./...`.

### 9. Test DB config duplicated and hardcoded
`db/sqlc/main_test.go` hardcodes `postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable` instead of reusing `config.LoadConfig`. Related: a `docker-compose.yaml` that used to provision Postgres on port 5433 (visible in an earlier commit) is missing from the current tree, so a fresh clone has no way to stand up the DB these tests expect.

### 10. `set-env.env` is dead/misleading
Root-level `set-env.env` exports `DB_URI` with a placeholder credential, but the app reads `DB_SOURCE` (via `app.env` + viper), so this file is never actually consumed. Either wire it up or delete it — as-is it's confusing for anyone new to the repo.

### 11. No `.gitignore`
Nothing prevents a real `app.env` (with real credentials) from being accidentally committed later. Add one covering `app.env`, build artifacts, and IDE files.

### 12. Gin always runs in debug mode
`gin.SetMode(...)` is never called, so `gin.Default()` always runs in debug mode regardless of environment — verbose logging and debug output would ship to production as-is. Wire mode selection to an env var.

### 13. Config has no defaults or validation
`internal/config/config.go` unmarshals into `Config` with no `viper.SetDefault` calls and no post-load check that required fields are non-empty. A missing key in `app.env` silently produces a zero-value string (e.g., an empty `DBSource` passed straight to `sql.Open`), failing far from the actual cause.

### 14. No structured logging
Logging is `log.Fatal` in `cmd/`, scattered `fmt.Println` in `cmd/migration/main.go`, and Gin's default plain-text request logger. No leveled/structured logging (zap, zerolog), no request IDs, no log line when a transfer is rolled back — makes production debugging harder than it needs to be.

### 15. Connection pool left at defaults
`sql.Open` in `cmd/server/main.go` is never followed by `SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime` — fine for a tutorial, worth tuning before any real load.

## 🟢 Low priority / cleanup

- `internal/api/router.go:14-16` — three commented-out routes for transaction list/detail that don't exist. Either implement or delete.
- `pkg/utils/utils.go` — empty file (just `package utils`), dead.
- `domain/constant/entries.go` — `ENTRY_TYPE_DEPOSIT` defined but never referenced; no deposit flow exists yet.
- `docs/001_init_tech_doc.md` — a 4-line unchecked checklist, effectively an empty stub.
- `rest-testing/main_test.go` — empty placeholder Go file sitting next to what's actually a VS Code REST Client `.http` file; misleading name.
- No API documentation (no Swagger/OpenAPI) — fine for a learning project, worth adding if this grows.
- Store layer wraps errors with `fmt.Errorf("tx err: %v, rb err: %v", ...)` using `%v` instead of `%w`, breaking `errors.Is`/`errors.As` chains for callers.
- `Owner` field has no length cap matching the DB's `VARCHAR(255)`, so an over-long value fails at the DB layer with an unhelpful raw error (compounds with #6).

## What's already solid

- sqlc-generated queries are fully parameterized — no SQL injection risk found anywhere.
- The `execTx` transaction wrapper (`db/store/store.go`) correctly rolls back on error and wraps both the original and rollback error if rollback also fails.
- `go.mod` dependencies are current and reasonably chosen (Gin, viper, testify, go.uber.org/mock all recent versions); Go 1.25 toolchain.
- Existing gomock-based unit tests for `transferTx` are well-structured (table-driven, cover success + per-step failure branches) — they just test call sequencing, not real balance arithmetic (see #1).
- DB integration tests correctly isolate via `Begin()` + `t.Cleanup(tx.Rollback)`.

## Suggested order of attack

1. Fix the balance-overwrite bug (#1) and add a concurrency test that would have caught it.
2. Add row locking for transfers (#3) and the same/currency validation (#4, #7).
3. Add authentication + ownership checks (#2) before this is exposed anywhere beyond localhost.
4. Clean up error handling/status codes (#6) and config validation (#13) — cheap, high-clarity wins.
5. Everything else (CI, structured logging, `.gitignore`, dead code) as time allows.
