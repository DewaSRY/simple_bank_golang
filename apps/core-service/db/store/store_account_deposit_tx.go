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
func (store *_store) DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error) {
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
