package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newTestRouterWithMiddleware wires the same middleware chain and ordering
// NewServer does, so these tests catch a regression in that ordering (a
// panic being "recovered" into an empty 200 instead of a rendered 500) even
// though NewServer itself isn't exercised here.
func newTestRouterWithMiddleware(log *slog.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.Use(loggingMiddleware(log))
	router.Use(errorHandlerMiddleware(log))
	router.Use(recoveryMiddleware(log))
	return router
}

func TestRecoveryMiddleware_RendersSanitized500(t *testing.T) {
	log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := newTestRouterWithMiddleware(log)
	router.GET("/panic", func(ctx *gin.Context) {
		panic("something exploded")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body errorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, errCodeInternal, body.Error.Code)
	require.NotContains(t, rec.Body.String(), "something exploded")
}

func TestErrorHandlerMiddleware_LogsCauseButDoesNotLeakIt(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithMiddleware(log)
	router.GET("/boom", func(ctx *gin.Context) {
		fail(ctx, InternalErr(errors.New("pq: connection reset by peer")))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "connection reset by peer")
	require.Contains(t, logBuf.String(), "connection reset by peer")
}

func TestRequestIDMiddleware_EchoesSuppliedID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := newTestRouterWithMiddleware(log)
	router.GET("/ping", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(requestIDHeader, "caller-supplied-id")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, "caller-supplied-id", rec.Header().Get(requestIDHeader))
}

func TestRequestIDMiddleware_GeneratesIDWhenAbsent(t *testing.T) {
	log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := newTestRouterWithMiddleware(log)
	router.GET("/ping", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.Header().Get(requestIDHeader))
}
