---
description: "Use when adding a new HTTP API module/domain/resource in apps/core-service: a brand-new package under internal/api/ sibling to account/auth/transfer (e.g. 'cards', 'notifications'). Trigger phrases: 'create a new module', 'add a new domain package', 'scaffold a new API resource', 'new feature module', 'add a new endpoint group'. Covers Handler struct, routes, DTOs, error mapping, Swagger, tests. Assumes the Storer method(s) it calls already exist — see core-service-new-model.instructions.md if they don't. Companion to core-service.instructions.md (general rules)."
---

# Creating a new "module" (HTTP API domain package) — core-service

A "module" is a new sibling package under `internal/api/` (like `account`,
`auth`, `transfer`) — the HTTP layer for one resource. This assumes whatever
`Storer` method(s) it needs already exist; if they don't, do that first via
`core-service-new-model.instructions.md`, then come back here for the HTTP
layer.

Read `apps/core-service/AGENTS.md` and, if this is genuinely a 4th domain
package, `apps/core-service/docs/API_MODULE_GUIDE.md` for the fully-worked
`account`-based example this file compresses.

**Adding one endpoint to an *existing* domain?** Skip straight to step 4 —
you never touch `router.go` for that.

## 1. Package skeleton

One directory under `internal/api/`, named after the resource, lowercase,
singular-ish to match convention (`account`, not `accounts`/`AccountApi`).
Split files the same way every existing domain does:

| File | Holds |
|---|---|
| `<domain>.go` | `Handler` struct + `RegisterRoutes` (or `RegisterPublicRoutes`/`RegisterAuthorizedRoutes`) |
| `handlers.go` (split further if large, e.g. `manage.go`) | Endpoint functions, one per route, each with a Swagger godoc block |
| `response.go` | Response DTOs + `toXResponse` mappers from `sqlc` row types |
| `apperror.go` | This domain's sentinel/DB-error → `*core.AppError` mapping function(s) |
| `<domain>_handler_test.go` | gomock-based handler tests, no DB |

## 2. `Handler` struct + route registration

Plain struct, only what its endpoints need — never more:

```go
// internal/api/something/something.go
package something

import (
	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Store store.Storer
}

// All-authenticated resource (most common — mirrors account/transfer)
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/somethings", h.createSomething)
	rg.GET("/somethings/:id", h.detailSomething)
}
```

If the resource has both public and authenticated endpoints (like `auth`),
use two methods instead — `RegisterPublicRoutes(rg *gin.RouterGroup)` and
`RegisterAuthorizedRoutes(rg *gin.RouterGroup)` — called from the two
different groups in `router.go`.

If multiple routes share a path param, define one shared struct at package
level (see `idPathParam` in `internal/api/account/account.go`) instead of
repeating it per handler.

## 3. Wire it into `router.go` (new domain package only)

`bindRouters` (`internal/api/router.go`) is the *only* place `Handler`
structs get constructed. Add two lines:

```go
somethingHandler := &something.Handler{Store: server.store}
somethingHandler.RegisterRoutes(authorized)   // or RegisterPublicRoutes/RegisterAuthorizedRoutes on v1 / authorized
```

Nothing stops you from forgetting this — a missing `RegisterRoutes` call
compiles fine and just silently 404s every request to that resource. No
route-registration test catches this; verify by actually hitting the route
(`rest-testing/core-service.http` or `make swag-gen` + Swagger UI).

## 4. Endpoint function — the 5-step shape (copy this every time)

```go
func (h *Handler) createSomething(ctx *gin.Context) {
	var req createSomethingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {                 // 1. bind
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := core.GetAuthPayload(ctx)                          // 2. read caller (authorized routes only)

	result, err := h.Store.SomeStorerMethod(ctx, store.SomeParams{    // 3. exactly ONE Storer call
		UserID: authPayload.ID,
		// ...
	})
	if err != nil {
		core.Fail(ctx, somethingAppError(err))                       // 4. map error (apperror.go)
		return
	}

	core.Succeed(ctx, http.StatusOK, toSomethingResponse(result), "Something created successfully") // 5. respond
}
```

