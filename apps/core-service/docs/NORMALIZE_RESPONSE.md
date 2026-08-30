# Normalized Response — As Implemented

## Who this doc is for

You're comfortable with Go and Gin, but you haven't necessarily used Gin's `ctx.Error`/`ctx.Errors` mechanism before, or you have and want to know what *this* service actually does with it. Section 0 is a short primer on that mechanism — skip it if you already know it.

Everything after that is verified against the source in this repo as of `8f68240` (2026-08-30), file:line cited throughout. This is **not** a design doc — it describes the response format as the code actually produces it, not the idealized shape in the original spec. Several things a reader would reasonably assume from a "normalized response" doc don't actually hold here (an error code used in an example that no handler ever emits, a `details` field that's omitted rather than empty, a mapper that only one of three eligible handlers calls) — those are called out explicitly in the sections below and summarized in "Known gaps."

## 0. Background primer — Gin's collect-then-render error model

| Approach | When the response is written | Who decides the shape | Typical use |
|---|---|---|---|
| Write-immediately | Inside the handler, via `ctx.JSON(...)` | Each handler, individually | Small APIs, no shared error shape needed |
| **Collect-then-render (this service)** | After the handler returns, by a middleware that runs post-`ctx.Next()` | One middleware, for every handler | Services that need every error response to look identical |

`ctx.Error(err)` (used here via the `fail` helper, §3) doesn't write anything to the response — it just appends `err` to `ctx.Errors`, a slice Gin carries on the context for exactly this purpose. `ctx.Abort()` stops any remaining handlers in the chain from running, but still doesn't write a response. A middleware registered *before* the handler can call `ctx.Next()`, let the handler (and any middleware after it) run, and only then — once `ctx.Next()` returns — inspect `ctx.Errors` and decide what to render. That's `errorHandlerMiddleware` (§3).

Gotchas that aren't obvious from the API surface:

1. **A handler that calls `fail()` and forgets to `return` keeps executing.** `fail` doesn't stop the function — only `ctx.Abort()` (which `fail` calls internally) stops *downstream Gin handlers* from running. If a handler's own code continues past `fail(ctx, ...)` without an explicit `return`, it can still call `succeed(...)` afterward, and since `ctx.JSON` can be called more than once without erroring at the Go level, the second write wins over the network but the first is silently wasted. Every handler in this codebase does return immediately after `fail` (confirmed by inspection of every call site in §6) — but nothing enforces it at the type level.
2. **Only the *last* recorded error is rendered.** `errorHandlerMiddleware` reads `ctx.Errors.Last()` (`internal/api/apperror.go:91`), not the first. If a handler ever called `fail` twice before returning, only the second would be visible in the response.
3. **A non-`*AppError` doesn't crash the response — it silently becomes a generic 500.** This is a deliberate safety net (§3), not a bug: it's what guarantees a handler can never leak raw internal error text just by forgetting to wrap an error in `*AppError`.

## 1. Architecture at a glance

The composition root is `internal/api/server.go` — `NewServer` (`internal/api/server.go:37`) builds the Gin engine and registers `errorHandlerMiddleware()` globally (`internal/api/server.go:49`) before `bindRouters` wires any routes. `NewServer` doesn't implement any response-formatting logic itself — it only wires the one piece (the error middleware) that needs to run for every request. There is no equivalent middleware or base-handler object for the success path; `succeed`/`succeedWithMeta` are plain functions each handler calls directly, not something injected through the request lifecycle.

