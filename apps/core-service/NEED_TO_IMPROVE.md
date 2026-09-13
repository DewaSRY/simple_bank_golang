# Core Service — Areas to Improve

Audit of `apps/core-service` (Go 1.25, Gin + sqlc + Postgres). Builds and vets clean. Overall the codebase is in good shape — a real `TxStore` pattern with correct row locking, a centralized error-response pipeline, JWT auth with per-request ownership checks, and Swagger docs kept in sync with handlers. The items below are what stand between this and production-ready.

A prior self-audit (`NEED_TO_IMPOROVE.md` / `STEEP_TO_IMPROVE.md`, deleted but recoverable via `git show d171fe3:apps/core-service/NEED_TO_IMPOROVE.md`) flagged several critical bugs — absolute-balance overwrites, missing row locking, `float64` money, missing auth, raw error leaks. **All of those are now fixed.** What's left below is what that audit also flagged as medium/low priority and is still open, plus a few things found fresh in this pass.

## High priority

- **Gin always runs in debug mode in production.** `gin.SetMode(...)` is never called outside test files (`internal/api/auth_middleware_test.go:16`, `auth_router_test.go:67`). `gin.Default()` in `internal/api/server.go` therefore runs with Gin's debug logger/warnings in every deployed environment. Call `gin.SetMode(gin.ReleaseMode)` in `main.go` (or drive it off config) before building the engine.

- **One handler still bypasses the centralized error pipeline.** `internal/api/account_transactions_router.go:150`:
  ```go
  ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
  ```
  This is the only remaining raw `ctx.JSON` error response in the codebase — every other handler goes through `fail(ctx, ...)` / `AppError` (`internal/api/apperror.go`). It breaks the normalized `{"error":{"code","message"}}` envelope and leaks the raw driver error string to the client. Replace with `fail(ctx, InternalErr())`.

- **No CI pipeline.** No `.github/workflows` (or any CI config) anywhere in the repo. Nothing gates merges to `main` — `go vet`/`go test` are manual-only via `make`. Add a workflow that at minimum runs `go build ./...`, `go vet ./...`, and the test suite (see below — needs Postgres available) on PRs.

- **No `docker-compose.yml`.** Several test suites (`db/sqlc/*_test.go`, `db/store/*_integration_test.go`) require a live Postgres at `localhost:5433` with hardcoded credentials, but there's no committed way to stand this database up. A fresh clone can't run `make test` or the integration tests without manually provisioning Postgres on that exact port/credentials. This also blocks adding CI. Add a `docker-compose.yml` (Postgres + migrations) and wire it into `make test`.

## Medium priority

- **Config has no defaults or startup validation.** `internal/config/config.go` binds env vars via viper but never calls `viper.SetDefault(...)` and never validates that required fields (`DBSource`, `JWTSecretKey`) are non-empty after load. A missing env var silently produces a zero-value string that fails far downstream (inside `sql.Open`, or `NewJWTMaker`'s min-length check) instead of a clear "missing config: X" error at startup. Add a validation step at the end of `LoadConfig`.

- **No structured logging.** Only `log.Fatal`/`log.Println`/`log.Printf` in `cmd/server/main.go`, `cmd/server/connect_db.go`, `cmd/migration/main.go`, plus Gin's default plaintext request logger. No leveled logs, no request IDs, and nothing is logged server-side when a transaction rolls back — the client sees a sanitized 500 but there's no corresponding server-side log line with the real error unless you're tailing raw stdout. Consider `slog` (stdlib, Go 1.25) with a request-ID middleware.

- **Repeated ownership-check block across 5+ handlers.** The same `account.UserID.Int64 != authPayload.ID → ForbiddenErr` check is copy-pasted in `account_manage_router.go:44-45,105-106,163-164` and `account_transactions_router.go:60-61,204-205`. Correct, but worth extracting into a shared helper (or better, filtering by `user_id` directly in the query) to avoid one of these copies drifting out of sync.

- **Orphaned code path: `ListAccountTransactionHistory`.** Fully generated (sqlc), mocked, mapped (`internal/api/account_mappers.go`), has response types and a dedicated test (`TestListAccountTransactionHistory` in `transactions_router_test.go`) — but no route or handler ever calls it. Looks like leftover from before entries listing was rewritten to use `ListAccountEntriesByAccountId`/the `account_entries` view (commit `0b3d90f`). Either wire it up or delete the query, mapper, response types, and test.

- **Duplicated row-mapping boilerplate.** `db/mapper/account.go` has 6 near-identical `XyzRowToAccount` functions, each copying the same ~8 fields from a different sqlc row type. `internal/api/account_response.go` has the same pattern (`toAccountResponse`, `toAccountuserResponse`, `toAccountuserResponseFromSearch`, `toPublicAccountResponse`, `toAccountEntriesViewResponse`, `toAccountListEntriesViewResponse`). Worth collapsing via a shared conversion helper or consolidating the underlying sqlc queries.

## Low priority / cleanup

- **`set-env.env` is dead.** Exports a placeholder `DB_URI`, but the app only ever reads `DB_SOURCE` via viper — this file is never consumed and is confusing for onboarding. Delete it or fix it to match what `config.go` actually reads.
- **Duplicated hardcoded test DSN.** `postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable` is duplicated in `db/sqlc/main_test.go:18` and `db/store/main_test.go:13` instead of being sourced from `config.LoadConfig` or a shared test helper.
- **Empty/dead files:** `domain/error/` (empty directory), `pkg/utils/utils.go` (empty, just `package utils`), `rest-testing/main_test.go` (1-line empty placeholder next to `core-service.http`).
- **Inconsistent import aliasing.** `db/store/store.go:6-7` aliases the same import path (`github.com/DewaSRY/core-service/db/sqlc`) twice under different names in one file; other files inconsistently call it `db` vs `sqlc`. Pick one alias convention.
- **Awkward naming:** `accountuserResponse` in `internal/api/account_response.go:18` is missing a camel-case boundary (`account` + `user`) next to properly-cased siblings like `publicAccountResponse`.
- **No linter config.** No `.golangci.yml`; only `go vet` via `make`. Consider adding `golangci-lint` with a baseline config, especially since CI doesn't exist yet either.

## Confirmed as already fixed (for context, not action items)

- Additive `IncrementAccountBalance` (not an absolute `SET`) — the historical balance-overwrite bug is gone.
- Row locking (`GetAccountByIdForUpdate`) in fixed ascending-ID order to avoid deadlocks, in both transfer and account-deletion sweep logic.
- `decimal.Decimal` used end-to-end for money instead of `float64`.
- JWT auth (HS256, pinned signing method, 32-byte min secret) + ownership checks on every account-scoped handler.
- Centralized `AppError` / `errorHandlerMiddleware` pattern sanitizes unexpected errors into generic 500s.
- `CHECK (balance >= 0)` / `CHECK (amount > 0)` constraints at the DB level (migration `000002`).
