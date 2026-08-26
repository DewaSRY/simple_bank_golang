package store

import (
	"context"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"
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
	var err error

	// Create transfer record
	result.Transfer, err = q.CreateTransfer(ctx, sqlc.CreateTransferParams{
		FromAccountID: arg.FromAccountID,
		ToAccountID:   arg.ToAccountID,
		Amount:        arg.Amount,
	})

	if err != nil {
		return result, err
	}

	negativeAmount := "-" + arg.Amount
	// Create entries for sending
	_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
		AccountID: arg.FromAccountID,
		Type:      constant.ENTRY_TYPE_SEND,
		Amount:    negativeAmount,
	})

	if err != nil {
		return result, err
	}

	//record for sending transaction
	fromAccount, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
		ID:      arg.FromAccountID,
		Balance: negativeAmount,
	})

	if err != nil {
		return result, err
	}
	result.FromAccount = mapper.UpdateBalanceAccountToAccount(fromAccount)

	_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
		AccountID: arg.ToAccountID,
		Type:      constant.ENTRY_TYPE_RECEIVED,
		Amount:    arg.Amount,
	})

	if err != nil {
		return result, err
	}

	//record for receiving transaction
	toAccount, err := q.IncrementAccountBalance(ctx, sqlc.IncrementAccountBalanceParams{
		ID:      arg.ToAccountID,
		Balance: arg.Amount,
	})

	if err != nil {
		return result, err
	}

	result.ToAccount = mapper.UpdateBalanceAccountToAccount(toAccount)

	return result, nil
}