| Concern | Owner (file) | Analogy |
|---|---|---|
| Envelope shapes (`successResponse`, `errorResponse`, `Meta`, `FieldError`) | `internal/api/response.go` | The paper every response gets printed on |
| Writing a success response | `succeed` / `succeedWithMeta` (`internal/api/response.go:67,73`) | The print button each handler presses |
| Classifying what went wrong into an `*AppError` | `internal/api/apperror.go` constructors + per-handler logic | Sorting mail into categories before it goes out |
| Rendering whatever was classified | `errorHandlerMiddleware` (`internal/api/apperror.go:82`) | The mailroom that actually stamps and sends, regardless of category |
| Turning a DB row into a response DTO (accounts only) | `internal/api/account_response.go` | A translator who's only fluent for one of the three callers who need one |
| Parsing shared `page`/`limit` query params | `paginationQuery` (`internal/api/response.go:55`) | The one ruler every list endpoint measures against |

This is split the way it is because "what went wrong" (business knowledge, different per endpoint) and "how a failure is rendered" (identical for every endpoint) are genuinely different concerns — a handler should only ever have to answer the first question. That split is also *why* there's no single central error-code table: codes are constants scattered across `apperror.go` plus a few ad hoc string literals written straight into handlers, because nothing forces classification through one funnel the way rendering is forced through one middleware. §7 (divergences) shows the concrete consequence of that.

## 2. Success envelope — `internal/api/response.go`

**Problem it solves:** give every successful response the same `{data, message}` (optionally `+meta`) shape without each handler re-deriving it.

```go
// internal/api/response.go:39-51
type successResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Meta    any    `json:"meta,omitempty"`
}

type Meta struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
	Total int64 `json:"total"`
}
```

Two functions write it, both simple wrappers over `ctx.JSON`:

```go
// internal/api/response.go:67-75
func succeed(ctx *gin.Context, status int, data any, message string) {
	ctx.JSON(status, successResponse{Data: data, Message: message})
}

func succeedWithMeta(ctx *gin.Context, status int, data any, message string, meta any) {
	ctx.JSON(status, successResponse{Data: data, Message: message, Meta: meta})
}
```

`Data` and `Meta` are typed `any` — nothing here validates that `data` is JSON-serializable or that `meta` is shaped like `Meta`. In practice every call site passes either a hand-built DTO (`accountResponse`, `profileResponse`, `AuthResponse`, `userResponse`) or, in three places, a raw sqlc row/slice straight through (§7, point 4). Every `succeedWithMeta` call in the codebase passes a literal `Meta{...}` for the `meta` argument — nothing else is ever passed there today.

Rough edge: `status` is a parameter on both functions, but every single call site in this codebase passes `http.StatusOK` (200) — including `createAccount`, which by REST convention might reasonably return 201. There's no enforced convention here; it's just that no handler has ever passed anything else.

## 3. Error envelope & `AppError` — `internal/api/apperror.go`

**Problem it solves:** let a handler say *what kind* of failure happened without knowing anything about how it gets turned into JSON.

```go
// internal/api/apperror.go:16-21
type AppError struct {
	Status  int
	Code    string
	Message string
	Details []FieldError
}

func (e *AppError) Error() string { return e.Message }
```

`AppError` has no JSON tags — it's never marshaled directly. It's an internal carrier; `errorHandlerMiddleware` copies its fields into the actual wire type:

```go
// internal/api/response.go:29-37
type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}
```

A fixed set of constructors (`internal/api/apperror.go:33-66`) is the only sanctioned way to build one — no handler constructs an `AppError{}` literal directly:

| Constructor | Status | Code | Notes |
|---|---|---|---|
| `ValidationErr(details ...FieldError)` | 400 | `VALIDATION_ERROR` | Message is always the fixed string `"One or more fields are invalid"` |
| `BadRequestErr(code, message string)` | 400 | caller-supplied | For business-rule violations not tied to one field |
| `NotFoundErr(message string)` | 404 | `NOT_FOUND` | |
| `UnauthorizedErr(message string)` | 401 | `UNAUTHORIZED` | |
| `ForbiddenErr(message string)` | 403 | `FORBIDDEN` | |
| `ConflictErr(code, message string)` | 409 | caller-supplied | |
| `InternalErr()` | 500 | `INTERNAL_ERROR` | Message is always the fixed string `"internal server error"` — never the real underlying error |

