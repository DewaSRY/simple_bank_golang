# Normalized Response — As Implemented

## Who this doc is for

You're comfortable with Go and Gin, but you haven't necessarily used Gin's
`ctx.Error`/`ctx.Errors` mechanism before, or you have and want to know what
*this* service actually does with it. Section 0 is a short primer on that
mechanism — skip it if you already know it.

Everything after that is verified against the source in this repo as of
`13f5fc4` (2026-09-19), file:line cited throughout. This is **not** a design
doc — it describes the response format as the code actually produces it, not
an idealized shape. This repo went through a refactor that split a flat
`internal/api` package into per-domain packages (`internal/api/core`,
`internal/api/account`, `internal/api/auth`, `internal/api/transfer`) — an
earlier version of this doc still cited the pre-refactor file layout
(`internal/api/response.go`, `internal/api/apperror.go`, `account_router.go`,
`user_router.go`, a `POST /api/v1/users` endpoint). None of those files or
that endpoint exist anymore. Everything below reflects the current layout.
Rough edges — inconsistent status codes, error codes reused across different
HTTP statuses, endpoints that skip the response-DTO layer entirely — are
called out explicitly as their own callouts, not smoothed over.

## 0. Background primer — Gin's collect-then-render error model

| Approach | When the response is written | Who decides the shape | Typical use |
|---|---|---|---|
| Write-immediately | Inside the handler, via `ctx.JSON(...)` | Each handler, individually | Small APIs, no shared error shape needed |
| **Collect-then-render (this service)** | After the handler returns, by a middleware that runs post-`ctx.Next()` | One middleware, for every handler | Services that need every error response to look identical |

`ctx.Error(err)` (used here via `core.Fail`, §3) doesn't write anything to the
response — it just appends `err` to `ctx.Errors`, a slice Gin carries on the
context for exactly this purpose. `ctx.Abort()` stops any remaining handlers
in the chain from running, but still doesn't write a response. A middleware
registered *before* the handler can call `ctx.Next()`, let the handler (and
any middleware after it) run, and only then — once `ctx.Next()` returns —
inspect `ctx.Errors` and decide what to render. That's
`core.ErrorHandlerMiddleware` (§3).

Gotchas that aren't obvious from the API surface:

1. **A handler that calls `core.Fail` and forgets to `return` keeps
   executing.** `Fail` doesn't stop the function — only `ctx.Abort()` (which
   `Fail` calls internally) stops *downstream Gin handlers* from running. If a
   handler's own code continues past `core.Fail(ctx, ...)` without an
   explicit `return`, it can still call `core.Succeed(...)` afterward, and
   since `ctx.JSON` can be called more than once without erroring at the Go
   level, the second write wins over the network but the first is silently
   wasted. Every handler in this codebase does return immediately after
   `Fail` (confirmed by inspection of every call site cited in §6/§7 below) —
   but nothing enforces it at the type level.
2. **Only the *last* recorded error is rendered.** `ErrorHandlerMiddleware`
   reads `ctx.Errors.Last().Err` (`internal/api/core/apperror.go:136`), not
   the first. If a handler ever called `Fail` twice before returning, only
   the second would be visible in the response.
3. **A non-`*AppError` doesn't crash the response — it silently becomes a
   generic 500.** This is a deliberate safety net (§3), not a bug: it's what
   guarantees a handler can never leak raw internal error text just by
   forgetting to wrap an error in `*AppError`.

## 1. Architecture at a glance

The composition root is `internal/api/server.go` — `NewServer`
(`internal/api/server.go:40`) builds the Gin engine and registers
`core.ErrorHandlerMiddleware` (`internal/api/server.go:72`) globally, before
`bindRouters` (`internal/api/router.go:21`) wires any domain routes.
`NewServer` doesn't implement any response-formatting logic itself — it only
wires the middleware chain (`RequestIDMiddleware` → `LoggingMiddleware` →
`corsMiddleware` → `ErrorHandlerMiddleware` → `RecoveryMiddleware`,
`internal/api/server.go:69-73`) that has to run for every request. There is
no equivalent middleware or base-handler object for the success path;
`core.Succeed`/`core.SucceedWithMeta` are plain functions each handler calls
directly, not something injected through the request lifecycle.

