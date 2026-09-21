# AGENTS.md

Guidance for AI agents working in this repository. This describes the code as it
actually exists (verified by reading it), not aspirations. Where `CLAUDE.md` or
`docs/*.md` disagree with the code below, **trust the code** — this repo went
through a refactor (splitting `internal/api` into per-domain packages) that
several docs haven't fully caught up with; see "Doc drift" at the end.

## What This Project Is

A double-entry ledger / simple banking HTTP API: users register, own one or more
accounts, and transfer money between accounts with full entry (ledger) history.
Stack: Go 1.25, Gin (HTTP), PostgreSQL via `sqlc` (no ORM), **passwordless**
JWT auth bound to a device fingerprint (`golang-jwt/jwt/v5`), an in-process
per-IP rate limiter, `shopspring/decimal` for money math, `viper` for config,
`slog` for structured logging, `gomock` for store/handler unit tests, Swagger
(`swaggo`) for API docs, `golang-migrate` for schema migrations, Terraform +
Docker for deployment to EC2 behind an nginx reverse proxy that rate-limits
at the edge (infra now lives at the repo root, `infra/terraform/` — not
under this app).

## Where to Look First (read this before exploring)

Match your task to a row, go straight to that file — don't grep the whole
repo first.

| Task | Start here |
|---|---|
| New/changed HTTP endpoint on an existing resource | `internal/api/<domain>/handlers.go` (or `manage.go`/`transactions_handlers.go`); register in that domain's `RegisterRoutes` in `<domain>.go` |
| New/changed multi-step business logic (>1 write, needs atomicity) | `internal/db/store/store_<name>_tx.go` (`FooTx`/`fooTx` + `execTx` shape); add method to `Storer` in `store.go` |
| New/changed single SQL query | `internal/db/query/*.sql` → `make sqlc` → `make generate-mock` → `go build ./...` |
| New/changed error → HTTP status mapping | domain's `apperror.go` (e.g. `transfer/apperror.go`); new sentinels go in `internal/db/store/errors.go` |
| New/changed response JSON shape | domain's `response.go` (`toXResponse` mapper) |
| Cross-cutting HTTP concern (middleware, response envelope, auth check) | `internal/api/core/*.go`; registration order is in `server.go` |
| Config value | `internal/config/config.go` (mapstructure tag = env var name) |
| Auth/login/session behavior | `internal/api/auth/*.go` (passwordless, device-fingerprint-bound — see below), `internal/token/` |
| Rate limiting | `internal/rate-limiter/limiter.go` (generic token bucket), `internal/api/core/rate_limit_middleware.go` (Gin adapter) |
| DB schema change | new pair in `internal/db/migrations/` via `make migrate-create name=...` |
| Handler test | domain's `*_handler_test.go` (gomock `MockStorer`, no DB) |
| Store/query test | `internal/db/sqlc/*_test.go` (real Postgres, `make db-up` first) |

If your task isn't a clean fit for any row (new domain package, cross-domain
refactor), read "How the Code Is Organized" and "How to Implement a New
Feature" below before writing code.

## What the Architecture Looks Like

Three layers, each only talking to the layer directly below:

```
internal/api/<domain>   HTTP handlers (Gin) — thin translators between HTTP and Storer
internal/db/store       Business logic + multi-step transactions — depends only on db.Querier
internal/db/sqlc        Generated, one method per SQL query — no business logic
```

```
Handler (internal/api/<domain>)
  ↓ calls
Storer  (internal/db/store — db.Querier + hand-written *Tx methods)
  ↓ wraps
sqlc.Querier (internal/db/sqlc — generated)
  ↓
Postgres
```

**`Storer` is the seam** (`internal/db/store/store.go`): it's `db.Querier` (every
generated query method) plus `TransferTx`, `CreateAccountTx`, `DepositTx`,
`DeleteAccountTx`. Every `Handler` struct holds a `store.Storer`, never a
concrete type, so `internal/db/mock.MockStorer` (gomock-generated) can stand in
for handler tests without a database.

## What the Major Modules Are

