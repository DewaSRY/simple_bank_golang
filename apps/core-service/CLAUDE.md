# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make db-up                 # start Postgres + run pending migrations (docker compose)
make db-down                # stop Postgres
make server                 # go run ./cmd/server
make test                   # db-up, then go test -v ./...
make test-router-coverage   # coverage for internal/api/... only

go test ./internal/db/store/...              # run one package's tests
go test -run TestTransferTx ./internal/db/store  # run a single test by name

make migrate-create name=add_something   # new migration pair (up/down)
make migrate-up
make migrate-down
make migrate-force version=N
make migrate-goto version=N

make sqlc                   # regenerate internal/db/sqlc from internal/db/query/*.sql
make generate-mock          # regenerate internal/db/mock/querier.go (gomock) from db.Querier
make swag-gen                # regenerate internal/docs (swagger) from handler annotations

make docker-build / docker-run / docker-stop
make docker-dep-build / docker-dep-push     # prod image, linux/amd64, pushed as sdewa/core-service-dep

make tf-init / tf-plan / tf-apply / tf-output / tf-destroy   # terraform, see docs/TERRAFORM_MAKE_COMMANDS.md
```

Tests are split into two kinds, and it matters which one you're running:
- Unit tests against a `gomock`-generated `MockQuerier` (e.g. `store_transaction_test.go`) — no database needed.
- Integration tests that hit a real Postgres (e.g. `store_transaction_integration_test.go`, everything under `internal/db/sqlc/*_test.go`) — require `make db-up` first, which `make test` does automatically.

`app.env` must exist at the repo root (copy from `app.env.example`) before running the server or integration tests locally; config also loads from real env vars if the file is absent (see `docs/CONFIG_ENV_VARIABLE.md`).

## Architecture

This is a double-entry ledger / banking API (Gin + Postgres via sqlc), organized in three layers that only talk to the layer directly below:

```
internal/api      HTTP handlers (Gin) — thin translators between HTTP and Storer
internal/db/store Business logic + transactions — depends only on db.Querier
internal/db/sqlc  Generated, one method per SQL query — no business logic
```

**`Storer` is the seam.** `internal/db/store/store.go` defines `Storer` as `db.Querier` (every generated query method) plus a handful of hand-written `*Tx` methods (`TransferTx`, `CreateAccountTx`, `DepositTx`, `DeleteAccountTx`). `Server` (`internal/api/server.go`) depends only on `Storer`, so `internal/db/mock` can swap in a `MockQuerier` for handler/store unit tests without a database.

**Transaction pattern.** Each `*Tx` method (e.g. `TransferTx` in `store_transaction.go`) wraps a plain function (`transferTx`) that takes a `sqlc.Querier` — not the concrete `*_store` — and is called through `execTx`, which opens a `sql.Tx`, runs the function against a tx-scoped `Queries`, and commits/rolls back based on its error. This is why `transferTx` can be unit-tested against `MockQuerier` (see `docs/GOMOCK_TESTING.md`) while still running for real inside a transaction in production. When adding a new multi-step DB operation, follow this shape rather than issuing multiple `Storer` calls directly from a handler.

**Money is `string`, not `float64` or an int column, end-to-end** — sqlc, `Querier` params, JSON bodies. Business logic parses with `shopspring/decimal` at the point of use (see `transferTx`) rather than converting once at a boundary. Preserve this when adding money-handling code; introducing a float anywhere in the chain reintroduces the precision bugs this was chosen to avoid.

**Locking order for transfers.** `transferTx` locks the two accounts via `GetAccountByIdForUpdate` in ascending-ID order regardless of which is `from`/`to`, specifically so two concurrent transfers between the same pair can't deadlock. Don't reorder those calls.

**Errors and responses are centralized, not per-handler.** Handlers call `fail(ctx, err)` with an `*AppError` (`internal/api/apperror.go`) or `succeed`/`succeedWithMeta` (`internal/api/response.go`); they never call `ctx.JSON` directly. `errorHandlerMiddleware` (registered once in `NewServer`) renders whatever was recorded on `ctx.Errors` into the normalized `{error: {code, message, details}}` shape, and defaults to a sanitized 500 for anything that isn't an `*AppError` — so a handler can't accidentally leak an internal error string just by returning a raw `error`. Store-layer sentinel errors (`store.ErrSameAccount`, `store.ErrInsufficientFunds`, etc., in `internal/db/store/errors.go`) get mapped to the right `*AppError` in small `*AppError`-mapping functions like `transferAppError` — add new store errors there, not with ad-hoc `errors.Is` checks in handlers. Full spec: `docs/NORMALIZE_RESPONSE.md`.

**Auth.** JWT-based (`internal/token`, `golang-jwt/jwt/v5`); `authMiddleware` in `internal/api/auth_middleware.go` verifies the bearer token and stores `*token.Payload` in the Gin context, read back via `getAuthPayload(ctx)`. All routes under `v1.Group("/")` in `internal/api/router.go` that go through the `authorized` group require it. Full spec, including known gaps between intended and actual behavior: `docs/IMPLEMENT_AUTH.md`.

**sqlc.** Queries live in `internal/db/query/*.sql`, schema/migrations in `internal/db/migrations/`, generated code lands in `internal/db/sqlc/`. `emit_exact_table_names: false` and no `db/mapper` package (it was deleted — see `docs/SQLC_ROW_MAPPING.md`, which supersedes parts of `docs/SQLC_SETUP.md`): prefer `SELECT *`/`RETURNING *` and `sqlc.embed()` to reuse existing row types instead of hand-writing a new mapper for a new query shape.

**CORS** is opt-in and origin-list-based, not `*` (`internal/api/server.go:corsMiddleware`, configured via `CORS_ALLOWED_ORIGINS`) — see `docs/CORS.md` for why a wildcard doesn't actually work here given `AllowCredentials: true`.

## Existing docs — read before changing that area

`docs/` already has deep, source-verified write-ups (file:line cited, gaps between intended and actual behavior called out explicitly) for most subsystems: `IMPLEMENT_AUTH.md`, `NORMALIZE_RESPONSE.md`, `SQLC_SETUP.md` + `SQLC_ROW_MAPPING.md`, `GOMOCK_TESTING.md`, `CONFIG_ENV_VARIABLE.md`, `CORS.md`, `GIN_HTTP_LAYER.md`, `MIGRATION_GUID.md`, `VSCODE_DEBUGGING.md`, `TERRAFORM_EC2_DEPLOY.md` + `TERRAFORM_MAKE_COMMANDS.md`. Read the relevant one before making a non-trivial change in that area — several contain corrections to claims made in an earlier doc or in the library's own docs. If your change makes one of them inaccurate, update it in the same change rather than letting it drift.

When writing a new doc in this style, follow `docs/DOC_STRUCTURE.md` (a template, not content) rather than inventing a new structure.
