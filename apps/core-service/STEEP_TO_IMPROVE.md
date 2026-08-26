# core-service — Step-by-Step Improvement Plan

Sequenced action plan derived from [NEED_TO_IMPOROVE.md](NEED_TO_IMPOROVE.md). Each step is small enough to land as its own PR. Do them in order — later steps assume earlier ones are done.

## Step 1 — Fix the balance-overwrite bug

- [ ] Change `IncrementAccountBalance` in `db/query/account.sql` from `SET balance = $2` to `SET balance = balance + $2` (an additive/signed amount, not an absolute value).
- [ ] Re-run `sqlc generate` to regenerate `db/sqlc/account.sql.go`.
- [ ] Update `transferTx` in `db/store/store_transaction.go` to pass `-amount` for the debit leg and `+amount` for the credit leg into the new additive query (remove the `"-" + arg.Amount` string-concat approach).
- [ ] Add an integration test in `db/store/` that opens a real DB transaction, seeds two accounts with known balances, runs `TransferTx` N times concurrently, and asserts the final balances equal `start ± (N × amount)`. This is the test that would have caught the original bug — the existing gomock-based test only checks call sequencing.
- [ ] Regenerate mocks (`make generate-mock`) if the `Querier` interface signature changed.

## Step 2 — Add transfer-safety checks

- [ ] In `transactionTransfer` (`internal/api/transactions_router.go`), reject a request where `FromAccountID == ToAccountID` with a 400.
- [ ] Before executing the transfer, fetch both accounts and reject with 400/422 if their currencies don't match.
- [ ] Add an insufficient-funds check (reject if `fromAccount.Balance < amount`) so the app doesn't rely solely on the DB constraint from Step 3.
- [ ] Add `CHECK (balance >= 0)` to a new migration (don't edit the existing `000001_init-transaction-feature` migration — add `000002_add_balance_check.up.sql` / `.down.sql`) as a backstop.

## Step 3 — Add row locking for concurrent transfers

- [ ] Add a `GetAccountForUpdate` query using `SELECT ... FOR UPDATE` in `db/query/account.sql`.
- [ ] In `transferTx`, lock both accounts before reading/updating balances, always locking the lower account ID first (consistent lock ordering prevents deadlocks between two transfers going opposite directions between the same pair of accounts).
- [ ] Extend the concurrency test from Step 1 to include transfers going both directions between the same account pair, and confirm no deadlock and correct final balances.

## Step 4 — Add authentication and ownership checks

- [ ] Decide on PASETO or JWT (PASETO is the more common choice in this tutorial lineage; either is fine).
- [ ] Add a `users` table + migration (username, hashed password, created_at).
- [ ] Add login/token endpoints and a Gin auth middleware that validates the token and injects the authenticated user into the request context.
- [ ] Apply the middleware to `/accounts` and `/transactions/transfer`.
- [ ] Add an `owner` check: a transfer's `from_account` must belong to the authenticated user; account creation should associate the new account with the authenticated user rather than an arbitrary `owner` string in the request body.

## Step 5 — Clean up error handling and status codes

- [ ] Replace the blanket `ctx.JSON(500, gin.H{"error": err.Error()})` pattern in the handlers with explicit `errors.Is(err, sql.ErrNoRows)` → 404, validation failures → 400/422, everything else → 500 with a generic message (don't leak `err.Error()` to the client — log it server-side instead).
- [ ] Switch `fmt.Errorf("tx err: %v, rb err: %v", ...)` in `db/store/store.go` to use `%w` so errors stay unwrappable.
- [ ] Add a length cap (`max=255` or matching the DB column) to the `Owner` field's binding tag.

## Step 6 — Config hardening

- [ ] Add `viper.SetDefault(...)` for any field that has a sane default (e.g. `SERVER_ADDRESS`).
- [ ] After `viper.Unmarshal`, validate that required fields (`DBDriver`, `DBSource`) are non-empty and return a clear error instead of letting a zero-value string reach `sql.Open`.
- [ ] Delete `set-env.env` (it's dead — the app reads `DB_SOURCE`, this file exports an unused `DB_URI`) or rewire it to actually be sourced somewhere.
- [ ] Add a `.gitignore` covering `app.env`, build binaries, and IDE files.

## Step 7 — Local dev environment

- [ ] Recreate `docker-compose.yaml` (it existed in an earlier commit, provisioning Postgres on port 5433 — `db/sqlc/main_test.go` still assumes this exists) so a fresh clone can run `docker compose up` and immediately run tests/migrations.
- [ ] Point `db/sqlc/main_test.go` at `config.LoadConfig` instead of a hardcoded DSN, so test config lives in one place.
- [ ] Add `docker-build`/`docker-up` targets to the `Makefile` and a `Dockerfile` for the service itself.

## Step 8 — CI

- [ ] Add `.github/workflows/ci.yml`: spin up Postgres as a service container, run migrations, `go vet ./...`, `go test ./...`.
- [ ] Gate merges to `main` on this workflow passing.

## Step 9 — Observability

- [ ] Set `gin.SetMode` based on an env var (debug locally, release in any deployed environment).
- [ ] Introduce a structured logger (zap or zerolog) for `cmd/` and handler-level logging; replace scattered `fmt.Println` in `cmd/migration/main.go`.
- [ ] Log (server-side) whenever a transfer transaction is rolled back, including the reason.
- [ ] Tune the DB connection pool (`SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime`) in `cmd/server/main.go`.

## Step 10 — Cleanup pass

- [ ] Remove or implement the three commented-out routes in `internal/api/router.go`.
- [ ] Delete the empty `pkg/utils/utils.go` if nothing has landed in it by this point.
- [ ] Either wire up `ENTRY_TYPE_DEPOSIT` (`domain/constant/entries.go`) into a real deposit flow or remove it.
- [ ] Replace the stub `docs/001_init_tech_doc.md` with real notes, or delete it.
- [ ] Rename/clarify `rest-testing/main_test.go` — it's not a Go test, it's a placeholder next to a `.http` file; the name is misleading.
- [ ] Switch money handling from `float64` to `github.com/shopspring/decimal` (already a dependency, currently unused in production code) end-to-end in request/response types and the transfer amount plumbing.

## Nice-to-have (after the above)

- [ ] Add Swagger/OpenAPI annotations once the API surface stabilizes post-auth.
- [ ] Add rate limiting on account creation and transfer endpoints.
- [ ] Add a `ListAccounts` endpoint/query if multi-account-per-user becomes relevant.
