package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	config "github.com/DewaSRY/core-service/internal/config"
	corelog "github.com/DewaSRY/core-service/internal/logger"
	"github.com/DewaSRY/core-service/internal/token"
)

// defaultMaxLoggedBodySize is the fallback for config.Config.LogMaxBodySize
// when it's zero/unset — see effectiveMaxBodySize.
const defaultMaxLoggedBodySize int64 = 1 << 20 // 1MiB

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

func loggingMiddleware(log *slog.Logger, cfg config.Config) gin.HandlerFunc {
	maxBodySize := effectiveMaxBodySize(cfg)

	return func(ctx *gin.Context) {
		start := time.Now()

		requestURL := ctx.Request.URL.Path
		if raw := ctx.Request.URL.RawQuery; raw != "" {
			requestURL += "?" + raw
		}

		var reqBody any
		if cfg.LogRequestBody {
			reqBody = captureRequestBody(ctx.Request, maxBodySize)
		}

		var respWriter *bodyLogWriter
		if cfg.LogResponseBody {
			respWriter = &bodyLogWriter{ResponseWriter: ctx.Writer, maxBytes: int(maxBodySize)}
			ctx.Writer = respWriter
		}

		ctx.Next()

		status := ctx.Writer.Status()
		respHeaders := corelog.RedactHeaders(ctx.Writer.Header())

		reqAttrs := []any{
			slog.String("method", ctx.Request.Method),
			slog.String("url", requestURL),
			slog.String("path", ctx.Request.URL.Path),
			slog.Any("query", ctx.Request.URL.Query()),
			slog.Any("headers", corelog.RedactHeaders(ctx.Request.Header)),
			slog.Any("params", routeParams(ctx)),
		}
		if cfg.LogRequestBody {
			reqAttrs = append(reqAttrs, slog.Any("body", reqBody))
		}

		respAttrs := []any{
			slog.Int("status", status),
			slog.Any("headers", respHeaders),
			slog.Int("size", ctx.Writer.Size()),
		}
		if cfg.LogResponseBody && respWriter != nil {
			raw := respWriter.buf.Bytes()
			truncated := ctx.Writer.Size() > respWriter.buf.Len()
			respAttrs = append(respAttrs, slog.Any("body", decodeLoggedBody(raw, ctx.Writer.Header().Get("Content-Type"), truncated)))
		}

		outcome := "success"
		if status >= http.StatusBadRequest {
			outcome = "error"
		}

		attrs := []any{
			slog.Group("client",
				slog.String("ip", ctx.ClientIP()),
				slog.String("user_agent", ctx.Request.UserAgent()),
			),
			slog.Group("request", reqAttrs...),
			slog.Group("response", respAttrs...),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("status", outcome),
		}
		if userID, ok := authenticatedUserID(ctx); ok {
			attrs = append(attrs, slog.Int64("user_id", userID))
		}

		reqLog := loggerFromContext(ctx, log)
		switch {
		case status >= 500:
			reqLog.Error("request completed", attrs...)
		case status >= 400:
			reqLog.Warn("request completed", attrs...)
		default:
			reqLog.Info("request completed", attrs...)
		}
	}
}

// effectiveMaxBodySize applies defaultMaxLoggedBodySize when cfg doesn't set
// one, the same "zero/unset falls back to a sane default" pattern
// internal/logger.parseLevel uses for LOG_LEVEL.
func effectiveMaxBodySize(cfg config.Config) int64 {
	if cfg.LogMaxBodySize <= 0 {
		return defaultMaxLoggedBodySize
	}
	return cfg.LogMaxBodySize
}

// routeParams captures Gin's path parameters (e.g. :id) as a plain map so
// they serialize as a JSON object — {} when the matched route has none,
// never null, matching how query/headers render.
func routeParams(ctx *gin.Context) map[string]string {
	out := make(map[string]string, len(ctx.Params))
	for _, p := range ctx.Params {
		out[p.Key] = p.Value
	}
	return out
}

type bodyLogWriter struct {
	gin.ResponseWriter
	buf      bytes.Buffer
	maxBytes int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.captureForLog(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.captureForLog([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *bodyLogWriter) captureForLog(b []byte) {
	remaining := w.maxBytes - w.buf.Len()
	if remaining <= 0 {
		return
	}
	if len(b) > remaining {
		b = b[:remaining]
	}
	w.buf.Write(b)
}

func captureRequestBody(req *http.Request, maxBytes int64) any {
	contentType := req.Header.Get("Content-Type")

	if req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	if isMultipartContentType(contentType) {
		return placeholder("multipart/form-data body omitted")
	}
	if isBinaryContentType(contentType) {
		return placeholder(fmt.Sprintf("non-text content-type %q omitted", contentType))
	}
	if req.ContentLength > maxBytes {
		return placeholder(fmt.Sprintf("body omitted: %d bytes exceeds configured LOG_MAX_BODY_SIZE", req.ContentLength))
	}

	peeked, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	if err != nil {
		return placeholder("body unreadable: " + err.Error())
	}

	// Whatever's left unread on the original body (nothing, unless peeked
	// hit the maxBytes+1 cap) is chained back on so the handler sees the
	// exact stream it would have without this middleware.
	req.Body = io.NopCloser(io.MultiReader(bytes.NewReader(peeked), req.Body))

	truncated := int64(len(peeked)) > maxBytes
	raw := peeked
	if truncated {
		raw = peeked[:maxBytes]
	}
	return decodeLoggedBody(raw, contentType, truncated)
}

// decodeLoggedBody turns raw body bytes into whatever should actually be
// placed in the log record: a parsed (and redacted) JSON value so it renders
// as real JSON rather than an escaped string, a plain string for other text
// content, or a placeholder for empty/binary content so logs never fill up
// with unreadable bytes.
func decodeLoggedBody(raw []byte, contentType string, truncated bool) any {
	if len(raw) == 0 {
		return nil
	}

	if strings.Contains(strings.ToLower(contentType), "json") {
		var parsed any
		if err := json.Unmarshal(raw, &parsed); err == nil {
			return corelog.RedactJSONValue(parsed)
		}
		// Declared JSON but didn't parse — most likely truncated at
		// maxBytes. Fall through to the raw-text rendering below so a
		// snippet is still visible instead of nothing at all.
	}

	if !utf8.Valid(raw) {
		return placeholder(fmt.Sprintf("binary content omitted (%d bytes)", len(raw)))
	}

	text := string(raw)
	if truncated {
		text += fmt.Sprintf(" …(truncated at %d bytes)", len(raw))
	}
	return text
}

func isMultipartContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/")
}

// isBinaryContentType recognizes the common media types not worth buffering
// for logging (images, archives, PDFs, ...). Anything it doesn't recognize
// still gets a final safety check in decodeLoggedBody (utf8.Valid) before
// being rendered as text.
func isBinaryContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case ct == "":
		return false
	case strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "video/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "font/"):
		return true
	case strings.Contains(ct, "octet-stream"),
		strings.Contains(ct, "pdf"),
		strings.Contains(ct, "zip"),
		strings.Contains(ct, "gzip"),
		strings.Contains(ct, "protobuf"),
		strings.Contains(ct, "msgpack"):
		return true
	default:
		return false
	}
}

func placeholder(msg string) string {
	return "<" + msg + ">"
}

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