| Concern | Owner (file) | Analogy |
|---|---|---|
| Envelope shapes (`successResponse`, `errorResponse`, `Meta`, `FieldError`) | `internal/api/core/response.go` | The paper every response gets printed on |
| Writing a success response | `core.Succeed` / `core.SucceedWithMeta` (`internal/api/core/response.go:54,60`) | The print button each handler presses |
| Classifying what went wrong into an `*AppError` | `internal/api/core/apperror.go` constructors + per-domain error-mapping functions | Sorting mail into categories before it goes out |
| Rendering whatever was classified | `core.ErrorHandlerMiddleware` (`internal/api/core/apperror.go:122`) | The mailroom that actually stamps and sends, regardless of category |
| Turning a DB row into a response DTO, per domain | `account/response.go`, `transfer/response.go`, inline structs in `auth/handlers.go` | Three translators, each fluent only in their own domain's row types |
| Parsing shared `page`/`limit` query params | `core.PaginationQuery` (`internal/api/core/response.go:42`) | The one ruler every list endpoint measures against |

This is split the way it is because "what went wrong" (business knowledge,
different per endpoint) and "how a failure is rendered" (identical for every
endpoint) are genuinely different concerns — a handler should only ever have
to answer the first question. That split is also *why* there's no single
central error-code table: the six generic codes (`core.ErrCodeValidation`,
`ErrCodeNotFound`, `ErrCodeInternal`, `ErrCodeUnauthorized`,
`ErrCodeForbidden`, `ErrCodeConflict` — `internal/api/core/apperror.go:20-27`)
live in `core`, but each domain package is free to define its own on top
(`account.errCodeMainAccount`, `transfer.errCodeCurrencyMismatch`/
`errCodeInsufficientFunds`), and `auth`'s handlers write a few more as inline
string literals rather than constants at all. §5 shows the concrete
consequence of that.

## 2. Success envelope — `internal/api/core/response.go`

**Problem it solves:** give every successful response the same
`{data, message}` (optionally `+meta`) shape without each handler re-deriving
it.

```go
// internal/api/core/response.go:26-38
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

Two exported functions write it, both simple wrappers over `ctx.JSON`:

```go
// internal/api/core/response.go:54-62
func Succeed(ctx *gin.Context, status int, data any, message string) {
	ctx.JSON(status, successResponse{Data: data, Message: message})
}

func SucceedWithMeta(ctx *gin.Context, status int, data any, message string, meta any) {
	ctx.JSON(status, successResponse{Data: data, Message: message, Meta: meta})
}
```

`Data` and `Meta` are typed `any` — nothing here validates that `data` is
JSON-serializable or that `meta` is shaped like `Meta`. In practice every
`SucceedWithMeta` call site in the codebase passes a literal `core.Meta{...}`
— nothing else is ever passed there today. `data` is always one of: a
hand-built response DTO (`accountResponse`, `accountuserResponse`,
`profileResponse`, `AuthResponse`, `accountEntriesViewResponse`,
`transactionHistoryItem`, `publicAccountResponse`), a slice of one of those
via `util.MapSlice`, or — for `/health` only — a bare `gin.H{"status": "ok"}`
(`internal/api/router.go:57`).

Rough edge: `status` is a parameter on both functions, but every call site in
this codebase passes `http.StatusOK` (200) — including `createAccount`, which
by REST convention might reasonably return 201. There's no enforced
convention here; it's just that no handler has ever passed anything else.

## 3. Error envelope & `AppError` — `internal/api/core/apperror.go`

**Problem it solves:** let a handler say *what kind* of failure happened
without knowing anything about how it gets turned into JSON.

```go
// internal/api/core/apperror.go:35-52
type AppError struct {
	Status  int
	Code    string
	Message string
	Details []FieldError

	// Cause is the underlying error that led to this AppError, if any. It is
	// never rendered in the HTTP response (errorBody has no field for it) —
	// its only job is to give ErrorHandlerMiddleware something real to log
	// for 500s, instead of every internal failure showing up in logs as the
	// same sanitized "internal server error" string with no way to tell one
	// failure from another.
	Cause error
}

