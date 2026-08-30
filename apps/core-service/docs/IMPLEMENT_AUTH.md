# Authentication — As Implemented

## Who this doc is for

You're comfortable with Go, Gin, and REST APIs, but you haven't necessarily used JWT before, or you have and want to know what *this* service actually does with it rather than the textbook version. Section 0 is a short primer on JWT vs. its siblings — skip it if you already know the tradeoffs.

Everything after that is verified against the source in this repo as of the commits leading up to `8f68240` (2026-08-30), file:line cited throughout. This is **not** a design doc — where the code's actual behavior diverges from what you'd reasonably expect (a status-code inconsistency, a test that's currently red, a column that's populated but never read), that's called out explicitly rather than smoothed over. Read the callouts before you build on top of a piece of this system; a couple of "should work" assumptions don't currently hold.

## 0. Background primer — why JWT, and what it isn't

| Approach | Where identity lives between requests | Server-side state | Revocation | Typical use |
|---|---|---|---|---|
| Session cookie | Opaque ID in a cookie | A session store (Redis/DB) | Instant — delete the row | Server-rendered apps, same-origin |
| **JWT (this service)** | Signed, self-contained token held by the client | None | None (expires only) | Stateless APIs, multiple backend services |
| PASETO | Same idea as JWT, versioned/encrypted format | None | None (expires only) | Same as JWT, avoids JWT's algorithm-confusion footguns by design |

Gotchas that aren't obvious from "JWT = a token with a signature":

1. **Encoded, not encrypted.** Anyone holding the token can base64-decode the payload and read `id`/`username`/`email` without knowing the secret. This is *why* §2 explicitly says never put sensitive data in `Payload` — the signature stops tampering, not reading.
2. **Stateless means unrevokable.** There's no server-side record of an issued token, so there is no way to invalidate one before it expires short of rotating the signing secret (which invalidates *every* token). This is why §11 lists "no logout / no blacklist" as a deliberate gap, not an oversight.
3. **The algorithm is part of the attack surface.** A JWT library that trusts the `alg` header in the token itself can be tricked into verifying with `alg: none`, or into treating an HMAC secret as an RSA public key. §2 shows the one line of code in this repo that exists solely to prevent that.

## 1. Architecture at a glance

The composition root is `internal/api/server.go` — `NewServer` (`internal/api/server.go:37`) builds the token maker, registers global middleware, and calls `bindRouters` (`internal/api/router.go:11`) to wire routes to handlers. Neither file implements any auth *logic* itself — they only assemble pieces owned elsewhere, which is what makes the table below possible.

| Concern | Owner (file) | Analogy |
|---|---|---|
| Prove *who* is calling | `internal/token/` (`Maker`, `JWTMaker`, `Payload`) | The wristband-stamping machine at a venue door |
| Prove a password is correct | `pkg/utils/password.go` | The deadbolt on the account itself |
| Gate every request behind a valid token | `internal/api/auth_middleware.go` | The bouncer checking wristbands at the door |
| Decide *whether* this caller may touch this resource | Per-handler checks in `account_router.go`, `transactions_router.go` | The coat-check ticket matching a name to a coat |
| Render every failure the same shape | `internal/api/apperror.go` | The front desk giving one consistent script for any problem |
| Supply secrets/durations from the environment | `internal/config/config.go` | The knobs the venue manager sets before opening night |

This is split because authentication (who) and authorization (can-touch-this) are genuinely different questions with different failure modes — a valid wristband (token) says nothing about which coat (resource) is yours. Keep that split in mind for §9: there's no generic "ownership middleware," and you'll see the consequence of that (an inconsistency between two ownership-check strategies) called out there.

## 2. Token package — `internal/token/`

**Problem it solves:** issue something a client can hold and present later that proves "this request comes from user N," without the server storing per-request state.

One interface, so `internal/api` never imports the JWT library directly:

```go
// internal/token/maker.go:7
type Maker interface {
    CreateToken(userID int64, username, email string, duration time.Duration) (string, *Payload, error)
    VerifyToken(token string) (*Payload, error)
}
```

If you ever swapped HS256 for PASETO or RS256 (§0 above), only `internal/token` would change.

