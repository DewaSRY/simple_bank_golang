package transfer

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/DewaSRY/core-service/internal/api/core"
	mockdb "github.com/DewaSRY/core-service/internal/db/mock"
	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/token"
)

const (
	testSecretKey = "12345678901234567890123456789012"
	testUserID    = int64(1)
	testUsername  = "dewa"
	testEmail     = "dewa@example.com"
)

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func mustDecimal(t *testing.T, s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

// newTestRouter wires a bare authorized group behind core.AuthMiddleware,
// exactly as NewServer wires the "authorized" group in production, so these
// tests exercise transfer's Handler the same way it runs for real.
func newTestRouter(t *testing.T, storer store.Storer) (*gin.Engine, token.Maker) {
	gin.SetMode(gin.TestMode)

	tokenMaker, err := token.NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	router := gin.New()
	router.Use(core.ErrorHandlerMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))

	authorized := router.Group("/api/v1")
	authorized.Use(core.AuthMiddleware(tokenMaker))

	h := &Handler{Store: storer}
	h.RegisterRoutes(authorized)

	return router, tokenMaker
}

func authHeaderFor(t *testing.T, tokenMaker token.Maker, userID int64) string {
	accessToken, _, err := tokenMaker.CreateToken(userID, testUsername, testEmail, time.Minute)
	require.NoError(t, err)
	return "Bearer " + accessToken
}

