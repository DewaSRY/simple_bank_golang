package api

import (
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
)

func doGetProfileRequest(t *testing.T, server *Server, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	if authHeader != "" {
		req.Header.Set(authorizationHeaderKey, authHeader)
	}

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
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
		authHeader    func(t *testing.T, server *Server) string
		buildStubs    func(q *mockdb.MockQuerier)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "returns the authenticated user's profile",
			authHeader: func(t *testing.T, server *Server) string {
				accessToken, _, err := server.tokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockQuerier) {
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
			authHeader: func(t *testing.T, server *Server) string {
				return ""
			},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserById(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "returns 404 when the token's user no longer exists",
			authHeader: func(t *testing.T, server *Server) string {
				accessToken, _, err := server.tokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockQuerier) {
				q.EXPECT().GetUserById(gomock.Any(), userID).Return(db.GetUserByIdRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "returns 500 when fetching the user hits a db error",
			authHeader: func(t *testing.T, server *Server) string {
				accessToken, _, err := server.tokenMaker.CreateToken(userID, username, email, time.Minute)
				require.NoError(t, err)
				return "Bearer " + accessToken
			},
			buildStubs: func(q *mockdb.MockQuerier) {
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
			q := mockdb.NewMockQuerier(ctrl)
			tc.buildStubs(q)

			server := newTestServerWithMockStore(t, q)
			recorder := doGetProfileRequest(t, server, tc.authHeader(t, server))
			tc.checkResponse(t, recorder)
		})
	}
}
