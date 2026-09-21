// Package core is the shared kernel every domain module (auth, account,
// transfer) depends on: the AppError/response envelope, and the
// cross-cutting middleware (auth, logging, error handling, recovery). It has
// no dependency on Server or Storer, so it never imports back up to
// internal/api or any domain package.
package core

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error codes returned in the "error.code" field of a normalized error
// response. Keep these stable — API consumers switch on them. Domain-specific
// codes (e.g. money-transfer or account-deletion failures) live in their own
// package instead of here.
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeRateLimited  = "RATE_LIMITED"
)

// AppError is the one error type handlers report through Fail(ctx, err). It
// carries everything ErrorHandlerMiddleware needs to render a normalized
// error response, so a handler only has to classify what went wrong — it
// never builds a JSON body itself. Any new endpoint gets the same
// {error: {code, message, details}} shape for free by constructing one of
// these instead of calling ctx.JSON.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details []FieldError

	// Cause is the underlying error that led to this AppError, if any. It is
	// never rendered in the HTTP response (errorBody has no field for it) —
	// its only job is to give ErrorHandlerMiddleware something real to log
	// for 500s, instead of every internal failure showing up in logs as the
	// same sanitized "internal server error" string with no way to tell one
	// failure from another.
	Cause error
}

func (e *AppError) Error() string {
	return e.Message
}

func newAppError(status int, code, message string, details ...FieldError) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

// ValidationErr builds a 400 VALIDATION_ERROR response, optionally with
// per-field details (typically from FieldErrorsFromBindErr).
func ValidationErr(details ...FieldError) *AppError {
	return newAppError(http.StatusBadRequest, ErrCodeValidation, "One or more fields are invalid", details...)
}

// BadRequestErr builds a 400 response with a custom code/message, for
// business-rule violations that aren't tied to a single field (e.g.
// "from and to account must differ").
func BadRequestErr(code, message string) *AppError {
	return newAppError(http.StatusBadRequest, code, message)
}

func NotFoundErr(message string) *AppError {
	return newAppError(http.StatusNotFound, ErrCodeNotFound, message)
}

// UnauthorizedErr builds a 401 response for authentication failures (missing,
// malformed, or invalid/expired token) — see ErrCodeUnauthorized.
func UnauthorizedErr(message string) *AppError {
	return newAppError(http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// ForbiddenErr builds a 403 response for a valid but insufficiently
// privileged caller (e.g. accessing another user's resource).
func ForbiddenErr(message string) *AppError {
	return newAppError(http.StatusForbidden, ErrCodeForbidden, message)
}

func ConflictErr(code, message string) *AppError {
	return newAppError(http.StatusConflict, code, message)
}

// TooManyRequestsErr builds a 429 response for a client that has exceeded a
// rate limit (see RateLimitMiddleware).
func TooManyRequestsErr(message string) *AppError {
	return newAppError(http.StatusTooManyRequests, ErrCodeRateLimited, message)
}

// InternalErr builds a sanitized 500 response for an unexpected failure. err
// is the real underlying error (a DB failure, a token-signing error, ...) —
// it's never sent to the client, but ErrorHandlerMiddleware logs it so a 500
// is actually debuggable instead of just "internal server error" with no
// further trace. Always pass the error that triggered this branch, never nil.
func InternalErr(err error) *AppError {
	appErr := newAppError(http.StatusInternalServerError, ErrCodeInternal, "internal server error")
	appErr.Cause = err
	return appErr
}

// Fail records the error on the gin context and stops the handler chain.
// ErrorHandlerMiddleware renders the actual response, so every endpoint
// reports failures the same way regardless of what kind of error it is.
func Fail(ctx *gin.Context, err *AppError) {
	ctx.Error(err) //nolint:errcheck // gin.Context.Error only returns for chaining
	ctx.Abort()
}

// ErrorHandlerMiddleware centralizes error-to-response rendering on top of
// Gin's own error-collection mechanism (ctx.Error/ctx.Errors) rather than a
// bespoke one, so it works for any handler registered after it. It renders
// whatever Fail(ctx, ...) recorded, and defaults to a sanitized 500 for
// anything that isn't an *AppError — a handler can never accidentally leak
// raw internal error text just by forgetting to wrap an error.
//
// Before rendering, it logs the failure via log (request_id/method/path plus
// whatever the error actually was) — a 500's AppError.Cause included — so a
// client-visible "internal server error" always has a matching log line to
// debug from, instead of vanishing the moment the sanitized response is
// written.
func ErrorHandlerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		reqLog := loggerFromContext(ctx, log).With(
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
		)

		var appErr *AppError
		if errors.As(ctx.Errors.Last().Err, &appErr) {
			logAppError(reqLog, appErr)
			ctx.JSON(appErr.Status, errorResponse{Error: errorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			}})
			return
		}

		reqLog.Error("unhandled error", slog.String("error", ctx.Errors.Last().Err.Error()))
		ctx.JSON(http.StatusInternalServerError, errorResponse{Error: errorBody{
			Code:    ErrCodeInternal,
			Message: "internal server error",
		}})
	}
}

// logAppError logs an *AppError at a level matching its severity: 500s are
// Error (need investigating — logged with Cause, the real underlying error,
// never sent to the client) and everything else is Warn (a normal
// client-caused failure, worth seeing but not paging anyone over).
func logAppError(log *slog.Logger, appErr *AppError) {
	if appErr.Status < http.StatusInternalServerError {
		log.Warn("request failed", slog.Int("status", appErr.Status), slog.String("code", appErr.Code), slog.String("message", appErr.Message))
		return
	}

	attrs := []any{slog.Int("status", appErr.Status), slog.String("code", appErr.Code)}
	if appErr.Cause != nil {
		attrs = append(attrs, slog.String("cause", appErr.Cause.Error()))
	}
	log.Error("request failed", attrs...)
}