`fail` records the error and stops the chain, but — per §0 — doesn't render anything:

```go
// internal/api/apperror.go:71-74
func fail(ctx *gin.Context, err *AppError) {
	ctx.Error(err)
	ctx.Abort()
}
```

`errorHandlerMiddleware` is the only place that renders an error response, for every route registered after it (i.e. every route, since it's registered globally in `NewServer`):

```go
// internal/api/apperror.go:82-105
func errorHandlerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		var appErr *AppError
		if errors.As(ctx.Errors.Last().Err, &appErr) {
			ctx.JSON(appErr.Status, errorResponse{Error: errorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			}})
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse{Error: errorBody{
			Code:    errCodeInternal,
			Message: "internal server error",
		}})
	}
}
```

The `default` branch (falling back to a sanitized 500) is the concrete mechanism behind §0 gotcha 3 — it's what makes it structurally impossible for a raw Go error to reach a client, even if a future handler pushes a plain `error` onto `ctx.Errors` instead of an `*AppError`.

## 4. Field-level validation errors — `fieldErrorsFromBindErr` / `validationMessage`

**Problem it solves:** turn a Gin/go-playground-validator binding failure into the `details: [{field, message}]` array documented for validation errors, without hand-parsing validator's error strings everywhere a handler binds a request.

```go
// internal/api/response.go:84-95
func fieldErrorsFromBindErr(err error) []FieldError {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}

	details := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		details = append(details, FieldError{Field: fe.Field(), Message: validationMessage(fe)})
	}
	return details
}
```

If the bind error isn't a `validator.ValidationErrors` at all (e.g. the request body is malformed JSON, not a field-level failure), this returns `nil` — the caller ends up with `ValidationErr()` and no field-level details at all, just the fixed message. Every handler that binds a request calls this the same way: `fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))` — e.g. `internal/api/account_router.go:32-34`, `internal/api/transactions_router.go:39-42`, `internal/api/auth_router.go:41-44,95-98`.

`fe.Field()` reports the *JSON* field name (e.g. `password_confirm`), not the Go struct field name (`PasswordConfirm`), because `registerValidatorFieldNames` (`internal/api/server.go:61-79`) is called once at startup and rewrites the validator library's field-name resolution to prefer the `json`/`uri`/`form` struct tag over the Go identifier. Without it, `details[].field` would read `PasswordConfirm` instead of the field the client actually sent — worth knowing since it lives in `server.go`, not in `response.go` alongside the rest of this logic.

`validationMessage` (`internal/api/response.go:97-112`) hand-writes English for five binding tags and falls through to the library's own wording for anything else:

```go
// internal/api/response.go:97-112
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " must be at least " + fe.Param()
	case "max":
		return fe.Field() + " must be at most " + fe.Param()
	case "gt":
		return fe.Field() + " must be greater than " + fe.Param()
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	default:
		return fe.Error()
	}
}
```

Worth flagging: the codebase uses more binding tags than this switch covers — `email` (`auth_router.go:17,76`), `alphanum` (`user_router.go:16`), `eqfield` (`auth_router.go:78`), `oneof` with multiple values (`account_router.go:14`). Any of those that fall to `default` produce validator's own library-formatted message (its default `Error()` string, e.g. something like `Key: 'req.Email' Error:Field validation for 'Email' failed on the 'email' tag`) rather than the app's own phrasing — a real gap between what the switch's five hand-written cases suggest ("every field error gets a friendly message") and what actually ships for the other tags.

## 5. Domain error classification — duplicate detection & `transferAppError`

**Problem it solves:** map a Postgres error or a store-layer sentinel error to the right `(status, code, message)` triple, per business rule, without leaking DB internals.

There is no central table keyed by error type; each handler (or, for transfers, one dedicated helper) classifies inline.

**Unique-constraint violations** are detected the same way in three places — a `*pq.Error` type assertion plus a check on the Postgres error code name:

```go
// internal/api/user_router.go:69-74 (also auth_router.go:143-148)
var pqErr *pq.Error
if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
	fail(ctx, ConflictErr(errCodeConflict, "username or email already exists"))
	return
}
fail(ctx, InternalErr())
```

**Transfer failures** go through a dedicated classifier, since `store.TransferTx` can fail for several distinct business reasons:

```go
// internal/api/transactions_router.go:181-202
func transferAppError(err error) *AppError {
	var pqErr *pq.Error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return NotFoundErr("account not found")
	case errors.Is(err, store.ErrSameAccount):
		return BadRequestErr(errCodeValidation, err.Error())
	case errors.Is(err, store.ErrInvalidAmount):
		return ValidationErr(FieldError{Field: "amount", Message: err.Error()})
	case errors.Is(err, store.ErrCurrencyMismatch):
		return BadRequestErr(errCodeCurrencyMismatch, err.Error())
	case errors.Is(err, store.ErrInsufficientFunds):
		return ConflictErr(errCodeInsufficientFunds, err.Error())
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
		return ConflictErr(errCodeInsufficientFunds, "insufficient funds")
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
		return BadRequestErr(errCodeNotFound, "account not found")
	default:
		return InternalErr()
	}
}
```

`store.ErrSameAccount`, `store.ErrCurrencyMismatch`, `store.ErrInvalidAmount`, and `store.ErrInsufficientFunds` are plain sentinel errors (`errors.New(...)`) defined in `db/store/errors.go:6-9` — not a custom error type, just package-level `error` values raised from inside `store.TransferTx` (`db/store/store_transaction.go:31,39,66,74`). Three of the six branches pass `err.Error()` straight through as the response `message` — this couples the client-facing message text directly to the sentinel's Go string, so renaming one of those `errors.New(...)` messages changes API response text as a side effect.

Worth flagging — `errCodeNotFound` (`"NOT_FOUND"`) is used at `transactions_router.go:198` as the `code` for a **400** (`BadRequestErr`) response, on a foreign-key violation. Every other place `errCodeNotFound` appears, it's paired with `NotFoundErr` and a genuine 404. A client that switches on `code == "NOT_FOUND"` to mean "404" will see it attached to a 400 here.

There's no dedicated custom-error-type package for this: `domain/error/` exists in the repo tree but is empty (confirmed — `ls domain/error` returns nothing). All domain errors are either the four sentinels above, or ad hoc string literals passed straight to `BadRequestErr`/`ConflictErr` inline in handlers (§7, point 1).

## 6. Pagination — `paginationQuery` & `Meta`

**Problem it solves:** parse and validate `page`/`limit` query params identically across every list endpoint, and attach consistent `meta` to the response.

```go
// internal/api/response.go:55-62
type paginationQuery struct {
	Page  int32 `form:"page,default=1" binding:"min=1"`
	Limit int32 `form:"limit,default=10" binding:"min=1,max=100"`
}

func (p paginationQuery) offset() int32 {
	return (p.Page - 1) * p.Limit
}
```

Every list handler follows the same four-step shape. `listAccounts` (`internal/api/account_router.go:128-162`):

```go
// internal/api/account_router.go:129-133, 137-144, 150-161
var query paginationQuery
if err := ctx.ShouldBindQuery(&query); err != nil {
	fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
	return
}
// ...
accounts, err := server.store.ListAccountsByUserId(ctx, db.ListAccountsByUserIdParams{
	UserID:      sql.NullInt64{Int64: authPayload.ID, Valid: true},
	LimitCount:  query.Limit,
	OffsetCount: query.offset(),
})
// ...
total, err := server.store.CountAccountsByUserId(ctx, sql.NullInt64{Int64: authPayload.ID, Valid: true})
// ...
succeedWithMeta(ctx, http.StatusOK, toListAccountResponse(accounts), "Accounts retrieved successfully", Meta{
	Page: query.Page, Limit: query.Limit, Total: total,
})
```

1. Bind `page`/`limit` from the query string via `paginationQuery` — a bad value (e.g. `limit=0` or `limit=500`) fails Gin binding and goes through the same `ValidationErr(fieldErrorsFromBindErr(err)...)` path as any other bind error (§4).
2. Pass `query.Limit` and `query.offset()` into the sqlc list query.
3. Fetch a **separate** total count via a dedicated `Count*` query (`CountAccountsByUserId`, `CountEntriesByAccount`, `CountTransfersByOwner` — `db/sqlc/account.sql.go:34`, `db/sqlc/entries.sql.go:18`, `db/sqlc/transfers.sql.go:20`).
4. Call `succeedWithMeta` with `Meta{Page: query.Page, Limit: query.Limit, Total: total}`.

The same four steps repeat at `internal/api/account_router.go:184-231` (`listAccountEntries`) and `internal/api/transactions_router.go:148-176` (`listTransactions`).

Worth flagging — the count is a **second round-trip**, not derived from the list query, and the two queries aren't run in the same transaction. Under concurrent writes between step 2 and step 3, `total` and the returned page can disagree (e.g. a row inserted between the two queries is counted but not listed, or vice versa). Nothing in the code addresses this; it's a plain read-then-read race, not handled.

`Meta.Page`/`Meta.Limit` are echoed straight from what the client sent (or the `default=` values), not derived from anything server-side. There's no `total_pages` field computed anywhere — only `page`/`limit`/`total`, matching what `docs/NORMALIZE_RESPONSE.md`'s Pagination Response section documents.

## 7. Account response mapper — `internal/api/account_response.go`

**Problem it solves:** convert a sqlc-generated row into the API-facing `accountResponse` DTO, so handlers don't build that literal by hand.

```go
// internal/api/account_response.go
type accountResponse struct {
	ID        int64  `json:"id"`
	Owner     string `json:"owner"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	UserID    int64  `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

func toAccountResponse(account db.ListAccountsByUserIdRow) accountResponse {
	return accountResponse{
		ID:        account.ID,
		Owner:     account.Owner,
		Balance:   account.Balance,
		Currency:  account.Currency,
		UserID:    account.UserID.Int64,
		CreatedAt: account.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toListAccountResponse(accounts []db.ListAccountsByUserIdRow) []accountResponse {
	accountResponses := make([]accountResponse, len(accounts))
	for i, account := range accounts {
		accountResponses[i] = toAccountResponse(account)
	}
	return accountResponses
}
```

This was added by the commit titled (with a preserved typo) "Create respomse mapper" (`bfced21`). It's the only thing in the codebase that fits the word "mapper," and it's narrower than the name suggests: it converts a `db.ListAccountsByUserIdRow`, and only that sqlc row type, into `accountResponse`. It's called from exactly one place — `listAccounts` (`internal/api/account_router.go:159`).

**Deliberately not used everywhere it could be** — `createAccount` (`internal/api/account_router.go:55-62`) and `getAccount` (`internal/api/account_router.go:105-112`) build a byte-for-byte identical `accountResponse{...}` literal by hand instead of calling `toAccountResponse`, because they hold a `db.Account` (from `CreateAccount`/`GetAccountById`), a *different* sqlc-generated struct than `db.ListAccountsByUserIdRow`, and `toAccountResponse`'s parameter type doesn't accept it. So the "response mapper" commit didn't unify response construction across all three account handlers — it added a mapper for the one handler whose row type happened to fit, and left the other two with hand-duplicated logic. A future change to how `accountResponse` is built from a row (e.g. a new field) has to be made in three places, not one, and nothing enforces that the three stay in sync beyond code review.

There's a second, independently-written mini-mapper of the same shape for users — `newUserResponse(user db.CreateUserRow) userResponse` (`internal/api/user_router.go:28-35`) — used only by `createUser`. `profile_router.go` and `transactions_router.go` build their response data with no mapper at all (§7 continues into the next section's divergence list, point 4).

