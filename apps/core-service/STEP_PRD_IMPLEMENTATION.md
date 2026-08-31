# Step Implementation

## Register user

### Currently

User can register with email

### Change

After success fully register, user will atomaticly get create the account with name main

## Create Account

### Currently

user create account with currency

### Change

User will create account with name and description, then system will automaticly genrate the number

By Default the system will store IDR as currency

## Update account

### Change

User able to update account name and description.

## Delete account

### Change

User able to delete the account, except for the main account, then the remain balacen will autoamticly transfer to main account

The account will mark as deleted by the system

For now, the deleted account cannot be recovered.

The balance transfer and the delete (soft delete) must happen in a single DB transaction, if either step fails everything rolls back.

## Deposit

### Currently

Deposit does not exist as a feature. Only transfer between two accounts is implemented (`internal/api/transactions_router.go`), and the `DEPOSIT` entry type constant is already defined but unused (`domain/constant/entries.go`).

### Change

User can deposit money into any account they own.

User selects the account, enters an amount (required, must be greater than zero) and an optional description. Currency is fixed to IDR, same as the account.

On success the system increases the account balance and writes a single entry with type `DEPOSIT`. A failed deposit must not change the account balance.

## Transfer

### Currently

User transfers by supplying `from_account_id` and `to_account_id` directly (`createTransactionTransferRequest`). The transfer already runs atomically in `TransferTx`/`transferTx` (`db/store/store_transaction.go`): it locks both accounts in ascending-ID order, checks the accounts aren't the same, checks currency match, checks sufficient balance, then creates the transfer record plus a `SEND`/`RECEIVED` entry pair and updates both balances.

### Change

Instead of picking the destination by raw account ID, the user selects it from a list of recently used destination accounts, or searches for it by account number.

Since every account will be IDR going forward, the existing currency-mismatch check stays only as a safety net and should no longer be reachable in normal use.

Everything else about the atomic transfer flow already matches the PRD and does not need to change.

## Transaction History & Filtering

### Currently

`GET /transactions` (`listTransactions`) lists raw `Transfer` rows for the authenticated user, with no entry direction and no deposits included. There is no per-account view and no date filtering.

### Change

Transaction history should combine deposit entries and transfer entries (`SEND`/`RECEIVED`) into one list per account, each row labeled as `Deposit`, `Transfer In`, or `Transfer Out`, showing date, amount, currency, description, and the counterparty account when applicable.

History must be filterable by month, defaulting to the current month, and must only ever return transactions belonging to the authenticated user's own accounts.

## Security & Authorization

### Currently

Existing account and transfer endpoints already check that the resource belongs to the authenticated user (`account.Owner != authPayload.Username` in `internal/api/account_router.go`, and the from/to account owner check in `internal/api/transactions_router.go`) before allowing access.

### Change

Every new endpoint (deposit, update account, delete account, transaction history) must apply the same ownership check pattern. A user must never be able to view, modify, or delete another user's account or transactions by changing an ID in the request.
