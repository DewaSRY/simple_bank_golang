package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/DewaSRY/core-service/db/mock"
	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
)

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func mustDecimal(t *testing.T, s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

const (
	testUserID   = int64(1)
	testUsername = "dewa"
	testEmail    = "dewa@example.com"
)

func authHeaderFor(t *testing.T, server *Server, userID int64) string {
	accessToken, _, err := server.tokenMaker.CreateToken(userID, testUsername, testEmail, time.Minute)
	require.NoError(t, err)
	return "Bearer " + accessToken
}

func doAuthenticatedRequest(t *testing.T, server *Server, method, path string, body any, authHeader string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set(authorizationHeaderKey, authHeader)
	}

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	return recorder
}

func TestCreateAccount(t *testing.T) {
	testCases := []struct {
		name          string
		body          createAccountRequest
		buildStorer   func(t *testing.T, storer *mockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "creates a non-main account owned by the authenticated user",
			body: createAccountRequest{Name: "Savings", Description: "for later"},
			buildStorer: func(t *testing.T, storer *mockStorer) {
				storer.createAccountTxFunc = func(_ context.Context, arg store.CreateAccountTxParams) (db.Account, error) {
					require.Equal(t, testUserID, arg.UserID.Int64)
					require.Equal(t, "Savings", arg.Name)
					require.False(t, arg.IsMain)
					return db.Account{ID: 2, Balance: "0", Currency: "IDR", UserID: arg.UserID, Name: sql.NullString{String: arg.Name, Valid: true}}, nil
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "rejects a request missing the required name",
			body: createAccountRequest{Description: "no name"},
			buildStorer: func(t *testing.T, storer *mockStorer) {
				storer.createAccountTxFunc = func(_ context.Context, arg store.CreateAccountTxParams) (db.Account, error) {
					t.Fatal("CreateAccountTx should not be called")
					return db.Account{}, nil
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
			storer := &mockStorer{MockQuerier: q}
			tc.buildStorer(t, storer)

			server := newTestServerWithStorer(t, storer)
			recorder := doAuthenticatedRequest(t, server, http.MethodPost, "/api/v1/accounts", tc.body, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateAccount(t *testing.T) {
	existingAccount := db.GetAccountByIdRow{
		ID: 5, Balance: "100", Currency: "IDR",
		UserID: sql.NullInt64{Int64: testUserID, Valid: true},
		Name:   sql.NullString{String: "Old Name", Valid: true},
	}

	testCases := []struct {
		name          string
		body          updateAccountRequest
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "updates the account name and description",
			body: updateAccountRequest{Name: "New Name", Description: "new description"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(existingAccount, nil)
				q.EXPECT().UpdateAccount(gomock.Any(), db.UpdateAccountParams{
					ID:          existingAccount.ID,
					Name:        sql.NullString{String: "New Name", Valid: true},
					Description: sql.NullString{String: "new description", Valid: true},
				}).Return(db.UpdateAccountRow{ID: existingAccount.ID, Name: sql.NullString{String: "New Name", Valid: true}}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "keeps the existing name when only description is sent",
			body: updateAccountRequest{Description: "new description only"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(existingAccount, nil)
				q.EXPECT().UpdateAccount(gomock.Any(), db.UpdateAccountParams{
					ID:          existingAccount.ID,
					Name:        sql.NullString{String: existingAccount.Name.String, Valid: true},
					Description: sql.NullString{String: "new description only", Valid: true},
				}).Return(db.UpdateAccountRow{ID: existingAccount.ID}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			body: updateAccountRequest{Name: "New Name"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(db.GetAccountByIdRow{}, sql.ErrNoRows)
				q.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects updating an account owned by another user",
			body: updateAccountRequest{Name: "New Name"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(
					db.GetAccountByIdRow{ID: existingAccount.ID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
				q.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
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
			path := "/api/v1/accounts/" + itoa(existingAccount.ID)
			recorder := doAuthenticatedRequest(t, server, http.MethodPut, path, tc.body, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	const accountID = int64(5)

	ownedAccount := db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockQuerier)
		buildStorer   func(storer *mockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "deletes a zero-balance account with no sweep",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.deleteAccountTxFunc = func(_ context.Context, arg store.DeleteAccountTxParams) (store.DeleteAccountTxResult, error) {
					require.Equal(t, accountID, arg.AccountID)
					require.Equal(t, testUserID, arg.UserID)
					return store.DeleteAccountTxResult{Account: db.Account{ID: accountID}}, nil
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data deleteAccountResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Nil(t, resp.Data.BalanceSweptToID)
			},
		},
		{
			name: "deletes a positive-balance account and reports the sweep destination",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.deleteAccountTxFunc = func(_ context.Context, arg store.DeleteAccountTxParams) (store.DeleteAccountTxResult, error) {
					return store.DeleteAccountTxResult{
						Account:       db.Account{ID: accountID},
						SweepTransfer: &db.Transfer{ID: 1, FromAccountID: accountID, ToAccountID: 2},
					}, nil
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data deleteAccountResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.NotNil(t, resp.Data.BalanceSweptToID)
				require.Equal(t, int64(2), *resp.Data.BalanceSweptToID)
			},
		},
		{
			name: "rejects deleting the main account with a conflict",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.deleteAccountTxFunc = func(_ context.Context, arg store.DeleteAccountTxParams) (store.DeleteAccountTxResult, error) {
					return store.DeleteAccountTxResult{}, store.ErrCannotDeleteMainAccount
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.GetAccountByIdRow{}, sql.ErrNoRows)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects deleting an account owned by another user",
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			buildStorer: func(storer *mockStorer) {},
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

			storer := &mockStorer{MockQuerier: q}
			tc.buildStorer(storer)

			server := newTestServerWithStorer(t, storer)
			path := "/api/v1/accounts/" + itoa(accountID)
			recorder := doAuthenticatedRequest(t, server, http.MethodDelete, path, nil, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeposit(t *testing.T) {
	const accountID = int64(5)
	ownedAccount := db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		body          depositRequest
		buildStubs    func(q *mockdb.MockQuerier)
		buildStorer   func(storer *mockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "deposits a positive amount",
			body: depositRequest{Amount: mustDecimal(t, "100"), Description: "salary"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
			},
			buildStorer: func(storer *mockStorer) {
				storer.depositTxFunc = func(_ context.Context, arg store.DepositTxParams) (store.DepositTxResult, error) {
					require.Equal(t, accountID, arg.AccountID)
					require.Equal(t, "100.00", arg.Amount)
					return store.DepositTxResult{Account: db.Account{ID: accountID, Balance: "100.00"}}, nil
				}
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "rejects a non-positive amount without touching the store",
			body: depositRequest{Amount: mustDecimal(t, "0")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "rejects depositing into an account owned by another user",
			body: depositRequest{Amount: mustDecimal(t, "100")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.GetAccountByIdRow{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			body: depositRequest{Amount: mustDecimal(t, "100")},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.GetAccountByIdRow{}, sql.ErrNoRows)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
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
			path := "/api/v1/accounts/" + itoa(accountID) + "/deposit"
			recorder := doAuthenticatedRequest(t, server, http.MethodPost, path, tc.body, authHeaderFor(t, server, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}