```
cmd/server/            entry point: loads config, connects DB, builds logger, calls api.NewServer, starts Gin
cmd/migration/         standalone CLI wrapping golang-migrate (create/up/down/force/goto)
internal/api/
  server.go            Server struct, NewServer (middleware chain, validator setup, CORS)
  router.go            bindRouters — the ONLY place route groups are composed
  core/                shared kernel: AppError, response envelope, auth/logging/recovery/request-id/rate-limit middleware
  auth/                login, register, profile — passwordless, device-fingerprint-bound (public + authorized routes)
  account/             create/list/search/get/update/delete account
  transfer/            deposit, cross-account transfer, entries, transaction history
internal/db/
  sqlc/                generated code (DO NOT hand-edit) — Querier interface + per-query methods
  store/                hand-written Storer implementation + *Tx business logic
  query/*.sql          hand-written SQL, source of truth for sqlc generation
  migrations/          golang-migrate up/down SQL pairs
  mock/querier.go       generated MockStorer (gomock) — DO NOT hand-edit
internal/token/        JWT Maker interface + implementation, Payload struct
internal/rate-limiter/ generic in-process per-key token-bucket limiter (no HTTP dependency)
internal/config/       viper-based Config struct + LoadConfig
internal/logger/       slog.Logger construction from Config (JSON/text/pretty)
internal/domain/constant/  small shared enums (e.g. entry types)
internal/util/         generic helpers: array.go (MapSlice), fingerprint.go (device fingerprint hash), like.go (SQL LIKE escaping)
internal/docs/         generated Swagger spec (DO NOT hand-edit)
docs/                  design-decision write-ups per subsystem (see "Existing docs" below)
```

**Module dependency direction:** `cmd` → `internal/api` → `internal/db/store` →
`internal/db/sqlc`. `internal/api/core` has no dependency on `Server` or any
domain package — it's imported *by* `auth`/`account`/`transfer`, never the
other way. `internal/db/store` depends only on `internal/db/sqlc`, never on
`internal/api`.

## What the Important Abstractions Are

- **`Storer`** (`internal/db/store/store.go`) — the interface every `Handler`
  depends on. Adding a method to it means implementing it on `*_store` (or,
  for a generated query, adding SQL + running `make sqlc`).
- **`Handler`** (one per domain package: `auth.Handler`, `account.Handler`,
  `transfer.Handler`) — holds a `store.Storer` (and, for `auth`, a
  `token.Maker`), and implements the single `RegisterRoutes(public,
  authorized *gin.RouterGroup)` method every domain package uses (the
  `router` interface in `internal/api/router.go`). A domain with no public
  endpoints (`account`, `transfer`) just ignores the `public` argument.
  Handlers never call `ctx.JSON` directly.
- **`*Tx` functions** (`store_transaction.go`, `store_account_create_tx.go`,
  `store_account_deposit_tx.go`, `store_account.go`) — the business-logic
  layer. Each has a public `FooTx(ctx, arg)` method on `*_store` that opens a
  transaction via `execTx`, and a private `fooTx(ctx, q sqlc.Querier, arg)`
  function containing the actual logic, parameterized on the `sqlc.Querier`
  interface (not the concrete `*sqlc.Queries`) so it's unit-testable with a mock.
- **`core.AppError`** (`internal/api/core/apperror.go`) — the one error type
  handlers report via `core.Fail(ctx, err)`. Carries `Status`, `Code`,
  `Message`, `Details []FieldError`, and an unexported-from-response `Cause`
  used only for server-side logging of 500s.
- **`*AppError`-mapping functions** — small per-domain functions
  (`transfer.transferAppError`, `account.deleteAccountAppError`) that turn a
  `store` sentinel error into the right `*AppError`. New store errors get a new
  `case` here, not ad-hoc `errors.Is` checks scattered in handlers.
- **Store sentinel errors** (`internal/db/store/errors.go`) — `ErrSameAccount`,
  `ErrCurrencyMismatch`, `ErrInvalidAmount`, `ErrInsufficientFunds`,
  `ErrCannotDeleteMainAccount`. Plain `errors.New`, matched with `errors.Is`.
- **`token.Maker` / `token.Payload`** (`internal/token/`) — JWT creation and
  verification; `Payload` is what `core.GetAuthPayload(ctx)` returns inside a
  handler behind `core.AuthMiddleware`.
- **Device fingerprint auth** (`internal/api/auth/handler_utils.go`,
  `internal/util/fingerprint.go`) — there is no password. `users.device_fingerprint_hash`
  is a SHA-256 hash derived from `User-Agent`/`Accept-Language`/`Accept-Encoding`
  headers via `util.ComputeDeviceFingerprint`. Register binds it; login rejects
  with 403 (`core.ForbiddenErr`) if the caller's current fingerprint doesn't
  match, unless the row is unbound (`""`, a pre-migration legacy row), in which
  case login binds it. `Handler.TrustedIPs` (config `DEVICE_FINGERPRINT_TRUSTED_IPS`)
  exempts specific IPs to a fixed mock fingerprint, purely so local tooling
  (Swagger UI, curl) doesn't get locked out as headers vary — never set this
  in production.