func (e *AppError) Error() string {
	return e.Message
}
```

`AppError` has no JSON tags — it's never marshaled directly. It's an internal
carrier; `ErrorHandlerMiddleware` copies its fields into the actual wire type:

```go
// internal/api/core/response.go:16-24
type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}
```

A fixed set of exported constructors (`internal/api/core/apperror.go:58-100`)
is the only sanctioned way to build one — no handler constructs an
`AppError{}` literal directly:

| Constructor | Status | Code | Notes |
|---|---|---|---|
| `ValidationErr(details ...FieldError)` | 400 | `VALIDATION_ERROR` | Message is always the fixed string `"One or more fields are invalid"` |
| `BadRequestErr(code, message string)` | 400 | caller-supplied | For business-rule violations not tied to one field |
| `NotFoundErr(message string)` | 404 | `NOT_FOUND` | |
| `UnauthorizedErr(message string)` | 401 | `UNAUTHORIZED` | |
| `ForbiddenErr(message string)` | 403 | `FORBIDDEN` | |
| `ConflictErr(code, message string)` | 409 | caller-supplied | |
| `InternalErr(err error)` | 500 | `INTERNAL_ERROR` | `Message` is always the fixed string `"internal server error"` — the real `err` is stored on `Cause` and logged by `ErrorHandlerMiddleware`, never sent to the client. Always pass the error that caused the branch, never `nil` |

`Fail` records the error and stops the chain, but — per §0 — doesn't render
anything:

```go
// internal/api/core/apperror.go:105-108
func Fail(ctx *gin.Context, err *AppError) {
	ctx.Error(err) //nolint:errcheck // gin.Context.Error only returns for chaining
	ctx.Abort()
}
```

`ErrorHandlerMiddleware` is the only place that renders an error response, for
every route registered after it (i.e. every route, since it's registered
globally in `NewServer`):

```go
// internal/api/core/apperror.go:122-152
func ErrorHandlerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		reqLog := loggerFromContext(ctx, log).With(
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
		)

		var appErr *AppError
		if errors.As(ctx.Errors.Last().Err, &appErr) {
			logAppError(reqLog, appErr)
			ctx.JSON(appErr.Status, errorResponse{Error: errorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			}})
			return
		}

		reqLog.Error("unhandled error", slog.String("error", ctx.Errors.Last().Err.Error()))
		ctx.JSON(http.StatusInternalServerError, errorResponse{Error: errorBody{
			Code:    ErrCodeInternal,
			Message: "internal server error",
		}})
	}
}
```

The fallback branch at the bottom (a sanitized 500 when the last error isn't
an `*AppError`) is the concrete mechanism behind §0 gotcha 3 — it's what
makes it structurally impossible for a raw Go error to reach a client, even
if a future handler pushes a plain `error` onto `ctx.Errors` instead of an
`*AppError`.

Before rendering, the middleware also logs the failure via `logAppError`
(`internal/api/core/apperror.go:154-169`): 5xx `AppError`s log at `Error`
level with `Cause` attached, everything else logs at `Warn` — see
`docs/LOGGING.md` for the full logging design (request IDs, panic recovery,
access logging). That's a separate concern from this doc; the response shape
documented here (`errorResponse{Error: errorBody{...}}`) is what matters for
API consumers.

## 4. Field-level validation errors — `core.FieldErrorsFromBindErr` / `validationMessage`

**Problem it solves:** turn a Gin/go-playground-validator binding failure
into the `details: [{field, message}]` array, without hand-parsing
validator's error strings everywhere a handler binds a request.

```go
// internal/api/core/response.go:71-82
func FieldErrorsFromBindErr(err error) []FieldError {
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

If the bind error isn't a `validator.ValidationErrors` at all (e.g. the
request body is malformed JSON, not a field-level failure), this returns
`nil` — the caller ends up with `ValidationErr()` and no field-level details
at all, just the fixed message. Every handler that binds a request calls this
the same way: `core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))`
— e.g. `account/handlers.go:37`, `account/manage.go:37,84,90,154`,
`auth/handlers.go:45,99`, `transfer/handlers.go:75,81,140,146,204,259`,
`transfer/transactions_handlers.go:42`.

`fe.Field()` reports the *JSON/URI/form* field name (e.g. `password_confirm`),
not the Go struct field name (`PasswordConfirm`), because
`registerValidatorFieldNames` (`internal/api/server.go:94-112`) is called once
at startup and rewrites the validator library's field-name resolution to
prefer the `json`/`uri`/`form` struct tag over the Go identifier. Without it,
`details[].field` would read `PasswordConfirm` instead of the field the
client actually sent — worth knowing since it lives in `server.go`, not in
`core/response.go` alongside the rest of this logic.

`validationMessage` (`internal/api/core/response.go:84-99`) hand-writes
English for five binding tags and falls through to the library's own wording
for anything else:

