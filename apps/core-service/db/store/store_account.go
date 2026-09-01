package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"
	"github.com/shopspring/decimal"
)

const closeAccountTransferDescription = "Account closure balance transfer"

// generateAccountNumber derives an account's human-facing number from its own
// DB id, which is immutable and never reused, so the result is guaranteed
// globally unique (a plain per-user or per-day counter is not).
func generateAccountNumber(id int64) string {
	date := strings.ToUpper(time.Now().Format("02-Jan-2006"))
	return fmt.Sprintf("ACT/%s/%03d", date, id)
}

type CreateAccountTxParams struct {
	UserID      sql.NullInt64
	Name        string
	Description string
	IsMain      bool
}

// CreateAccountTx creates an account and assigns it a globally-unique number
// derived from its own id, which is only known after the insert. The insert
// and the follow-up number update happen in one DB transaction so an account
// can never be left with a permanently-NULL number.
func (store *Store) CreateAccountTx(ctx context.Context, arg CreateAccountTxParams) (sqlc.Account, error) {
	var result sqlc.Account

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = createAccountTx(ctx, q, arg)
		return err
	})

	return result, err
}

func createAccountTx(ctx context.Context, q sqlc.Querier, arg CreateAccountTxParams) (sqlc.Account, error) {
	inserted, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
		Name:        sql.NullString{String: arg.Name, Valid: arg.Name != ""},
		Description: sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		Balance:     "0",
		Currency:    "IDR",
		UserID:      arg.UserID,
		IsMain:      arg.IsMain,
	})
	if err != nil {
		return sqlc.Account{}, err
	}

	updated, err := q.UpdateAccountNumber(ctx, sqlc.UpdateAccountNumberParams{
		ID:     inserted.ID,
		Number: sql.NullString{String: generateAccountNumber(inserted.ID), Valid: true},
	})
	if err != nil {
		return sqlc.Account{}, err
	}

	return mapper.UpdateAccountNumberRowToAccount(updated), nil
}

type DepositTxParams struct {
	AccountID   int64
	Amount      string
	Description string
}

type DepositTxResult struct {
	Account sqlc.Account
	Entry   sqlc.CreateEntriesRow
}

// DepositTx increases an account's balance and records a DEPOSIT entry.
func (store *Store) DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error) {
	var result DepositTxResult

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = depositTx(ctx, q, arg)
		return err
	})

	return result, err
}

// depositTx contains the business logic and depends only on the sqlc.Querier
// interface, so it can be unit tested with a gomock-generated mock without a
// real database.
func depositTx(ctx context.Context, q sqlc.Querier, arg DepositTxParams) (DepositTxResult, error) {
	var result DepositTxResult

	amount, err := decimal.NewFromString(arg.Amount)
	if err != nil {
		return result, fmt.Errorf("invalid deposit amount %q: %w", arg.Amount, err)
	}
	if !amount.IsPositive() {
		return result, ErrInvalidAmount
	}

	if _, err := q.GetAccountByIdForUpdate(ctx, arg.AccountID); err != nil {
		return result, err
	}

	result.Entry, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
		AccountID:   arg.AccountID,
		Type:        constant.ENTRY_TYPE_DEPOSIT,
		Amount:      arg.Amount,
		Description: sql.NullString{String: arg.Description, Valid: arg.Description != ""},
	})
	if err != nil {
		return result, err
	}

	updated, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
		ID:      arg.AccountID,
		Balance: arg.Amount,
	})
	if err != nil {
		return result, err
	}
	result.Account = mapper.UpdateBalanceAccountToAccount(updated)

	return result, nil
}

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
func (store *Store) DeleteAccountTx(ctx context.Context, arg DeleteAccountTxParams) (DeleteAccountTxResult, error) {
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
