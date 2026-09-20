package store

import (
	"context"
	"fmt"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	sqlc "github.com/DewaSRY/core-service/internal/db/sqlc"

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

// execTxLockTimeout bounds how long a statement inside a store transaction
// will wait on a row lock. Correctness here otherwise relies entirely on the
// manual ascending-ID FOR UPDATE ordering (see transferTx/deleteAccountTx);
// this is a server-side backstop against an unrelated stalled transaction
// wedging the pool, not a substitute for that ordering.
const execTxLockTimeout = "5s"

func (store *_store) execTx(ctx context.Context, fn func(sqlc.Querier) error) error {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%s'", execTxLockTimeout)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %w, rb err: %v", err, rbErr)
		}
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