## Cross-feature coupling

- **`auth_middleware.go` uses the same `fail`/`AppError` convention**, not something local to it: `authMiddleware` (`internal/api/auth_middleware.go`) calls `fail(ctx, UnauthorizedErr(err.Error()))` on any token failure, which is rendered by the same global `errorHandlerMiddleware` as every other handler's errors. If you're only reading `response.go`/`apperror.go`, it's easy to miss that auth failures flow through the identical pipeline documented here rather than something bespoke.
- **`transferAppError` reads sentinel errors owned by the `store` package**, coupling `internal/api`'s response text to `db/store/errors.go`'s error message strings (§5). The `store` package itself knows nothing about HTTP status codes or response envelopes — the translation is entirely on the `api` side, which is why adding a new failure mode to `TransferTx` requires a matching `case` in `transferAppError`, or it silently falls into the generic `InternalErr()` branch.
- **Middleware registration order in `NewServer`** (`internal/api/server.go:47-52`) matters here specifically: `errorHandlerMiddleware()` must be registered *before* `bindRouters` wires any route, since Gin's `ctx.Next()`/post-processing model only lets a middleware inspect errors from handlers registered after it in the chain. It's registered right after `corsMiddleware`, ahead of every route — swapping that order would mean errors from at least the first-bound routes never get rendered by it.