```go
// internal/api/core/response.go:84-99
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

Worth flagging: the codebase uses more binding tags than this switch covers —
`email` (`auth/handlers.go:20,79`), `eqfield` (`auth/handlers.go:81`). Either
of those falling to `default` produces validator's own library-formatted
message (its default `Error()` string, e.g. something like
`Key: 'registerUserRequest.Email' Error:Field validation for 'Email' failed
on the 'email' tag`) rather than the app's own phrasing — a real gap between
what the switch's five hand-written cases suggest ("every field error gets a
friendly message") and what actually ships for `email`/`eqfield`.

## 5. Domain error classification

**Problem it solves:** map a Postgres error or a store-layer sentinel error
to the right `(status, code, message)` triple, per business rule, without
leaking DB internals into the response.

There is no central table keyed by error type. Per the "How to Handle
Errors" convention in `AGENTS.md`, each domain owns its own mapping: a small
`*AppError`-returning function for multi-cause failures
(`transfer.transferAppError`, `account.deleteAccountAppError`), or an inline
`errors.Is`/`errors.As` check for a single-cause one (`auth`'s two
unique-violation checks, every `sql.ErrNoRows` check in `account`/`transfer`).

**Unique-constraint violations** are detected the same way in two places in
`auth/handlers.go` — a `*pq.Error` type assertion plus a check on the
Postgres error code name:

```go
// internal/api/auth/handlers.go:146-153
var pqErr *pq.Error
if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
	core.Fail(ctx, core.ConflictErr(core.ErrCodeConflict, "username or email already exists"))
	return
}
core.Fail(ctx, core.InternalErr(err))
```

This is the only unique-violation check left in the codebase — the
pre-refactor `POST /api/v1/users` endpoint that had its own copy of this
logic no longer exists (see "Who this doc is for").

**Transfer and deposit failures** go through a dedicated classifier, since
`store.TransferTx`/`store.DepositTx` can each fail for several distinct
business reasons and both endpoints (`transfer/transactions_handlers.go:81`,
`transfer/handlers.go:100`) route their store error through it:

```go
// internal/api/transfer/apperror.go
func transferAppError(err error) *core.AppError {
	var pqErr *pq.Error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return core.NotFoundErr("account not found")
	case errors.Is(err, store.ErrSameAccount):
		return core.BadRequestErr(core.ErrCodeValidation, err.Error())
	case errors.Is(err, store.ErrInvalidAmount):
		return core.ValidationErr(core.FieldError{Field: "amount", Message: err.Error()})
	case errors.Is(err, store.ErrCurrencyMismatch):
		return core.BadRequestErr(errCodeCurrencyMismatch, err.Error())
	case errors.Is(err, store.ErrInsufficientFunds):
		return core.ConflictErr(errCodeInsufficientFunds, err.Error())
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
		return core.ConflictErr(errCodeInsufficientFunds, "insufficient funds")
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
		return core.BadRequestErr(core.ErrCodeNotFound, "account not found")
	default:
		return core.InternalErr(err)
	}
}
```

`store.ErrSameAccount`, `store.ErrCurrencyMismatch`, `store.ErrInvalidAmount`,
and `store.ErrInsufficientFunds` are plain sentinel errors (`errors.New(...)`)
defined in `internal/db/store/errors.go:6-9` — package-level `error` values
raised from inside `transferTx` (`internal/db/store/store_transaction.go:39,
47,74,82`) and, for `ErrInvalidAmount`, also from `depositTx`
(`internal/db/store/store_account_deposit_tx.go:44`). Three of the six
switch branches pass `err.Error()` straight through as the response
`message` — this couples the client-facing message text directly to the
sentinel's Go string, so renaming one of those `errors.New(...)` messages
changes API response text as a side effect.

Worth flagging — `core.ErrCodeNotFound` (`"NOT_FOUND"`) is used at
`transfer/apperror.go`'s foreign-key-violation branch as the `code` for a
**400** (`BadRequestErr`) response. Every other place `ErrCodeNotFound`
appears, it's paired with `NotFoundErr` and a genuine 404. A client that
switches on `code == "NOT_FOUND"` to mean "404" will see it attached to a 400
here.

Worth flagging — **both `deposit` and `transactionTransfer` validate
`amount.IsPositive()` in the handler before ever calling the store**
(`transfer/handlers.go:85`, `transfer/transactions_handlers.go:51`), and
**`transactionTransfer` also checks `FromAccountID == ToAccountID` in the
handler** (`transfer/transactions_handlers.go:46`) before calling
`TransferTx`. The store-level checks that raise `ErrInvalidAmount`/
`ErrSameAccount` (§ above) are therefore effectively unreachable through the
HTTP API today — they exist for `transferTx`/`depositTx`'s own callers (and
any future one that skips the handler-level check), not because a client
request can currently trigger them.

**Account deletion** goes through its own classifier, since
`DeleteAccountTx` has a single business-rule failure mode beyond the generic
ones:

```go
// internal/api/account/manage.go:191-201
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

