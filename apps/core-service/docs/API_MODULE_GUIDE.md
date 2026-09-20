# Adding a New API Module — As Implemented

## Who this doc is for

You're comfortable with Go and Gin, and you already know this repo has
`account`, `auth`, and `transfer` as sibling domain packages under
`internal/api/`. This doc walks through building a **fourth one** from
scratch — the HTTP layer only. It deliberately stops at the `store.Storer`
boundary: schema, SQL, migrations, and `*Tx` business logic are a separate
concern with their own workflow in `AGENTS.md` §"How to Work With the
Database" — this doc assumes whatever `Storer` methods you need either
already exist or are being added alongside this work by someone following
that other workflow.

Everything below is verified against the source as of the commits on
`epic/update_core_services` as of 2026-09-19, file:line cited throughout,
using `internal/api/account` as the running worked example because it's the
most structurally complete of the three existing modules (create, list,
search, get, update, delete). This is **not** a design aspiration — where a
step is a convention rather than something enforced by the compiler, that's
called out so you know where you can genuinely get it wrong without an error.

## 1. Architecture at a glance

The composition root is `internal/api/router.go` — `bindRouters`
(`internal/api/router.go:21`) is the *only* place `Handler` structs get
constructed and handed a route group. It doesn't implement any endpoint logic
itself; it wires. `internal/api/server.go`'s `NewServer`
(`internal/api/server.go:40`) builds the `Server`, registers global
middleware, then calls `bindRouters` once.

| Concern | Owner | Analogy |
|---|---|---|
| Compose route groups, construct handlers | `internal/api/router.go` (`bindRouters`) | The switchboard operator plugging each department into the right line |
| One resource's endpoints | `internal/api/<domain>/` (`Handler`, e.g. `account.Handler`) | A teller window for one specific kind of transaction |
| Cross-cutting request/response mechanics | `internal/api/core` | The bank's shared paperwork templates — every teller uses the same deposit slip |
| Business rules + persistence | `internal/db/store` (`Storer`) | The vault and ledger room behind the tellers — **out of scope for this doc** |

This split exists so a `Handler` can be unit-tested against a mock `Storer`
with zero database, and so `internal/api/core` never has to know a specific
domain exists (`internal/api/core/apperror.go:1`'s package doc says this
explicitly: core "has no dependency on Server or any domain package"). The
consequence you'll feel directly: every new module gets its own
`apperror.go` for domain-specific error codes rather than adding cases to
`core`'s (see §7) — `core` only owns the *generic* codes
(`internal/api/core/apperror.go:20-27`).

## 2. Step 1 — Create the package skeleton

One directory per resource under `internal/api/`, named after the resource in
lowercase (`account`, `auth`, `transfer` — not `accounts`, not
`AccountApi`). Inside it, the three existing modules split files the same way
— follow it even though nothing enforces the split:

| File | Holds |
|---|---|
| `<domain>.go` | `Handler` struct + `RegisterRoutes` (or `RegisterPublicRoutes`/`RegisterAuthorizedRoutes`) |
| `handlers.go` (or split further, e.g. `manage.go`, `transactions_handlers.go`) | The endpoint functions, one per route, each with a Swagger godoc block |
| `response.go` | Response DTOs + `toXResponse` mappers from `sqlc` row types |
| `apperror.go` | This domain's sentinel-error → `*core.AppError` mapping function(s) |
| `<domain>_handler_test.go` | gomock-based handler tests, no DB |

A brand-new resource is the *only* time you touch `router.go` — see §4. Every
endpoint added to an *existing* domain later only touches that domain's own
files.

## 3. Step 2 — Define the `Handler` struct and route registration

`Handler` is a plain struct holding exactly what its endpoints need — never
more. `account.Handler` (`internal/api/account/account.go:13`) needs only a
store:

```go
// internal/api/account/account.go:13
type Handler struct {
    Store store.Storer
}
```

`auth.Handler` (`internal/api/auth/auth.go:16`) needs more, because it issues
tokens:

```go
// internal/api/auth/auth.go:16
type Handler struct {
    Store               store.Storer
    TokenMaker          token.Maker
    AccessTokenDuration time.Duration
}
```

