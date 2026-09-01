package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	mockdb "github.com/DewaSRY/core-service/db/mock"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateAccountTx(t *testing.T) {
	now := time.Now()

	arg := CreateAccountTxParams{
		UserID:      sql.NullInt64{Int64: 1, Valid: true},
		Name:        "Savings",
		Description: "for later",
		IsMain:      false,
	}

	insertedAccount := sqlc.CreateAccountRow{ID: 42, Balance: "0", Currency: "IDR", UserID: arg.UserID, CreatedAt: now}
	updatedAccount := sqlc.UpdateAccountNumberRow{
		ID: 42, Balance: "0", Currency: "IDR", UserID: arg.UserID,
		Number: sql.NullString{String: "ACT/whatever/042", Valid: true},
		Name:   sql.NullString{String: "Savings", Valid: true}, CreatedAt: now,
	}

	t.Run("creates an account and assigns it a number derived from its own id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		gomock.InOrder(
			q.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p sqlc.CreateAccountParams) (sqlc.CreateAccountRow, error) {
					require.False(t, p.Number.Valid, "number must be NULL on first insert")
					require.Equal(t, arg.Name, p.Name.String)
					require.Equal(t, arg.Description, p.Description.String)
					require.Equal(t, arg.IsMain, p.IsMain)
					require.Equal(t, arg.UserID, p.UserID)
					require.Equal(t, "IDR", p.Currency)
					return insertedAccount, nil
				},
			),
			q.EXPECT().UpdateAccountNumber(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p sqlc.UpdateAccountNumberParams) (sqlc.UpdateAccountNumberRow, error) {
					require.Equal(t, insertedAccount.ID, p.ID)
					require.True(t, p.Number.Valid)
					require.NotEmpty(t, p.Number.String)
					return updatedAccount, nil
				},
			),
		)

		result, err := createAccountTx(context.Background(), q, arg)
		require.NoError(t, err)
		require.Equal(t, updatedAccount.ID, result.ID)
		require.Equal(t, updatedAccount.Number.String, result.Number.String)
	})

	t.Run("stops before assigning a number when the insert fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		q.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Return(sqlc.CreateAccountRow{}, errors.New("insert error"))
		q.EXPECT().UpdateAccountNumber(gomock.Any(), gomock.Any()).Times(0)

		_, err := createAccountTx(context.Background(), q, arg)
		require.ErrorContains(t, err, "insert error")
	})

	t.Run("bubbles up a failure to assign the number", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		q.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Return(insertedAccount, nil)
		q.EXPECT().UpdateAccountNumber(gomock.Any(), gomock.Any()).Return(sqlc.UpdateAccountNumberRow{}, errors.New("update number error"))

		_, err := createAccountTx(context.Background(), q, arg)
		require.ErrorContains(t, err, "update number error")
	})
}

func TestDepositTx(t *testing.T) {
	now := time.Now()
	const accountID = int64(1)

	lockedAccount := sqlc.GetAccountByIdForUpdateRow{ID: accountID, Balance: "500", Currency: "IDR", CreatedAt: now}
	createdEntry := sqlc.CreateEntriesRow{ID: 1, AccountID: accountID, Type: constant.ENTRY_TYPE_DEPOSIT, Amount: "100", CreatedAt: now}
	creditedAccount := sqlc.IncrementAccountBalanceRow{ID: accountID, Balance: "600", Currency: "IDR", CreatedAt: now}

	t.Run("deposits money and returns the updated balance", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		gomock.InOrder(
			q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), accountID).Return(lockedAccount, nil),
			q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
				AccountID:   accountID,
				Type:        constant.ENTRY_TYPE_DEPOSIT,
				Amount:      "100",
				Description: sql.NullString{String: "salary", Valid: true},
			}).Return(createdEntry, nil),
			q.EXPECT().IncrementAccountBalance(gomock.Any(), sqlc.IncrementAccountBalanceParams{ID: accountID, Balance: "100"}).Return(creditedAccount, nil),
		)

		result, err := depositTx(context.Background(), q, DepositTxParams{AccountID: accountID, Amount: "100", Description: "salary"})
		require.NoError(t, err)
		require.Equal(t, createdEntry, result.Entry)
		require.Equal(t, creditedAccount.Balance, result.Account.Balance)
	})

	t.Run("rejects a non-positive amount without touching the db", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)
		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), gomock.Any()).Times(0)

		_, err := depositTx(context.Background(), q, DepositTxParams{AccountID: accountID, Amount: "0"})
		require.ErrorIs(t, err, ErrInvalidAmount)
	})

	t.Run("rejects an unparseable amount without touching the db", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)
		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), gomock.Any()).Times(0)

		_, err := depositTx(context.Background(), q, DepositTxParams{AccountID: accountID, Amount: "not-a-number"})
		require.Error(t, err)
	})

	t.Run("stops when the account can't be found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)
		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), accountID).Return(sqlc.GetAccountByIdForUpdateRow{}, sql.ErrNoRows)
		q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)

		_, err := depositTx(context.Background(), q, DepositTxParams{AccountID: accountID, Amount: "100"})
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("does not touch the balance when the entry can't be recorded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), accountID).Return(lockedAccount, nil)
		q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Return(sqlc.CreateEntriesRow{}, errors.New("create entry error"))
		q.EXPECT().IncrementAccountBalance(gomock.Any(), gomock.Any()).Times(0)

		_, err := depositTx(context.Background(), q, DepositTxParams{AccountID: accountID, Amount: "100"})
		require.ErrorContains(t, err, "create entry error")
	})
}

