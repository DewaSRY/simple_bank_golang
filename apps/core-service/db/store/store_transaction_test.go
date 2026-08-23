package store

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return NewStore(db), mock
}

var transferColumns = []string{"id", "from_account_id", "to_account_id", "amount", "created_at"}
var entryColumns = []string{"id", "account_id", "type", "amount", "created_at"}
var accountColumns = []string{"id", "owner", "balance", "currency", "created_at"}

func TestTransferTx(t *testing.T) {
	arg := sqlc.CreateTransferParams{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        100,
	}
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()

		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))

		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(1, arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount, now))

		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.FromAccountID, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.FromAccountID, "owner1", 900, "USD", now))

		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(2, arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount, now))

		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.ToAccountID, "owner2", 1100, "USD", now))

		mock.ExpectCommit()

		result, err := store.TransferTx(context.Background(), arg)

		require.NoError(t, err)
		require.Equal(t, arg.FromAccountID, result.Transfer.FromAccountID)
		require.Equal(t, arg.ToAccountID, result.Transfer.ToAccountID)
		require.Equal(t, int64(900), result.FromAccount.Balance)
		require.Equal(t, int64(1100), result.ToAccount.Balance)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("BeginTxError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "begin error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateTransferError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnError(errors.New("create transfer error"))
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "create transfer error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateFromEntryError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnError(errors.New("create from entry error"))
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "create from entry error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateFromAccountBalanceError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(1, arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.FromAccountID, -arg.Amount).
			WillReturnError(errors.New("update from balance error"))
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "update from balance error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateToEntryError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(1, arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.FromAccountID, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.FromAccountID, "owner1", 900, "USD", now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount).
			WillReturnError(errors.New("create to entry error"))
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "create to entry error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateToAccountBalanceError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(1, arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.FromAccountID, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.FromAccountID, "owner1", 900, "USD", now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(2, arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.ToAccountID, arg.Amount).
			WillReturnError(errors.New("update to balance error"))
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "update to balance error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RollbackError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnError(errors.New("create transfer error"))
		mock.ExpectRollback().WillReturnError(errors.New("rollback error"))

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "create transfer error")
		require.ErrorContains(t, err, "rollback error")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CommitError", func(t *testing.T) {
		store, mock := newTestStore(t)

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO transfers").
			WithArgs(arg.FromAccountID, arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(transferColumns).
				AddRow(1, arg.FromAccountID, arg.ToAccountID, arg.Amount, now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(1, arg.FromAccountID, constant.ENTRY_TYPE_SEND, -arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.FromAccountID, -arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.FromAccountID, "owner1", 900, "USD", now))
		mock.ExpectQuery("INSERT INTO entries").
			WithArgs(arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount).
			WillReturnRows(sqlmock.NewRows(entryColumns).
				AddRow(2, arg.ToAccountID, constant.ENTRY_TYPE_RECEIVED, arg.Amount, now))
		mock.ExpectQuery("UPDATE accounts").
			WithArgs(arg.ToAccountID, arg.Amount).
			WillReturnRows(sqlmock.NewRows(accountColumns).
				AddRow(arg.ToAccountID, "owner2", 1100, "USD", now))
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		_, err := store.TransferTx(context.Background(), arg)

		require.ErrorContains(t, err, "commit error")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