func doAuthenticatedRequest(t *testing.T, router *gin.Engine, method, path string, body any, authHeader string) *httptest.ResponseRecorder {
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
		req.Header.Set("Authorization", authHeader)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestDeposit(t *testing.T) {
	const accountID = int64(5)

	ownedAccount := db.Account{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}, Currency: "IDR"}

	testCases := []struct {
		name          string
		body          depositRequest
		buildStubs    func(t *testing.T, q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "deposits into an owned account",
			body: depositRequest{Amount: mustDecimal(t, "100.00"), Description: "top up"},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().DepositTx(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg store.DepositTxParams) (store.DepositTxResult, error) {
						require.Equal(t, accountID, arg.AccountID)
						require.Equal(t, "100.00", arg.Amount)
						return store.DepositTxResult{Entry: db.Entry{ID: 7}}, nil
					},
				)
				q.EXPECT().AccountEntriesByAccountId(gomock.Any(), int64(7)).Return(db.AccountEntriesByAccountIdRow{
					AccountEntriesView: db.AccountEntriesView{ID: 7, AccountID: accountID, Amount: "100.00"},
				}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "rejects a non-positive amount without touching the db",
			body: depositRequest{Amount: mustDecimal(t, "0")},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().DepositTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			body: depositRequest{Amount: mustDecimal(t, "50")},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.Account{}, sql.ErrNoRows)
				q.EXPECT().DepositTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects depositing into an account owned by another user",
			body: depositRequest{Amount: mustDecimal(t, "50")},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.Account{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
				q.EXPECT().DepositTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 400 when DepositTx rejects the amount",
			body: depositRequest{Amount: mustDecimal(t, "50")},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().DepositTx(gomock.Any(), gomock.Any()).Return(store.DepositTxResult{}, store.ErrInvalidAmount)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(t, q)

			router, tokenMaker := newTestRouter(t, q)
			path := "/api/v1/accounts/" + itoa(accountID) + "/deposit"
			recorder := doAuthenticatedRequest(t, router, http.MethodPost, path, tc.body, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountEntriesByAccountId(t *testing.T) {
	const accountID = int64(5)

	testCases := []struct {
		name          string
		query         string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "lists entries for the account",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Return([]db.ListAccountEntriesByAccountIdRow{
					{AccountEntriesView: db.AccountEntriesView{ID: 1, AccountID: accountID}},
				}, nil)
				q.EXPECT().CountAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []accountEntriesViewResponse `json:"data"`
					Meta core.Meta                    `json:"meta"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, int64(1), resp.Meta.Total)
			},
		},
		{
			name:  "rejects an out-of-range month",
			query: "?month=13",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "returns 500 when listing entries hits a db error",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "returns 500 when counting entries hits a db error",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Return([]db.ListAccountEntriesByAccountIdRow{}, nil)
				q.EXPECT().CountAccountEntriesByAccountId(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router, tokenMaker := newTestRouter(t, q)
			path := "/api/v1/accounts/" + itoa(accountID) + "/entries" + tc.query
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, path, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListRecentTransferDestinations(t *testing.T) {
	const accountID = int64(5)

	ownedAccount := db.Account{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "lists recent destinations for an owned account",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListRecentTransferDestinations(gomock.Any(), db.ListRecentTransferDestinationsParams{
					FromAccountID: accountID,
					OffsetCount:   0,
					LimitCount:    10,
				}).Return([]db.ListRecentTransferDestinationsRow{
					{ID: 9, Name: sql.NullString{String: "Savings", Valid: true}, Username: "bob", UserID: 2},
				}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []publicAccountResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, int64(9), resp.Data[0].ID)
			},
		},
		{
			name: "returns 404 when the source account does not exist",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects a source account owned by another user",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.Account{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 500 when listing destinations hits a db error",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListRecentTransferDestinations(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router, tokenMaker := newTestRouter(t, q)
			path := "/api/v1/accounts/" + itoa(accountID) + "/recent-destinations"
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, path, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountTransactionHistory(t *testing.T) {
	const accountID = int64(5)

	ownedAccount := db.Account{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "lists transaction history for an owned account",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListAccountTransactionHistory(gomock.Any(), gomock.Any()).Return([]db.ListAccountTransactionHistoryRow{
					{ID: 1, Type: "DEPOSIT", Amount: "10.00", CreatedAt: time.Now()},
				}, nil)
				q.EXPECT().CountAccountTransactionHistory(gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []transactionHistoryItem `json:"data"`
					Meta core.Meta                `json:"meta"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, "Deposit", resp.Data[0].Label)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects an account owned by another user",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.Account{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 500 when listing history hits a db error",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(ownedAccount, nil)
				q.EXPECT().ListAccountTransactionHistory(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router, tokenMaker := newTestRouter(t, q)
			path := "/api/v1/accounts/" + itoa(accountID) + "/transactions"
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, path, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestTransactionTransfer(t *testing.T) {
	const (
		fromAccountID = int64(5)
		toAccountID   = int64(6)
	)

	ownedFromAccount := db.Account{ID: fromAccountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}, Currency: "IDR"}

	validReq := createTransactionTransferRequest{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        mustDecimal(t, "25.00"),
		Description:   "lunch money",
	}

	testCases := []struct {
		name          string
		body          createTransactionTransferRequest
		buildStubs    func(t *testing.T, q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "transfers between accounts",
			body: validReq,
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(ownedFromAccount, nil)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
						require.Equal(t, fromAccountID, arg.FromAccountID)
						require.Equal(t, toAccountID, arg.ToAccountID)
						require.Equal(t, "25.00", arg.Amount)
						return store.TransferTxResult{FromEntry: db.Entry{ID: 11}}, nil
					},
				)
				q.EXPECT().AccountEntriesByAccountId(gomock.Any(), int64(11)).Return(db.AccountEntriesByAccountIdRow{
					AccountEntriesView: db.AccountEntriesView{ID: 11, AccountID: fromAccountID, Amount: "-25.00"},
				}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "rejects a non-positive amount without touching the db",
			body: createTransactionTransferRequest{FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: mustDecimal(t, "0")},
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "returns 404 when the source account does not exist",
			body: validReq,
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(db.Account{}, sql.ErrNoRows)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects transferring from an account owned by another user",
			body: validReq,
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(
					db.Account{ID: fromAccountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
				)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 409 when the account has insufficient funds",
			body: validReq,
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(ownedFromAccount, nil)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Return(store.TransferTxResult{}, store.ErrInsufficientFunds)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "returns 400 when the accounts have mismatched currencies",
			body: validReq,
			buildStubs: func(t *testing.T, q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), fromAccountID).Return(ownedFromAccount, nil)
				q.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Return(store.TransferTxResult{}, store.ErrCurrencyMismatch)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(t, q)

			router, tokenMaker := newTestRouter(t, q)
			recorder := doAuthenticatedRequest(t, router, http.MethodPost, "/api/v1/transactions/transfer", tc.body, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}
