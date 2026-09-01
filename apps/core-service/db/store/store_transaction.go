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

func (store *Store) TransferTx(ctx context.Context, arg sqlc.CreateTransferParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q sqlc.Querier) error {
		var err error
		result, err = transferTx(ctx, q, arg)
		return err
	})

	return result, err
}

// transferTx contains the business logic and depends only on the sqlc.Querier interface,
// so it can be unit tested with a gomock-generated mock without a real database.
func transferTx(ctx context.Context, q sqlc.Querier, arg sqlc.CreateTransferParams) (TransferTxResult, error) {
	var result TransferTxResult

	if arg.FromAccountID == arg.ToAccountID {
		return result, ErrSameAccount
	}

	amount, err := decimal.NewFromString(arg.Amount)
	if err != nil {
		return result, fmt.Errorf("invalid transfer amount %q: %w", arg.Amount, err)
	}
	if !amount.IsPositive() {
		return result, ErrInvalidAmount
	}

	// Accounts are locked in a fixed ascending-ID order (rather than
	// from-then-to) so two concurrent transfers between the same pair of
	// accounts always acquire their row locks in the same order and can't
	// deadlock against each other.
	firstID, secondID := arg.FromAccountID, arg.ToAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	firstAccount, err := q.GetAccountByIdForUpdate(ctx, firstID)
	if err != nil {
		return result, err
	}
	secondAccount, err := q.GetAccountByIdForUpdate(ctx, secondID)
	if err != nil {
		return result, err
	}

	fromAccount, toAccount := firstAccount, secondAccount
	if arg.FromAccountID != firstID {
		fromAccount, toAccount = secondAccount, firstAccount
	}

	if fromAccount.Currency != toAccount.Currency {
		return result, ErrCurrencyMismatch
	}

	fromBalance, err := decimal.NewFromString(fromAccount.Balance)
	if err != nil {
		return result, fmt.Errorf("invalid account balance %q: %w", fromAccount.Balance, err)
	}
	if fromBalance.LessThan(amount) {
		return result, ErrInsufficientFunds
	}

	// Create transfer record
	result.Transfer, err = q.CreateTransfer(ctx, sqlc.CreateTransferParams{
		FromAccountID: arg.FromAccountID,
		ToAccountID:   arg.ToAccountID,
		Amount:        arg.Amount,
		Description:   arg.Description,
	})

	if err != nil {
		return result, err
	}

	transferID := sql.NullInt64{Int64: result.Transfer.ID, Valid: true}

	negativeAmount := "-" + arg.Amount
	// Create entries for sending
	_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
		AccountID:   arg.FromAccountID,
		Type:        constant.ENTRY_TYPE_SEND,
		Amount:      negativeAmount,
		Description: arg.Description,
		TransferID:  transferID,
	})

	if err != nil {
		return result, err
	}

	//record for sending transaction
	debitedFromAccount, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
		ID:      arg.FromAccountID,
		Balance: negativeAmount,
	})

	if err != nil {
		return result, err
	}
	result.FromAccount = mapper.UpdateBalanceAccountToAccount(debitedFromAccount)

	_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
		AccountID:   arg.ToAccountID,
		Type:        constant.ENTRY_TYPE_RECEIVED,
		Amount:      arg.Amount,
		Description: arg.Description,
		TransferID:  transferID,
	})

	if err != nil {
		return result, err
	}

	//record for receiving transaction
	creditedToAccount, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
		ID:      arg.ToAccountID,
		Balance: arg.Amount,
	})

	if err != nil {
		return result, err
	}

	result.ToAccount = mapper.UpdateBalanceAccountToAccount(creditedToAccount)

	return result, nil
}
