# Authentication Implementation — Gin + JWT (HS256)

This documents how authentication is actually implemented in this service, file by file. It assumes you know what a JWT is and have written a REST API before; it focuses on the concrete design decisions and where the code lives, not on JWT theory.

Stack: **Gin**, **sqlc**, **PostgreSQL**, [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt), `golang.org/x/crypto/bcrypt`, `viper` for config.

## 1. Request flow

```text
POST /api/v1/auth/register or /api/v1/auth/login
        │
        ▼
  auth_router.go handler
        │
        ├── validate request body (Gin binding tags)
        ├── look up / create user (sqlc Querier via Server.store)
        ├── bcrypt.CompareHashAndPassword (login) or HashPassword (register)
        └── tokenMaker.CreateToken(userID, username, email, duration)
        ▼
  200 { access_token, token_type: "Bearer", expires_in }

Client stores the token, then sends it on every subsequent request:

  Authorization: Bearer <token>
        │
        ▼
  authMiddleware (internal/api/auth_middleware.go)
        │
        ├── split header into ["Bearer", "<token>"]
        ├── tokenMaker.VerifyToken(token)   ← signature + expiry check
        └── ctx.Set("authorization_payload", *token.Payload)
        ▼
  handler calls getAuthPayload(ctx) to read who's calling
        │
        └── (per-handler) ownership check against the resource being accessed
```

There is no separate "session" or "authorize" layer in the middleware — authentication (verifying the token) and authorization (can this user touch this resource) are deliberately split: the middleware only proves *who* is calling, and each handler that touches an owned resource (accounts, entries, transactions) decides *whether* that caller is allowed to see it.

## 2. Token package — `internal/token/`

Three files, one interface:

```go
// internal/token/maker.go
type Maker interface {
    CreateToken(userID int64, username, email string, duration time.Duration) (string, *Payload, error)
    VerifyToken(token string) (*Payload, error)
}
```

`Maker` exists purely so `Server` depends on an interface, not on `jwt-go` directly — `internal/api` never imports the JWT library. If you ever swapped HS256 for PASETO or RS256, only `internal/token` would change.

**Payload** (`internal/token/payload.go`) embeds `jwt.RegisteredClaims` and adds the fields handlers actually need:

```go
type Payload struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    jwt.RegisteredClaims
}
```

Only `iat`/`exp` (from `RegisteredClaims`) plus `id`/`username`/`email` are in the token. No password hash, no role/permission list — keep reading below for why that matters once you add roles.

**JWTMaker** (`internal/token/jwt_maker.go`):

