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

### 3.1 Account Management

Users can create and manage bank accounts.

Each account should contain:

- Unique account ID
- Account owner
- Current balance
- Currency
- Creation timestamp
- Last updated timestamp

#### Required Operations

- Create an account
- Get account by ID
- Get all accounts
- Update account information if necessary

### Business Rules

- Account balance must not be negative.
- Account currency must be valid.
- Only authorized users should access their accounts.

---

## 3.2 Money Transfer

Users can transfer money from one account to another.

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

- Transfer amount must be greater than zero.
- Source and destination accounts cannot be the same.
- The source account must have sufficient balance.
- Both accounts must exist.
- The transfer must be executed atomically.
- If any operation fails, all changes must be rolled back.

### Important Requirement: Database Transaction

The following operations must be executed inside a single database transaction:

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

---

## 3.3 Transaction Records

Every money movement should be recorded.

The system should maintain a history of transactions for auditing purposes.

### Transaction Information

A transaction record should contain:

- Transaction ID
- Source account
- Destination account
- Transfer amount
- Currency
- Transaction status
- Creation timestamp

Possible transaction statuses:

- `PENDING`
- `SUCCESS`
- `FAILED`

---

## 3.4 Account Entries

An account entry represents a balance change for a specific account.

For example:

### Transfer of 100 USD from Account A to Account B

Account A:

```text
Amount: -100
Type: DEBIT
```

Account B:

```text
Amount: +100
Type: CREDIT
```

Each transfer should generate entries for both accounts.

This allows the system to maintain a clear history of all balance changes.

---

## 3.5 Authentication

The system should support basic user authentication.

### Required Features

- User registration
- User login
- Password hashing
- JWT authentication

Protected endpoints should require a valid authentication token.

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

### Authentication APIs

| Method | Endpoint         | Description         |
| ------ | ---------------- | ------------------- |
| `POST` | `/auth/register` | Register a new user |
| `POST` | `/auth/login`    | Authenticate a user |

### Account APIs

| Method | Endpoint        | Description             |
| ------ | --------------- | ----------------------- |
| `POST` | `/accounts`     | Create a new account    |
| `GET`  | `/accounts/:id` | Get account information |
| `GET`  | `/accounts`     | Get user accounts       |

### Transfer APIs

| Method | Endpoint         | Description                     |
| ------ | ---------------- | ------------------------------- |
| `POST` | `/transfers`     | Transfer money between accounts |
| `GET`  | `/transfers/:id` | Get transfer details            |
| `GET`  | `/transfers`     | Get transfer history            |

### Account Entry APIs

| Method | Endpoint                | Description                     |
| ------ | ----------------------- | ------------------------------- |
| `GET`  | `/accounts/:id/entries` | Get account transaction history |

---

## 5. Database Design

The application should contain the following core tables.

### Users

```text
users
├── id
├── username
├── password_hash
├── created_at
└── updated_at
```

### Accounts

```text
accounts
├── id
├── owner_id
├── balance
├── currency
├── created_at
└── updated_at
```

### Transfers

```text
transfers
├── id
├── from_account_id
├── to_account_id
├── amount
├── currency
├── status
└── created_at
```

### Account Entries

```text
entries
├── id
├── account_id
├── amount
├── entry_type
├── transfer_id
└── created_at
```

---

## 6. Response Format

All API responses should follow a consistent response structure.

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

### Data Validation

Validate all incoming requests.

Examples:

- Required fields cannot be empty.
- Transfer amount must be greater than zero.
- Currency must be supported.
- Account IDs must be valid.
- Users cannot transfer money from accounts they do not own.

---

### Error Handling

The API should return structured errors and should not expose internal database errors.

Example:

```text
Database Error
        ↓
Application Error
        ↓
Normalized API Error Response
```

---

### Concurrency

The application must safely handle multiple transfers happening at the same time.

Example:

```text
Account Balance: 100

Transfer A: -70
Transfer B: -50
```

The system must prevent both transfers from succeeding if the total amount exceeds the available balance.

The implementation should use appropriate database transactions and locking or atomic update queries.

---

## 8. Non-Functional Requirements

### Security

- Passwords must be hashed.
- Authentication tokens must be validated.
- Users cannot access other users' accounts.
- Database errors should not be exposed directly.

### Reliability

- Financial transfers must be atomic.
- Failed transfers must not partially modify account balances.
- Transaction history should remain consistent.

### Testing

The application should include:

- Unit tests for business logic.
- Repository/database tests.
- API handler tests.
- Transfer transaction tests.

Important scenarios to test:

- Successful transfer.
- Insufficient balance.
- Invalid account.
- Unauthorized transfer.
- Concurrent transfers.
- Transaction rollback when an operation fails.

---

## 9. Suggested Development Milestones

### Phase 1 — Foundation

- Set up the Go project.
- Configure the database.
- Create database migrations.
- Implement normalized API responses.
- Implement error handling.

### Phase 2 — Account Management

- Create account APIs.
- Retrieve account APIs.
- Add account ownership validation.

### Phase 3 — Transfers

- Implement money transfers.
- Implement database transactions.
- Create account entries.
- Add transfer history.

### Phase 4 — Authentication

- Implement user registration.
- Implement login.
- Add JWT authentication middleware.
- Add authorization checks.

### Phase 5 — Testing and Improvements

- Add unit tests.
- Add integration tests.
- Test concurrent transfers.
- Improve logging and error handling.

---

## 10. Definition of Done

The Simple Bank Application backend is considered complete when:

- Users can register and log in.
- Users can create and view their bank accounts.
- Users can securely transfer money between accounts.
- Account balances remain consistent after transfers.
- Every balance change is recorded.
- Transfers are executed using database transactions.
- Unauthorized access is prevented.
- API responses follow the normalized response format.
- Core functionality is covered by automated tests.

---

## 11. Future Features

The following features are outside the initial scope but can be added later:

- Multi-currency conversion.
- Scheduled transfers.
- Transfer cancellation.
- Email notifications.
- Account statements.
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
