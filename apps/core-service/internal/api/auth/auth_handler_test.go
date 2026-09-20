package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/DewaSRY/core-service/internal/api/core"
	mockdb "github.com/DewaSRY/core-service/internal/db/mock"
	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	store "github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/token"
	"github.com/DewaSRY/core-service/internal/util"
)

const testSecretKey = "12345678901234567890123456789012"

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestHandler(t *testing.T, storer store.Storer) *Handler {
	tokenMaker, err := token.NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	return &Handler{Store: storer, TokenMaker: tokenMaker, AccessTokenDuration: time.Minute}
}

// newTestRouter wires the same route groups NewServer does (public "/api/v1"
// routes, then an authorized group behind core.AuthMiddleware), so these
// tests exercise auth's Handler exactly as it runs in production.
func newTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(core.ErrorHandlerMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))

	v1 := router.Group("/api/v1")

	authorized := v1.Group("/")
	authorized.Use(core.AuthMiddleware(h.TokenMaker))
	h.RegisterRoutes(v1, authorized)

	return router
}

func doRegisterRequest(t *testing.T, router *gin.Engine, body registerUserRequest) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
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
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "registers successfully and returns an access token",
			body: validReq,
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
						require.Equal(t, validReq.Username, arg.Username)
						require.Equal(t, validReq.Email, arg.Email)
						require.NoError(t, util.CheckPassword(validReq.Password, arg.HashedPassword))
						return db.CreateUserRow{ID: 1, Username: arg.Username, Email: arg.Email, CreatedAt: time.Now()}, nil
					},
				)
				q.EXPECT().CreateAccountTx(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg store.CreateAccountTxParams) (db.Account, error) {
						require.Equal(t, int64(1), arg.UserID.Int64)
						require.Equal(t, "Main Account", arg.Name)
						require.True(t, arg.IsMain)
						return db.Account{ID: 1, UserID: arg.UserID, IsMain: true, Name: sql.NullString{String: arg.Name, Valid: true}}, nil
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
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
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.CreateUserRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "returns 500 when creating the main account fails",
			body: validReq,
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.CreateUserRow{ID: 1, Username: validReq.Username, Email: validReq.Email}, nil)
				q.EXPECT().CreateAccountTx(gomock.Any(), gomock.Any()).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "rejects a request missing required fields without touching the db",
			body: registerUserRequest{Email: "dewa@example.com", Password: "password123", PasswordConfirm: "password123"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				var resp errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, core.ErrCodeValidation, resp.Error.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router := newTestRouter(newTestHandler(t, q))
			recorder := doRegisterRequest(t, router, tc.body)
			tc.checkResponse(t, recorder)
		})
	}
}

func doLoginRequest(t *testing.T, router *gin.Engine, body loginUserRequest) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestLoginUser(t *testing.T) {
	const (
		email    = "dewa@example.com"
		password = "password123"
	)

	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	existingUser := db.GetUserByEmailRow{
		ID: 1, Username: "dewa", Email: email, HashedPassword: hashedPassword, CreatedAt: time.Now(),
	}

	testCases := []struct {
		name          string
		body          loginUserRequest
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "logs in successfully and returns an access token",
			body: loginUserRequest{Email: email, Password: password},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(existingUser, nil)
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
			name: "rejects a request missing required fields without touching the db",
			body: loginUserRequest{Email: email},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				var resp errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, core.ErrCodeValidation, resp.Error.Code)
			},
		},
		{
			name: "returns 401 when the email is not registered",
			body: loginUserRequest{Email: email, Password: password},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "returns 500 when looking up the user hits a db error",
			body: loginUserRequest{Email: email, Password: password},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(db.GetUserByEmailRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "returns 401 when the password is wrong",
			body: loginUserRequest{Email: email, Password: "wrong-password"},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(existingUser, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			q := mockdb.NewMockStorer(ctrl)
			tc.buildStubs(q)

			router := newTestRouter(newTestHandler(t, q))
			recorder := doLoginRequest(t, router, tc.body)
			tc.checkResponse(t, recorder)
		})
	}
}

func doGetProfileRequest(t *testing.T, router *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestGetProfile(t *testing.T) {
	const (
		userID   = int64(1)
		username = "dewa"
		email    = "dewa@example.com"
	)

	testCases := []struct {
		name          string
		authHeader    func(t *testing.T, h *Handler) string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "returns the authenticated user's profile",
			authHeader: func(t *testing.T, h *Handler) string {
				accessToken, _, err := h.TokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserById(gomock.Any(), userID).Return(db.GetUserByIdRow{
					ID:        userID,
					Username:  username,
					Email:     email,
					CreatedAt: time.Now(),
				}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Data profileResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.Equal(t, userID, resp.Data.ID)
				require.Equal(t, username, resp.Data.Username)
				require.Equal(t, email, resp.Data.Email)
			},
		},
		{
			name: "rejects a request without an access token",
			authHeader: func(t *testing.T, h *Handler) string {
				return ""
			},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserById(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "returns 404 when the token's user no longer exists",
			authHeader: func(t *testing.T, h *Handler) string {
				accessToken, _, err := h.TokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserById(gomock.Any(), userID).Return(db.GetUserByIdRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "returns 500 when fetching the user hits a db error",
			authHeader: func(t *testing.T, h *Handler) string {
				accessToken, _, err := h.TokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserById(gomock.Any(), userID).Return(db.GetUserByIdRow{}, sql.ErrConnDone)
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

			h := newTestHandler(t, q)
			router := newTestRouter(h)
			recorder := doGetProfileRequest(t, router, tc.authHeader(t, h))
			tc.checkResponse(t, recorder)
		})
	}
}