**Payload** (`internal/token/payload.go:14`) embeds `jwt.RegisteredClaims` and adds exactly the fields handlers need:

```go
type Payload struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    jwt.RegisteredClaims
}
```

| Field | Source | Notes |
|---|---|---|
| `ID`, `Username`, `Email` | Passed in at `CreateToken` call site | No password hash, no role/scope — see §0 gotcha 1 and §11 |
| `IssuedAt`, `ExpiresAt` | Set in `NewPayload` (`internal/token/payload.go:22`) | No `jti` (unique token ID) — a leaked-token detection scheme would need one; not present here |

**JWTMaker** (`internal/token/jwt_maker.go`):

- `NewJWTMaker` (`internal/token/jwt_maker.go:20`) rejects any secret under 32 bytes, so a weak `.env` value fails at startup instead of producing a brute-forceable token.
- `CreateToken` (`internal/token/jwt_maker.go:27`) signs with `jwt.SigningMethodHS256` exclusively.
- `VerifyToken` (`internal/token/jwt_maker.go:39`) does two things worth separating:
  ```go
  keyFunc := func(t *jwt.Token) (interface{}, error) {
      if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
          return nil, jwt.ErrTokenSignatureInvalid
      }
      return []byte(maker.secretKey), nil
  }
  jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc,
      jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
  ```
  The `keyFunc` method-type check *and* `jwt.WithValidMethods` both pin the algorithm. This double-pinning is the concrete answer to §0 gotcha 3: without it, a token forged with `alg: none`, or `alg: RS256` using the server's own HS256 secret reinterpreted as a public key, could bypass verification. `TestJWTMakerInvalidAlgNone` (`internal/token/jwt_maker_test.go:56`) exists specifically to pin this down.
- Expired tokens are translated to a sentinel `ErrExpiredToken` (`internal/token/jwt_maker.go:10`) so callers *could* distinguish "expired" from "otherwise invalid" — worth flagging: nothing in `internal/api` currently branches on this sentinel (see §6), so today it has no observable effect on the HTTP response.

## 3. Password hashing — `pkg/utils/password.go`

**Problem it solves:** never store a password (or anything reversible to it) at rest.

```go
// pkg/utils/password.go:10
func HashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    ...
}

func CheckPassword(password, hashedPassword string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

Plain bcrypt at the library's default cost (10), no pepper. `hashed_password` is the only password-related column on `users` (`db/query/user.sql:2`) — there is no plaintext or reversible-encryption path anywhere in this codebase.

If you're new to bcrypt: unlike a plain hash function, `CompareHashAndPassword` is intentionally slow and its output already includes the salt, which is why there's no separate salt column to manage.

## 4. Configuration — `internal/config/config.go`

JWT-relevant fields on `Config` (`internal/config/config.go:12`):

```go
JWTSecretKey           string        `mapstructure:"JWT_SECRET_KEY"`
JWTAccessTokenDuration time.Duration `mapstructure:"JWT_ACCESS_TOKEN_DURATION"`
```

Loaded from `app.env` if present, otherwise from real environment variables. If you're new to viper: `viper.AutomaticEnv()` only picks up keys viper already knows about from a config file — with no `app.env` mounted (e.g. Docker Compose), viper knows *no* keys, so env vars would be silently ignored. `LoadConfig` (`internal/config/config.go:23`) works around this by explicitly calling `viper.BindEnv` for every `mapstructure` tag on `Config` (`internal/config/config.go:45-54`) before unmarshalling.

There is currently **no refresh-token duration field** — only a single access token is issued for both register and login (see §11).

## 5. Auth endpoints — `internal/api/auth_router.go`

**Problem it solves:** turn a set of credentials into an access token, or reject them without telling the caller more than necessary.

### `POST /api/v1/auth/register`

```text
bind + validate request
  → password == password_confirm ?          400 password_mismatch
  → email already taken? (GetUserByEmail)   400 email_exists
  → username already taken?                 400 username_exists
  → HashPassword
  → store.CreateUser                        409 (on DB unique_violation race)
  → tokenMaker.CreateToken
  → 200 { access_token, token_type, expires_in }