- **`ratelimiter.Limiter`** (`internal/rate-limiter/limiter.go`) — a generic,
  in-process per-key token bucket (no HTTP dependency, no shared store —
  resets on restart, not shared across instances). `core.RateLimitMiddleware`
  (`internal/api/core/rate_limit_middleware.go`) is the Gin adapter: keys it
  by `ctx.ClientIP()`, applies globally before any auth check (public and
  authorized routes alike), and is only registered when
  `config.RateLimitEnabled` is true.
- **`db.Querier`** (`internal/db/sqlc/querier.go`, generated) — one method per
  SQL query. This is the mockable seam for business-logic unit tests (see
  `docs/GOMOCK_TESTING.md`, though see the note on drift below).
- **`config.Config`** (`internal/config/config.go`) — every runtime setting,
  `mapstructure`-tagged, loaded from `app.env` and/or real env vars.

## What Technologies Are Used

| Concern | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP framework | Gin (`gin-gonic/gin`) |
| Database | PostgreSQL, accessed via `lib/pq` |
| Query layer | `sqlc` (generated, no ORM, no hand-written mapper package) |
| Migrations | `golang-migrate`, invoked via `cmd/migration` |
| Money type | `string` end-to-end, parsed with `shopspring/decimal` at point of use |
| Auth | JWT (`golang-jwt/jwt/v5`), stateless, bearer token, **passwordless** (device-fingerprint-bound) |
| Rate limiting | in-process per-IP token bucket (`golang.org/x/time/rate`), no shared store |
| Config | `spf13/viper` + `go-viper/mapstructure` |
| Logging | stdlib `log/slog` (JSON, text, or pretty-JSON handler) |
| Validation | `go-playground/validator/v10` (via Gin binding) |
| Unit testing (store/handlers) | `go.uber.org/mock` (gomock) against `MockStorer`/generated interfaces |
| Integration testing | real Postgres, `stretchr/testify/require` |
| API docs | `swaggo/swag` + `gin-swagger`, generated into `internal/docs` |
| CORS | `gin-contrib/cors`, origin-list based (never `*`) |
| Containerization | Docker (`Dockerfile`, `Dockerfile.prod`) |
| Infra | Terraform, targeting a single EC2 instance behind an nginx reverse proxy/rate limiter — lives at the repo root now (`infra/terraform/`, not under this app; see `infra/terraform/docs/TERRAFORM_*.md`) |

## How the Code Is Organized

Within `internal/api/<domain>`, files typically split as:
- `<domain>.go` — `Handler` struct + `RegisterRoutes`
- `handlers.go` / `manage.go` / `transactions_handlers.go` — the actual
  endpoint functions, one per route, each with a Swagger `godoc` comment block
- `response.go` — response DTOs and `toXResponse`/`toXuserResponse` mapping
  functions from `sqlc` row types
- `apperror.go` — the domain's `errors.Is`/`errors.As` → `*core.AppError`
  mapping function(s)
- `<domain>_handler_test.go` — gomock-based handler tests

`internal/api/core` holds everything cross-cutting: `apperror.go`,
`response.go` (success envelope + pagination), `auth_middleware.go`,
`logging_middleware.go`, `recovery`/`error handler` middleware (also in
`logging_middleware.go`/inline in `server.go`'s comments — check the file for
current registration order).

## How to Implement a New Feature

Follow the shape already used by `account`/`transfer`/`auth`:

1. **Schema** — add/alter a table via a new migration pair in
   `internal/db/migrations/` (see "How to Work With the Database").
2. **Query** — write the SQL in `internal/db/query/*.sql` (new file or append
   to the relevant existing one), then run `make sqlc` to regenerate
   `internal/db/sqlc/`.
3. **Mock** — run `make generate-mock` (regenerates `internal/db/mock/querier.go`
   from the now-changed `Storer` interface). Treat `make sqlc` and
   `make generate-mock` as one step — nothing currently enforces running the
   second after the first.
4. **Business logic** — if the feature is a single query, call `h.Store.X`
   directly from the handler. If it's multi-step (more than one write that
   must commit/rollback together), add a new `store_<name>_tx.go` in
   `internal/db/store/` following the `FooTx`/`fooTx(ctx, q sqlc.Querier, arg)`
   + `execTx` shape (see `store_transaction.go`, `store_account_create_tx.go`,
   `store_account.go`), and add the new method to the `Storer` interface in
   `store.go`.
