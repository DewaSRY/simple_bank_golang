---
applyTo: "apps/core-service/**"
---

# Copilot instructions — core-service

Go 1.25 double-entry ledger API (Gin + sqlc/Postgres, passwordless JWT auth
bound to device fingerprint). Full rationale lives in `apps/core-service/AGENTS.md`
— this file is the fast-lookup version so you don't have to re-derive it.

## Jump table — find the file before writing code

| Task | Go here |
|---|---|
| New/changed HTTP endpoint | `internal/api/<domain>/handlers.go` (or `manage.go`/`transactions_handlers.go`); route in that domain's `RegisterRoutes` |
| Multi-step business logic (>1 write, needs atomicity) | `internal/db/store/store_<name>_tx.go` — `FooTx`/`fooTx` + `execTx`; add to `Storer` in `store.go` |
| Single SQL query | `internal/db/query/*.sql` → `make sqlc` → `make generate-mock` → `go build ./...` |
| Error → HTTP status mapping | domain's `apperror.go`; new sentinels in `internal/db/store/errors.go` |
| Response JSON shape | domain's `response.go` (`toXResponse`) |
| Middleware / response envelope / auth check | `internal/api/core/*.go` |
| Config value | `internal/config/config.go` (mapstructure tag = env var) |
| Auth/session | `internal/api/auth/*.go`, `internal/token/` |
| Rate limiting | `internal/rate-limiter/limiter.go`, `internal/api/core/rate_limit_middleware.go` |
| DB schema change | new pair in `internal/db/migrations/` via `make migrate-create name=...` |
| Handler test | domain's `*_handler_test.go` (gomock `MockStorer`, no DB) |
| Store/query test | `internal/db/sqlc/*_test.go` (real Postgres, `make db-up` first) |

## Layering — never skip a layer or call upward

```
internal/api/<domain>  → Gin handlers, thin translators (bind → 1 Storer call → map error → respond)
internal/db/store      → business logic, *Tx transactions — depends only on internal/db/sqlc
internal/db/sqlc       → generated, one method per query — no business logic, DO NOT hand-edit
```
`Storer` (`internal/db/store/store.go`) is the seam every `Handler` depends on —
never a concrete type — so `internal/db/mock.MockStorer` can replace it in tests.

## Hard rules (violating these is a bug, not a style nit)

- **Money is always `string`** end-to-end (params, sqlc types, JSON). Parse with
  `decimal.NewFromString` only at point of use. Never introduce `float64` or a
  numeric DB column in the money path.
- **Account locking order**: always `GetAccountByIdForUpdate` in ascending-ID
  order, never request/logical order — this is what prevents deadlocks in
  `transferTx`/`deleteAccountTx`. Do not reorder these calls in any new
  multi-account `*Tx`.
- **No password field.** Identity is email + device fingerprint
  (`internal/util/fingerprint.go`, `users.device_fingerprint_hash`). Don't
  reintroduce a password column/flow.
- **Auth is stateless, no logout/blacklist.** Don't build revocation logic
  without adding server-side token tracking first.
- **No role/permission system.** Authorization is per-resource ownership
  (`authPayload.ID == row.UserID`) checked in the handler, returning
  `core.ForbiddenErr(...)`. Don't bake roles into the JWT.
- **CORS**: never set `AllowOrigins: []string{"*"}` — it's a no-op because
  `AllowCredentials: true` is always set. Add the origin to
  `CORS_ALLOWED_ORIGINS` instead.
- **Never hand-edit generated files**: `internal/db/sqlc/*.sql.go`,
  `internal/db/mock/querier.go`, `internal/docs/*`. Edit the source
  (`internal/db/query/*.sql`, `Storer` interface, godoc comments) and
  regenerate.
- **Middleware order in `server.go` is load-bearing**:
  `RequestIDMiddleware` → `LoggingMiddleware` → `corsMiddleware` →
  `ErrorHandlerMiddleware` → `RateLimitMiddleware` (if enabled) →
  `RecoveryMiddleware`. `ErrorHandlerMiddleware` must precede
  `RecoveryMiddleware` or a panic recovers into an empty 200.

## Handler pattern (copy this shape)

```go
func (h *Handler) DoThing(ctx *gin.Context) {
    var req doThingRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
        return
    }
    result, err := h.Store.SomeStorerMethod(ctx, arg)
    if err != nil {
        core.Fail(ctx, domainAppError(err)) // domain's apperror.go mapping fn
        return
    }
    core.Succeed(ctx, toDoThingResponse(result), "message")
}
```
- Never call `ctx.JSON` directly. Never return a raw `error` to the client.
- Success body: `{"data", "message", "meta"}`. Error body: `{"error": {"code","message","details"}}`.
- List endpoints: embed `core.PaginationQuery`, respond via `core.SucceedWithMeta`
  with `core.Meta{Page, Limit, Total}` (Total from a paired `*Count` query).

## Errors

Sentinel errors live in `internal/db/store/errors.go` (`ErrSameAccount`,
`ErrCurrencyMismatch`, `ErrInvalidAmount`, `ErrInsufficientFunds`,
`ErrCannotDeleteMainAccount`), matched with `errors.Is`. A `pq.Error` with code
`check_violation` on a balance-changing query = insufficient funds → 409, not 500.
Every new sentinel needs a `case` in the relevant domain's `apperror.go` — don't
scatter `errors.Is` checks in handlers. `core.InternalErr(err)` always takes the
real cause (logged server-side only, never `nil`).

## Workflow for a new feature

1. Migration (`internal/db/migrations/`, via `make migrate-create`)
2. SQL in `internal/db/query/*.sql` → `make sqlc` → `make generate-mock` (always in that order) → `go build ./...`
3. If multi-write: `store_<name>_tx.go` (`FooTx`/`fooTx` + `execTx`), add to `Storer`
4. Sentinel error in `internal/db/store/errors.go` if callers must distinguish failure modes
5. Handler in the domain package, following the pattern above; register route in that domain's `RegisterRoutes`
6. `apperror.go` case for any new error
7. `response.go` DTO + mapper if the client shape differs from the sqlc row
8. Swagger godoc (`@Summary`/`@Router`) → `make swag-gen`
9. Tests: gomock handler test; store `*Tx` test against a mocked `sqlc.Querier` if you touched lock order/decimal math (there is currently **no dedicated unit test for `transferTx`/`deleteAccountTx`/`createAccountTx`** — handler tests mock the whole `Storer` and won't catch a regression inside the `*Tx` function itself)

## Commands

```bash
make db-up             # start Postgres + apply migrations
make server            # go run ./cmd/server
make test              # db-up, then go test -v ./...
make sqlc               # regenerate internal/db/sqlc from internal/db/query/*.sql
make generate-mock      # regenerate internal/db/mock/querier.go (run after make sqlc)
make swag-gen           # regenerate internal/docs from godoc annotations
```

## Don't trust blindly

`CLAUDE.md`/`docs/*.md` in this app predate a package-split refactor (flat
`internal/api` → per-domain packages). Trust the code and `AGENTS.md` over
`docs/*.md` for file paths, function names, and "current test status" claims —
the design rationale in `docs/*.md` is still fine, the file/function names in
some of them are stale.
