package store

import (
	"context"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"
)

func (store *Store) TransferTx(ctx context.Context, arg sqlc.CreateTransferParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *sqlc.Queries) error {
		var err error

		// Create transfer record
		result.Transfer, err = q.CreateTransfer(ctx, sqlc.CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})

		if err != nil {
			return err
		}

		// Create entries for sending
		_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
			AccountID: arg.FromAccountID,
			Type:      constant.ENTRY_TYPE_SEND,
			Amount:    -arg.Amount,
		})

		if err != nil {
			return err
		}

		//record for sending transaction
		fromAccount, err := q.UpdateAccountBalance(ctx, sqlc.UpdateAccountBalanceParams{
			ID:      arg.FromAccountID,
			Balance: -arg.Amount,
		})

		if err != nil {
			return err
		}
		result.FromAccount = mapper.UpdateBalanceAccountToAccount(fromAccount)

		_, err = q.CreateEntries(ctx, sqlc.CreateEntriesParams{
			AccountID: arg.ToAccountID,
			Type:      constant.ENTRY_TYPE_RECEIVED,
			Amount:    arg.Amount,
		})

		if err != nil {
			return err
		}

		//record for receiving transaction
		toAccount, err := q.UpdateAccountBalance(ctx, sqlc.UpdateAccountBalanceParams{
			ID:      arg.ToAccountID,
			Balance: arg.Amount,
		})

		if err != nil {
			return err
		}

		result.ToAccount = mapper.UpdateBalanceAccountToAccount(toAccount)

		return nil
	})

	return result, err
}
