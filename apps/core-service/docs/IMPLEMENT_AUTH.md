# JWT Authentication Implementation Guide — Go + Gin

This guide gives you a practical, production-oriented sequence for implementing JWT authentication in a Go project using **Gin**, **sqlc**, and **PostgreSQL**.

## 1. Define the Authentication Flow

Before writing code, establish the flow:

```text
Client
  │
  │ POST /auth/login
  ▼
Gin Handler
  │
  ▼
Service
  │
  ├── Find user
  ├── Verify password
  └── Generate JWT
  │
  ▼
Client
  │
  │ Authorization: Bearer <token>
  ▼
JWT Middleware
  │
  ├── Extract token
  ├── Verify signature
  ├── Validate expiration
  └── Extract user information
  │
  ▼
Protected Handler
```

---

# 2. Decide What Goes Inside the JWT

Keep the JWT payload small.

A reasonable access-token payload is:

```json
{
  "sub": "<user_id>",
  "email": "dewa",
  "iat": 1756400000,
  "exp": 1756403600
}
```

Recommended claims:

| Claim   | Purpose                            |
| ------- | ---------------------------------- |
| `sub`   | User/account ID                    |
| `iat`   | Token creation time                |
| `exp`   | Token expiration                   |
| `email` | Optional authorization information |

Avoid putting sensitive information such as:

- Password
- Password hash
- Bank balance
- Personal secrets
- Large user objects

JWT payloads are **encoded, not encrypted**.

---

# 3. Choose Your JWT Strategy

For a typical REST API, use:

- **Access token** — short-lived
- **Refresh token** — longer-lived

For example:

```text
Access Token
Expiration: 15 minutes

Refresh Token
Expiration: 7 days
```

Initially, you can implement only the access token. Add refresh tokens afterward.

---

# 4. Add the JWT Dependency

A common choice is:

```bash
go get github.com/golang-jwt/jwt/v5
```

Then verify your `go.mod` contains the dependency.

---

# 5. Create Your Environment Configuration

Add JWT configuration to your environment:

```env
JWT_SECRET=your-very-long-random-secret
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=168h
```

For production, don't commit the secret to Git.

Your configuration might eventually look like:

```text
PORT
DB_URI
SECRET_KEY
JWT_ACCESS_TOKEN_DURATION
JWT_REFRESH_TOKEN_DURATION
```

---

# 6. Create a JWT Utility / Service

Create something similar to:

```text
token/
├── maker.go
└── payload.go
```

Define a token maker interface:

```go
type Maker interface {
    CreateToken(username string, duration time.Duration) (string, error)
    VerifyToken(token string) (*Payload, error)
}
```

This abstraction is useful because your application doesn't need to know how JWT itself works.

---

# 7. Create the JWT Payload

For example:

```go
type Payload struct {
    Email string `json:"email"`
    ID       int64  `json:"id"`
    jwt.RegisteredClaims
}
```

When creating the token:

```go
claims := Payload{
    Email: email,
    ID:       userID,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
    },
}
```

---

# 8. Generate the JWT

Create a signing method and sign the token using your secret.

Conceptually:

```text
Payload
   +
Secret Key
   ↓
JWT
```

For example:

```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

signedToken, err := token.SignedString(secretKey)
```

For a first implementation, **HS256** is perfectly reasonable.

---

# 9. Verify the JWT

When a request arrives with:

```http
Authorization: Bearer eyJhbGciOiJIUzI1Ni...
```

Your JWT service should verify:

1. Token exists
2. Token format is valid
3. Signing algorithm is expected
4. Signature is valid
5. Token isn't expired
6. Claims can be parsed

The important principle is:

```text
Never trust the claims before verifying the signature.
```

---

# 10. Create the Login Endpoint

Create:

```http
POST /auth/login
```

Request:

```json
{
  "username": "dewa",
  "password": "password123"
}
```

The flow should be:

```text
Handler
   ↓
Validate request
   ↓
Service
   ↓
Get user from PostgreSQL
   ↓
Compare password hash
   ↓
Generate access token
   ↓
Return token
```

Response:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

---

# 11. Hash Passwords Properly

Never store plaintext passwords.

Use a password hashing algorithm such as:

```text
bcrypt
```

or preferably a modern password-hashing strategy such as:

```text
Argon2id
```

The database should contain something like:

```text
username: dewa
password_hash: $2a$10$...
```

During login:

```text
Incoming password
        ↓
Password verification
        ↓
Stored password hash
```

Do not decrypt passwords because properly hashed passwords are not supposed to be decrypted.

---

# 12. Create the Gin JWT Middleware

Create something like:

```text
api/
├── middleware/
│   └── auth.go
```

The middleware should:

```text
Request
   ↓
Authorization header
   ↓
Extract Bearer token
   ↓
Verify JWT
   ↓
Extract user ID
   ↓
Store user ID in Gin context
   ↓
Next handler
```

Example:

```go
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        authorizationHeader := ctx.GetHeader("Authorization")

        // Validate Bearer token
        // Verify JWT
        // Put user information into context

        ctx.Next()
    }
}
```

---

# 13. Put Authentication Information in Gin Context

After successfully validating the token:

```go
ctx.Set("user_id", payload.ID)
ctx.Set("email", payload.Email)
```

Then your handler can retrieve it:

```go
userID, exists := ctx.Get("user_id")
```

This allows protected endpoints to know who is making the request.

---

# 14. Protect Your Routes

Public routes:

```go
router.POST("/auth/login", server.login)
router.POST("/users", server.createUser)
```

Protected routes:

```go
authorized := router.Group("/")
authorized.Use(authMiddleware(tokenMaker))

authorized.GET("/accounts", server.getAccounts)
authorized.GET("/accounts/:id", server.getAccount)
authorized.POST("/transfers", server.createTransfer)
```

The result is:

```text
POST /auth/login
       │
       └── Public

GET /accounts
       │
       └── JWT required

POST /transfers
       │
       └── JWT required
```

---

# 15. Check Authorization Separately From Authentication

Authentication answers:

> Who are you?

Authorization answers:

> Are you allowed to do this?

For example:

```text
JWT
 ↓
User ID = 123
 ↓
Is account 456 owned by user 123?
 ↓
YES → continue
NO  → 403 Forbidden
```

This is extremely important for applications involving accounts and money.

Don't assume:

```text
Valid JWT = permission to access everything
```

---

# 16. Implement Resource Ownership Checks

For an account endpoint:

```http
GET /accounts/10
```

Don't only check whether the JWT is valid.

Also verify:

```sql
SELECT *
FROM accounts
WHERE id = $1
  AND owner = $2;
```

or whatever ownership model your application uses.

This prevents:

```text
User A
  ↓
Valid JWT
  ↓
Requests User B's account
  ↓
❌ Access denied
```

---

# 17. Return Appropriate HTTP Errors

Use consistent responses.

### Missing token

```http
401 Unauthorized
```

### Invalid token

```http
401 Unauthorized
```

### Expired token

```http
401 Unauthorized
```

### Valid token but insufficient permission

```http
403 Forbidden
```

A useful distinction:

```text
401 = You are not authenticated.

403 = You are authenticated, but you aren't allowed to do this.
```

---

# 18. Add Tests for the Token Maker

Test at least:

### Create token

```text
✓ token is generated
✓ token contains correct user ID
✓ token contains correct username
✓ expiration is correct
```

### Verify token

```text
✓ valid token passes
✓ expired token fails
✓ malformed token fails
✓ modified token fails
✓ wrong secret fails
```

This is one of the most important parts to test.

---

# 19. Add Middleware Tests

Test:

```text
✓ valid Authorization header
✓ missing Authorization header
✓ malformed Authorization header
✓ missing Bearer prefix
✓ invalid token
✓ expired token
✓ valid token reaches handler
```

For example:

```text
Authorization: Bearer <valid-token>
```

should reach the handler.

But:

```text
Authorization: invalid
```

should return:

```http
401
```

---

# 20. Add Login Integration Tests

Test the complete flow:

```text
Create user
    ↓
Hash password
    ↓
Insert user
    ↓
POST /auth/login
    ↓
Verify password
    ↓
Generate JWT
    ↓
Return 200
```

Also test:

```text
wrong username
wrong password
missing username
missing password
```

---

# 21. Test a Protected Endpoint

Create a test like:

```text
1. Create user
2. Login
3. Get access token
4. Send request to protected endpoint
5. Add Authorization header
6. Expect HTTP 200
```

Then test:

```text
1. Send request without JWT
2. Expect HTTP 401
```

And:

```text
1. Send request with invalid JWT
2. Expect HTTP 401
```

This gives you confidence that the complete authentication system works.

---

# 22. Recommended Implementation Order

Implement it in this order:

- [ ] Add `golang-jwt/jwt/v5`
- [ ] Add JWT configuration to environment
- [ ] Create `Payload`
- [ ] Create `Maker` interface
- [ ] Implement JWT token creation
- [ ] Implement JWT verification
- [ ] Write token unit tests
- [ ] Implement password hashing
- [ ] Implement login service
- [ ] Implement `POST /auth/login`
- [ ] Write login tests
- [ ] Implement Gin JWT middleware
- [ ] Extract `Authorization: Bearer <token>`
- [ ] Validate JWT in middleware
- [ ] Store authenticated user information in Gin context
- [ ] Protect routes
- [ ] Add authorization/ownership checks
- [ ] Write middleware tests
- [ ] Write protected-endpoint integration tests

---

# 23. Final Architecture

Once implemented, your application should roughly follow:

```text
                    ┌──────────────┐
                    │    Client    │
                    └──────┬───────┘
                           │
                    POST /auth/login
                           │
                           ▼
                    ┌──────────────┐
                    │ Gin Handler  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │    Service   │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
        PostgreSQL                Password Verify
              │                         │
              └────────────┬────────────┘
                           ▼
                     JWT Maker
                           │
                           ▼
                    Access Token
                           │
                           ▼
                         Client
                           │
                           │ Authorization:
                           │ Bearer <token>
                           ▼
                  ┌─────────────────┐
                  │  JWT Middleware │
                  └────────┬────────┘
                           │
                    Verify signature
                    Check expiration
                    Extract user ID
                           │
                           ▼
                  ┌─────────────────┐
                  │ Protected Route │
                  └────────┬────────┘
                           │
                           ▼
                  Authorization check
                           │
                           ▼
                       Database
```

## 24. Important Security Rules

Keep these rules throughout the implementation:

1. **Never store plaintext passwords.**
2. **Never put passwords or secrets inside JWT claims.**
3. **Use a strong random JWT secret.**
4. **Keep access tokens short-lived.**
5. **Always verify the JWT signature before trusting claims.**
6. **Check resource ownership separately from JWT validity.**
7. **Return `401` for authentication failures and `403` for authorization failures.**
8. **Don't log JWTs or passwords.**
9. **Use HTTPS in production.**
10. **Write tests for both authentication and authorization.**

For your Go project, I would implement the JWT system as a **small independent `token` package + Gin middleware**, rather than putting JWT logic directly inside your handlers. This keeps the architecture clean and makes the token system very easy to unit-test and mock.
