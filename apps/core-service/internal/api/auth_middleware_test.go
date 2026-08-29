package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/DewaSRY/core-service/internal/token"
)

func newTestRouterWithAuthMiddleware(t *testing.T, tokenMaker token.Maker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(errorHandlerMiddleware())

	router.GET("/protected", authMiddleware(tokenMaker), func(ctx *gin.Context) {
		payload := getAuthPayload(ctx)
		require.NotNil(t, payload)
		ctx.JSON(http.StatusOK, gin.H{"username": payload.Username})
	})

	return router
}

func TestAuthMiddleware(t *testing.T) {
	tokenMaker, err := token.NewJWTMaker(testSecretKeyForMiddleware)
	require.NoError(t, err)

	validToken, _, err := tokenMaker.CreateToken(1, "dewa", "dewa@example.com", time.Minute)
	require.NoError(t, err)

	expiredToken, _, err := tokenMaker.CreateToken(1, "dewa", "dewa@example.com", -time.Minute)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token reaches handler",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed authorization header",
			authHeader:     "invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing bearer prefix",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouterWithAuthMiddleware(t, tokenMaker)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set(authorizationHeaderKey, tc.authHeader)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(t, tc.expectedStatus, recorder.Code)
		})
	}
}

const testSecretKeyForMiddleware = "12345678901234567890123456789012"
