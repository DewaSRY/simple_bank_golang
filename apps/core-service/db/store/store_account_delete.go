package store

import (
	"context"
	"database/sql"
	"fmt"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"
	"github.com/shopspring/decimal"
)

const closeAccountTransferDescription = "Account closure balance transfer"

type DeleteAccountTxParams struct {
	AccountID int64
	UserID    int64
}

type DeleteAccountTxResult struct {
	Account       sqlc.Account
	SweepTransfer *sqlc.Transfer
}

// DeleteAccountTx soft-deletes an account, sweeping any remaining balance to
// the user's main account first (recorded as an ordinary transfer) so no
// money is ever silently lost.
func (store *_store) DeleteAccountTx(ctx context.Context, arg DeleteAccountTxParams) (DeleteAccountTxResult, error) {
	var result DeleteAccountTxResult

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = deleteAccountTx(ctx, q, arg)
		return err
	})

	return result, err
}

// deleteAccountTx contains the business logic and depends only on the
// sqlc.Querier interface, so it can be unit tested with a gomock-generated
// mock without a real database.
func deleteAccountTx(ctx context.Context, q sqlc.Querier, arg DeleteAccountTxParams) (DeleteAccountTxResult, error) {
	var result DeleteAccountTxResult

	account, err := q.GetAccountById(ctx, arg.AccountID)
	if err != nil {
		return result, err
	}
	if account.IsMain {
		return result, ErrCannotDeleteMainAccount
	}

	mainAccount, err := q.GetMainAccountByUserId(ctx, sql.NullInt64{Int64: arg.UserID, Valid: true})
	if err != nil {
		return result, err
	}

	// Lock both accounts in a fixed ascending-ID order, same as transferTx,
	// so a concurrent transfer touching this same pair of accounts can never
	// deadlock against this delete. The balance is read from the locked row
	// (not the unlocked account fetched above) so a concurrent change to the
	// balance between that read and acquiring the lock can't cause a stale
	// amount to be swept.
	firstID, secondID := arg.AccountID, mainAccount.ID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	firstLocked, err := q.GetAccountByIdForUpdate(ctx, firstID)
	if err != nil {
		return result, err
	}
	secondLocked, err := q.GetAccountByIdForUpdate(ctx, secondID)
	if err != nil {
		return result, err
	}

	lockedAccount := firstLocked
	if arg.AccountID != firstID {
		lockedAccount = secondLocked
	}

	balance, err := decimal.NewFromString(lockedAccount.Balance)
	if err != nil {
		return result, fmt.Errorf("invalid account balance %q: %w", lockedAccount.Balance, err)
	}

	var sweepTransfer *sqlc.Transfer
	if balance.IsPositive() {
		transfer, err := q.CreateTransfer(ctx, sqlc.CreateTransferParams{
			FromAccountID: arg.AccountID,
			ToAccountID:   mainAccount.ID,
			Amount:        lockedAccount.Balance,
			Description:   sql.NullString{String: closeAccountTransferDescription, Valid: true},
		})
		if err != nil {
			return result, err
		}

		negativeAmount := "-" + lockedAccount.Balance
		if _, err := q.CreateEntries(ctx, sqlc.CreateEntriesParams{
			AccountID:   arg.AccountID,
			Type:        constant.ENTRY_TYPE_SEND,
			Amount:      negativeAmount,
			Description: sql.NullString{String: closeAccountTransferDescription, Valid: true},
			TransferID:  sql.NullInt64{Int64: transfer.ID, Valid: true},
		}); err != nil {
			return result, err
		}
		if _, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
			ID:      arg.AccountID,
			Balance: negativeAmount,
		}); err != nil {
			return result, err
		}

		if _, err := q.CreateEntries(ctx, sqlc.CreateEntriesParams{
			AccountID:   mainAccount.ID,
			Type:        constant.ENTRY_TYPE_RECEIVED,
			Amount:      lockedAccount.Balance,
			Description: sql.NullString{String: closeAccountTransferDescription, Valid: true},
			TransferID:  sql.NullInt64{Int64: transfer.ID, Valid: true},
		}); err != nil {
			return result, err
		}
		if _, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
			ID:      mainAccount.ID,
			Balance: lockedAccount.Balance,
		}); err != nil {
			return result, err
		}

		sweepTransfer = &transfer
	}

	deleted, err := q.SoftDeleteAccount(ctx, arg.AccountID)
	if err != nil {
		return result, err
	}

	result.Account = mapper.SoftDeleteAccountRowToAccount(deleted)
	result.SweepTransfer = sweepTransfer

	return result, nil
}