There's no DI container — whatever a handler needs becomes a struct field,
set by hand in `router.go` (§4).

**Route registration** is a method on `*Handler` that takes the `*gin.RouterGroup`
it should register onto and calls `.GET`/`.POST`/etc. Two shapes exist,
depending on whether the resource has both public and authenticated
endpoints:

- **All-authenticated resource** (account, transfer): one method,
  `RegisterRoutes(rg *gin.RouterGroup)`
  (`internal/api/account/account.go:25`):
  ```go
  func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
      rg.POST("/accounts", h.createAccount)
      rg.GET("/accounts/search-by-number", h.searchAccountByNumber)
      rg.GET("/accounts/me", h.listmeAccounts)
      rg.GET("/accounts/:id", h.detailAccount)
      rg.PUT("/accounts/:id", h.updateAccount)
      rg.DELETE("/accounts/:id", h.deleteAccount)
  }
  ```
- **Mixed public/authenticated resource** (auth): two methods,
  `RegisterPublicRoutes` and `RegisterAuthorizedRoutes`
  (`internal/api/auth/auth.go:24-33`) — one per group, called from the two
  different groups in `router.go`.

If any of your new resource's routes share a URL prefix (`:id`), define a
shared path-param struct once at the package level rather than per-handler —
see `idPathParam` (`internal/api/account/account.go:19`), reused by
`detailAccount`, `updateAccount`, and `deleteAccount`.

## 4. Step 3 — Wire the handler into `router.go`

This is the one step that touches shared code. `bindRouters`
(`internal/api/router.go:21-47`) does three things per module: construct the
`Handler` with its dependencies, then call the right registration method(s)
on the right group.

```go
// internal/api/router.go:42-46 — the account module's entire wiring
accountHandler := &account.Handler{Store: server.store}
accountHandler.RegisterRoutes(authorized)

transferHandler := &transfer.Handler{Store: server.store}
transferHandler.RegisterRoutes(authorized)
```

For your new module, add the equivalent two lines. If it's all-authenticated,
add them after the `authorized := v1.Group("/")` /
`authorized.Use(core.AuthMiddleware(...))` block
(`internal/api/router.go:37-40`) — same as account/transfer. If it needs
public routes too, follow auth's pattern instead: construct the handler once,
call `RegisterPublicRoutes(v1)` on the pre-auth group, and
`RegisterAuthorizedRoutes(authorized)` on the post-auth one
(`internal/api/router.go:28-40`).

**Rough edge worth knowing:** nothing stops you from registering the same
`Handler` on both groups, or forgetting to add it to either — a missing
`RegisterRoutes` call compiles fine and just silently 404s every request to
that resource. There's no route-registration test that would catch this; a
handler test only proves the handler works *if* it's wired, and a handler
test builds its own bare router (§9) rather than exercising `bindRouters`
directly.

## 5. Step 4 — Write an endpoint function

Every endpoint function in this codebase follows the same five-step shape.
`createAccount` (`internal/api/account/handlers.go:34-58`) is close to the
minimum version:

```go
// internal/api/account/handlers.go:34-58
func (h *Handler) createAccount(ctx *gin.Context) {
    var req createAccountRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
        return
    }

    authPayload := core.GetAuthPayload(ctx)

    account, err := h.Store.CreateAccountTx(ctx, store.CreateAccountTxParams{
        UserID:      sql.NullInt64{Int64: authPayload.ID, Valid: true},
        Name:        req.Name,
        Description: req.Description,
        IsMain:      false,
    })
    if err != nil {
        core.Fail(ctx, core.InternalErr(err))
        return
    }

    core.Succeed(ctx, http.StatusOK, toAccountResponse(account), "Account created successfully")
}
```

