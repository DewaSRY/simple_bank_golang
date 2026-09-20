---
name: core-service-dev
description: How to correctly implement, modify, review, or test anything in apps/core-service — the Go/Gin double-entry-ledger banking API in this simple_bank_golang monorepo (handler → store → sqlc layering, money-as-string, account-locking order, passwordless device-fingerprint auth, error/response envelope, migrations, rate limiting, config). Use this whenever a task touches internal/api, internal/db, internal/config, internal/token, or internal/rate-limiter under apps/core-service — even if the request doesn't say "core-service" or name a file, e.g. "add an endpoint to freeze an account", "why does this transfer 500 instead of 409", "add a query param filter to account search", "make login also accept a phone number", "add a new rate limit tier", "write a test for deleteAccountTx". Do NOT use for apps/portal (separate frontend stack) or for anything outside apps/core-service.
---

# Core Service Development Workflow

This skill exists so an AI working on `apps/core-service` doesn't have to
rediscover the architecture from scratch, and doesn't quietly reintroduce a
bug this codebase already paid to fix once (float money, wrong lock order, a
password field, `ctx.JSON` on an error path). It is a **compressed workflow**,
not the full picture — the full picture is
[`AGENTS.md`](../../../AGENTS.md) at the `core-service` root. Read that file
if it's not already in context; this skill assumes its content and doesn't
repeat all of it.

## Non-negotiable rules

These are correctness/safety properties, not style preferences. Violating one
usually means silently reintroducing a bug this repo already fixed once:

1. **Money is `string` end-to-end** — params, sqlc types, JSON. Parse with
   `decimal.NewFromString` only at the point you do arithmetic/comparison.
   Never introduce `float64` or a numeric DB column in the money path.
2. **Multi-account transactions lock accounts in ascending-ID order**, always
   via `GetAccountByIdForUpdate`, regardless of logical from/to roles (see
   `transferTx`, `deleteAccountTx` in `internal/db/store/`). Reordering these
   calls reintroduces a real deadlock risk under concurrent load.
3. **There is no password.** Auth is email + a device-fingerprint hash
   (`internal/api/auth/handler_utils.go`, `internal/util/fingerprint.go`).
   Don't add a password field/flow back without an explicit ask.
4. **A handler never calls `ctx.JSON` on an error path.** Every error becomes
   a `*core.AppError` (via `core.ValidationErr`/`NotFoundErr`/`ForbiddenErr`/
   `ConflictErr`/`InternalErr`/a domain `*AppError`-mapping function) and goes
   through `core.Fail(ctx, err)`. `core.ErrorHandlerMiddleware` is the only
   place that renders an error response.
5. **Never hand-edit generated files**: `internal/db/sqlc/*.sql.go`,
   `internal/db/mock/querier.go`, `internal/docs/*`. Change the source
   (`internal/db/query/*.sql`, the `Storer` interface, godoc comments) and
   regenerate (`make sqlc && make generate-mock`, or `make swag-gen`).
6. **Business logic lives in `internal/db/store` (`*Tx` functions), never in
   a handler.** A handler binds the request, calls one `Storer` method, maps
   the error, writes the response — nothing else.
7. **Middleware registration order in `server.go` is load-bearing**
   (`RequestID` → `Logging` → `CORS` → `ErrorHandler` → `RateLimit` →
   `Recovery`). Don't reorder without reading the comment above
   `router.Use(...)` in `internal/api/server.go`.

## Decision flow: implementing a change

1. **Does it need a new/changed DB column or table?** → new migration pair in
   `internal/db/migrations/` via `make migrate-create name=...` → `make sqlc`
   (schema *is* the sqlc source of truth) → `make generate-mock`.
2. **Does it need a new/changed SQL query, no schema change?** → write it in
   `internal/db/query/*.sql` (prefer `SELECT *`/`RETURNING *`/`sqlc.embed(...)`
   over a new hand-written mapper) → `make sqlc` → `make generate-mock`.
