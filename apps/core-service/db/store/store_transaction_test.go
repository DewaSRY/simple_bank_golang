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

// TestTransferTx checks that transferTx runs its steps in the right order
// (create transfer -> debit sender -> credit receiver) and that it stops
// and bubbles up the error as soon as any one of those steps fails.
func TestTransferTx(t *testing.T) {
	// A simple scenario reused by every case below: owner1 sends 100 to owner2.
	const transferAmount = "100"
	const negatedTransferAmount = "-100"

	arg := sqlc.CreateTransferParams{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        transferAmount,
	}

	now := time.Now()

	transfer := sqlc.Transfer{
		ID:            10,
		FromAccountID: arg.FromAccountID,
		ToAccountID:   arg.ToAccountID,
		Amount:        arg.Amount,
		CreatedAt:     now,
	}

	sentEntry := sqlc.Entry{ID: 1, AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: negatedTransferAmount, CreatedAt: now}
	receivedEntry := sqlc.Entry{ID: 2, AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: transferAmount, CreatedAt: now}

	debitedFromAccount := sqlc.IncrementAccountBalanceRow{ID: arg.FromAccountID, Owner: "owner1", Balance: "900", Currency: "USD", CreatedAt: now}
	creditedToAccount := sqlc.IncrementAccountBalanceRow{ID: arg.ToAccountID, Owner: "owner2", Balance: "1100", Currency: "USD", CreatedAt: now}

	// Small helpers so each test case can describe "what happens next" without
	// re-typing the same params struct every time.
	sendEntryParams := func() sqlc.CreateEntriesParams {
		return sqlc.CreateEntriesParams{AccountID: arg.FromAccountID, Type: constant.ENTRY_TYPE_SEND, Amount: negatedTransferAmount}
	}
	receiveEntryParams := func() sqlc.CreateEntriesParams {
		return sqlc.CreateEntriesParams{AccountID: arg.ToAccountID, Type: constant.ENTRY_TYPE_RECEIVED, Amount: transferAmount}
	}
	debitFromAccountParams := func() sqlc.IncrementAccountBalanceParams {
		return sqlc.IncrementAccountBalanceParams{ID: arg.FromAccountID, Balance: negatedTransferAmount}
	}
	creditToAccountParams := func() sqlc.IncrementAccountBalanceParams {
		return sqlc.IncrementAccountBalanceParams{ID: arg.ToAccountID, Balance: transferAmount}
	}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, result TransferTxResult, err error)
	}{
		{
			name: "transfers money and returns the updated balances",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sendEntryParams()).Return(sentEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), debitFromAccountParams()).Return(debitedFromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), receiveEntryParams()).Return(receivedEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), creditToAccountParams()).Return(creditedToAccount, nil),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.NoError(t, err)
				require.Equal(t, transfer, result.Transfer)
				require.Equal(t, debitedFromAccount.Balance, result.FromAccount.Balance)
				require.Equal(t, creditedToAccount.Balance, result.ToAccount.Balance)
			},
		},
		{
			name: "stops immediately when the transfer record can't be created",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(sqlc.Transfer{}, errors.New("create transfer error"))
				// Nothing after CreateTransfer should ever run.
				q.EXPECT().CreateEntries(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().IncrementAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create transfer error")
			},
		},
		{
			name: "stops when the sender's entry can't be recorded",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sendEntryParams()).Return(sqlc.Entry{}, errors.New("create from entry error")),
				)
				// Balances should never be touched if the entry failed.
				q.EXPECT().IncrementAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create from entry error")
			},
		},
		{
			name: "stops when the sender's balance can't be debited",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sendEntryParams()).Return(sentEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), debitFromAccountParams()).Return(sqlc.IncrementAccountBalanceRow{}, errors.New("update from balance error")),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "update from balance error")
			},
		},
		{
			name: "stops when the receiver's entry can't be recorded",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sendEntryParams()).Return(sentEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), debitFromAccountParams()).Return(debitedFromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), receiveEntryParams()).Return(sqlc.Entry{}, errors.New("create to entry error")),
				)
			},
			checkResponse: func(t *testing.T, result TransferTxResult, err error) {
				require.ErrorContains(t, err, "create to entry error")
			},
		},
		{
			name: "stops when the receiver's balance can't be credited",
			buildStubs: func(q *mockdb.MockQuerier) {
				gomock.InOrder(
					q.EXPECT().CreateTransfer(gomock.Any(), arg).Return(transfer, nil),
					q.EXPECT().CreateEntries(gomock.Any(), sendEntryParams()).Return(sentEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), debitFromAccountParams()).Return(debitedFromAccount, nil),
					q.EXPECT().CreateEntries(gomock.Any(), receiveEntryParams()).Return(receivedEntry, nil),
					q.EXPECT().IncrementAccountBalance(gomock.Any(), creditToAccountParams()).Return(sqlc.IncrementAccountBalanceRow{}, errors.New("update to balance error")),
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