- `NewJWTMaker(secretKey string)` rejects any secret under 32 bytes (`minSecretKeySize`), so a weak `.env` value fails fast at startup rather than producing a brute-forceable token.
- `CreateToken` signs with `jwt.SigningMethodHS256` exclusively.
- `VerifyToken` uses `jwt.ParseWithClaims(..., jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))`. This is the important line: without pinning the accepted algorithm, a token forged with `alg: none` or `alg: RS256` (using the server's own HS256 secret reinterpreted as an RSA public key) could bypass verification — the classic "alg confusion" JWT attack. `jwt_maker_test.go`'s `TestJWTMakerInvalidAlgNone` exists specifically to pin this down.
- Expired tokens are translated to a sentinel `ErrExpiredToken` so callers can distinguish "expired" from "otherwise invalid" if they ever need to (e.g. to trigger a refresh flow later).

## 3. Password hashing — `pkg/utils/password.go`

```go
func HashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    ...
}

func CheckPassword(password, hashedPassword string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

Plain bcrypt at the library's default cost (10). `hashed_password` is the only password-related column in `users` (see `db/query/user.sql`) — there's no plaintext or reversible encryption path anywhere in the codebase.

## 4. Configuration — `internal/config/config.go`

JWT-relevant fields on `Config`:

```go
JWTSecretKey           string        `mapstructure:"JWT_SECRET_KEY"`
JWTAccessTokenDuration time.Duration `mapstructure:"JWT_ACCESS_TOKEN_DURATION"`
```

Loaded from `app.env` if present, otherwise from real environment variables — `LoadConfig` explicitly calls `viper.BindEnv` for every `mapstructure` tag because `viper.AutomaticEnv()` alone only picks up keys viper already knows about from a config file, which would silently break env-only deployments (e.g. Docker Compose with no `app.env` mounted). There is currently **no refresh-token duration field** — only a single access token is issued (see §8, gaps).

## 5. Auth endpoints — `internal/api/auth_router.go`

### `POST /api/v1/auth/register`

```text
bind + validate request
  → password == password_confirm ?
  → email already taken? (GetUserByEmail)
  → username already taken? (CheckIsUsernameExist)
  → HashPassword
  → store.CreateUser
  → tokenMaker.CreateToken
  → 200 { access_token, token_type, expires_in }
```

Validation on `registerUserRequest` uses Gin binding tags: `email` (`binding:"required,email"`), `password` (`binding:"required,min=8"`), and `password_confirm` (`binding:"required,eqfield=Password"`). Uniqueness is checked in the handler *before* insert (two round-trips), and the DB unique constraint is still the real backstop — if a race slips through, `CreateUser` returns a Postgres `unique_violation`, which is caught via `errors.As(err, &pqErr)` and mapped to a `409 Conflict`.

### `POST /api/v1/auth/login`

```text
bind + validate
  → GetUserByEmail (sql.ErrNoRows → 401, not 404 — see below)
  → utils.CheckPassword
  → tokenMaker.CreateToken
  → 200 { access_token, token_type, expires_in }
```

Both "user does not exist" and "wrong password" return the same `401 invalid username or password` — this is intentional and matters: returning 404 for an unknown email vs 401 for a wrong password would let an attacker enumerate which emails are registered.

Response shape (`AuthResponse`) is identical for register and login:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

## 6. Middleware — `internal/api/auth_middleware.go`

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

`parseAuthHeader` handles the header parsing edge cases explicitly rather than relying on `strings.TrimPrefix`:

- missing header → `errAuthHeaderMissing`
- `strings.Fields(header)` must produce exactly 2 tokens (so `"Bearer"` alone, or `"Bearer x y"`, both fail) → `errAuthHeaderInvalid`
- scheme must case-insensitively equal `"bearer"` → `errAuthTypeUnsupported`
- only then is `tokenMaker.VerifyToken` called

All three failure modes surface as `401`, which matches the "you are not authenticated" semantics from §7 — there's no reason to distinguish "you sent garbage" from "you sent an expired token" to the client.

`getAuthPayload(ctx)` is how every downstream handler gets the caller's identity:

```go
func getAuthPayload(ctx *gin.Context) *token.Payload {
    payload, ok := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
    ...
}
```

It uses `MustGet`, so it will panic if called on a route that isn't behind `authMiddleware`. That's deliberate — it's a programmer error to call it from a public route, and Gin's recovery middleware turns that into a 500 rather than a silent nil-payload bug. Don't add a nil-safe fallback here; fix the route wiring instead.

## 7. Route wiring — `internal/api/router.go`

```go
v1 := router.Group("/api/v1")

// public
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

Splitting into `v1` vs `authorized := v1.Group("/")` with `.Use(authMiddleware(...))` is the standard Gin pattern for "protect everything registered after this point in this group" — any new authenticated route just needs to be added under `authorized`, not wrapped individually.

Note: `router.go:20` still has `v1.POST("/users", server.createUser)` marked `// TODO: DELETE this end point` — it's a public, unauthenticated user-creation endpoint left over from before `/auth/register` existed. It duplicates `registerUser` without issuing a token. See §8.

`server.go` builds the token maker once at startup (`token.NewJWTMaker(cfg.JWTSecretKey)`) and stores it on `Server`, alongside `errorHandlerMiddleware()` and `corsMiddleware()` registered globally before routes are bound.

## 8. Error semantics — `internal/api/apperror.go`

Every handler reports failures through `fail(ctx, err)` with a typed `*AppError`, rendered centrally by `errorHandlerMiddleware`:

| Helper | Status | Used for |
|---|---|---|
| `UnauthorizedErr` | 401 | missing/malformed/expired/invalid token, or bad login credentials |
| `ForbiddenErr` | 403 | valid token, but the resource belongs to someone else |
| `ValidationErr` | 400 | request body/query/uri failed binding |
| `ConflictErr` | 409 | unique constraint violation (duplicate email/username) |

This is what makes `403` mean "authenticated but not permitted" everywhere in this codebase — see the ownership checks below. Any handler that used `NotFoundErr` (404) for "not your resource" would leak which resource IDs exist to unauthorized callers; this codebase consistently prefers 403 for that case once the row is found, and 404 only when the row genuinely doesn't exist.

## 9. Authorization / ownership checks — per handler, not centralized

There's no generic "ownership middleware." Each handler that reads a specific resource fetches it first, then compares to the caller:

```go
// internal/api/account_router.go — getAccount
account, err := server.store.GetAccountById(ctx, params.ID)
// ... sql.ErrNoRows → 404 ...

authPayload := getAuthPayload(ctx)
if account.Owner != authPayload.Username {
    fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
    return
}
```

The same pattern repeats in `listAccountEntries`. List endpoints (`listAccounts`, `listTransactions`) instead scope the *query itself* to `authPayload.ID` (`ListAccountsByUserIdParams{UserID: ...}`), so there's nothing to leak in the first place — ownership is enforced by the WHERE clause, not a post-hoc check. When you add a new "get one resource by ID" endpoint, follow the `getAccount` pattern: fetch → compare owner → 403 if mismatched, *before* returning any field of the resource.

## 10. Tests

| File | Covers |
|---|---|
| `internal/token/jwt_maker_test.go` | create/verify round-trip, expired token, `alg: none` forgery attempt, malformed token, tampered token, wrong secret, short-secret rejection |
| `internal/api/auth_middleware_test.go` | header missing / malformed / wrong scheme / invalid token / expired token / valid token reaching the handler |
| `internal/api/auth_router_test.go` | register + login against a mocked `Storer` (via `mockdb.MockQuerier` + gomock), including the duplicate-email/username and password-mismatch paths |
| `internal/api/profile_router_test.go` | authenticated profile fetch |

`auth_router_test.go` builds a real `*Server` with a mocked store (`newTestServerWithMockStore`) and drives requests through `server.router.ServeHTTP`, so these are router-level integration tests, not just handler-function unit tests — they exercise binding, middleware, and error-rendering together.

## 11. Known gaps / deliberate simplifications

If you extend this, be aware of what's *not* here yet:

- **No refresh token.** Only a single access token is issued (`JWTAccessTokenDuration`, currently used for both register and login). There's no revocation path — a leaked token is valid until it expires. If you add refresh tokens, they should *not* go inside the JWT itself; store them server-side (e.g. a `refresh_tokens` table with hash + expiry) so they can be revoked.
- **No token blacklist / logout endpoint.** JWTs here are entirely stateless; "logout" only means "the client discards the token."
- **`POST /api/v1/users` is still public and unauthenticated** (`internal/api/router.go:20`, `user_router.go`). It creates a user and hashes the password but does **not** issue a token, and it predates `/auth/register`. It's marked for deletion — don't build new features on it.
- **Roles/permissions aren't modeled.** `Payload` has no role or scope claim, so every authenticated user has identical capabilities beyond resource ownership. If you add roles, put them in the DB and look them up per-request rather than trusting a `role` claim baked into a long-lived JWT (a role change wouldn't take effect until the token expires).
- **No rate limiting on `/auth/login`.** Nothing here currently slows down credential-stuffing attempts.
