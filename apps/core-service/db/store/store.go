package store

import (
	"context"
	"fmt"

	db "github.com/DewaSRY/core-service/db/sqlc"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"

	"database/sql"
)

type Storer interface {
	db.Querier
	TransferTx(ctx context.Context, arg db.CreateTransferParams) (TransferTxResult, error)
	CreateAccountTx(ctx context.Context, arg CreateAccountTxParams) (db.Account, error)
	DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error)
	DeleteAccountTx(ctx context.Context, arg DeleteAccountTxParams) (DeleteAccountTxResult, error)
}

type _store struct {
	*sqlc.Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Storer {
	return &_store{
		db:      db,
		Queries: sqlc.New(db),
	}
}

func (store *_store) execTx(ctx context.Context, fn func(sqlc.Querier) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := sqlc.New(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %w, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

type TransferTxResult struct {
	Transfer    sqlc.Transfer `json:"transfer"`
	FromAccount sqlc.Account  `json:"from_account"`
	ToAccount   sqlc.Account  `json:"to_account"`
}
