package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/DewaSRY/core-service/db/mock"
	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
)

func TestTransactionTransfer(t *testing.T) {
	const fromAccountID = int64(1)
	const toAccountID = int64(2)
	fromAccount := db.GetAccountByIdRow{ID: fromAccountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}, Currency: "IDR"}

	testCases := []struct {
		name          string
		body          createTransactionTransferRequest
		buildStubs    func(q *mockdb.MockQuerier)
		buildStorer   func(storer *mockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "transfers between two accounts",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "100"), Description: "rent"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(fromAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.transferTxFunc = func(_ context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
					require.Equal(t, fromAccountID, arg.FromAccountID)
					require.Equal(t, toAccountID, arg.ToAccountID)
					require.Equal(t, "100.00", arg.Amount)
					require.Equal(t, "rent", arg.Description.String)
					return store.TransferTxResult{Transfer: db.Transfer{ID: 1}}, nil
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "rejects a non-positive amount without touching the store",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "0")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "rejects transferring from an account owned by another user",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "100")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(
					db.GetAccountByIdRow{ID: fromAccountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "maps insufficient funds to a conflict",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "100")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(fromAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.transferTxFunc = func(_ context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
					return store.TransferTxResult{}, store.ErrInsufficientFunds
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "maps same-account transfers to a bad request",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "100")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(fromAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.transferTxFunc = func(_ context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
					return store.TransferTxResult{}, store.ErrSameAccount
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			storer := &mockStorer{MockQuerier: q}
			tc.buildStorer(storer)

			server := newTestServerWithStorer(t, storer)
			recorder := doAuthenticatedRequest(t, server, http.MethodPost, "/api/v1/transactions/transfer", tc.body, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListRecentTransferDestinations(t *testing.T) {
	const accountID = int64(1)
	ownedAccount := db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "lists recent destinations for an owned account",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListRecentTransferDestinations(gomock.Any(), db.ListRecentTransferDestinationsParams{FromAccountID: accountID, LimitCount: 5}).Return(
					[]db.ListRecentTransferDestinationsRow{
						{ID: 2, Name: sql.NullString{String: "Savings", Valid: true}, Number: sql.NullString{String: "ACT/1", Valid: true}},
					}, nil,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []publicAccountResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, int64(2), resp.Data[0].ID)
			},
		},
		{
			name: "rejects listing destinations for an account owned by another user",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			server := newTestServerWithMockStore(t, q)
			path := "/api/v1/accounts/" + itoa(accountID) + "/recent-destinations"
			recorder := doAuthenticatedRequest(t, server, http.MethodGet, path, nil, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountTransactionHistory(t *testing.T) {
	const accountID = int64(1)
	ownedAccount := db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}
	now := time.Now().UTC()

	testCases := []struct {
		name          string
		path          string
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "defaults to the current month and labels each row",
			path: "/api/v1/accounts/" + itoa(accountID) + "/transactions",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListAccountTransactionHistory(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.ListAccountTransactionHistoryParams) ([]db.ListAccountTransactionHistoryRow, error) {
						require.Equal(t, accountID, arg.AccountID)
						require.Equal(t, now.Year(), arg.PeriodStart.Year())
						require.Equal(t, now.Month(), arg.PeriodStart.Month())
						return []db.ListAccountTransactionHistoryRow{
							{ID: 1, Type: "DEPOSIT", Amount: "100", CreatedAt: now},
							{ID: 2, Type: "SEND", Amount: "-50", CreatedAt: now,
								CounterpartyAccountID:     sql.NullInt64{Int64: 9, Valid: true},
								CounterpartyAccountName:   sql.NullString{String: "Savings", Valid: true},
								CounterpartyAccountNumber: sql.NullString{String: "ACT/9", Valid: true}},
							{ID: 3, Type: "RECEIVED", Amount: "25", CreatedAt: now,
								CounterpartyAccountID: sql.NullInt64{Int64: 8, Valid: true}},
						}, nil
					},
				)
				q.EXPECT().CountAccountTransactionHistory(gomock.Any(), gomock.Any()).Return(int64(3), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []transactionHistoryItem `json:"data"`
					Meta Meta                     `json:"meta"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 3)
				require.Equal(t, "Deposit", resp.Data[0].Label)
				require.Nil(t, resp.Data[0].Counterparty)
				require.Equal(t, "Transfer Out", resp.Data[1].Label)
				require.NotNil(t, resp.Data[1].Counterparty)
				require.Equal(t, "Savings", resp.Data[1].Counterparty.Name)
				require.Equal(t, "Transfer In", resp.Data[2].Label)
				require.Equal(t, int64(3), resp.Meta.Total)
			},
		},
		{
			name: "rejects viewing history for an account owned by another user",
			path: "/api/v1/accounts/" + itoa(accountID) + "/transactions",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "honors an explicit month and year",
			path: "/api/v1/accounts/" + itoa(accountID) + "/transactions?month=1&year=2025",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListAccountTransactionHistory(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.ListAccountTransactionHistoryParams) ([]db.ListAccountTransactionHistoryRow, error) {
						require.Equal(t, 2025, arg.PeriodStart.Year())
						require.Equal(t, time.January, arg.PeriodStart.Month())
						require.Equal(t, time.February, arg.PeriodEnd.Month())
						return nil, nil
					},
				)
				q.EXPECT().CountAccountTransactionHistory(gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			server := newTestServerWithMockStore(t, q)
			recorder := doAuthenticatedRequest(t, server, http.MethodGet, tc.path, nil, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}
