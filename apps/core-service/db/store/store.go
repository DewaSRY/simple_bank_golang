package store

import (
	"context"
	"fmt"

	sqlc "github.com/DewaSRY/core-service/db/sqlc"

	"database/sql"
)

type Store struct {
	*sqlc.Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: sqlc.New(db),
	}
}

func (store *Store) execTx(ctx context.Context, fn func(sqlc.Querier) error) error {
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