| Step | Do | Never |
|---|---|---|
| 1. Bind | `ShouldBindJSON`/`ShouldBindUri`/`ShouldBindQuery` into a request struct | Hand-parse `ctx.Param`/`ctx.Query` outside a struct |
| 2. Read caller | `core.GetAuthPayload(ctx)` (authorized routes only) | Re-parse the `Authorization` header |
| 3. Store call | Call exactly one `Storer` method | Call the store more than once when the calls must be atomic (that's a `*Tx`, see the model instructions) |
| 4. Map error | Domain `*AppError`-mapping function, or one inline `errors.Is(err, sql.ErrNoRows)` for a single obvious case | `ctx.JSON` on an error path, or return the raw `error` |
| 5. Respond | `core.Succeed(...)` / `core.SucceedWithMeta(...)` for lists | Build the JSON body by hand |

**Ownership checks** (does this resource belong to the caller?) are a
per-handler comparison, not middleware — copy this pattern, don't invent a
shared helper (none of the existing domains have one):

```go
if result.UserID.Int64 != authPayload.ID {
	core.Fail(ctx, core.ForbiddenErr("something does not belong to the authenticated user"))
	return
}
```

## 5. Request DTOs

One struct per endpoint, defined directly above the handler that uses it,
validated entirely through Gin binding tags:

```go
type createSomethingRequest struct {
	Name string `json:"name" binding:"required"`
}
```

List/search endpoints embed `core.PaginationQuery` instead of hand-rolling
`page`/`limit`:

```go
type listSomethingsQuery struct {
	core.PaginationQuery
	Name string `form:"name" binding:"omitempty"`
}
```
Pass `query.Offset()` / `query.Limit` straight into the `Storer` call.

## 6. Response DTOs (`response.go`)

Never return a raw `sqlc` row to the client. Always a hand-written struct +
`toXResponse` mapper, with money fields kept as `string`:

```go
type somethingResponse struct {
	ID        int64  `json:"id"`
	Amount    string `json:"amount"` // string all the way through
	CreatedAt string `json:"created_at"`
}

func toSomethingResponse(s db.Something) somethingResponse {
	return somethingResponse{
		ID:        s.ID,
		Amount:    s.Amount,
		CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
```

If several endpoints share the same `sqlc.embed(...)` view, write **one**
mapper for it and reuse it — don't hand-write a mapper per query.

## 7. Error mapping (`apperror.go`)

```go
func somethingAppError(err error) *core.AppError {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return core.NotFoundErr("something not found")
	case errors.Is(err, store.ErrSomethingInvalid):
		return core.ConflictErr(errCodeSomethingInvalid, err.Error())
	default:
		return core.InternalErr(err) // always the real err, never nil
	}
}
```

| `core` helper | Status | When |
|---|---|---|
| `core.ValidationErr(details...)` | 400 | Field-level bind failure — pass `core.FieldErrorsFromBindErr(err)...` |
| `core.BadRequestErr(code, msg)` | 400 | Business-rule violation not tied to one field |
| `core.NotFoundErr(msg)` | 404 | `sql.ErrNoRows`, or an explicit "doesn't exist" check |
| `core.ForbiddenErr(msg)` | 403 | Valid caller, wrong owner |
| `core.ConflictErr(code, msg)` | 409 | State conflict — a new sentinel error, or a `pq.Error` `check_violation` |
| `core.InternalErr(err)` | 500 | Anything unexpected |

A domain-specific error code (`errCodeSomethingInvalid`) is a local `const`
in that domain's `apperror.go` — never add it to `core.ErrCode*`, which only
owns generic codes. Add one `case` per new sentinel error the `Storer` call
can return; don't scatter `errors.Is` checks across handlers instead.

## 8. Swagger annotations

Godoc block directly above every handler:

```go
// createSomething godoc
// @Summary      Create a new something
// @Tags         somethings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createSomethingRequest  true  "payload"
// @Success      200      {object}  successResponse{data=somethingResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /somethings [post]
func (h *Handler) createSomething(ctx *gin.Context) {
```

List every `@Failure` your `apperror.go` mapping can actually produce, set
`@Security BearerAuth` only on routes behind `core.AuthMiddleware`, then:

```bash
make swag-gen   # regenerates internal/docs — never hand-edit it; check the diff
```

## 9. Handler tests (`*_handler_test.go`)

gomock-based, no database. Copy the three helpers every existing test file
defines at the top rather than importing them from somewhere shared (none of
the domains currently share this plumbing):

```go
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

Then one table-driven test function per endpoint using
`mockdb.NewMockStorer(gomock.NewController(t))`, `.EXPECT()` with
`gomock.Any()` for `ctx` and exact business params, fired through
`httptest`. Cover at minimum:

| Case | Proves |
|---|---|
| Happy path | Correct store call, 200 with correct shape |
| Bind failure (missing field, bad pagination) | `.Times(0)` on the store call — validation short-circuits |
| Not found (`sql.ErrNoRows`) | 404 |
| Wrong owner | 403, mutating store call `.Times(0)` |
| Unexpected store error (`sql.ErrConnDone`) | 500 via `core.InternalErr`, no leaked error string |

## 10. Verify

```bash
go build ./...
go test ./internal/api/something/...
make swag-gen && git diff internal/docs   # only if you touched a godoc annotation
```

## Hard rules (see `core-service.instructions.md` for the full list)

- Handlers never call `ctx.JSON` on an error path — always `core.Fail`.
- Business logic never lives in a handler — bind → one `Storer` call → map
  error → respond, nothing else.
- A brand-new domain package is the *only* time `router.go` gets touched; a
  new endpoint on an existing domain never does.