## Presentational layer

This is an API-only service — no client/UI code lives in this repo. The closest thing to a presentation layer is the Swagger/OpenAPI annotations on each handler (e.g. `internal/api/account_router.go:17-29`) and the generated docs under `internal/docs/` (`docs.go`, `swagger.json`, `swagger.yaml`), served at `/swagger/*any` (`internal/api/router.go:14`). Those files were regenerated as part of the same commit that added the account response mapper (`bfced21`) — they document the shapes described here but contain no response-building logic of their own; they're generated output, not hand-written.

## Summary / data flow

**Success path (e.g. `listAccounts`):**

```text
handler binds request (query/body/uri)
        │
        ▼
handler calls server.store.* (sqlc query / TransferTx)
        │
        ▼
handler maps the result to a DTO — via a mapper (§7) if one exists
for this row type, by hand otherwise, or not at all (raw sqlc row/slice)
        │
        ▼
succeed(ctx, status, data, message)              — single resource
succeedWithMeta(ctx, status, data, message, meta) — paginated list, meta from
                                                     query.Page/Limit + a
                                                     separate Count* query (§6)
        │
        ▼
ctx.JSON writes { data, message[, meta] } immediately — no middleware involved
```

**Error path (any handler):**

```text
something fails (bind error, sql.ErrNoRows, pq unique/check/fk violation,
store sentinel error, ownership mismatch, unexpected error)
        │
        ▼
handler (or transferAppError, §5) classifies it into one *AppError
via a ValidationErr/NotFoundErr/ConflictErr/... constructor
        │
        ▼
fail(ctx, err)  →  ctx.Error(err) + ctx.Abort()   — nothing written yet (§0)
        │
        ▼
errorHandlerMiddleware runs after ctx.Next() returns, finds the *AppError
via errors.As(ctx.Errors.Last().Err, &appErr)
        │
        ▼
ctx.JSON(appErr.Status, { error: { code, message, details } })
— or a sanitized 500 if the last error isn't an *AppError at all
```