```

Validation on `registerUserRequest` (`internal/api/auth_router.go:74`) uses Gin binding tags: `email` (`required,email`), `password` (`required,min=8`), `password_confirm` (`required,eqfield=Password`).

**Worth flagging — two status codes for the same underlying problem.** The pre-insert uniqueness checks (`internal/api/auth_router.go:107-126`) return **400** via `BadRequestErr("email_exists"/"username_exists", ...)`. The DB-level backstop for the same conflict — a race where two requests slip past both pre-checks — returns **409** via `ConflictErr` (`internal/api/auth_router.go:142-151`). A client can't rely on one status code meaning "duplicate email/username"; it has to check both 400 and 409. `auth_router_test.go` documents both paths directly: the pre-check case at `internal/api/auth_router_test.go:119-134` and `:147-166` expects 400, the race case at `internal/api/auth_router_test.go:183-193` expects 409 with a comment calling it "belt-and-suspenders."

### `POST /api/v1/auth/login`

```text
bind + validate
  → GetUserByEmail (sql.ErrNoRows → 401, not 404)
  → utils.CheckPassword                      → 401 on mismatch
  → tokenMaker.CreateToken
  → 200 { access_token, token_type, expires_in }
```

Both "user does not exist" and "wrong password" return the identical `401 invalid username or password` (`internal/api/auth_router.go:49` and `:57`) — intentional: a 404 for unknown email vs. 401 for wrong password would let an attacker enumerate registered emails.

Actual response envelope (both endpoints wrap `AuthResponse` in the service-wide success shape documented in [`NORMALIZE_RESPONSE.md`](NORMALIZE_RESPONSE.md); the `message` string differs — "Login successful" vs. "Registration successful"):

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 60
  },
  "message": "Login successful"
}
```

## 6. Middleware — `internal/api/auth_middleware.go`

**Problem it solves:** run the "is this token valid" check once, in one place, for every route that needs it.

```go
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        payload, err := parseAuthHeader(ctx.GetHeader(authorizationHeaderKey), tokenMaker)
        if err != nil {
            fail(ctx, UnauthorizedErr(err.Error()))
            return
        }
        ctx.Set(authorizationPayloadKey, payload)
        ctx.Next()
    }
}
```

`parseAuthHeader` (`internal/api/auth_middleware.go:40`) handles header parsing explicitly rather than `strings.TrimPrefix`:

| Failure | Trigger | Sentinel |
|---|---|---|
| Missing header | Empty `Authorization` value | `errAuthHeaderMissing` |
| Malformed header | `strings.Fields` doesn't produce exactly 2 tokens (`"Bearer"` alone, or `"Bearer x y"`) | `errAuthHeaderInvalid` |
| Wrong scheme | First field isn't case-insensitively `"bearer"` | `errAuthTypeUnsupported` |
| Bad/expired token | `tokenMaker.VerifyToken` fails | Whatever `VerifyToken` returns, including `ErrExpiredToken` (§2) |

All four collapse to the same `401` — there's no reason to tell a caller "you sent garbage" vs. "you sent an expired token."

`getAuthPayload(ctx)` (`internal/api/auth_middleware.go:64`) is how every downstream handler learns who's calling:

```go
func getAuthPayload(ctx *gin.Context) *token.Payload {
    payload, ok := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
    if !ok {
        return nil
    }
    return payload
}
```

