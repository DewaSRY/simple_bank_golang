# Product Requirements Document (PRD)

# Simple Bank Application — Backend

## 1. Overview

The Simple Bank Application is a backend REST API that allows users to manage bank accounts and perform money transfers between accounts.

The main goal of this project is to practice and demonstrate backend engineering concepts, including:

- REST API design
- Database design and relationships
- Authentication and authorization
- Database transactions
- Data validation
- Error handling
- Concurrency and data consistency
- Unit and integration testing

The application will focus only on backend functionality. A frontend application is not required.

**Stack (as built):** Go + [Gin](https://github.com/gin-gonic/gin), PostgreSQL, [sqlc](https://sqlc.dev/) for type-safe queries (no ORM), `golang-migrate` for schema migrations, `golang-jwt/jwt/v5` for auth tokens, `bcrypt` for password hashing, `shopspring/decimal` for money math, `go.uber.org/mock` for repository mocking.

---

## 1.1 Implementation Status (updated 2026-08-29)

This section tracks what is actually built versus what the PRD originally scoped, so the document stays a source of truth rather than an aspirational spec. Legend: ✅ Done · ⚠️ Partial / diverges from spec · ❌ Not implemented.

| Feature | Status | Notes |
| --- | --- | --- |
| User registration | ⚠️ | Implemented as `POST /users`, not `/auth/register` |
| User login (JWT) | ✅ | `POST /auth/login`, bcrypt + JWT (HS256), access token only |
| Auth middleware / route protection | ✅ | Bearer token middleware on protected route group |
| Refresh tokens / logout / revocation | ❌ | Stateless access-token-only; no blacklist |
| Create account | ✅ | `POST /accounts`; owner forced from JWT, not request body |
| Get account by ID | ✅ | `GET /accounts/:id`, with ownership check (403 if not owner) |
| List accounts (current user) | ✅ | `GET /accounts`, paginated (`page`/`limit`), scoped to the caller |
| Update account | ❌ | Not implemented — out of scope for this pass |
| Money transfer | ✅ | `POST /transactions/transfer` (not `/transfers`); fully atomic |
| Transfer DB transaction | ✅ | `Store.execTx` pattern wrapping a `sql.Tx` |
| Row-level locking / deadlock avoidance | ✅ | `SELECT ... FOR UPDATE` with fixed ascending-ID lock order |
| Atomic balance updates | ✅ | `balance = balance + $delta` increment query, not read-modify-write |
| Get transfer by ID / transfer history | ✅ | `GET /transactions/:id` (visible to either party) and `GET /transactions` (paginated, either-side ownership) |
| Transfer `status` (PENDING/SUCCESS/FAILED) | ❌ | No `status` column; transfers are only ever fully committed or rolled back — deliberately deferred, not built this pass |
| Transfer `currency` column | ❌ | Not stored on `transfers` (only implied via account currency) — deliberately deferred |
| Account entries (double-entry bookkeeping) | ⚠️ | Created on every transfer (`SEND`/`RECEIVED`, not `DEBIT`/`CREDIT`); now readable via `GET /accounts/:id/entries`, but still no `transfer_id` FK |
| `GET /accounts/:id/entries` | ✅ | Implemented, paginated, ownership-checked |
| Normalized success/error response envelope | ✅ | Fully implemented, matches spec (`internal/api/response.go`, `apperror.go`) |
| Centralized error handling | ✅ | `errorHandlerMiddleware`; DB/internal errors never leak to clients |
| Request validation | ✅ | `go-playground/validator` via Gin binding, with field-name-aware error messages |
| Concurrency correctness tests | ✅ | Integration tests proving no lost updates and no deadlock under concurrent transfers |
| Unit tests for transfer business logic | ✅ | Table-driven, mocked-DB tests covering every failure branch |
| HTTP handler-level tests | ❌ | No `httptest` coverage for account/user/auth/transfer handlers — not addressed this pass, still the main testing gap |
| DB-level integrity constraints | ✅ (extra) | `CHECK (balance >= 0)`, `CHECK (amount > 0)` — not originally scoped, added as defense-in-depth |
| `owner_id` FK from accounts to users | ❌ | `accounts.owner` is a plain string matched against JWT username, not a foreign key |
| CORS / rate limiting / structured logging / request tracing | ❌ | None implemented; only Gin's default logger + recovery — deliberately deferred, not a correctness gap |
| Health check endpoint | ✅ (extra) | `GET /health`, not originally scoped |
| Pagination (`page`/`limit`/`meta.total`) | ✅ (new) | Implemented once, shared via a `paginationQuery` helper, reused by all three list endpoints |

**Summary:** The financial core — atomic transfers, deadlock-safe row locking, non-negative-balance guarantees, double-entry-style records, normalized responses, and JWT authentication with ownership checks — is solidly implemented and well-tested. As of this update, the previously-missing read/listing endpoints (list accounts, transfer history/detail, account entries) are now implemented and pagination is wired through the standard `{data, message, meta}` envelope. Remaining gaps: a few schema fields (`transfers.status`, `transfers.currency`, `entries.transfer_id`) left as deliberate scope decisions, route-naming divergence from the original spec (kept intentionally to avoid a breaking change), HTTP handler-level test coverage, an account update endpoint, and infra hardening (CORS, rate limiting, structured logging).

---

## 2. Goals

The backend should provide APIs that allow users to:

1. Create and manage bank accounts.
2. View account information and balances.
3. Transfer money between accounts.
4. View transaction history.
5. Authenticate users and protect private resources.

The system must ensure that financial operations are safe and that account balances remain consistent.

---

## 3. Core Features

### 3.1 Account Management — ✅ Mostly implemented

Users can create and manage bank accounts.

Each account contains (as built):

- Unique account ID
- Account owner (⚠️ stored as a plain string matched against the JWT username, not a foreign key to `users.id`)
- Current balance
- Currency
- Creation timestamp
- Last updated timestamp

#### Required Operations

- ✅ Create an account — owner is always derived from the authenticated JWT, ignoring any `owner` field a client sends, so a user can never create an account for someone else.
- ✅ Get account by ID — enforces ownership (403 if the requester doesn't own the account).
- ✅ Get all accounts — `GET /accounts`, paginated (`?page=&limit=`), scoped to the caller's own accounts via `ListAccountsByOwner`/`CountAccountsByOwner`.
- ❌ Update account information — not implemented (deliberately out of scope; no concrete use case identified yet).

### Business Rules

- ✅ Account balance must not be negative — enforced both in application logic and via a DB-level `CHECK (balance >= 0)` constraint.
- ✅ Account currency must be valid — restricted to `USD | EUR | GBP | IDR` via validation tag.
- ✅ Only authorized users should access their accounts — enforced in the `getAccount` handler.

### Improvement Opportunities

- ~~Add `GET /accounts` to list the authenticated user's accounts~~ — done.
- Add an account update endpoint if a real use case emerges (e.g. renaming, deactivating).
- Consider normalizing `accounts.owner` into a real `owner_id BIGINT REFERENCES users(id)` foreign key instead of a denormalized username string, to avoid drift if usernames ever change.

---

## 3.2 Money Transfer — ✅ Implemented, most mature part of the codebase

Users can transfer money from one account to another via `POST /transactions/transfer` (note: PRD originally specified `POST /transfers`; the implemented path diverges — see §4 for the reconciliation).

### Transfer Flow

1. User provides the source account.
2. User provides the destination account.
3. User provides the transfer amount.
4. The system validates the request.
5. The system checks the source account balance.
6. The amount is deducted from the source account.
7. The amount is added to the destination account.
8. The transfer is recorded in the database.

### Business Rules

- ✅ Transfer amount must be greater than zero — validated in both the handler and the transaction logic, plus a DB `CHECK (amount > 0)`.
- ✅ Source and destination accounts cannot be the same.
- ✅ The source account must have sufficient balance — checked against the row-locked balance to avoid TOCTOU races.
- ✅ Both accounts must exist.
- ✅ Currency of both accounts must match (not originally called out in this PRD section, but implemented and tested).
- ✅ The transfer must be executed atomically.
- ✅ If any operation fails, all changes are rolled back.
- ⚠️ Ownership is enforced only on the **source** account (the caller must own `from_account_id`); there is no restriction on transferring *to* an account you don't own, which is intentional (you can pay other people) but worth stating explicitly since the PRD doesn't.

### Important Requirement: Database Transaction — ✅ Implemented as `Store.execTx`

The following operations are executed inside a single database transaction (`db/store/store_transaction.go`):

```text
Start Transaction
    ↓
Validate Accounts
    ↓
Deduct Balance From Source Account
    ↓
Add Balance To Destination Account
    ↓
Create Transfer Record
    ↓
Create Account Entries
    ↓
Commit Transaction
```

If any step fails:

```text
Rollback Transaction
```

This ensures that money is never deducted without being added to the destination account.

**Concurrency safety (as built):** Both accounts are locked via `SELECT ... FOR UPDATE`, always in **ascending account-ID order** regardless of which is "from" or "to". This prevents the classic deadlock where two concurrent, opposite-direction transfers each hold one lock and wait on the other. Balance changes use an atomic `balance = balance + $delta` query rather than a read-then-write, closing the lost-update race even without the row lock. This is proven by two dedicated integration tests: `TestTransferTxConcurrent` (10 concurrent same-direction transfers, asserts exact final balances) and `TestTransferTxConcurrentReverse` (10 transfers in each direction simultaneously, the deadlock-prone shape, asserts all complete and balances net out correctly).

---

## 3.3 Transaction Records — ⚠️ Partially implemented

Every money movement is recorded, but the record is simpler than originally scoped: there is no status field, so a transfer is either fully committed or entirely absent (rolled back) — there is no `PENDING` intermediate state to observe.

### Transaction Information (as built)

A `transfers` row actually contains:

- Transaction ID
- Source account (`from_account_id`)
- Destination account (`to_account_id`)
- Transfer amount
- Creation timestamp

Not present, despite being scoped: ❌ `currency` column, ❌ `status` column (so the `PENDING` / `SUCCESS` / `FAILED` state machine described below does not exist — there is no code path that writes a `FAILED` transfer row; failed attempts simply never get inserted, per the atomic rollback above). These were evaluated and deliberately deferred (see below) rather than added in this pass.

Transfer history and detail are now readable: `GET /transactions` (paginated, scoped to any account the caller owns on either side) and `GET /transactions/:id` (visible if the caller owns the source **or** destination account).

### Improvement Opportunities

- ~~Add `GET /transactions/:id` and `GET /transactions` (history)~~ — done.
- If asynchronous or multi-step transfers are ever introduced (e.g. external payment rails), add a `status` column and the `PENDING`/`SUCCESS`/`FAILED` states as originally scoped. For the current fully-synchronous, single-DB-transaction transfer flow, the binary "row exists = success" model is arguably sufficient and simpler — this was evaluated during the read-endpoint work and deliberately deferred rather than added speculatively.
- Add a `currency` column to `transfers` for auditability (currently only inferable from the accounts involved, which is fragile if account currency could ever change) — same deferral reasoning as `status`.

---

## 3.4 Account Entries — ⚠️ Mostly implemented (read path added, `transfer_id` still missing)

An account entry represents a balance change for a specific account.

For example:

### Transfer of 100 USD from Account A to Account B

Account A (as built, entry type `SEND` rather than `DEBIT`):

```text
Amount: -100
Type: SEND
```

Account B (as built, entry type `RECEIVED` rather than `CREDIT`):

```text
Amount: +100
Type: RECEIVED
```

Each transfer generates entries for both accounts. A `DEPOSIT` entry type constant is also defined for a future non-transfer top-up flow, but nothing currently creates one.

⚠️ Gap remaining versus the original scope:

- `entries` still has no `transfer_id` foreign key back to the `transfers` table — the link between an entry and the transfer that created it is only inferable indirectly (matching `account_id` + timestamp + type), not queryable directly. This was in scope for consideration but deferred since it requires a migration; see §5.

✅ Resolved: `GET /accounts/:id/entries` is now implemented — paginated (`page`/`limit`), ownership-checked (403 if the account isn't the caller's), backed by `ListEntriesByAccount`/`CountEntriesByAccount`. This was the highest-value missing read endpoint and is the first way to actually see "why did my balance change" via the API.

### Improvement Opportunities

- Add `transfer_id BIGINT REFERENCES transfers(id)` to `entries` so entry history can be joined back to its originating transfer reliably (requires a new migration — not done this pass).
- ~~Implement `GET /accounts/:id/entries`~~ — done.
- Rename entry types to `DEBIT`/`CREDIT` to match this PRD, or update the PRD's terminology to `SEND`/`RECEIVED` to match the code — pick one vocabulary and make it consistent.

---

## 3.5 Authentication — ✅ Implemented (access-token only)

The system supports basic user authentication.

### Required Features

- ✅ User registration — `POST /users` (PRD originally specified `POST /auth/register`; see §4 for reconciliation).
- ✅ User login — `POST /auth/login`.
- ✅ Password hashing — `bcrypt`, default cost, unique salt per hash (tested in `pkg/utils/password_test.go`).
- ✅ JWT authentication — HS256, `golang-jwt/jwt/v5`, payload carries `id`/`username`/`email` plus standard registered claims (`iat`, `exp`); secret key must be ≥32 bytes.
- ❌ Refresh tokens, logout, or token revocation — not implemented (access-token-only, stateless, no blacklist). This was already listed under "Future Features" in this PRD, so it's a known and intentional gap, not a regression.

Auth token security has above-and-beyond test coverage: expired tokens, tampered signatures, wrong secret key, malformed tokens, and an `alg:none` algorithm-confusion attack are all explicitly tested in `internal/token/jwt_maker_test.go`.

Protected endpoints require a valid authentication token.

Example:

```text
Authorization: Bearer <token>
```

### Authorization Rule

Users should only be able to access accounts that they own.

For example:

```text
User A → Can access Account A
User A → Cannot access Account B owned by User B
```

---

## 4. API Requirements

Status reflects what's live today. Where the implemented path diverges from this PRD's original naming, both are noted.

### Authentication APIs

| Method | Endpoint      | Description                                                  | Status |
| ------ | ------------- | -------------------------------------------------------------- | ------ |
| `POST` | `/users`      | Register a new user (originally scoped as `/auth/register`)  | ✅ Implemented, path diverges |
| `POST` | `/auth/login` | Authenticate a user                                           | ✅ Implemented |

### Account APIs

| Method  | Endpoint        | Description                                        | Status |
| ------- | --------------- | ----------------------------------------------------- | ------ |
| `POST`  | `/accounts`     | Create a new account (owner is always the caller)  | ✅ Implemented |
| `GET`   | `/accounts/:id` | Get account information (ownership-checked)        | ✅ Implemented |
| `GET`   | `/accounts`     | Get user accounts, paginated (`?page=&limit=`)     | ✅ Implemented |
| `PATCH` | `/accounts/:id` | Update account information                         | ❌ Not implemented |

### Transfer APIs

| Method | Endpoint                 | Description                                                          | Status |
| ------ | ------------------------ | ----------------------------------------------------------------------- | ------ |
| `POST` | `/transactions/transfer` | Transfer money between accounts (originally scoped as `/transfers`) | ✅ Implemented, path diverges |
| `GET`  | `/transactions/:id`      | Get transfer details (visible to either the source or destination account owner) | ✅ Implemented |
| `GET`  | `/transactions`          | Get transfer history, paginated, scoped to accounts the caller owns on either side | ✅ Implemented |

### Account Entry APIs

| Method | Endpoint                | Description                                   | Status |
| ------ | ----------------------- | ------------------------------------------------ | ------ |
| `GET`  | `/accounts/:id/entries` | Get account transaction history, paginated, ownership-checked | ✅ Implemented |

### Other

| Method | Endpoint  | Description     | Status |
| ------ | --------- | ------------------ | ------ |
| `GET`  | `/health` | Liveness check  | ✅ Implemented (not originally scoped) |

### Pagination

`GET /accounts`, `GET /transactions`, and `GET /accounts/:id/entries` all accept `?page=` (default 1, min 1) and `?limit=` (default 10, min 1, max 100), and return `meta: {page, limit, total}` per §6/§5's pagination envelope. Implemented once as a shared `paginationQuery` binder (`internal/api/response.go`) reused by all three handlers.

### Improvement Opportunities

- Decide whether to rename `/users` → `/auth/register` and `/transactions/transfer` → `/transfers` to match this PRD and conventional REST naming, or formally adopt the implemented naming in this PRD instead (current decision: keep the implemented names, documented above, to avoid a breaking change).
- ~~Implement the stubbed read endpoints (`GET /accounts`, `GET /transactions/:id`, `GET /transactions`) plus `GET /accounts/:id/entries`~~ — done.
- Add `PATCH /accounts/:id` if/when a real update use case emerges.

---

## 5. Database Design

The tables below show the **originally scoped** design next to what's **actually in the migrations** (`db/migrations/`), so gaps are visible at a glance.

### Users — ✅ matches scope (plus an `email` column)

```text
users                     (as built)
├── id                    ├── id
├── username              ├── username (unique)
├── password_hash         ├── email (unique)
├── created_at            ├── hashed_password
└── updated_at            ├── created_at
                          └── updated_at
```

### Accounts — ⚠️ `owner_id` was never added; `owner` is a plain string

```text
accounts (scoped)         accounts (as built)
├── id                    ├── id
├── owner_id  ❌          ├── owner   ⚠️ VARCHAR, matched against JWT username — not a FK
├── balance               ├── balance   NUMERIC(20,2), CHECK (balance >= 0)
├── currency              ├── currency  VARCHAR(3)
├── created_at            ├── created_at
└── updated_at            └── updated_at
```

### Transfers — ⚠️ missing `currency` and `status`

```text
transfers (scoped)        transfers (as built)
├── id                    ├── id
├── from_account_id       ├── from_account_id  (FK → accounts.id)
├── to_account_id         ├── to_account_id    (FK → accounts.id)
├── amount                ├── amount   NUMERIC(20,2), CHECK (amount > 0)
├── currency  ❌          │
├── status    ❌          │
└── created_at            └── created_at
```

### Account Entries — ⚠️ missing `transfer_id`; type naming differs

```text
entries (scoped)          entries (as built)
├── id                    ├── id
├── account_id            ├── account_id  (FK → accounts.id)
├── amount                ├── amount
├── entry_type            ├── type        VARCHAR(50) — values are SEND / RECEIVED / DEPOSIT, not DEBIT / CREDIT
├── transfer_id  ❌        │
└── created_at            └── created_at
```

### Improvement Opportunities

- Add `transfers.currency` and `transfers.status` if the `PENDING`/`SUCCESS`/`FAILED` lifecycle from §3.3 is still wanted, or drop those columns from the scope permanently if the current fully-synchronous transfer model (row exists ⇒ success) is the intended long-term design.
- Add `entries.transfer_id` as a foreign key to `transfers.id` to make entry-to-transfer joins reliable instead of inferred.
- Consider replacing `accounts.owner` (string) with a proper `owner_id BIGINT REFERENCES users(id)` foreign key — the current design works only because account creation always derives the owner from the JWT, but it's one bug away from an orphaned or mismatched account if that invariant is ever violated.

---

## 6. Response Format — ✅ Fully implemented

All API responses follow a consistent response structure, implemented in `internal/api/response.go` and `internal/api/apperror.go`, and enforced globally via `errorHandlerMiddleware`. A `meta` field for pagination is already scaffolded in the success envelope (`succeedWithMeta`) but currently unused since no list endpoints exist yet (see §4).

### Success Response

```json id="c21bsz"
{
  "data": {},
  "message": "Success"
}
```

### Field Validation Error

```json id="k2tq0p"
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [
      {
        "field": "amount",
        "message": "Amount must be greater than zero"
      }
    ]
  }
}
```

### Business Error

```json id="jmrk09"
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "The source account does not have sufficient balance"
  }
}
```

---

## 7. Important Backend Requirements

### Data Validation — ✅ Implemented

All incoming requests are validated via `go-playground/validator` struct tags through Gin's binding, with a custom `RegisterTagNameFunc` so validation error `details[].field` values match the JSON field names the client sent rather than Go struct field names.

Covered:

- ✅ Required fields cannot be empty.
- ✅ Transfer amount must be greater than zero.
- ✅ Currency must be supported (`oneof=USD EUR GBP IDR`).
- ✅ Account IDs must be valid (`min=1` on URI/body bindings).
- ✅ Users cannot transfer money from accounts they do not own (checked in the transfer handler, not a validator tag).

---

### Error Handling — ✅ Implemented

The API returns structured errors and never exposes internal database errors, via `errorHandlerMiddleware` (global) + the `AppError` type. Any error not explicitly classified as an `AppError` is defaulted to a sanitized `500 INTERNAL_ERROR`.

```text
Database Error
        ↓
Application Error
        ↓
Normalized API Error Response
```

⚠️ Improvement opportunity: DB-error-to-`AppError` translation (e.g. `sql.ErrNoRows` → 404, Postgres `unique_violation`/`check_violation`/`foreign_key_violation` codes → 409/400) is currently duplicated per-handler (`user_router.go`, `transactions_router.go`) rather than centralized in one generic translator. Extracting a shared `pqErrorToAppError(err)` helper would reduce duplication as more handlers are added.

---

### Concurrency — ✅ Implemented and tested

The application safely handles multiple transfers happening at the same time.

```text
Account Balance: 100

Transfer A: -70
Transfer B: -50
```

The system prevents both transfers from succeeding if the total amount exceeds the available balance, via row-level `SELECT ... FOR UPDATE` locking (in a fixed ascending-account-ID order to avoid deadlocks) plus atomic `balance = balance + $delta` updates. Correctness under concurrent load is proven by `TestTransferTxConcurrent` and `TestTransferTxConcurrentReverse` in `db/store/store_transaction_integration_test.go`.

---

## 8. Non-Functional Requirements

### Security — ✅ Implemented

- ✅ Passwords must be hashed — bcrypt, unique salt per hash.
- ✅ Authentication tokens must be validated — JWT signature, expiry, and algorithm are all checked (including a test for `alg:none` attacks).
- ✅ Users cannot access other users' accounts — ownership checks on account read and transfer-source.
- ✅ Database errors should not be exposed directly — sanitized via `errorHandlerMiddleware`.
- ⚠️ Not yet covered: no CORS policy, no rate limiting, no CSRF considerations (less relevant for a pure JSON API, but worth a deliberate note), no refresh-token/logout/revocation story.

### Reliability — ✅ Implemented

- ✅ Financial transfers must be atomic — single DB transaction via `Store.execTx`.
- ✅ Failed transfers must not partially modify account balances — proven by unit tests asserting no downstream calls happen after any failure point, plus integration tests under real concurrency.
- ✅ Transaction history should remain consistent — within the current scope (no `status`/`currency` columns yet; see §3.3 and §5).

### Testing — ✅ Strong on transfer logic and auth; ❌ gap on HTTP handler tests

The application includes:

- ✅ Unit tests for business logic — `db/store/store_transaction_test.go`, table-driven, mocked `Querier` (gomock), covers every failure branch of `transferTx`.
- ✅ Repository/database tests — `db/sqlc/*_test.go`, run against a real Postgres instance wrapped in a rolled-back transaction per test.
- ❌ API handler tests — no `httptest`-based tests exist for `createAccount`, `getAccount`, `createUser`, `loginUser`, or `transactionTransfer`. Only the auth *middleware* has HTTP-level tests (`internal/api/auth_middleware_test.go`), not the business handlers themselves.
- ✅ Transfer transaction tests — both unit-level (mocked) and integration-level (real DB, concurrent) as described in §3.2.

Scenario coverage:

- ✅ Successful transfer.
- ✅ Insufficient balance.
- ✅ Invalid account.
- ⚠️ Unauthorized transfer — covered at the auth-middleware level (bad/missing/expired tokens), but not as an end-to-end handler test asserting a 403 when a real authenticated user tries to transfer from an account they don't own.
- ✅ Concurrent transfers — two dedicated integration tests, including the deadlock-prone opposite-direction case.
- ✅ Transaction rollback when an operation fails — every step of `transferTx` has a corresponding "fails here, nothing downstream runs" test case.

### Improvement Opportunities

- Add `httptest`-based handler tests for the account, user, and transfer endpoints — this is the single biggest testing gap, since all current coverage sits either below the handler (business logic, DB queries) or beside it (auth middleware).
- Add CORS middleware and a rate limiter (even a simple in-memory token bucket) before any real deployment — currently there is no protection against abuse beyond what Postgres/JWT provide.
- Add structured/leveled logging (e.g. `zerolog`/`zap`) in place of Gin's default logger for production observability.

---

## 9. Suggested Development Milestones

Status per item reflects the current codebase; this also roughly matches the actual commit order (foundation → store/mocks → account+transfer endpoints → concurrency hardening → normalized responses → auth, added last).

### Phase 1 — Foundation — ✅ Done

- ✅ Set up the Go project.
- ✅ Configure the database.
- ✅ Create database migrations.
- ✅ Implement normalized API responses.
- ✅ Implement error handling.

### Phase 2 — Account Management — ⚠️ Mostly done

- ✅ Create account API.
- ✅ Retrieve account API (`GET /accounts/:id`).
- ✅ Add account ownership validation.
- ✅ List accounts API (`GET /accounts`), paginated.
- ❌ Update account API — not started (no concrete use case yet).

### Phase 3 — Transfers — ✅ Done

- ✅ Implement money transfers.
- ✅ Implement database transactions (with deadlock-safe row locking).
- ✅ Create account entries.
- ✅ Add transfer history (`GET /transactions`, paginated) and detail (`GET /transactions/:id`).

### Phase 4 — Authentication — ✅ Done (for the scoped access-token model)

- ✅ Implement user registration (as `POST /users`).
- ✅ Implement login.
- ✅ Add JWT authentication middleware.
- ✅ Add authorization checks.

### Phase 5 — Testing and Improvements — ⚠️ Partially done

- ✅ Add unit tests (transfer business logic, JWT, password hashing).
- ✅ Add integration tests (sqlc queries, concurrent transfers).
- ✅ Test concurrent transfers.
- ❌ Add HTTP handler-level tests — not started.
- ❌ Improve logging (structured logging) — not started; only Gin's default logger is in place.

### Phase 6 — Remaining Read/List Endpoints — ✅ Done (2026-08-29)

- ✅ Implemented `GET /accounts` (list current user's accounts), paginated.
- ✅ Implemented `GET /transactions` and `GET /transactions/:id` (transfer history/detail), paginated, ownership-checked on either side of the transfer.
- ✅ Implemented `GET /accounts/:id/entries` (account statement), paginated, ownership-checked. The `entries.transfer_id` FK was evaluated and deliberately deferred (requires a migration; not needed for this endpoint to work correctly since it's scoped by `account_id`) — tracked in §3.4 and §5.

---

## 10. Definition of Done

The Simple Bank Application backend is considered complete when:

- ✅ Users can register and log in.
- ✅ Users can create and view their bank accounts, including listing all of their own accounts.
- ✅ Users can securely transfer money between accounts.
- ✅ Account balances remain consistent after transfers.
- ✅ Every balance change is recorded and now readable via `GET /accounts/:id/entries`.
- ✅ Transfers are executed using database transactions.
- ✅ Unauthorized access is prevented.
- ✅ API responses follow the normalized response format, including pagination `meta` for list endpoints.
- ⚠️ Core functionality is covered by automated tests — strong coverage on transfer logic, DB queries, and auth; **no HTTP handler-level tests** yet, including for the newly added list/detail endpoints.

**Current overall status: all originally-scoped CRUD/read functionality is implemented. The one remaining gap against this Definition of Done is HTTP handler-level test coverage — the harder concurrency/atomicity problem was already solved and well-tested.**

---

## 11. Future Features

The following features remain outside current scope. Items marked ⚠️ have some groundwork already in place.

- Multi-currency conversion.
- Scheduled transfers.
- Transfer cancellation.
- Email notifications.
- ⚠️ Account statements — `GET /accounts/:id/entries` is now implemented; the remaining piece is the `entries.transfer_id` FK (see §3.4/§5) for reliable joins back to the originating transfer.
- Refresh tokens.
- Role-based access control.
- Audit logs.
- Idempotency keys for transfer requests.
- Rate limiting.
- Two-factor authentication.
- Event-driven transaction processing.

---

# Summary

This project is designed as a simple but realistic backend banking system. The most important part is not the number of features, but the correctness of financial operations.

The core principle of the system is:

> **Money must never disappear, duplicate, or be partially transferred.**

Every transfer should be handled safely, consistently, and atomically.
