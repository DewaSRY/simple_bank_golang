package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/token"
)

const (
	requestIDHeader     = "X-Request-Id"
	requestIDContextKey = "request_id"
)

// requestIDMiddleware assigns every request a short opaque ID — reusing one
// supplied by the caller (e.g. an upstream proxy/load balancer) via
// X-Request-Id, or generating one otherwise — and echoes it back on the
// response. Every log line for a request carries this ID, so a client-
// reported problem can be traced to its exact log lines by asking for the
// header value instead of grepping by timestamp.
func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetHeader(requestIDHeader)
		if id == "" {
			id = generateRequestID()
		}

		ctx.Set(requestIDContextKey, id)
		ctx.Writer.Header().Set(requestIDHeader, id)
		ctx.Next()
	}
}

func generateRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing means the OS entropy source is broken; a
		// timestamp-based fallback still gives every request a distinct,
		// merely non-random, ID instead of taking the process down for it.
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func getRequestID(ctx *gin.Context) string {
	id, _ := ctx.Get(requestIDContextKey)
	s, _ := id.(string)
	return s
}

// loggerFromContext returns a *slog.Logger already carrying request_id, so
// call sites (handlers, error mapping) don't have to thread it through
// separately.
func loggerFromContext(ctx *gin.Context, base *slog.Logger) *slog.Logger {
	return base.With(slog.String("request_id", getRequestID(ctx)))
}

// loggingMiddleware logs one structured line per request after it completes,
// so every request is accounted for and can be correlated by request_id with
// whatever errorHandlerMiddleware/recoveryMiddleware logged for it. Level
// scales with outcome: 5xx responses log at Error (something broke and needs
// looking at), 4xx at Warn (a caller did something invalid, less urgent),
// everything else at Info.
func loggingMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		if raw := ctx.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		ctx.Next()

		status := ctx.Writer.Status()
		attrs := []any{
			slog.String("request_id", getRequestID(ctx)),
			slog.String("method", ctx.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", ctx.ClientIP()),
		}
		if userID, ok := authenticatedUserID(ctx); ok {
			attrs = append(attrs, slog.Int64("user_id", userID))
		}

		switch {
		case status >= 500:
			log.Error("request completed", attrs...)
		case status >= 400:
			log.Warn("request completed", attrs...)
		default:
			log.Info("request completed", attrs...)
		}
	}
}

// authenticatedUserID reads the caller's ID off the token payload set by
// authMiddleware, without panicking on routes that never run it (health,
// login, register) — unlike getAuthPayload, which is only safe to call from
// handlers that authMiddleware guarantees ran first.
func authenticatedUserID(ctx *gin.Context) (int64, bool) {
	v, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		return 0, false
	}
	payload, ok := v.(*token.Payload)
	if !ok {
		return 0, false
	}
	return payload.ID, true
}

// recoveryMiddleware replaces gin.Recovery(): it catches a panic anywhere
// downstream, logs it (with a stack trace, request_id, method and path) at
// Error level, and renders the same normalized 500 body every other
// unhandled failure gets, instead of gin's plain-text default page.
func recoveryMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				loggerFromContext(ctx, log).Error("panic recovered",
					slog.Any("panic", r),
					slog.String("method", ctx.Request.Method),
					slog.String("path", ctx.Request.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)
				fail(ctx, InternalErr(fmt.Errorf("panic: %v", r)))
			}
		}()
		ctx.Next()
	}
}