## Final reference table

Every endpoint that returns a normalized response, and how each one uses the pieces above.

| Method | Endpoint | Success helper | Error paths |
|---|---|---|---|
| GET | `/health` | `succeed` | none |
| POST | `/api/v1/users` (deprecated, marked for deletion — `router.go:20`) | `succeed` (`newUserResponse` mapper) | `ValidationErr`, `ConflictErr`, `InternalErr` |
| POST | `/api/v1/auth/register` | `succeed` | `ValidationErr`, `BadRequestErr` (×3, ad hoc codes), `ConflictErr`, `InternalErr` |
| POST | `/api/v1/auth/login` | `succeed` | `ValidationErr`, `UnauthorizedErr`, `InternalErr` |
| GET | `/api/v1/auth/profile` | `succeed` | `NotFoundErr`, `InternalErr` |
| POST | `/api/v1/accounts` | `succeed` (hand-built DTO) | `ValidationErr`, `InternalErr` |
| GET | `/api/v1/accounts/:id` | `succeed` (hand-built DTO) | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| GET | `/api/v1/accounts` | `succeedWithMeta` (`toAccountResponse` mapper) | `ValidationErr`, `InternalErr` |
| GET | `/api/v1/accounts/:id/entries` | `succeedWithMeta` (raw `[]db.Entry`, no mapper) | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| POST | `/api/v1/transactions/transfer` | `succeed` (raw `store.TransferTxResult`) | `ValidationErr`, `NotFoundErr`, `BadRequestErr`, `ForbiddenErr`, `ConflictErr`, `InternalErr` (via `transferAppError`, §5) |
| GET | `/api/v1/transactions/:id` | `succeed` (raw `db.Transfer`) | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| GET | `/api/v1/transactions` | `succeedWithMeta` (raw `[]db.Transfer`) | `ValidationErr`, `InternalErr` |