3. **Is it more than one write that must commit/rollback together** (or has a
   business rule beyond "run this query", e.g. "insufficient funds",
   "accounts must differ")? → add/extend a `store_<name>_tx.go` in
   `internal/db/store/`: public `FooTx(ctx, arg)` opens a tx via `execTx`,
   private `fooTx(ctx, q sqlc.Querier, arg)` holds the logic (parameterized on
   the interface, not `*sqlc.Queries`, so it's mockable). Add the method to
   `Storer` in `store.go`. New failure mode the caller must distinguish? add a
   sentinel `errors.New(...)` to `internal/db/store/errors.go`.
4. **Otherwise it's a single query** → call `h.Store.X` straight from the
   handler.
5. **Handler**: bind (`ShouldBindJSON`/`ShouldBindUri`/`ShouldBindQuery`) → on
   bind error, `core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))`
   → call the one `Storer` method chosen above → map any error (extend the
   domain's `apperror.go` with a `case` for any new sentinel/`pq.Error`) →
   `core.Succeed`/`core.SucceedWithMeta`. New/changed shape? add it to that
   domain's `response.go`.
6. **New endpoint** registers on the group its domain's `RegisterRoutes(public,
   authorized *gin.RouterGroup)` receives. A brand-new domain package (not
   `account`/`auth`/`transfer`) is wired into the `routers` slice in
   `internal/api/router.go`'s `bindRouters` — an endpoint on an *existing*
   domain never touches `router.go`.
7. **Swagger**: add `@Summary`/`@Router`/etc. godoc above the handler, then
   `make swag-gen`.
8. **Tests**: extend that domain's `*_handler_test.go` (gomock `MockStorer`,
   no DB — follow the file's existing `newTestRouter`/`authHeaderFor` helpers).
   If you touched a `*Tx` function, note handler tests mock the *entire*
   `Storer` call and won't catch a regression inside it — add a
   `internal/db/store/*_test.go` against a mocked `sqlc.Querier` instead of
   assuming the handler test covers it (there is currently no dedicated
   `store`-level test suite; this is a known gap, not a pattern to copy from
   an existing file).

## Where to look first

| Task | File(s) |
|---|---|
| New/changed HTTP endpoint on an existing resource | `internal/api/<domain>/handlers.go` (or `manage.go`/`transactions_handlers.go`) + `RegisterRoutes` in `<domain>.go` |
| New/changed multi-step business logic | `internal/db/store/store_<name>_tx.go`; add method to `Storer` in `store.go` |
| New/changed single SQL query | `internal/db/query/*.sql` → `make sqlc` → `make generate-mock` |
| New/changed error → HTTP status mapping | domain's `apperror.go`; new sentinels in `internal/db/store/errors.go` |
| New/changed response JSON shape | domain's `response.go` |
| Middleware / response envelope / auth check | `internal/api/core/*.go`; order wired in `server.go` |
| Config value | `internal/config/config.go` (mapstructure tag = env var name) |
| Auth/login/session behavior | `internal/api/auth/*.go`, `internal/token/` |
| Rate limiting | `internal/rate-limiter/limiter.go` (generic bucket), `internal/api/core/rate_limit_middleware.go` (Gin adapter) |
| DB schema change | new pair in `internal/db/migrations/` via `make migrate-create` |
| Handler test | domain's `*_handler_test.go` |
| Store/query test | `internal/db/sqlc/*_test.go` (needs `make db-up`) |

## Verification — match depth to what changed

```
Handler / request-response shape        → go build ./...  →  that domain's *_handler_test.go
                                           → touched a godoc annotation? make swag-gen, check internal/docs diff

Store / *Tx business logic              → go test ./internal/db/store/...
                                           → go test ./internal/db/sqlc/...  (make db-up first)
                                           → the calling domain's *_handler_test.go (won't catch a *Tx
                                             regression by itself — it mocks the whole Storer call)

New/changed SQL query                   → make sqlc → make generate-mock (always second)
                                           → go build ./...  (catches every call site the signature broke)
                                           → go test ./internal/db/sqlc/...

New migration                           → make migrate-up (or make db-up) → make migrate-down to confirm
                                           it's symmetric → make sqlc → make generate-mock

Config change                           → update app.env.example
                                           → update docs/CONFIG_ENV_VARIABLE.md if precedence/defaults changed
```

`make test` runs everything (`make db-up` then `go test -v ./...`). Before
calling a change done, look at the actual diff — several files here are
generated (`internal/db/sqlc/*.sql.go`, `internal/db/mock/querier.go`,
`internal/docs/*`), and an unexpected diff there almost always means a
generator ran against stale input, not that the generator is wrong.

## Deep-dive references (`docs/*.md`)

Each doc's *design rationale* (why a decision was made) is reliable; treat
any *file path, function name, or "current status" claim* in one as
possibly stale — verify against the actual code before relying on it (see
the "Doc drift" note at the end of `AGENTS.md` for why). Open one of these
only when the task actually calls for that depth — don't read them
speculatively:

| Doc | Open it when you're... |
|---|---|
| `API_MODULE_GUIDE.md` | building a genuinely new domain package (4th one beyond `account`/`auth`/`transfer`) |
| `GIN_HTTP_LAYER.md` | questioning why Gin vs. another framework — background only, not a how-to |
| `NORMALIZE_RESPONSE.md` | touching `core.Succeed`/`Fail`/the `ctx.Errors` mechanism itself |
| `IMPLEMENT_AUTH.md` | changing login/register/JWT behavior beyond a small tweak |
| `SQLC_SETUP.md` | new to sqlc's codegen model, or debugging an unexpected `make sqlc` diff |
| `SQLC_ROW_MAPPING.md` | a new query would otherwise need a hand-written mapper — this explains the `sqlc.embed(...)` alternative |
| `MIGRATION_GUID.md` | writing a migration more complex than a single `ALTER TABLE` |
| `GOMOCK_TESTING.md` | the gomock patterns (`InOrder`, `.Times(0)`) aren't already clear from a sibling `*_handler_test.go` |
| `CONFIG_ENV_VARIABLE.md` | adding a config field, or debugging a value that's silently wrong (documents two real past incidents) |
| `CORS.md` | a CORS error shows up — read before reaching for `AllowOrigins: []string{"*"}`, which is a no-op here |
| `RATE_LIMITING.md` | changing limiter behavior, keying, or adding a per-route/per-role tier |
| `LOGGING.md` | changing what gets logged or how request_id propagates |
| `TERRAFORM_EC2_DEPLOY.md` / `TERRAFORM_MAKE_COMMANDS.md` | touching `terraform/` or the `tf-*` Makefile targets |
| `VSCODE_DEBUGGING.md` | setting up step-through debugging, not code changes |
| `DOC_STRUCTURE.md` | writing a new doc in this `docs/` style, not a code task |

## If AGENTS.md and a docs/*.md file disagree

Trust the code, then `AGENTS.md`, then the specific `docs/*.md` file, in that
order — `AGENTS.md` is kept current against the actual source; several
`docs/*.md` files predate the `internal/api` per-domain-package refactor and
say so themselves. If you find a claim here or in `AGENTS.md` that the code
now contradicts, fix the doc as part of your change rather than working
around the stale claim silently.