`errCodeMainAccount = "MAIN_ACCOUNT"` (`account/manage.go:18`) is defined
locally in `account`, not in `core`, because it's the only place it's used —
following the same "generic codes in `core`, domain-specific codes next to
their one caller" split as `transfer.errCodeCurrencyMismatch`/
`errCodeInsufficientFunds` (`transfer/apperror.go:14-15`).

**Every other `sql.ErrNoRows` check is inline**, not routed through a shared
helper: `account/manage.go:43,96,160` (get/update/delete-by-id),
`transfer/handlers.go:22` (`requireOwnedAccount`, shared by three
`transfer` handlers), `transfer/transactions_handlers.go:58` (transfer's
`from_account` lookup), and `auth/handlers.go:207` (profile lookup) all
repeat the same `errors.Is(err, sql.ErrNoRows) → core.NotFoundErr(...)`
shape by hand rather than calling one function. There's no dedicated
custom-error-type package for domain errors: `internal/db/store/errors.go`'s
five sentinels are the only ones in the codebase; everything else is either
`sql.ErrNoRows`, a `*pq.Error`, or an ad hoc string literal passed straight
to `BadRequestErr`/`ConflictErr` inline in a handler (e.g. `auth`'s
`"password_mismatch"`/`"email_exists"`/`"username_exists"`, §7).

## 6. Ownership checks — the recurring "does this belong to the caller" shape

**Problem it solves:** every account-scoped endpoint needs to confirm the
authenticated caller actually owns the account being read or mutated, not
just that the account exists.

This isn't a shared helper across packages — it's the same four-line shape
copy-pasted into every handler that needs it, with one partial exception.
`account/manage.go` repeats it three times inline (`detailAccount:51-55`,
`updateAccount:104-108`, `deleteAccount:168-172`):

```go
// internal/api/account/manage.go:104-108
authPayload := core.GetAuthPayload(ctx)
if account.UserID.Int64 != authPayload.ID {
	core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
	return
}
```

`transfer` factors its version into one shared helper,
`requireOwnedAccount` (`transfer/handlers.go:19-37`), which bundles the
`GetAccountById` lookup, the `sql.ErrNoRows` → `NotFoundErr` mapping, and the
ownership check into one call returning `(db.Account, bool)` — used by
`deposit`, `listRecentTransferDestinations`, and
`listAccountTransactionHistory`. `transactionTransfer`
(`transfer/transactions_handlers.go:56-70`) does **not** call
`requireOwnedAccount` even though it needs the identical check for
`from_account_id` — it repeats the lookup and ownership check inline instead,
because it also needs the fetched `db.Account` for `TransferTx`'s
`arg.FromAccountID` in a way that happened not to get unified with the
helper. So there are, in total, two independently-written copies of "does
this account belong to the caller" in the `transfer` package alone, plus
`account`'s three separate inline copies — five copies of the same four-line
check across two packages, none sharing code.

## 7. Response DTOs — one mapper set per domain

**Problem it solves:** convert a sqlc-generated row into the API-facing
response shape, so handlers don't build response literals by hand — and, for
the endpoints that need it, hide fields a caller shouldn't see (another
user's account balance, `user_id`) when the row being displayed might belong
to someone other than the caller.

Each domain owns its own mapper(s); there's no shared mapper package.

| Domain | Mapper | Source row type(s) | Used by |
|---|---|---|---|
| `account` | `toAccountResponse` (`account/response.go:19`) | `db.Account` | `createAccount`, `updateAccount`, `deleteAccount` (via `deleteAccountResponse`) |
| `account` | `toAccountuserResponse` (`account/response.go:51`) | `db.AccountUserDetailsView` (embedded in three different `Row` types) | `detailAccount`, `listmeAccounts`, `searchAccountByNumber` |
| `transfer` | `toAccountEntriesViewResponse` (`transfer/response.go:49`) | `db.AccountEntriesView` (embedded) | `deposit`, `transactionTransfer`, `listAccountEntriesByAccountId` |
| `transfer` | `toPublicAccountResponseFromDestination` (`transfer/response.go:19`) | `db.ListRecentTransferDestinationsRow` | `listRecentTransferDestinations` |
| `transfer` | `toTransactionHistoryItem` (`transfer/response.go:82`) | `db.ListAccountTransactionHistoryRow` | `listAccountTransactionHistory` |
| `auth` | none — literal struct built inline | `db.User` | `getProfile` (`profileResponse`, `auth/handlers.go:184`) |