5. **Domain errors** — if the new logic has failure modes callers need to
   distinguish, add a sentinel `errors.New(...)` to `internal/db/store/errors.go`.
6. **Handler** — add the endpoint function to the owning domain package (or
   create a new domain package under `internal/api/` if it's a genuinely new
   resource), following the bind → call Store → map error → `core.Succeed`
   shape below. Register the route in that package's `RegisterRoutes(public,
   authorized *gin.RouterGroup)`. Wire a brand-new domain package's `Handler`
   into the `routers` slice in `internal/api/router.go`'s `bindRouters` —
   existing domains never need `router.go` touched for a new endpoint on an
   existing resource.
7. **Error mapping** — extend or add an `*AppError`-mapping function in that
   domain's `apperror.go` for any new sentinel/DB error the handler can hit.
8. **Response DTO** — add a response struct + `toXResponse` mapper in that
   domain's `response.go` if the shape returned to clients differs from the
   raw `sqlc` row.
9. **Swagger annotations** — add `@Summary`/`@Router`/etc. godoc comments above
   the handler function, then run `make swag-gen`.
10. **Tests** — add/extend the domain's `_handler_test.go` (gomock,
    `MockStorer`, no DB) and, if you touched `internal/db/sqlc` queries
    directly, a corresponding case in the relevant `internal/db/sqlc/*_test.go`
    (real Postgres, `make db-up` first).

## How to Modify Existing Code

- **Business logic** lives in `internal/db/store` (the `*Tx` functions), never
  in a handler. A handler should be a thin translator: bind request → call one
  `Storer` method → map error → write response.
- **Database queries** live in `internal/db/query/*.sql`; never write raw SQL
  inline in Go code, and never hand-edit `internal/db/sqlc/*.sql.go`.
- **Validation** of request shape (required fields, types, ranges) happens via
  Gin binding tags (`binding:"required,min=1"`) on request structs, surfaced
  through `core.FieldErrorsFromBindErr`. Business-rule validation (e.g. "from
  and to account must differ", "insufficient funds") happens in the store's
  `*Tx` function and comes back as a sentinel error, not a bind-time check.
- **Errors**: a handler never returns a raw `error` to the client and never
  calls `ctx.JSON` on an error path. It always converts to `*core.AppError`
  via `core.ValidationErr`/`NotFoundErr`/`UnauthorizedErr`/`ForbiddenErr`/
  `ConflictErr`/`BadRequestErr`/`InternalErr`, or a domain `*AppError`-mapping
  function, then calls `core.Fail(ctx, err)`.
- **Dependency injection** is manual and minimal: `Handler` structs take their
  dependencies (`store.Storer`, `token.Maker`, a `time.Duration`) as plain
  struct fields set at construction time in `router.go`'s `bindRouters`. There
  is no DI container/framework.
- **Shared functionality** that multiple domain packages need belongs in
  `internal/api/core` (HTTP-layer concerns), `internal/util` (generic
  helpers), or `internal/domain/constant` (shared enums) — not duplicated
  per-package.
- **Money**: always `string` in `sqlc`/`Querier`/JSON; parse with
  `decimal.NewFromString` at the point you need to do arithmetic or compare.
  Never introduce a `float64` or numeric-typed money field anywhere in the
  chain — this was a deliberate choice to avoid float precision bugs.
- **Locking order for any multi-account transaction**: always lock accounts in
  ascending-ID order via `GetAccountByIdForUpdate`, regardless of logical
  from/to roles, then figure out which locked row is which afterward (see
  `transferTx` and `deleteAccountTx`). This is what prevents two concurrent
  transactions on the same pair of accounts from deadlocking. Don't reorder
  these calls or lock accounts in request-order.

## How to Implement APIs

- **Routing**: `/api/v1` prefix, split into a public group (no auth) and an
  `authorized` group behind `core.AuthMiddleware`. Register new routes on the
  group the owning domain's `Handler.RegisterRoutes` receives — never add
  routes directly in `router.go` beyond the `bindRouters` wiring.
- **Handler structure**: `ctx.ShouldBindJSON`/`ShouldBindUri`/`ShouldBindQuery`
  → on error, `core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))`
  → call one `Storer` method → on error, map it (sentinel/`sql.ErrNoRows`/`pq.Error`
  → `*core.AppError`) → `core.Succeed`/`core.SucceedWithMeta`.
- **Response shape** (`internal/api/core/response.go`): every success response
  is `{"data": ..., "message": "...", "meta": ...}` (`meta` omitted when nil);
  every error response is `{"error": {"code", "message", "details"}}`. Never
  deviate from this — `core.Succeed`/`SucceedWithMeta`/`Fail` are the only
  sanctioned way to write a response.
- **Pagination**: embed `core.PaginationQuery` (`page`/`limit` form fields,
  `min=1`/`max=100`) in list query-param structs, use `.Offset()` for the SQL
  offset, and return via `core.SucceedWithMeta` with a `core.Meta{Page, Limit,
  Total}`. `Total` comes from a paired `*Count` sqlc query.
- **Filtering**: optional query params bind with `binding:"omitempty"` and are
  passed to sqlc as `sql.NullString`/similar, matching the `WHERE ... IS NULL
  OR ...`-style query pattern already used (see `account/handlers.go`).
- **Authentication**: `core.AuthMiddleware(tokenMaker)` on the `authorized`
  group parses `Authorization: Bearer <token>`, verifies it, and stores
  `*token.Payload` in the Gin context. Read it in a handler with
  `core.GetAuthPayload(ctx)` — never re-parse the header yourself.
- **Authorization**: resource ownership is checked per-handler by comparing
  `authPayload.ID` against the row's `UserID`/owner column, returning
  `core.ForbiddenErr(...)` on mismatch (see `account/manage.go`). There is no
  role/permission system — every authenticated user has identical capabilities
  beyond resource ownership.

## How to Work With the Database

- **Migrations** live in `internal/db/migrations/`, as `NNNNNN_description.up.sql`
  / `.down.sql` pairs (sequential 6-digit prefix). Create a new pair with
  `make migrate-create name=add_something` (never hand-number a new pair).
  Apply with `make migrate-up` (or `make db-up`, which starts Postgres via
  docker-compose and runs pending migrations). `make migrate-force
  version=N` / `make migrate-goto version=N` exist for recovering from a dirty
  migration state — don't reach for them casually.
- **Schema is the source of truth for sqlc** — `sqlc.yaml` points its `schema:`
  at `internal/db/migrations/`, so sqlc's generated Go types come from the
  *migrations*, not a separate model file.
- **Queries**: written by hand in `internal/db/query/*.sql`, one file roughly
  per table/feature (`account.sql`, `account-manage.sql`,
  `account-transaction.sql`, `entries.sql`, `transfers.sql`, `user.sql`).
  Prefer `SELECT *` / `RETURNING *` and `sqlc.embed(...)` to reuse an existing
  generated row type for a new query shape, rather than hand-writing a new
  mapper — there is no `db/mapper` package (deleted; see
  `docs/SQLC_ROW_MAPPING.md`).
- **Regenerate** with `make sqlc` after any query/schema change, then
  `make generate-mock` (the mock is generated *from* the sqlc-generated
  `Querier`/`Storer` interface, so it must come second).
- **Transactions**: any operation touching more than one row/table
  atomically goes through `execTx` (`internal/db/store/store.go`) — open a
  `*sql.Tx`, run a function against a `Queries` built on that tx, commit/rollback
  based on the function's returned error. Never issue multiple independent
  `Storer` calls from a handler when they need to be atomic.
- **Constraints worth knowing**: accounts have DB-level balance check
  constraints (migration `000002_add-balance-constraints`) — a `pq.Error`
  with code `check_violation` on a balance-changing query means insufficient
  funds and should map to a 409, not a 500 (see `transferAppError`). Account
  deletion is a **soft delete** (`SoftDeleteAccount`), and a main account
  (`IsMain`) cannot be deleted (`ErrCannotDeleteMainAccount`) — deleting a
  non-main account with a positive balance sweeps that balance to the user's
  main account as an ordinary transfer first (see `deleteAccountTx` in
  `store_account.go`).

## How to Handle Errors

```
store sentinel error / sql.ErrNoRows / *pq.Error
         ↓
domain's *AppError-mapping function (e.g. transferAppError, deleteAccountAppError)
         ↓
core.Fail(ctx, appErr)   — records on ctx.Errors, aborts the chain
         ↓
core.ErrorHandlerMiddleware — the ONLY place that calls ctx.JSON for an error
         ↓
{"error": {"code", "message", "details"}}  — sanitized 500 for anything not an *AppError
```

- Never call `ctx.JSON` for an error directly in a handler.
- Every new store-layer sentinel error needs a `case` added to the relevant
  domain's mapping function — don't scatter `errors.Is` checks across handlers.
- `core.InternalErr(err)` always takes the real underlying error as `Cause`
  (logged, never sent to the client) — never call it with `nil`.
- 5xx `AppError`s are logged at `Error` level (with `Cause`); everything else
  logs at `Warn`. This happens automatically in `ErrorHandlerMiddleware` — you
  don't log manually in a handler.

## How to Handle Logging

- One shared `*slog.Logger`, built once in `cmd/server/main.go` via
  `logger.New(cfg)` and threaded through `api.NewServer`. Don't construct a
  second logger or reach for the stdlib `log` package inside `internal/api`
  or `internal/db`.
- `LOG_LEVEL` (`debug`/`info`/`warn`/`error`) and `LOG_FORMAT` (`json`
  default, or `text`) are config-driven (`internal/config/config.go`,
  `internal/logger/logger.go`). `LOG_PRETTY_JSON` indents JSON for local dev
  only — never enable it where a log aggregator expects one line per record.
- Every request gets a `request_id` (from `X-Request-Id` if the caller sent
  one, generated otherwise) via `core.RequestIDMiddleware`, echoed back on the
  response and attached to every log line for that request
  (`core.LoggingMiddleware`).
- `LOG_REQUEST_BODY`/`LOG_RESPONSE_BODY` opt into capturing bodies on the
  access log line, capped by `LOG_MAX_BODY_SIZE` (default 1MiB) — treat these
  as a local-debugging aid, not something to enable in production without
  checking for sensitive data exposure (see `internal/logger/redact.go` for
  existing redaction).
- Middleware order matters: `RequestIDMiddleware` → `LoggingMiddleware` →
  `corsMiddleware` → `ErrorHandlerMiddleware` → `RateLimitMiddleware` (only if
  `RateLimitEnabled`) → `RecoveryMiddleware` (see the detailed comment in
  `server.go` on *why* `ErrorHandlerMiddleware`/`RateLimitMiddleware` must be
  registered before `RecoveryMiddleware` — don't reorder without re-reading it).

## How Configuration Works

- `config.LoadConfig(".")` reads `app.env` (viper, `env` format) if present,
  and always binds every `mapstructure`-tagged `Config` field to a real
  environment variable too — so the service runs identically with or without
  an `app.env` file (e.g. in docker-compose/CI where only real env vars are
  set). See `docs/CONFIG_ENV_VARIABLE.md` for the exact precedence rules.
- Copy `app.env.example` → `app.env` for local dev; never commit real secrets
  in `app.env`. `app.prod.env` and `core-service-key.pem` exist in this repo
  root — treat both as sensitive, don't print their contents, and don't add
  new secrets as plain files under version control.
- **No startup validation** on required fields today: a missing/empty
  `JWT_SECRET_KEY` or `DB_SOURCE` isn't caught by `LoadConfig` or `NewServer`
  up front — a too-short JWT secret fails loudly via `NewJWTMaker`'s length
  check, but other missing values fail later and less clearly (`sql.Open`,
  etc.). If you add a new required config field, don't assume validation
  happens elsewhere — there currently is none.
- `CORS_ALLOWED_ORIGINS` is a comma-separated list; empty means CORS is fully
  disabled (no `Access-Control-*` headers at all), not "allow everything." A
  wildcard `*` doesn't actually work here because `AllowCredentials: true` is
  always set — see `docs/CORS.md`.
- `RATE_LIMIT_ENABLED` (bool) gates whether `core.RateLimitMiddleware` is
  registered at all; `RATE_LIMIT_REQUESTS_PER_SECOND` (float64) and
  `RATE_LIMIT_BURST` (int) size the per-IP token bucket.
- `DEVICE_FINGERPRINT_TRUSTED_IPS` is a comma-separated IP list; empty (the
  default) disables the fingerprint bypass. See "Device fingerprint auth"
  above — never set this in production.

## How to Write Tests

Two distinct kinds — know which one you're writing/running:

1. **Unit tests against gomock** — e.g. `internal/api/account/account_handler_test.go`,
   `internal/api/transfer/transfer_handler_test.go`, `internal/api/auth/auth_handler_test.go`.
   No database. Pattern: build a bare `gin.Engine` with the same middleware a
   real request would hit (`core.ErrorHandlerMiddleware`, `core.AuthMiddleware`
   for authorized routes), construct the domain's `Handler` with a
   `mockdb.NewMockStorer(gomock.NewController(t))`, set `.EXPECT()`
   expectations, fire a request with `httptest`, assert on the recorded
   response. Follow the existing `newTestRouter`/`authHeaderFor`/
   `doAuthenticatedRequest` helpers already defined per test file rather than
   inventing new plumbing.
2. **Integration tests against real Postgres** — `internal/db/sqlc/*_test.go`
   (`account_test.go`, `entries_test.go`, `transfers_test.go`). Each test opens
   a transaction via `createTestQueries(t)` (from `main_test.go`) and rolls it
   back in `t.Cleanup`, so tests don't pollute each other or need manual
   fixture cleanup. Require `make db-up` first.
- **Mocking strategy**: gomock, generated from the `Storer` interface
  (`internal/db/mock/querier.go`, `make generate-mock`). Match `ctx` with
  `gomock.Any()`; match business parameters exactly. Use `gomock.InOrder(...)`
  when call sequence is part of correctness (e.g. lock-then-write ordering),
  and `.Times(0)` to assert a call must *not* happen (e.g. proving an early
  validation failure short-circuits before touching the store). See
  `docs/GOMOCK_TESTING.md` for the full rationale — but see "Doc drift" below:
  that doc's file paths for store-level tests no longer exist.
- **Running tests**:
  - `make test` — `make db-up` then `go test -v ./...` (everything).
  - `go test ./internal/db/store/...` — one package.
  - `go test -run TestTransferTx ./internal/db/store` — one test by name
    (note: as of this writing there is no test named this — see "Doc drift").
  - `make test-coverage` — coverage for `internal/api/...` only, written to
    `coverage.out`.
  - `make test-failed` — runs the full suite and greps for failures only.

## How to Run and Verify Changes

```bash
make db-up     # start Postgres + apply pending migrations
make server    # go run ./cmd/server  (needs app.env or equivalent env vars)
make test      # db-up, then go test -v ./... (all unit + integration tests)
```

After making changes, match validation depth to what changed:

```
Handler / request-response shape change
  → go build ./...
  → that domain's *_handler_test.go
  → make swag-gen if you touched a godoc annotation, then check internal/docs diff

Store / *Tx business logic change
  → go test ./internal/db/store/...
  → go test ./internal/db/sqlc/...   (make db-up first)
  → the calling domain's *_handler_test.go, since handler tests mock the whole
    Storer and won't catch a *Tx regression — see "Doc drift" for why this
    matters more than it used to

New/changed SQL query
  → make sqlc
  → make generate-mock   (always the second step, from the regenerated interface)
  → go build ./...        (catches every call site sqlc's signature change broke)
  → go test ./internal/db/sqlc/...

New migration
  → make migrate-up (or make db-up) against a disposable local DB
  → make migrate-down to confirm the down migration is correct and symmetric
  → make sqlc (schema changed) → make generate-mock

Config change
  → update app.env.example
  → update docs/CONFIG_ENV_VARIABLE.md if precedence/defaults changed
```

Before finishing any change, review the actual diff — this codebase has
several generated files (below) where an unexpected diff usually means you
ran a generator against stale input, not that the generator is wrong.

## Important Constraints

- **Never hand-edit generated files**: `internal/db/sqlc/*.sql.go` (from
  `make sqlc`), `internal/db/mock/querier.go` (from `make generate-mock`),
  `internal/docs/*` (from `make swag-gen`). All three carry a
  `// Code generated ... DO NOT EDIT.` header. Change the source instead
  (`internal/db/query/*.sql`, the `Storer` interface, or handler godoc
  comments) and regenerate.
- **Locking order in multi-account transactions is a correctness property,
  not a style choice** — reordering the `GetAccountByIdForUpdate` calls in
  `transferTx`/`deleteAccountTx` (or writing a new multi-account `*Tx` without
  the same ascending-ID lock order) reintroduces a real deadlock risk under
  concurrent load.
- **Money must stay `string` end-to-end.** Introducing a `float64` or a
  numeric DB column anywhere in the money path (params, sqlc types, JSON)
  reintroduces the precision bugs this design avoids. Always parse with
  `decimal.NewFromString` at the point of use.
- **CORS**: don't "fix" a CORS issue by setting `AllowOrigins: []string{"*"}`
  — it's a no-op here because `AllowCredentials: true` is always set, and
  browsers reject the combination. Add the real origin to
  `CORS_ALLOWED_ORIGINS` instead.
- **Auth is stateless — there is no logout/blacklist mechanism.** A leaked
  token is valid until `JWTAccessTokenDuration` expires; there's no
  server-side record of issued tokens. Don't build a feature that assumes
  tokens can be revoked without first adding server-side token tracking.
- **There is no password, deliberately** — `users` has no password column
  (dropped in migration `000013`); identity is email + device fingerprint
  (see "Device fingerprint auth" above). Don't add a password field/flow back
  in without an explicit ask — this was an intentional design change, not an
  oversight.
- **No role/permission system.** `token.Payload` carries no role/scope claim.
  All authorization is per-resource ownership comparison in the handler. If
  you add roles, look them up per-request from the DB — don't bake a role
  into a long-lived JWT.
- **No startup config validation** — see "How Configuration Works." Don't
  assume a missing required env var fails fast and clearly; it may fail
  deep in a stdlib call instead.
- **Middleware registration order in `server.go` is load-bearing.**
  `ErrorHandlerMiddleware` must be registered before `RecoveryMiddleware` or a
  panic recovers into a response that never gets written (client sees an
  empty 200). Read the comment above `router.Use(...)` calls in
  `internal/api/server.go` before touching that block.
- **`app.env`, `app.prod.env`, `core-service-key.pem`** live at the repo root.
  Treat all three as secrets — never log their contents, never include them
  in a diff you're about to show/commit, and don't add new plaintext secret
  files alongside them.

## Legacy / Special Cases

- **`internal/domain/constant`** holds only entry-type enums today
  (`ENTRY_TYPE_SEND`/`ENTRY_TYPE_RECEIVED`) — it's a real, small shared
  package, not dead scaffolding; extend it rather than redefining an entry
  type constant locally in a new file.
- **Account numbers** are generated deterministically from the account ID
  after insert (`generateAccountNumber`, `store_account_create_tx.go`) via a
  two-step insert-then-update (`CreateAccount` then `UpdateAccountNumber`),
  because the number depends on the row's own auto-generated ID. Don't try to
  collapse this into a single `RETURNING *` insert.
- **Account deletion is a sweep-then-soft-delete**, not a hard delete and not
  a no-op on balance: `deleteAccountTx` moves any positive balance to the
  user's main account as a real transfer (with its own entries) before
  calling `SoftDeleteAccount`. A main account itself can never be deleted
  (`ErrCannotDeleteMainAccount`).

## Doc drift — read this before trusting `CLAUDE.md` or `docs/*.md` blindly

This repository was recently refactored: `internal/api` was split from a
flatter layout into per-domain packages (`internal/api/account`,
`internal/api/auth`, `internal/api/transfer`, `internal/api/core`), and
`internal/db/store`'s dedicated `store_transaction_test.go` /
`store_transaction_integration_test.go` / `main_test.go` files described in
`docs/GOMOCK_TESTING.md` **no longer exist in this repo** — they were removed
in a "update store codebase" commit. Concretely, as of this writing:

- `transferTx`'s business logic (locking order, decimal math, short-circuit
  validation) has **no dedicated unit test** against a mocked `sqlc.Querier`
  anymore. The only coverage is indirect, through
  `internal/api/transfer/transfer_handler_test.go`, which mocks the *entire*
  `Storer.TransferTx` call — so it cannot catch a regression inside
  `transferTx` itself (wrong lock order, wrong call sequence). If you touch
  `transferTx`/`deleteAccountTx`/`createAccountTx`, consider adding a
  `internal/db/store/*_test.go` against a mocked `sqlc.Querier` (the pattern
  `docs/GOMOCK_TESTING.md` describes still applies conceptually — its file
  paths are just stale) rather than assuming existing tests cover it.
- Function/type names in `CLAUDE.md` and `docs/IMPLEMENT_AUTH.md` /
  `docs/NORMALIZE_RESPONSE.md` (e.g. `fail(ctx, err)`, `internal/api/apperror.go`,
  `internal/api/user_router.go`, `POST /api/v1/users`) reflect the pre-refactor
  layout. The current, correct names are `core.Fail`/`core.Succeed` in
  `internal/api/core/`, and there is no `/users`/`user_router.go` in the
  current router at all.
- Treat every `docs/*.md` file's *design rationale* (why a decision was made)
  as reliable, but verify any *file path, function name, or "current test
  status" claim* against the actual code before relying on it — several of
  these docs predate the per-domain package split.
