package account

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

// mockStorer adapts a *mockdb.MockStorer (generated only from sqlc.Querier)
// into the store.Storer interface Handler.Store requires, which additionally
// needs the hand-written store transactions. Each transaction method has an
// optional override func so a test can assert on its args/control its
// result; tests that don't care about a given transaction get a harmless
// zero result.
type mockStorer struct {
	*mockdb.MockStorer
	createAccountTxFunc func(ctx context.Context, arg store.CreateAccountTxParams) (db.Account, error)
	deleteAccountTxFunc func(ctx context.Context, arg store.DeleteAccountTxParams) (store.DeleteAccountTxResult, error)
}

func (m *mockStorer) TransferTx(ctx context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
	return store.TransferTxResult{}, nil
}

func (m *mockStorer) CreateAccountTx(ctx context.Context, arg store.CreateAccountTxParams) (db.Account, error) {
	if m.createAccountTxFunc != nil {
		return m.createAccountTxFunc(ctx, arg)
	}
	return db.Account{}, nil
}

func (m *mockStorer) DepositTx(ctx context.Context, arg store.DepositTxParams) (store.DepositTxResult, error) {
	return store.DepositTxResult{}, nil
}

func (m *mockStorer) DeleteAccountTx(ctx context.Context, arg store.DeleteAccountTxParams) (store.DeleteAccountTxResult, error) {
	if m.deleteAccountTxFunc != nil {
		return m.deleteAccountTxFunc(ctx, arg)
	}
	return store.DeleteAccountTxResult{}, nil
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

// newTestRouter wires a bare authorized group behind core.AuthMiddleware,
// exactly as NewServer wires the "authorized" group in production, so these
// tests exercise account's Handler the same way it runs for real.
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
			q := mockdb.NewMockStorer(ctrl)
			storer := &mockStorer{MockStorer: q}
			tc.buildStorer(t, storer)

			router, tokenMaker := newTestRouter(t, storer)
			recorder := doAuthenticatedRequest(t, router, http.MethodPost, "/api/v1/accounts", tc.body, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateAccount(t *testing.T) {
	existingAccount := db.Account{
		ID: 5, Balance: "100", Currency: "IDR",
		UserID: sql.NullInt64{Int64: testUserID, Valid: true},
		Name:   sql.NullString{String: "Old Name", Valid: true},
	}

	testCases := []struct {
		name          string
		body          updateAccountRequest
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "updates the account name and description",
			body: updateAccountRequest{Name: "New Name", Description: "new description"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(existingAccount, nil)
				q.EXPECT().UpdateAccount(gomock.Any(), db.UpdateAccountParams{
					ID:          existingAccount.ID,
					Name:        sql.NullString{String: "New Name", Valid: true},
					Description: sql.NullString{String: "new description", Valid: true},
				}).Return(db.Account{ID: existingAccount.ID, Name: sql.NullString{String: "New Name", Valid: true}}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "keeps the existing name when only description is sent",
			body: updateAccountRequest{Description: "new description only"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(existingAccount, nil)
				q.EXPECT().UpdateAccount(gomock.Any(), db.UpdateAccountParams{
					ID:          existingAccount.ID,
					Name:        sql.NullString{String: existingAccount.Name.String, Valid: true},
					Description: sql.NullString{String: "new description only", Valid: true},
				}).Return(db.Account{ID: existingAccount.ID}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			body: updateAccountRequest{Name: "New Name"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(db.Account{}, sql.ErrNoRows)
				q.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects updating an account owned by another user",
			body: updateAccountRequest{Name: "New Name"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), existingAccount.ID).Return(
					db.Account{ID: existingAccount.ID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
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
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router, tokenMaker := newTestRouter(t, &mockStorer{MockStorer: q})
			path := "/api/v1/accounts/" + itoa(existingAccount.ID)
			recorder := doAuthenticatedRequest(t, router, http.MethodPut, path, tc.body, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDetailAccount(t *testing.T) {
	const accountID = int64(5)

	ownedAccountView := db.GetAccountViewByIdRow{
		AccountUserDetailsView: db.AccountUserDetailsView{
			ID:     accountID,
			UserID: sql.NullInt64{Int64: testUserID, Valid: true},
			Name:   sql.NullString{String: "Savings", Valid: true},
		},
	}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "returns the account details for the owning user",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountViewById(gomock.Any(), accountID).Return(ownedAccountView, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data accountuserResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, accountID, resp.Data.ID)
			},
		},
		{
			name: "returns 404 when the account does not exist",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountViewById(gomock.Any(), accountID).Return(db.GetAccountViewByIdRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects viewing an account owned by another user",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountViewById(gomock.Any(), accountID).Return(db.GetAccountViewByIdRow{
					AccountUserDetailsView: db.AccountUserDetailsView{
						ID:     accountID,
						UserID: sql.NullInt64{Int64: 999, Valid: true},
					},
				}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "returns 500 on an unexpected db error",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountViewById(gomock.Any(), accountID).Return(db.GetAccountViewByIdRow{}, sql.ErrConnDone)
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

			router, tokenMaker := newTestRouter(t, &mockStorer{MockStorer: q})
			path := "/api/v1/accounts/" + itoa(accountID)
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, path, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListmeAccounts(t *testing.T) {
	testCases := []struct {
		name          string
		query         string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "lists the authenticated user's accounts",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListMeAccountsByUserId(gomock.Any(), db.ListMeAccountsByUserIdParams{
					UserID:      sql.NullInt64{Int64: testUserID, Valid: true},
					Name:        sql.NullString{Valid: true},
					OffsetCount: 0,
					LimitCount:  10,
				}).Return([]db.ListMeAccountsByUserIdRow{
					{AccountUserDetailsView: db.AccountUserDetailsView{ID: 1, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}},
				}, nil)
				q.EXPECT().ListMeAccountsByUserIdCount(gomock.Any(), db.ListMeAccountsByUserIdCountParams{
					UserID: sql.NullInt64{Int64: testUserID, Valid: true},
					Name:   sql.NullString{Valid: true},
				}).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []accountuserResponse `json:"data"`
					Meta core.Meta             `json:"meta"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, int64(1), resp.Meta.Total)
			},
		},
		{
			name:  "rejects an out-of-range limit without touching the db",
			query: "?limit=1000",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListMeAccountsByUserId(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().ListMeAccountsByUserIdCount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "returns 500 when listing accounts hits a db error",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListMeAccountsByUserId(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
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

			router, tokenMaker := newTestRouter(t, &mockStorer{MockStorer: q})
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, "/api/v1/accounts/me"+tc.query, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestSearchAccountByNumber(t *testing.T) {
	testCases := []struct {
		name          string
		query         string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "finds accounts matching the number",
			query: "?number=12345",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountsSearchByUserNumber(gomock.Any(), db.ListAccountsSearchByUserNumberParams{
					Number:      sql.NullString{String: "12345", Valid: true},
					OffsetCount: 0,
					LimitCount:  10,
				}).Return([]db.ListAccountsSearchByUserNumberRow{
					{AccountUserDetailsView: db.AccountUserDetailsView{ID: 2, Number: sql.NullString{String: "12345", Valid: true}}},
				}, nil)
				q.EXPECT().CountAccountsSearchByUserNumber(gomock.Any(), sql.NullString{String: "12345", Valid: true}).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data []accountuserResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Len(t, resp.Data, 1)
				require.Equal(t, "12345", resp.Data[0].Number)
			},
		},
		{
			name:  "rejects a request missing the required number without touching the db",
			query: "",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountsSearchByUserNumber(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().CountAccountsSearchByUserNumber(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "returns 500 when the search hits a db error",
			query: "?number=12345",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().ListAccountsSearchByUserNumber(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
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

			router, tokenMaker := newTestRouter(t, &mockStorer{MockStorer: q})
			recorder := doAuthenticatedRequest(t, router, http.MethodGet, "/api/v1/accounts/search-by-number"+tc.query, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	const accountID = int64(5)

	ownedAccount := db.Account{ID: accountID, UserID: sql.NullInt64{Int64: testUserID, Valid: true}}

	testCases := []struct {
		name          string
		buildStubs    func(q *mockdb.MockStorer)
		buildStorer   func(storer *mockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "deletes a zero-balance account with no sweep",
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(db.Account{}, sql.ErrNoRows)
			},
			buildStorer: func(storer *mockStorer) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "rejects deleting an account owned by another user",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetAccountById(gomock.Any(), accountID).Return(
					db.Account{ID: accountID, UserID: sql.NullInt64{Int64: 999, Valid: true}}, nil,
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
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			storer := &mockStorer{MockStorer: q}
			tc.buildStorer(storer)

			router, tokenMaker := newTestRouter(t, storer)
			path := "/api/v1/accounts/" + itoa(accountID)
			recorder := doAuthenticatedRequest(t, router, http.MethodDelete, path, nil, authHeaderFor(t, tokenMaker, testUserID))
			tc.checkResponse(t, recorder)
		})
	}
}