`toAccountuserResponse` and `toAccountEntriesViewResponse` are each
deliberately written to take the *embedded view type*
(`db.AccountUserDetailsView`, `db.AccountEntriesView`), not the outer
`...Row` struct sqlc generates per query — per the comment above each
(`account/response.go:46-50`, `transfer/response.go:45-48`), this is what
lets one mapper serve every query that does `sqlc.embed(v)` against that
view, instead of needing a new near-identical mapper per query. This is the
`docs/SQLC_ROW_MAPPING.md` pattern (`SELECT *`/`RETURNING *` +
`sqlc.embed(...)` over a hand-written per-query struct) actually paying off:
three account-view queries and two entries-view queries share one mapper
each.

**`publicAccountResponse` (`transfer/response.go:11-17`) is the one response
DTO in the codebase whose shape is a deliberate security boundary, not just a
translation convenience** — its doc comment says so directly: it "never
exposes balance or user_id" because `listRecentTransferDestinations` can
return accounts that belong to a different user than the caller (any account
the caller has previously sent money to). Every other response DTO in this
codebase is only ever built from a row the ownership checks (§6) already
confirmed belongs to the caller, so hiding fields was never a requirement for
them — `publicAccountResponse` is the only one where it is, and that's why
it's a narrower struct than `accountResponse`/`accountuserResponse` rather
than reusing one of them.

`getProfile` builds `profileResponse` as a literal directly in the handler
(`auth/handlers.go:215-220`) rather than through a `toXResponse` function —
it's the only response DTO in the codebase with no dedicated mapper function,
because it has exactly one caller and one source row type.

## Cross-feature coupling

- **`core.AuthMiddleware` uses the same `Fail`/`AppError` convention**, not
  something local to it: `AuthMiddleware` (`internal/api/core/auth_middleware.go:27-38`)
  calls `Fail(ctx, UnauthorizedErr(err.Error()))` on any token failure, which
  is rendered by the same global `ErrorHandlerMiddleware` as every other
  handler's errors. If you're only reading `response.go`/`apperror.go`, it's
  easy to miss that auth failures flow through the identical pipeline
  documented here rather than something bespoke.
- **`core.RecoveryMiddleware` also reports through `Fail`/`InternalErr`**
  (`internal/api/core/logging_middleware.go:316-331`): a panic is recovered,
  logged with a full stack trace, and turned into `InternalErr(fmt.Errorf("panic: %v", r))`
  — a caller sees the same generic 500 shape as any other unhandled error,
  with no indication a panic (rather than, say, a DB failure) caused it.
