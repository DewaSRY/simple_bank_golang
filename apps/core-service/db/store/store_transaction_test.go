package store

import (
	"context"
	"errors"
	"testing"
	"time"

	mockdb "github.com/DewaSRY/core-service/db/mock"
	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	constant "github.com/DewaSRY/core-service/domain/constant"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTransferTx(t *testing.T) {
	arg := sqlc.CreateTransferParams{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        "100",
	}

	now := time.Now()

	transfer := sqlc.Transfer{
		ID:            10,
		FromAccountID: arg.FromAccountID,
		ToAccountID:   arg.ToAccountID,
		Amount:        arg.Amount,
		CreatedAt:     now,
	}
	fromEntry := sqlc.Entry{ID: 1, AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100", CreatedAt: now}
	toEntry := sqlc.Entry{ID: 2, AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: "100", CreatedAt: now}
	fromAccount := sqlc.UpdateAccountBalanceRow{ID: arg.FromAccountID, Owner: "owner1", Balance: "900", Currency: "USD", CreatedAt: now}
	toAccount := sqlc.UpdateAccountBalanceRow{ID: arg.ToAccountID, Owner: "owner2", Balance: "1100", Currency: "USD", CreatedAt: now}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, result TransferTxResult, err error)
	}{
		{
			name: "Success",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100",
					}).Return(fromEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.FromAccountID, Balance: "-100",
					}).Return(fromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: arg.Amount,
					}).Return(toEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.ToAccountID, Balance: "100",
					}).Return(toAccount, nil),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.NoError(t, err)
				require.Equal(t, transfer, result.Transfer)
				require.Equal(t, "900", result.FromAccount.Balance)
				require.Equal(t, "1100", result.ToAccount.Balance)
			},
		},
		{
			name: "CreateTransferError",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(sqlc.Transfer{}, errors.New("create transfer error"))
				q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().UpdateAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create transfer error")
			},
		},
		{
			name: "CreateFromEntryError",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100",
					}).Return(sqlc.Entry{}, errors.New("create from entry error")),
				)
				q.EXPECT().UpdateAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create from entry error")
			},
		},
		{
			name: "UpdateFromAccountBalanceError",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100",
					}).Return(fromEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.FromAccountID, Balance: "-100",
					}).Return(sqlc.UpdateAccountBalanceRow{}, errors.New("update from balance error")),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "update from balance error")
			},
		},
		{
			name: "CreateToEntryError",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100",
					}).Return(fromEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.FromAccountID, Balance: "-100",
					}).Return(fromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: arg.Amount,
					}).Return(sqlc.Entry{}, errors.New("create to entry error")),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create to entry error")
			},
		},
		{
			name: "UpdateToAccountBalanceError",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: "-100",
					}).Return(fromEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.FromAccountID, Balance: "-100",
					}).Return(fromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sqlc.CreateEntriesParams{
						AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: arg.Amount,
					}).Return(toEntry, nil),
					q.EXPECT().UpdateAccountBalance(gomock.Any(), sqlc.UpdateAccountBalanceParams{
						ID: arg.ToAccountID, Balance: arg.Amount,
					}).Return(sqlc.UpdateAccountBalanceRow{}, errors.New("update to balance error")),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "update to balance error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			result, err := transferTx(context.Background(), q, arg)
			tc.checkResponse(t, result, err)
		})
	}
}