func TestDeleteAccountTx(t *testing.T) {
	now := time.Now()
	const userID = int64(7)
	const closingAccountID = int64(5)
	const mainAccountID = int64(2)

	closingAccount := sqlc.GetAccountByIdRow{ID: closingAccountID, Balance: "300", Currency: "IDR", IsMain: false, UserID: sql.NullInt64{Int64: userID, Valid: true}, CreatedAt: now}
	mainAccountRow := sqlc.GetMainAccountByUserIdRow{ID: mainAccountID, Balance: "1000", Currency: "IDR", IsMain: true, UserID: sql.NullInt64{Int64: userID, Valid: true}, CreatedAt: now}
	zeroBalanceClosingAccount := sqlc.GetAccountByIdRow{ID: closingAccountID, Balance: "0", Currency: "IDR", IsMain: false, UserID: sql.NullInt64{Int64: userID, Valid: true}, CreatedAt: now}

	lockAscending := func(q *mockdb.MockQuerier, closingLocked, mainLocked interface{}) {
		// mainAccountID (2) < closingAccountID (5), so main is locked first.
		gomock.InOrder(
			q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), mainAccountID).Return(mainLocked, nil),
			q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), closingAccountID).Return(closingLocked, nil),
		)
	}

	t.Run("rejects deleting the main account without mutating anything", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		q.EXPECT().GetAccountById(gomock.Any(), closingAccountID).Return(
			sqlc.GetAccountByIdRow{ID: closingAccountID, IsMain: true, UserID: sql.NullInt64{Int64: userID, Valid: true}}, nil,
		)
		q.EXPECT().GetMainAccountByUserId(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().IncrementAccountBalance(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().SoftDeleteAccount(gomock.Any(), gomock.Any()).Times(0)

		_, err := deleteAccountTx(context.Background(), q, DeleteAccountTxParams{AccountID: closingAccountID, UserID: userID})
		require.ErrorIs(t, err, ErrCannotDeleteMainAccount)
	})

	t.Run("soft-deletes a zero-balance account without sweeping anything", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		zeroLocked := sqlc.GetAccountByIdForUpdateRow{ID: closingAccountID, Balance: "0", Currency: "IDR", CreatedAt: now}
		mainLocked := sqlc.GetAccountByIdForUpdateRow{ID: mainAccountID, Balance: "1000", Currency: "IDR", CreatedAt: now}
		deletedRow := sqlc.SoftDeleteAccountRow{ID: closingAccountID, Balance: "0", Currency: "IDR", CreatedAt: now}

		q.EXPECT().GetAccountById(gomock.Any(), closingAccountID).Return(zeroBalanceClosingAccount, nil)
		q.EXPECT().GetMainAccountByUserId(gomock.Any(), gomock.Any()).Return(mainAccountRow, nil)
		lockAscending(q, zeroLocked, mainLocked)
		q.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().IncrementAccountBalance(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().SoftDeleteAccount(gomock.Any(), closingAccountID).Return(deletedRow, nil)

		result, err := deleteAccountTx(context.Background(), q, DeleteAccountTxParams{AccountID: closingAccountID, UserID: userID})
		require.NoError(t, err)
		require.Nil(t, result.SweepTransfer)
	})

	t.Run("sweeps a positive balance to the main account before soft-deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		closingLocked := sqlc.GetAccountByIdForUpdateRow{ID: closingAccountID, Balance: "300", Currency: "IDR", CreatedAt: now}
		mainLocked := sqlc.GetAccountByIdForUpdateRow{ID: mainAccountID, Balance: "1000", Currency: "IDR", CreatedAt: now}
		sweepTransfer := sqlc.Transfer{ID: 99, FromAccountID: closingAccountID, ToAccountID: mainAccountID, Amount: "300", CreatedAt: now}
		deletedRow := sqlc.SoftDeleteAccountRow{ID: closingAccountID, Balance: "0", Currency: "IDR", CreatedAt: now}

		q.EXPECT().GetAccountById(gomock.Any(), closingAccountID).Return(closingAccount, nil)
		q.EXPECT().GetMainAccountByUserId(gomock.Any(), gomock.Any()).Return(mainAccountRow, nil)
		lockAscending(q, closingLocked, mainLocked)

		gomock.InOrder(
			q.EXPECT().CreateTransfer(gomock.Any(), sqlc.CreateTransferParams{
				FromAccountID: closingAccountID, ToAccountID: mainAccountID, Amount: "300",
				Description: sql.NullString{String: closeAccountTransferDescription, Valid: true},
			}).Return(sweepTransfer, nil),
			q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
				AccountID: closingAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-300",
				Description: sql.NullString{String: closeAccountTransferDescription, Valid: true},
				TransferID:  sql.NullInt64{Int64: sweepTransfer.ID, Valid: true},
			}).Return(sqlc.CreateEntriesRow{}, nil),
			q.EXPECT().IncrementAccountBalance(gomock.Any(), sqlc.IncrementAccountBalanceParams{ID: closingAccountID, Balance: "-300"}).Return(sqlc.IncrementAccountBalanceRow{}, nil),
			q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
				AccountID: mainAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: "300",
				Description: sql.NullString{String: closeAccountTransferDescription, Valid: true},
				TransferID:  sql.NullInt64{Int64: sweepTransfer.ID, Valid: true},
			}).Return(sqlc.CreateEntriesRow{}, nil),
			q.EXPECT().IncrementAccountBalance(gomock.Any(), sqlc.IncrementAccountBalanceParams{ID: mainAccountID, Balance: "300"}).Return(sqlc.IncrementAccountBalanceRow{}, nil),
			q.EXPECT().SoftDeleteAccount(gomock.Any(), closingAccountID).Return(deletedRow, nil),
		)

		result, err := deleteAccountTx(context.Background(), q, DeleteAccountTxParams{AccountID: closingAccountID, UserID: userID})
		require.NoError(t, err)
		require.NotNil(t, result.SweepTransfer)
		require.Equal(t, sweepTransfer.ID, result.SweepTransfer.ID)
	})

	t.Run("stops when the main account can't be found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		q.EXPECT().GetAccountById(gomock.Any(), closingAccountID).Return(closingAccount, nil)
		q.EXPECT().GetMainAccountByUserId(gomock.Any(), gomock.Any()).Return(sqlc.GetMainAccountByUserIdRow{}, sql.ErrNoRows)
		q.EXPECT().GetAccountByIdForUpdate(gomock.Any(), gomock.Any()).Times(0)

		_, err := deleteAccountTx(context.Background(), q, DeleteAccountTxParams{AccountID: closingAccountID, UserID: userID})
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("stops when the sweep transfer can't be created", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		q := mockdb.NewMockQuerier(ctrl)

		closingLocked := sqlc.GetAccountByIdForUpdateRow{ID: closingAccountID, Balance: "300", Currency: "IDR", CreatedAt: now}
		mainLocked := sqlc.GetAccountByIdForUpdateRow{ID: mainAccountID, Balance: "1000", Currency: "IDR", CreatedAt: now}

		q.EXPECT().GetAccountById(gomock.Any(), closingAccountID).Return(closingAccount, nil)
		q.EXPECT().GetMainAccountByUserId(gomock.Any(), gomock.Any()).Return(mainAccountRow, nil)
		lockAscending(q, closingLocked, mainLocked)
		q.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(sqlc.Transfer{}, errors.New("create transfer error"))
		q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)
		q.EXPECT().SoftDeleteAccount(gomock.Any(), gomock.Any()).Times(0)

		_, err := deleteAccountTx(context.Background(), q, DeleteAccountTxParams{AccountID: closingAccountID, UserID: userID})
		require.ErrorContains(t, err, "create transfer error")
	})
}