- **`transferAppError`/`deleteAccountAppError` read sentinel errors owned by
  the `store` package**, coupling `internal/api`'s response text to
  `internal/db/store/errors.go`'s error message strings (§5). The `store`
  package itself knows nothing about HTTP status codes or response envelopes
  — the translation is entirely on the `api` side, which is why adding a new
  failure mode to `TransferTx`/`DeleteAccountTx` requires a matching `case`
  in the domain's mapping function, or it silently falls into the generic
  `InternalErr(err)` branch (this is also the exact rule `AGENTS.md`'s "How
  to Handle Errors" section states).
- **Middleware registration order in `NewServer` is load-bearing**, and not
  just relative to `bindRouters`: `ErrorHandlerMiddleware` must be registered
  *before* `RecoveryMiddleware` (`internal/api/server.go:72-73`), not after.
  Gin middleware nests through `ctx.Next()` — a panic unwinds past every
  `ctx.Next()` call up to the first `recover()` above it in the chain,
  skipping any post-`ctx.Next()` code (like `ErrorHandlerMiddleware`'s
  response rendering) that sits *below* that `recover()`. Registering
  `ErrorHandlerMiddleware` first means it's still above `RecoveryMiddleware`
  in the chain, so once `RecoveryMiddleware`'s `recover()` calls
  `Fail(ctx, ...)` and returns normally, control passes back up through
  `ErrorHandlerMiddleware`'s own `ctx.Next()` call and it renders the 500.
  Get the order backwards and a panic gets "recovered" into a response
  nothing ever writes — the client sees an empty `200`. The full comment
  explaining this lives right above the `router.Use(...)` calls in
  `server.go:59-67`; see `docs/LOGGING.md` for the rest of the middleware
  chain.

## Presentational layer

This is an API-only service — no client/UI code lives in this repo. The
closest thing to a presentation layer is the Swagger/OpenAPI annotations on
each handler (e.g. `account/handlers.go:21-33`) and the generated docs under
`internal/docs/` (`docs.go`, `swagger.json`, `swagger.yaml`, all
`// Code generated ... DO NOT EDIT.`, regenerated via `make swag-gen`),
served at `/swagger/*any` (`internal/api/router.go:24`). Those files document
the shapes described here but contain no response-building logic of their
own.

## Summary / data flow

**Success path (e.g. `listmeAccounts`):**

```text
handler binds request (query/body/uri)
        │
        ▼
handler calls h.Store.* (sqlc query / *Tx business logic)
        │
        ▼
handler maps the result to a DTO — via that domain's toXResponse
mapper (§7) if the row type has one, by hand otherwise (auth/profile)
        │
        ▼
core.Succeed(ctx, status, data, message)              — single resource
core.SucceedWithMeta(ctx, status, data, message, meta) — paginated list, meta
                                                          from query.Page/Limit
                                                          + a separate Count*
                                                          query (see below)
        │
        ▼
ctx.JSON writes { data, message[, meta] } immediately — no middleware involved
```

Every list endpoint (`listmeAccounts`, `searchAccountByNumber`,
`listAccountEntriesByAccountId`, `listAccountTransactionHistory`) follows the
same shape: bind `core.PaginationQuery` (embedded in a request-specific query
struct), pass `query.Limit`/`query.Offset()` to the list query, fetch the
total via a **separate** `*Count`/`Count*` query (e.g.
`ListMeAccountsByUserIdCount`, `CountAccountEntriesByAccountId`,
`CountAccountTransactionHistory`), then call `SucceedWithMeta` with
`core.Meta{Page: query.Page, Limit: query.Limit, Total: total}`.
`listRecentTransferDestinations` binds `core.PaginationQuery` too but calls
plain `core.Succeed` with no `meta` — it's the one list endpoint in the
codebase that paginates its query without returning pagination metadata to
the client.

Worth flagging — the count is a **second round-trip**, not derived from the
list query, and the two queries aren't run in the same transaction. Under
concurrent writes between the list query and the count query, `total` and
the returned page can disagree (e.g. a row inserted in between is counted
but not listed, or vice versa). Nothing in the code addresses this; it's a
plain read-then-read race, not handled. `Meta.Page`/`Meta.Limit` are echoed
straight from what the client sent (or the `default=` values), not derived
from anything server-side — there's no `total_pages` field computed
anywhere, only `page`/`limit`/`total`.

**Error path (any handler):**

```text
something fails (bind error, sql.ErrNoRows, pq unique/check/fk violation,
store sentinel error, ownership mismatch, panic, unexpected error)
        │
        ▼
handler (or transferAppError/deleteAccountAppError, §5) classifies it into
one *AppError via a ValidationErr/NotFoundErr/ConflictErr/... constructor
        │
        ▼
core.Fail(ctx, err)  →  ctx.Error(err) + ctx.Abort()   — nothing written yet (§0)
        │
        ▼
core.ErrorHandlerMiddleware runs after ctx.Next() returns, finds the
*AppError via errors.As(ctx.Errors.Last().Err, &appErr), logs it (Warn for
4xx, Error + Cause for 5xx)
        │
        ▼
ctx.JSON(appErr.Status, { error: { code, message, details } })
— or a sanitized 500 if the last error isn't an *AppError at all
```

## Final reference table

Every endpoint that returns a normalized response, and how each one uses the
pieces above. All routes are under `/api/v1` except `/health`; "auth: no"
marks the two public auth routes, everything else sits behind
`core.AuthMiddleware`.

| Method | Endpoint | Auth | Success helper | DTO / mapper | Error paths |
|---|---|---|---|---|---|
| GET | `/health` | no | `Succeed` | `gin.H{"status":"ok"}`, no mapper | none |
| POST | `/auth/register` | no | `Succeed` | `AuthResponse` | `ValidationErr`, `BadRequestErr` (×3, ad hoc codes: `password_mismatch`/`email_exists`/`username_exists`), `ConflictErr`, `InternalErr` |
| POST | `/auth/login` | no | `Succeed` | `AuthResponse` | `ValidationErr`, `UnauthorizedErr`, `InternalErr` |
| GET | `/auth/profile` | yes | `Succeed` | `profileResponse` (inline, no mapper) | `NotFoundErr`, `InternalErr` |
| POST | `/accounts` | yes | `Succeed` | `accountResponse` via `toAccountResponse` | `ValidationErr`, `InternalErr` |
| GET | `/accounts/me` | yes | `SucceedWithMeta` | `[]accountuserResponse` via `toAccountuserResponse` | `ValidationErr`, `InternalErr` |
| GET | `/accounts/search-by-number` | yes | `SucceedWithMeta` | `[]accountuserResponse` via `toAccountuserResponse` | `ValidationErr`, `InternalErr` |
| GET | `/accounts/:id` | yes | `Succeed` | `accountuserResponse` via `toAccountuserResponse` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| PUT | `/accounts/:id` | yes | `Succeed` | `accountResponse` via `toAccountResponse` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| DELETE | `/accounts/:id` | yes | `Succeed` | `deleteAccountResponse{Account, BalanceSweptToID}` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `ConflictErr` (`deleteAccountAppError`), `InternalErr` |
| POST | `/accounts/:id/deposit` | yes | `Succeed` | `accountEntriesViewResponse` via `toAccountEntriesViewResponse` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `ConflictErr` (`transferAppError`), `InternalErr` |
| GET | `/accounts/:id/entries` | yes | `SucceedWithMeta` | `[]accountEntriesViewResponse` via `toAccountEntriesViewResponse` | `ValidationErr`, `InternalErr` |
| GET | `/accounts/:id/recent-destinations` | yes | `Succeed` (no meta, despite paginating) | `[]publicAccountResponse` via `toPublicAccountResponseFromDestination` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| GET | `/accounts/:id/transactions` | yes | `SucceedWithMeta` | `[]transactionHistoryItem` via `toTransactionHistoryItem` | `ValidationErr`, `NotFoundErr`, `ForbiddenErr`, `InternalErr` |
| POST | `/transactions/transfer` | yes | `Succeed` | `accountEntriesViewResponse` via `toAccountEntriesViewResponse` | `ValidationErr`, `NotFoundErr`, `BadRequestErr`, `ForbiddenErr`, `ConflictErr` (`transferAppError`), `InternalErr` |

## Known gaps / divergences

- **`core.ErrCodeNotFound` is used as the `code` for a 400 response** in
  `transferAppError`'s foreign-key-violation branch (§5) — everywhere else,
  `NOT_FOUND` implies a 404.
- **Duplicate-user handling has no dedicated error code of its own for the
  pre-check path.** `auth.registerUser` pre-checks email/username existence
  and reports them via `BadRequestErr` with inline string codes
  (`"email_exists"`, `"username_exists"`, plus `"password_mismatch"` for a
  mismatched confirmation) rather than constants — the comment above
  `core`'s `ErrCode*` block (`internal/api/core/apperror.go:16-19`, "Keep
  these stable — API consumers switch on them") only covers the six generic
  codes; these three domain-specific ones aren't held to that same
  stability discipline since they're inline literals, not named constants
  anywhere.
- **`details` is omitted, not emitted as `[]`, when there are no field
  errors.** `Details []FieldError \`json:"details,omitempty"\`` — for every
  error built via `NotFoundErr`, `UnauthorizedErr`, `ForbiddenErr`,
  `InternalErr`, `ConflictErr`, or `BadRequestErr` (none of which take field
  errors), `Details` is `nil`, and `omitempty` drops the key entirely rather
  than serializing an empty array.
- **The store-level `ErrInvalidAmount`/`ErrSameAccount` sentinel checks are
  unreachable through the HTTP API** — both are re-checked in the handler
  before the store call ever runs (§5). Not a bug, but worth knowing before
  assuming a test that only exercises the HTTP layer is exercising that
  store-level branch.
- **Ownership checks are duplicated five times across two packages with no
  shared helper**, and one of `transfer`'s own two call sites
  (`transactionTransfer`) doesn't use the one helper (`requireOwnedAccount`)
  the rest of that package shares (§6).
- **`getProfile` is the only response DTO with no dedicated `toXResponse`
  mapper** — it's built as a literal inline (§7). Not wrong (it has exactly
  one caller), but inconsistent with every other domain's convention of a
  named mapper function.
- **`validationMessage`'s hand-written wording only covers 5 of the binding
  tags actually used** (`required`, `min`, `max`, `gt`, `oneof`) —
  `email` and `eqfield` (both used only in `auth`) fall through to the
  validator library's own default message text (§4).
- **Every `succeed`/`succeedWithMeta` call site passes `http.StatusOK`**,
  including `createAccount`, which creates a resource — no handler in this
  codebase has ever used a 201 (§2).
