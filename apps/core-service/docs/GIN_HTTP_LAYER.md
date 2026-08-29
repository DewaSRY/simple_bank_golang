# Why Gin — HTTP Layer

## The decision

The REST API is built on [gin-gonic/gin](https://github.com/gin-gonic/gin), not the standard library's `net/http` directly and not a heavier batteries-included framework. Gin is a thin router + middleware chain + binding/validation layer over `net/http` — it doesn't dictate project structure, an ORM, or a templating engine, which matters because this project already made those decisions independently (sqlc for the database, a hand-rolled normalized-response envelope for JSON — see [NORMALIZE_RESPONSE.md](NORMALIZE_RESPONSE.md)).

## Why not the alternatives

**Why not plain `net/http`.** The standard library is a perfectly viable choice for a small API, but you end up hand-rolling three things this codebase needs constantly: path-parameter extraction (`/accounts/:id`), request-body binding with validation (turning `{"currency": "XX"}` into a rejected request with a field-level error), and a way to run shared logic — auth, error rendering — around every handler without repeating it. Gin gives you all three (`gin.Context`, `ShouldBindJSON`/`ShouldBindUri`/`ShouldBindQuery`, and middleware) as the framework's core job, not something you build yourself.

**Why not a heavier framework** (e.g. something with baked-in ORM/ports conventions). This project's boundaries are already opinionated elsewhere — sqlc owns the database layer, a custom `AppError`/`errorHandlerMiddleware` pair owns error shape (see below). A heavier framework would either fight those decisions or make them redundant. Gin's job here is narrowly "route requests to handlers, bind and validate input, run middleware" — it doesn't try to own more than that.

**The trade-off you're accepting.** Gin's binding validation (`binding:"required,min=1"` tags, powered by `go-playground/validator`) is convenient but its error messages are validator-internal and not written for end users — see below for how this codebase compensates. Gin also encourages putting a fair amount of logic directly in handler functions; this codebase pushes business logic out to `Store`/`Querier` (see [SQLC_SETUP.md](SQLC_SETUP.md)) precisely so handlers stay thin translators between HTTP and the store, not a place where transfer logic accidentally accumulates.

## How it's wired in this codebase

```
internal/api/
├── server.go              <- Server struct, gin.Engine setup, registerValidatorFieldNames
├── router.go               <- route table (bindRouters)
├── auth_middleware.go       <- JWT auth as gin middleware
├── apperror.go              <- AppError type + errorHandlerMiddleware
├── response.go              <- success/error envelope, pagination, validator error mapping
├── account_router.go        <- handlers, one file per resource
├── auth_router.go
├── transactions_router.go
└── user_router.go
```

Four mechanics worth understanding, because they're the parts a newcomer to this codebase would otherwise have to reverse-engineer from handler code:

1. **Errors flow through `ctx.Error()`, not `ctx.JSON()`, from every handler.** A handler that fails never builds a JSON error body itself — it calls `fail(ctx, SomeErr(...))` (see [internal/api/apperror.go](../internal/api/apperror.go)), which does `ctx.Error(err)` + `ctx.Abort()` and returns. `errorHandlerMiddleware`, registered once in [server.go](../internal/api/server.go) (`router.Use(errorHandlerMiddleware())`), runs `ctx.Next()` first, then inspects `ctx.Errors` after every handler in the chain has finished, and renders exactly one JSON body from whatever was recorded. This is Gin's own error-collection mechanism (`gin.Context.Errors`), not a bespoke one — the payoff is that a handler literally cannot leak a raw internal error string by forgetting to wrap it: anything that isn't an `*AppError` falls through to a sanitized generic 500.

2. **Validation errors are translated at the boundary, not left as Go struct-field names.** `go-playground/validator` (Gin's default binding engine) reports failures using the Go struct field name (`FromAccountID`) by default, which is meaningless to an API consumer who sent `from_account_id`. `registerValidatorFieldNames` in [server.go](../internal/api/server.go) registers a custom tag-name function on the validator engine, once, at server construction, so validation errors report the `json`/`uri`/`form` tag name instead. `fieldErrorsFromBindErr` and `validationMessage` in [response.go](../internal/api/response.go) then turn the validator's `ValidationErrors` into this project's `[]FieldError` shape, with a small per-tag message table (`required`, `min`, `max`, `oneof`, ...) — anything Gin/validator did not already have a friendlier message for falls back to the validator's own `fe.Error()`.

3. **Three binding methods cover the three places request data lives.** `ShouldBindJSON` for the request body (`createAccountRequest`), `ShouldBindUri` for path parameters (`getAccountParams{ID int64 \`uri:"id"\`}`), `ShouldBindQuery` for query strings (`paginationQuery{Page, Limit}` in [response.go](../internal/api/response.go), reused by every list endpoint). All three report errors the same way — bind, check `err`, call `fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))` — so a handler never branches on which kind of binding failed.

4. **Auth is a middleware that writes into `gin.Context`, not a value threaded through function signatures.** `authMiddleware(tokenMaker)` (see [internal/api/auth_middleware.go](../internal/api/auth_middleware.go)) verifies the `Authorization: Bearer <token>` header and, on success, stores the decoded payload in the request context; `getAuthPayload(ctx)` retrieves it in any handler downstream (e.g. `createAccount` in [account_router.go](../internal/api/account_router.go) reads `authPayload.Username` to set `Owner`). It's applied to a route *group*, not per-route: `router.go` splits routes into an unauthenticated group (`/users`, `/auth/login`) and an `authorized := router.Group("/")` group with `authorized.Use(authMiddleware(...))` — adding a new protected endpoint means registering it on `authorized`, not remembering to attach middleware by hand each time. See [IMPLEMENT_AUTH.md](IMPLEMENT_AUTH.md) for the full auth design.

## Day-to-day integration

**Adding a new endpoint:**

1. Add the route to `bindRouters` in [internal/api/router.go](../internal/api/router.go), on `router` (public) or `authorized` (requires a valid JWT).
2. Write the handler in the resource's router file (or a new one, following the `*_router.go` naming convention) as `func (server *Server) yourHandler(ctx *gin.Context)`.
3. Define a request struct with `binding` tags for whatever Gin should validate, and bind it with the matching `ShouldBind*` call.
4. On failure, call `fail(ctx, ...)` with one of the constructors in [apperror.go](../internal/api/apperror.go) (`ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `BadRequestErr`, `ConflictErr`, `InternalErr`) — add a new one there if the failure doesn't fit an existing status/code.
5. On success, call `succeed`/`succeedWithMeta` from [response.go](../internal/api/response.go) — never `ctx.JSON` directly, so every response keeps the `{data, message}` (or `{data, message, meta}`) envelope documented in [NORMALIZE_RESPONSE.md](NORMALIZE_RESPONSE.md).

**Testing handlers/middleware:** tests build a real `*gin.Engine` in `gin.TestMode`, register just the middleware/routes under test, and drive it with `httptest.NewRecorder()` + `router.ServeHTTP(...)` — see [internal/api/auth_middleware_test.go](../internal/api/auth_middleware_test.go). This exercises the actual Gin routing and middleware chain rather than calling handler functions directly, which is what lets these tests catch, e.g., a middleware that forgets to `ctx.Abort()`.