| Step | What it looks like | Never do instead |
|---|---|---|
| 1. Bind the request | `ctx.ShouldBindJSON`/`ShouldBindUri`/`ShouldBindQuery` into a request struct, `core.Fail` + `core.ValidationErr(core.FieldErrorsFromBindErr(err)...)` on error | Hand-parse `ctx.Param`/`ctx.Query` outside a struct, or write your own field-error formatting |
| 2. Read the caller | `core.GetAuthPayload(ctx)` — only valid on routes behind `core.AuthMiddleware` | Re-parse the `Authorization` header yourself |
| 3. Call exactly one `Storer` method | `h.Store.<Method>(ctx, ...)` | Call the store more than once from a handler when the calls need to be atomic together (that belongs in a `*Tx`, out of scope here) |
| 4. Map the error | A domain `*AppError`-mapping function, or an inline `errors.Is(err, sql.ErrNoRows)` check for a single obvious case (see `detailAccount`, `internal/api/account/manage.go:41-49`) | Call `ctx.JSON` on an error path, or return the raw `error` |
| 5. Respond | `core.Succeed(ctx, status, data, message)` or `core.SucceedWithMeta(...)` for a list | Build the JSON body by hand |

Ownership checks (does this resource belong to the caller?) are a per-handler
comparison, not a middleware — see `detailAccount`
(`internal/api/account/manage.go:51-55`):

```go
authPayload := core.GetAuthPayload(ctx)
if account.AccountUserDetailsView.UserID.Int64 != authPayload.ID {
    core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
    return
}
```

There's no generic "ownership middleware" in this codebase — every new
single-resource endpoint (`GET/PUT/DELETE /accounts/:id`-shaped) repeats this
same three-line check by hand. If your new resource has this shape, copy the
pattern rather than inventing a shared helper — none of the three existing
domains have factored it out, so a new shared helper would be introducing an
abstraction the rest of the codebase doesn't use yet.

## 6. Step 5 — Request DTOs

A request struct per endpoint, defined right above the handler function that
uses it, with Gin binding tags doing all shape validation
(`internal/api/account/handlers.go:16-19`):

```go
type createAccountRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}
```

For list/search endpoints, embed `core.PaginationQuery`
(`internal/api/core/response.go:42-49`) rather than hand-rolling `page`/`limit`
fields:

```go
// internal/api/account/handlers.go:60-63
type listmeAccountsQuery struct {
    core.PaginationQuery
    Name string `form:"name" binding:"omitempty"`
}
```

`PaginationQuery.Offset()` does the page→offset arithmetic for you — pass
`query.Offset()` and `query.Limit` straight into the `Storer` call
(`internal/api/account/handlers.go:94-95`).

## 7. Step 6 — Response DTOs and mappers

Never return a raw `sqlc` row type to the client — always a hand-written
response struct with explicit `json` tags, built by a `toXResponse` function
in `response.go`. `toAccountResponse`
(`internal/api/account/response.go:19-31`):

```go
func toAccountResponse(account db.Account) accountResponse {
    return accountResponse{
        ID:          account.ID,
        Balance:     account.Balance,   // string all the way through — see AGENTS.md's money rule
        Currency:    account.Currency,
        UserID:      account.UserID.Int64,
        Number:      account.Number.String,
        Name:        account.Name.String,
        Description: account.Description.String,
        IsMain:      account.IsMain,
        CreatedAt:   account.CreatedAt.Format("2006-01-02 15:04:05"),
    }
}
```

If several of your endpoints return rows built from the same `sqlc.embed(...)`
view, write **one** mapper for that embedded type and reuse it — don't
hand-write a mapper per query. `toAccountuserResponse`
(`internal/api/account/response.go:51-64`) is reused by three different
queries that all embed `AccountUserDetailsView`; the comment above it
(`internal/api/account/response.go:46-50`) spells out why a fourth query
against the same view won't need a fourth mapper.

## 8. Step 7 — Error mapping (`apperror.go`)

A handler never inspects a sentinel error inline beyond the single-case,
obvious situations shown in §5. Anything with more than one failure mode gets
its own mapping function that turns whatever the store returned into a
`*core.AppError`. `deleteAccountAppError`
(`internal/api/account/manage.go:192-201`):

```go
func deleteAccountAppError(err error) *core.AppError {
    switch {
    case errors.Is(err, sql.ErrNoRows):
        return core.NotFoundErr("account not found")
    case errors.Is(err, store.ErrCannotDeleteMainAccount):
        return core.ConflictErr(errCodeMainAccount, err.Error())
    default:
        return core.InternalErr(err)
    }
}
```

`transferAppError` (`internal/api/transfer/apperror.go:18-39`) is the fuller
version — worth reading before you write your own, because it shows the
`*pq.Error` pattern for constraint violations:

