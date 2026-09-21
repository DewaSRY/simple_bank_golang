package core_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/DewaSRY/core-service/internal/api/core"
	config "github.com/DewaSRY/core-service/internal/config"
)

func newRateLimitedRouter(t *testing.T, rps float64, burst int) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(core.ErrorHandlerMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))

	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	router.Use(core.RateLimitMiddleware(config.Config{
		RateLimitRequestsPerSecond: rps,
		RateLimitBurst:             burst,
	}, stop))

	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return router
}

func doGet(router *gin.Engine, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRateLimitMiddleware_AllowsUpToBurstThenRejects(t *testing.T) {
	router := newRateLimitedRouter(t, 1, 3)

	for i := 0; i < 3; i++ {
		w := doGet(router, "1.2.3.4:1111")
		require.Equal(t, http.StatusOK, w.Code, "request %d within burst should be allowed", i+1)
	}

	w := doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitMiddleware_TracksIPsIndependently(t *testing.T) {
	router := newRateLimitedRouter(t, 1, 1)

	w := doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusOK, w.Code)

	w = doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusTooManyRequests, w.Code, "second request from the same IP should be throttled")

	w = doGet(router, "5.6.7.8:2222")
	require.Equal(t, http.StatusOK, w.Code, "a different IP should have its own bucket")
}

func TestRateLimitMiddleware_RefillsOverTime(t *testing.T) {
	router := newRateLimitedRouter(t, 20, 1)

	w := doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusOK, w.Code)

	w = doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusTooManyRequests, w.Code)

	time.Sleep(100 * time.Millisecond)

	w = doGet(router, "1.2.3.4:1111")
	require.Equal(t, http.StatusOK, w.Code, "bucket should have refilled after waiting past the rate")
}
