package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	config "github.com/DewaSRY/core-service/internal/config"
)

// newTestRouterWithMiddleware wires the same middleware chain and ordering
// NewServer does, so these tests catch a regression in that ordering (a
// panic being "recovered" into an empty 200 instead of a rendered 500) even
// though NewServer itself isn't exercised here.
func newTestRouterWithMiddleware(log *slog.Logger) *gin.Engine {
	return newTestRouterWithConfig(log, config.Config{LogRequestBody: true, LogResponseBody: true})
}

func newTestRouterWithConfig(log *slog.Logger, cfg config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware(log, cfg))
	router.Use(ErrorHandlerMiddleware(log))
	router.Use(RecoveryMiddleware(log))
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
	require.Equal(t, ErrCodeInternal, body.Error.Code)
	require.NotContains(t, rec.Body.String(), "something exploded")
}

func TestErrorHandlerMiddleware_LogsCauseButDoesNotLeakIt(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithMiddleware(log)
	router.GET("/boom", func(ctx *gin.Context) {
		Fail(ctx, InternalErr(errors.New("pq: connection reset by peer")))
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

func TestLoggingMiddleware_RedactsRequestBodyAndHeadersButHandlerSeesOriginal(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithConfig(log, config.Config{LogRequestBody: true, LogResponseBody: true})

	var handlerSawBody []byte
	router.POST("/users", func(ctx *gin.Context) {
		handlerSawBody, _ = io.ReadAll(ctx.Request.Body)
		ctx.JSON(http.StatusCreated, gin.H{"id": "123", "name": "John"})
	})

	reqBody := `{"email":"john@example.com","password":"hunter2"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	// The handler must still see the exact, un-redacted original body —
	// capturing it for logging must not consume or alter it.
	require.JSONEq(t, reqBody, string(handlerSawBody))

	logged := logBuf.String()
	require.Contains(t, logged, `"password":"[REDACTED]"`)
	require.NotContains(t, logged, "hunter2")
	require.Contains(t, logged, `"authorization":"[REDACTED]"`)
	require.NotContains(t, logged, "super-secret-token")
	require.Contains(t, logged, "john@example.com")
}

func TestLoggingMiddleware_CapturesResponseBody(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithConfig(log, config.Config{LogResponseBody: true})
	router.GET("/users/1", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"id": "123", "name": "John"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"id":"123","name":"John"}`, rec.Body.String())
	require.Contains(t, logBuf.String(), `"name":"John"`)
}

func TestLoggingMiddleware_OmitsBinaryBodyFromLogButHandlerStillReadsIt(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithConfig(log, config.Config{LogRequestBody: true})

	binary := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	var handlerSawBody []byte
	router.POST("/upload", func(ctx *gin.Context) {
		handlerSawBody, _ = io.ReadAll(ctx.Request.Body)
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(binary))
	req.Header.Set("Content-Type", "image/jpeg")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, binary, handlerSawBody)
	require.Contains(t, logBuf.String(), "non-text content-type")
	require.NotContains(t, logBuf.String(), "JFIF")
}

func TestLoggingMiddleware_DoesNotCaptureBodyWhenDisabled(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	router := newTestRouterWithConfig(log, config.Config{LogRequestBody: false, LogResponseBody: false})

	var handlerSawBody []byte
	router.POST("/users", func(ctx *gin.Context) {
		handlerSawBody, _ = io.ReadAll(ctx.Request.Body)
		ctx.JSON(http.StatusCreated, gin.H{"secret": "do-not-log-me"})
	})

	reqBody := `{"name":"John"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.JSONEq(t, reqBody, string(handlerSawBody))
	require.NotContains(t, logBuf.String(), "do-not-log-me")
}
