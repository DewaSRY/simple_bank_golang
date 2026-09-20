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

func newTestHandlerWithTrustedIPs(t *testing.T, storer store.Storer, trustedIPs []string) *Handler {
	h := newTestHandler(t, storer)
	h.TrustedIPs = trustedIPs
	return h
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

// fingerprintFor mirrors requestDeviceFingerprint for a request that only
// sets User-Agent (Accept-Language/Accept-Encoding absent, as in these tests).
func fingerprintFor(userAgent string) string {
	return util.ComputeDeviceFingerprint(userAgent, "", "")
}

func doRegisterRequest(t *testing.T, router *gin.Engine, body registerUserRequest, userAgent string) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestRegisterUser(t *testing.T) {
	const registerUserAgent = "register-agent/1.0"

	validReq := registerUserRequest{
		Username: "dewa",
		Email:    "dewa@example.com",
	}

	testCases := []struct {
		name          string
		body          registerUserRequest
		userAgent     string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "registers successfully, binding the account to the requesting device",
			body:      validReq,
			userAgent: registerUserAgent,
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), validReq.Email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
				q.EXPECT().CheckIsUsernameExist(gomock.Any(), validReq.Username).Return(false, nil)
				q.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
						require.Equal(t, validReq.Username, arg.Username)
						require.Equal(t, validReq.Email, arg.Email)
						require.Equal(t, fingerprintFor(registerUserAgent), arg.DeviceFingerprintHash)
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
			body: registerUserRequest{Email: "dewa@example.com"},
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
			recorder := doRegisterRequest(t, router, tc.body, tc.userAgent)
			tc.checkResponse(t, recorder)
		})
	}
}

func doLoginRequest(t *testing.T, router *gin.Engine, body loginUserRequest, userAgent string) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestLoginUser(t *testing.T) {
	const (
		email          = "dewa@example.com"
		boundUserAgent = "bound-agent/1.0"
	)

	boundUser := db.GetUserByEmailRow{
		ID: 1, Username: "dewa", Email: email, DeviceFingerprintHash: fingerprintFor(boundUserAgent), CreatedAt: time.Now(),
	}

	testCases := []struct {
		name          string
		body          loginUserRequest
		userAgent     string
		buildStubs    func(q *mockdb.MockStorer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "logs in successfully from the bound device and returns an access token",
			body:      loginUserRequest{Email: email},
			userAgent: boundUserAgent,
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(boundUser, nil)
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
			body: loginUserRequest{},
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
			body: loginUserRequest{Email: email},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "returns 500 when looking up the user hits a db error",
			body: loginUserRequest{Email: email},
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(db.GetUserByEmailRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "returns 403 when the request comes from a different device",
			body:      loginUserRequest{Email: email},
			userAgent: "some-other-agent/9.9",
			buildStubs: func(q *mockdb.MockStorer) {
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(boundUser, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:      "binds and succeeds on first login for a legacy, unbound account",
			body:      loginUserRequest{Email: email},
			userAgent: "first-login-agent/1.0",
			buildStubs: func(q *mockdb.MockStorer) {
				unboundUser := db.GetUserByEmailRow{ID: 2, Username: "legacy", Email: email, DeviceFingerprintHash: "", CreatedAt: time.Now()}
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(unboundUser, nil)
				q.EXPECT().BindUserDeviceFingerprint(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.BindUserDeviceFingerprintParams) (db.BindUserDeviceFingerprintRow, error) {
						require.Equal(t, int64(2), arg.ID)
						require.Equal(t, fingerprintFor("first-login-agent/1.0"), arg.DeviceFingerprintHash)
						return db.BindUserDeviceFingerprintRow{ID: 2, Username: "legacy", Email: email, CreatedAt: time.Now()}, nil
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
			},
		},
		{
			name:      "returns 500 when binding a legacy account's device fails",
			body:      loginUserRequest{Email: email},
			userAgent: "first-login-agent/1.0",
			buildStubs: func(q *mockdb.MockStorer) {
				unboundUser := db.GetUserByEmailRow{ID: 2, Username: "legacy", Email: email, DeviceFingerprintHash: "", CreatedAt: time.Now()}
				q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(unboundUser, nil)
				q.EXPECT().BindUserDeviceFingerprint(gomock.Any(), gomock.Any()).Return(db.BindUserDeviceFingerprintRow{}, sql.ErrConnDone)
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

			router := newTestRouter(newTestHandler(t, q))
			recorder := doLoginRequest(t, router, tc.body, tc.userAgent)
			tc.checkResponse(t, recorder)
		})
	}
}

// TestLoginUser_TrustedIPBypassesDeviceFingerprint covers Handler.TrustedIPs:
// a request from a configured trusted IP must log in even with a device
// fingerprint that doesn't match the bound one, since it gets the fixed mock
// fingerprint instead of one derived from headers.
func TestLoginUser_TrustedIPBypassesDeviceFingerprint(t *testing.T) {
	const (
		email          = "dewa@example.com"
		trustedIP      = "192.0.2.1" // httptest.NewRequest's default RemoteAddr
		boundUserAgent = "bound-agent/1.0"
	)

	boundUser := db.GetUserByEmailRow{
		ID: 1, Username: "dewa", Email: email, DeviceFingerprintHash: trustedIPMockFingerprint, CreatedAt: time.Now(),
	}

	ctrl := gomock.NewController(t)
	q := mockdb.NewMockStorer(ctrl)
	q.EXPECT().GetUserByEmail(gomock.Any(), email).Return(boundUser, nil)

	h := newTestHandlerWithTrustedIPs(t, q, []string{trustedIP})
	router := newTestRouter(h)

	// A user agent that would normally mismatch the bound fingerprint and be
	// rejected with 403 — the trusted IP bypass must still let it through.
	recorder := doLoginRequest(t, router, loginUserRequest{Email: email}, boundUserAgent+"-different")
	require.Equal(t, http.StatusOK, recorder.Code)
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