```go
case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
    return core.ConflictErr(errCodeInsufficientFunds, "insufficient funds")
case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
    return core.BadRequestErr(core.ErrCodeNotFound, "account not found")
```

| `core` helper | Status | When to reach for it |
|---|---|---|
| `core.ValidationErr(details...)` | 400 | Field-level bind failure — pass `core.FieldErrorsFromBindErr(err)...` |
| `core.BadRequestErr(code, msg)` | 400 | Business-rule violation not tied to one field |
| `core.NotFoundErr(msg)` | 404 | `sql.ErrNoRows`, or an explicit "doesn't exist" check |
| `core.UnauthorizedErr(msg)` | 401 | Auth failures — normally only inside `core.AuthMiddleware` itself |
| `core.ForbiddenErr(msg)` | 403 | Valid caller, wrong owner |
| `core.ConflictErr(code, msg)` | 409 | A state conflict — insufficient funds, "can't delete the main account," a DB constraint violation |
| `core.InternalErr(err)` | 500 | Anything unexpected — **always** pass the real error as `err`, never `nil` (`internal/api/core/apperror.go:95` says so directly) |

A **domain-specific error code** (like `errCodeMainAccount` or
`errCodeInsufficientFunds`) is a local `const` in that domain's `apperror.go`,
not an addition to `core.ErrCode*` — `core` only owns the generic codes
(`internal/api/core/apperror.go:20-27`). Add a new `case` to your mapping
function for every new sentinel error your `Storer` call can return; don't
scatter `errors.Is` checks across multiple handlers instead.

## 9. Step 8 — Swagger annotations

Every handler function gets a godoc block immediately above it, in the shape
`createAccount` uses (`internal/api/account/handlers.go:21-33`):

```go
// createAccount godoc
// @Summary      Create a new account
// @Description  Create a bank account for the authenticated user
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createAccountRequest  true  "Account creation payload"
// @Success      200      {object}  successResponse{data=accountResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /accounts [post]
func (h *Handler) createAccount(ctx *gin.Context) {
```

Match `@Tags` to the resource name, list every `@Failure` status your error
mapping can actually produce (checking `apperror.go` is the fastest way to
get this right instead of guessing), and set `@Security BearerAuth` only on
routes actually behind `core.AuthMiddleware`. After annotating, run
`make swag-gen` and check the diff under `internal/docs/` — that package is
generated (`// Code generated ... DO NOT EDIT.`), so a hand-edit there is
always wrong and a *missing* diff after `make swag-gen` usually means an
annotation typo the generator silently skipped.

## 10. Step 9 — Handler tests

Handler tests are gomock-based and never touch a database. Every existing
`*_handler_test.go` file defines the same three helpers at the top — copy
them into your new test file rather than importing them from somewhere
shared, since none of the three domains currently share this plumbing (see
`internal/api/account/account_handler_test.go:38-84`):

```go
// newTestRouter wires a bare authorized group behind core.AuthMiddleware,
// exactly as NewServer wires the "authorized" group in production
func newTestRouter(t *testing.T, storer store.Storer) (*gin.Engine, token.Maker) {
    gin.SetMode(gin.TestMode)
    tokenMaker, _ := token.NewJWTMaker(testSecretKey)

    router := gin.New()
    router.Use(core.ErrorHandlerMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))

    authorized := router.Group("/api/v1")
    authorized.Use(core.AuthMiddleware(tokenMaker))

    h := &Handler{Store: storer}
    h.RegisterRoutes(authorized)

    return router, tokenMaker
}
```

Then one table-driven test function per endpoint: build a
`mockdb.NewMockStorer(gomock.NewController(t))`, set `.EXPECT()`s (matching
`ctx` with `gomock.Any()`, business params exactly), fire the request through
`doAuthenticatedRequest`, assert on the status code and, where it matters,
decode the JSON body. Cover at minimum:

| Case | What it proves |
|---|---|
| Happy path | The handler calls the store with the right params and returns 200 with the right shape |
| A bind failure (missing required field, out-of-range pagination) | `.Times(0)` on the store call — validation short-circuits before touching persistence |
| The resource not found (`sql.ErrNoRows`) | 404, mapped correctly |
| The resource owned by another user | 403, and the store's mutating call is `.Times(0)` |
| An unexpected store error (`sql.ErrConnDone`) | 500 via `core.InternalErr`, not a leaked error string |

See `TestDeleteAccount` (`internal/api/account/account_handler_test.go:436-534`)
for the fullest version of this pattern, including a domain-specific sentinel
error (`store.ErrCannotDeleteMainAccount`) asserted as a 409.

## 11. Cross-module coupling to be aware of

- **Middleware order is global, not per-module.** Every request to your new
  module passes through the same chain `NewServer` builds once —
  `RequestIDMiddleware` → `LoggingMiddleware` → CORS → `ErrorHandlerMiddleware`
  → `RecoveryMiddleware` (`internal/api/server.go:69-73`). You don't register
  any of this per-module, but a panic in your handler is only turned into a
  response *because* `ErrorHandlerMiddleware` sits above `RecoveryMiddleware`
  in that chain — see the comment at `internal/api/server.go:54-67` before
  ever touching that ordering.
- **`core` is a one-way dependency.** Your new module imports
  `internal/api/core`; `core` must never import your module back. If you find
  yourself wanting `core` to know something domain-specific (a new error
  code, a new response shape), that's a sign it belongs in your module's own
  `apperror.go`/`response.go` instead (§7, §7-adjacent).
- **`core.GetAuthPayload` panics outside `core.AuthMiddleware`.** It's built
  on `ctx.MustGet` (`internal/api/core/auth_middleware.go:65`) — calling it
  from a route registered on the *public* group (no `AuthMiddleware` in front
  of it) panics at request time, not at compile time. If your module has both
  public and authorized routes (the `auth` shape, §3), only ever call it from
  functions registered via `RegisterAuthorizedRoutes`.

## 12. Data flow — a new endpoint end to end

Walking `POST /api/v1/accounts` (`createAccount`) from wire to response, the
same shape any new endpoint follows:

1. Request hits `RequestIDMiddleware` → `LoggingMiddleware` → CORS →
   `ErrorHandlerMiddleware` → `RecoveryMiddleware`, all registered once in
   `NewServer` (`internal/api/server.go:69-73`).
2. It reaches `authorized.Use(core.AuthMiddleware(...))`
   (`internal/api/router.go:37-38`) — token verified, `*token.Payload` stored
   on the Gin context.
3. Gin routes it to `accountHandler.createAccount`, registered via
   `RegisterRoutes` at `internal/api/router.go:43`.
4. Inside the handler (§5): bind → `core.GetAuthPayload` → one
   `h.Store.CreateAccountTx` call → `toAccountResponse` → `core.Succeed`.
5. On any error instead: `core.Fail(ctx, appErr)` records the error on
   `ctx.Errors` and aborts (`internal/api/core/apperror.go:105-108`) —
   nothing is written to the response yet.
6. Control unwinds back up to `ErrorHandlerMiddleware`
   (`internal/api/core/apperror.go:122-152`), which is the *only* place that
   calls `ctx.JSON` on an error path — it renders `{"error": {code, message,
   details}}`, logging 5xx at `Error` (with `Cause`) and everything else at
   `Warn`.

## 13. Checklist / reference

| Step | Files touched | Command to verify |
|---|---|---|
| 1. Package skeleton | new `internal/api/<domain>/` dir | — |
| 2. `Handler` + route registration | `<domain>.go` | `go build ./...` |
| 3. Wire into router | `internal/api/router.go` | `go build ./...` |
| 4. Endpoint function(s) | `handlers.go` / split files | `go build ./...` |
| 5. Request DTOs | same file as the endpoint | — |
| 6. Response DTOs + mappers | `response.go` | — |
| 7. Error mapping | `apperror.go` | — |
| 8. Swagger annotations | godoc above each handler | `make swag-gen`, check `internal/docs/` diff |
| 9. Handler tests | `<domain>_handler_test.go` | `go test ./internal/api/<domain>/...` |

Final full-repo sanity check for any new module: `go build ./...` (catches
any interface mismatch against `store.Storer`) followed by
`go test ./internal/api/...`.
