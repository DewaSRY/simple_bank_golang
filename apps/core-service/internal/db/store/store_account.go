package store

import (
	"context"
	"database/sql"
	"fmt"

	sqlc "github.com/DewaSRY/core-service/internal/db/sqlc"
	constant "github.com/DewaSRY/core-service/internal/domain/constant"
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

func (store *_store) DeleteAccountTx(ctx context.Context, arg DeleteAccountTxParams) (DeleteAccountTxResult, error) {
	var result DeleteAccountTxResult

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = deleteAccountTx(ctx, q, arg)
		return err
	})

	return result, err
}

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

	result.Account = deleted
	result.SweepTransfer = sweepTransfer

	return result, nil
}