**Worth flagging — this now has a nil-safe fallback.** `ctx.MustGet` still panics if the key was never set (i.e. the route isn't behind `authMiddleware`), but the `, ok :=` type assertion means a *present-but-wrong-typed* value returns `nil` instead of panicking. In practice only `authMiddleware` ever sets this key, and always with the correct type, so the fallback is currently dead code — but it also means if that ever changes, callers get a silent `nil` instead of a loud panic. Every handler that calls `getAuthPayload` (`getAccount`, `listAccountEntries`, `createAccount`, `listAccounts`, `GetProfile`, …) immediately dereferences a field on the result (e.g. `authPayload.Username` at `internal/api/account_router.go:100`) with no nil check, so a `nil` here would surface as an unhandled panic anyway — just one HTTP request downstream of where it actually went wrong, which is harder to debug than a panic at the source.

## 7. Route wiring — `internal/api/router.go`

```go
v1 := router.Group("/api/v1")

// public
v1.POST("/users", server.createUser) // TODO: DELETE this end point
v1.POST("/auth/login", server.loginUser)
v1.POST("/auth/register", server.registerUser)

// everything below requires a valid token
authorized := v1.Group("/")
authorized.Use(authMiddleware(server.tokenMaker))

authorized.GET("/auth/profile", server.GetProfile)
authorized.POST("/accounts", server.createAccount)
authorized.GET("/accounts/:id", server.getAccount)
authorized.GET("/accounts", server.listAccounts)
authorized.GET("/accounts/:id/entries", server.listAccountEntries)
authorized.POST("/transactions/transfer", server.transactionTransfer)
authorized.GET("/transactions/:id", server.getTransaction)
authorized.GET("/transactions", server.listTransactions)
```

Splitting into `v1` vs. `authorized := v1.Group("/")` with `.Use(authMiddleware(...))` is the standard Gin pattern for "protect everything registered after this point in this group" — a new authenticated route just needs to go under `authorized`.

`server.go` (`internal/api/server.go:37`) builds the token maker once at startup and registers `corsMiddleware(cfg.CORSAllowedOrigins)` and `errorHandlerMiddleware()` globally, *before* `bindRouters` runs — see §Cross-Feature Coupling for why the CORS piece matters to auth specifically.

Two public, unauthenticated routes exist alongside the two `/auth/*` ones: `/health` and `/swagger/*any` (`internal/api/router.go:12-14`) — neither is auth-related, both listed for completeness in the final reference table.

## 8. Error semantics — `internal/api/apperror.go`

Every handler reports failures through `fail(ctx, err)` with a typed `*AppError`, rendered centrally by `errorHandlerMiddleware` (`internal/api/apperror.go:82`) into the envelope documented in [`NORMALIZE_RESPONSE.md`](NORMALIZE_RESPONSE.md).

| Helper | Status | Used for (in the auth flow) |
|---|---|---|
| `ValidationErr` | 400 | Request body failed Gin binding (missing field, bad email format, `min=8`) |
| `BadRequestErr` | 400 | Business-rule violation not tied to a single field: password mismatch, duplicate email/username pre-check (§5) |
| `UnauthorizedErr` | 401 | Missing/malformed/expired/invalid token; bad login credentials |
| `ForbiddenErr` | 403 | Valid token, but the resource belongs to someone else (§9) |
| `NotFoundErr` | 404 | Row genuinely doesn't exist (e.g. profile lookup after the user row was deleted) |
| `ConflictErr` | 409 | Unique-constraint race on insert (§5) |
| `InternalErr` | 500 | Anything unexpected — never leaks the underlying error text |

`fail` (`internal/api/apperror.go:71`) never builds JSON itself; any handler that forgets to wrap an error still gets a sanitized 500 by default (`internal/api/apperror.go:100`), so a raw internal error can't leak to a client just from a missing `errors.As` check.

A related, non-auth-specific piece: `registerValidatorFieldNames` (`internal/api/server.go:61`) makes `ValidationErr`'s field-level details report the request's JSON field name (`password_confirm`) instead of the Go struct field name (`PasswordConfirm`) — this is what makes the 400 body for a bad register request point at a field the client actually sent.

## 9. Authorization / ownership checks — per handler, not centralized

**Problem it solves:** a valid token proves *who* is calling, not that they're allowed to see a specific row.

There's no generic "ownership middleware" (§1). Two different strategies exist side by side:

**Strategy A — scope the query itself**, used by list endpoints: `listAccounts` builds `ListAccountsByUserIdParams{UserID: authPayload.ID}` (`internal/api/account_router.go:137-144`), so there's nothing to leak in the first place.

**Strategy B — fetch, then compare**, used by single-resource endpoints:

```go
// internal/api/account_router.go:89-103 — getAccount
account, err := server.store.GetAccountById(ctx, params.ID)
// ... sql.ErrNoRows → 404 ...

authPayload := getAuthPayload(ctx)
if account.Owner != authPayload.Username {
    fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
    return
}
```

The same pattern repeats in `listAccountEntries` (`internal/api/account_router.go:207-211`).

| Endpoint | Strategy | Compares on |
|---|---|---|
| `listAccounts` | A (query-scoped) | `accounts.user_id` |
| `getAccount` | B (fetch + compare) | `accounts.owner` (username string) |
| `listAccountEntries` | B (fetch + compare) | `accounts.owner` (username string) |

**Worth flagging — Strategy B never looks at `user_id`.** Migration `000004_authorization_by_user_tabel.up.sql` (`db/migrations/000004_authorization_by_user_tabel.up.sql:6-22`) added a nullable `user_id` FK to `accounts` specifically to support authorization by user, and `createAccount` populates it on every new row (`internal/api/account_router.go:43-46`). Strategy A already uses that column. But `getAccount` and `listAccountEntries` still compare against the older `owner` string column instead — both columns should agree for any account created after this migration, so this isn't wrong today, but it means two different columns are the source of truth for "who owns this" depending on which endpoint you're reading, and a future rename of `owner` (or an account whose `user_id` and `owner` drift) would only break half the ownership checks.

`403` consistently means "authenticated but not permitted" — `NotFoundErr` (404) is deliberately never used for "not your resource," since that would leak which resource IDs exist to unauthorized callers. When adding a new "get one resource by ID" endpoint, follow the `getAccount` pattern: fetch → compare owner → 403 if mismatched, before returning any field.

## Cross-Feature Coupling

Two pieces registered outside `internal/token`/`internal/api/auth_*` still shape how authentication behaves end-to-end — easy to miss if you're only reading the auth files:

- **CORS and the `Authorization` header.** `corsMiddleware` (`internal/api/server.go:88`) explicitly allowlists `Authorization` in `AllowHeaders` and sets `AllowCredentials: true`. If a browser client's origin isn't in `CORSAllowedOrigins`, the browser will block the request *before* `authMiddleware` ever runs — a token problem and a CORS problem look identical from the client (request never reaches the handler), but only one of them is fixed in `internal/token`. An empty `CORSAllowedOrigins` disables CORS handling entirely (`internal/api/server.go:89-91`), which only works for server-to-server callers, not browsers.
- **Validator field naming.** `registerValidatorFieldNames` (`internal/api/server.go:61`, called once from `NewServer`) is global — it affects every `ValidationErr` in the service, not just auth's. It's mentioned here because it's easy to attribute register/login's nicely-named validation errors to `auth_router.go` when the actual behavior lives in `server.go`.

## Presentational layer

This is an API-only service — there's no client/UI code in this repo to document. The closest thing to a presentation layer is the Swagger/OpenAPI annotations on each handler (e.g. `internal/api/auth_router.go:27-38`, `internal/api/profile_router.go:19-29`), served at `/swagger/*any` (`internal/api/router.go:14`). Authenticated endpoints are marked `@Security BearerAuth`; that annotation is documentation only and has no runtime effect — the actual enforcement is `authMiddleware` in §6.

## Summary / data flow

**Register / Login (happy path):**

```text
POST /api/v1/auth/register or /api/v1/auth/login
        │
        ▼
  auth_router.go handler
        ├── validate request body (Gin binding tags)
        ├── look up / create user (sqlc Querier via Server.store)
        ├── bcrypt.CompareHashAndPassword (login) or HashPassword (register)
        └── tokenMaker.CreateToken(userID, username, email, duration)
        ▼
  200 { data: { access_token, token_type, expires_in }, message }
```

**Authenticated request (happy path):**

```text
Authorization: Bearer <token>
        │
        ▼
  authMiddleware (§6)
        ├── split header into ["Bearer", "<token>"]
        ├── tokenMaker.VerifyToken(token)   ← signature + expiry + alg check
        └── ctx.Set("authorization_payload", *token.Payload)
        ▼
  handler calls getAuthPayload(ctx)
        └── (per-handler) ownership check against the resource being accessed (§9)
```

**Error path (any auth failure):**

```text
missing header / bad scheme / malformed token / wrong signature / expired token
        │
        ▼
  fail(ctx, UnauthorizedErr(...))  →  errorHandlerMiddleware  →  401 { error: {...} }
```

## Test coverage

| File | Covers | Status |
|---|---|---|
| `internal/token/jwt_maker_test.go` | create/verify round-trip, expired token, `alg: none` forgery, malformed/tampered token, wrong secret, short-secret rejection | Passing |
| `internal/api/auth_middleware_test.go` | header missing / malformed / wrong scheme / invalid / expired / valid token reaching the handler | Passing |
| `internal/api/auth_router_test.go` | register + login against a mocked `Storer`, including duplicate email/username (400) and unique-constraint race (409) | Passing |
| `internal/api/profile_router_test.go` | authenticated profile fetch | **Failing** |

**Worth flagging — `TestGetProfile` is currently red.** `doGetProfileRequest` (`internal/api/profile_router_test.go:19`) sends requests to `/api/v1/profile`, but the route is registered at `/api/v1/auth/profile` (`internal/api/router.go:30`). Every subtest gets a `404` from Gin's router before it ever reaches `GetProfile`, `authMiddleware`, or the mock — confirmed by running `go test ./internal/api/... -run TestGetProfile`. This isn't a doc-accuracy issue, it's a real bug in the test file (or the route, if `/api/v1/profile` was the intended path) that predates this doc update — fix the path in one file to match the other before relying on this test as a safety net for §6.

`auth_router_test.go` builds a real `*Server` with a mocked store (`newTestServerWithMockStore`, `internal/api/auth_router_test.go:37`) and drives requests through `server.router.ServeHTTP`, so it's a router-level integration test — binding, middleware, and error-rendering are exercised together, not just the handler function.

## Final reference table

| Method | Endpoint | Auth required | Purpose |
|---|---|---|---|
| GET | `/health` | No | Liveness check |
| GET | `/swagger/*any` | No | OpenAPI docs UI |
| POST | `/api/v1/users` | No | **Deprecated** — duplicates `/auth/register` without issuing a token; marked for deletion (`internal/api/router.go:20`) |
| POST | `/api/v1/auth/register` | No | Create a user, return an access token |
| POST | `/api/v1/auth/login` | No | Authenticate, return an access token |
| GET | `/api/v1/auth/profile` | Yes | Return the authenticated user's profile |
| POST | `/api/v1/accounts` | Yes | Create an account owned by the caller |
| GET | `/api/v1/accounts/:id` | Yes | Get one account (403 if not the owner) |
| GET | `/api/v1/accounts` | Yes | List the caller's accounts |
| GET | `/api/v1/accounts/:id/entries` | Yes | List ledger entries for an owned account |
| POST | `/api/v1/transactions/transfer` | Yes | Transfer between accounts |
| GET | `/api/v1/transactions/:id` | Yes | Get one transaction |
| GET | `/api/v1/transactions` | Yes | List transactions |

## Known gaps / deliberate simplifications

- **No refresh token.** Only a single access token is issued (`JWTAccessTokenDuration`, used for both register and login). A leaked token is valid until it expires. If you add refresh tokens, don't put them inside the JWT — store them server-side (e.g. a `refresh_tokens` table with hash + expiry) so they can be revoked.
- **No token blacklist / logout endpoint.** JWTs here are entirely stateless (§0); "logout" only means "the client discards the token."
- **`POST /api/v1/users` is still public and unauthenticated** (`internal/api/router.go:20`, `internal/api/user_router.go`). It hashes the password and creates the user but issues no token, and predates `/auth/register`. Marked for deletion — don't build on it.
- **Roles/permissions aren't modeled.** `Payload` has no role or scope claim, so every authenticated user has identical capabilities beyond resource ownership. If you add roles, look them up per-request from the DB rather than trusting a `role` claim baked into a long-lived JWT (a role change wouldn't take effect until the token expires).
- **No rate limiting on `/auth/login`.** Nothing here slows down credential-stuffing attempts.
- **Two status codes for one problem** (§5): duplicate email/username is 400 on the pre-check path, 409 on the DB-race path.
- **`getAuthPayload`'s nil-safe fallback is unexercised dead code that would surface as a downstream panic if it ever triggered** (§6).
- **Ownership checks read two different columns depending on the endpoint** — `user_id` (FK) for list queries, `owner` (username string) for single-resource fetch-then-compare checks (§9).
- **`TestGetProfile` is currently failing** due to a URL mismatch between the test and the route (Test coverage, above) — not an auth-logic bug, but treat this test as not currently providing coverage until it's fixed.
