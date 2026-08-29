package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/DewaSRY/core-service/db/mock"
	db "github.com/DewaSRY/core-service/db/sqlc"
	store "github.com/DewaSRY/core-service/db/store"
	config "github.com/DewaSRY/core-service/internal/config"
	"github.com/DewaSRY/core-service/pkg/utils"
)

// mockStorer adapts a *mockdb.MockQuerier (generated only from sqlc.Querier)
// into the Server's Storer interface, which additionally requires TransferTx.
// None of the auth tests exercise transfers, so TransferTx just returns a
// zero result.
type mockStorer struct {
	*mockdb.MockQuerier
}

func (m *mockStorer) TransferTx(ctx context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error) {
	return store.TransferTxResult{}, nil
}

func newTestServerWithMockStore(t *testing.T, q *mockdb.MockQuerier) *Server {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		JWTSecretKey:           testSecretKeyForMiddleware,
		JWTAccessTokenDuration: time.Minute,
	}

	server, err := NewServer(&mockStorer{MockQuerier: q}, cfg)
	require.NoError(t, err)
	return server
}

func doRegisterRequest(t *testing.T, server *Server, body registerUserRequest) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	return recorder
}

func TestRegisterUser(t *testing.T) {
	validReq := registerUserRequest{
		Username:        "dewa",
		Email:           "dewa@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}

	testCases := []struct {
		name          string
		body          registerUserRequest
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "registers successfully and returns an access token",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
						require.Equal(t, validReq.Username, arg.Username)
						require.Equal(t, validReq.Email, arg.Email)
						require.NoError(t, utils.CheckPassword(validReq.Password, arg.HashedPassword))
						return db.CreateUserRow{ID: 1, Username: arg.Username, Email: arg.Email, CreatedAt: time.Now()}, nil
					},
				)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data AuthResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.NotEmpty(t, resp.Data.AccessToken)
				require.Equal(t, "Bearer", resp.Data.TokenType)
			},
		},
		{
			name: "rejects a mismatched password confirmation without touching the db",
			body: registerUserRequest{
				Username:        "dewa",
				Email:           "dewa@example.com",
				Password:        "password123",
				PasswordConfirm: "somethingelse",
			},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "rejects an email that is already registered",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{ID: 1, Email: validReq.Email}, nil)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				var resp errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, "email_exists", resp.Error.Code)
			},
		},
		{
			name: "returns 500 when checking the email hits a db error",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrConnDone)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), gomock.Any()).Times(0)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			// Regression test: the username was never checked for uniqueness,
			// so registering with a taken username reached CreateUser, failed
			// on the DB's unique constraint, and that raw error was mapped to
			// a generic 500 instead of a proper conflict response.
			name: "rejects a username that is already taken",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(true, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				var resp errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, "username_exists", resp.Error.Code)
			},
		},
		{
			name: "returns 500 when checking the username hits a db error",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, sql.ErrConnDone)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			// Belt-and-suspenders: even if the pre-checks race with another
			// request and CreateUser itself hits the unique constraint, that
			// must surface as a 409 conflict, not a 500.
			name: "returns 409 when CreateUser hits a unique constraint race",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.CreateUserRow{}, &pq.Error{Code: "23505"})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "returns 500 when CreateUser fails for an unrelated reason",
			body: validReq,
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.CreateUserRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "rejects a request missing required fields without touching the db",
			body: registerUserRequest{Email: "dewa@example.com", Password: "password123", PasswordConfirm: "password123"},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				var resp errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, errCodeValidation, resp.Error.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			server := newTestServerWithMockStore(t, q)
			recorder := doRegisterRequest(t, server, tc.body)
			tc.checkResponse(t, recorder)
		})
	}
}