## Known gaps / divergences from the original spec

- **`DUPLICATE_RESOURCE`, the only error code named in `docs/NORMALIZE_RESPONSE.md`'s worked example, doesn't exist anywhere in the code.** Real duplicate-user handling uses `errCodeConflict = "CONFLICT"` (`internal/api/response.go:20`, used at `user_router.go:72` and `auth_router.go:146`) for the DB-race path, and two ad hoc, non-constant string codes — `"email_exists"` (`auth_router.go:109`) and `"username_exists"` (`auth_router.go:124`) — for the pre-check path, plus `"password_mismatch"` (`auth_router.go:102`). The comment above the `errCode*` constant block (`response.go:10-11`, "Keep these stable — API consumers switch on them") is already not honored for these three codes, since they're inline literals rather than constants.
- **`details` is omitted, not emitted as `[]`, when there are no field errors.** `docs/NORMALIZE_RESPONSE.md`'s plain error example shows `"details": []`. The real field is `Details []FieldError \`json:"details,omitempty"\`` (`internal/api/response.go:32`) — for every error built via `NotFoundErr`, `UnauthorizedErr`, `ForbiddenErr`, `InternalErr`, `ConflictErr`, or `BadRequestErr` (none of which take field errors), `Details` is `nil`, and `omitempty` drops the key entirely rather than serializing an empty array.
- **`errCodeNotFound` is used as the `code` for a 400 response** in `transferAppError`'s foreign-key-violation branch (`transactions_router.go:198`) — see §5. Everywhere else, `NOT_FOUND` implies a 404.
- **Two more sqlc-row-exposing endpoints bypass any DTO/mapper entirely.** `listAccountEntries` returns raw `[]db.Entry` as `data` (`account_router.go:229`), and `getTransaction`/`listTransactions` return raw `db.Transfer`/`[]db.Transfer` (`transactions_router.go:132,173`). These expose sqlc-generated field names and types (including `sql.NullXxx` wrapper types) directly in the API response, unlike the account/user/profile endpoints, which map to a dedicated response struct.
- **The response mapper only covers one of three account handlers** — see §7's "Deliberately not used everywhere it could be" callout.
- **`validationMessage`'s hand-written wording only covers 5 of the binding tags actually used** (`required`, `min`, `max`, `gt`, `oneof`) — `email`, `alphanum`, and `eqfield` fall through to the validator library's own default message text (§4).
- **`POST /api/v1/users` remains live and wired through this same normalized-response path** despite being marked `// TODO: DELETE this end point` (`router.go:20`) — it duplicates `/auth/register` but issues no token.
