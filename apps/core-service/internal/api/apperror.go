package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// AppError is the one error type handlers report through fail(ctx, err).
// It carries everything errorHandlerMiddleware needs to render a normalized
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
	// its only job is to give errorHandlerMiddleware something real to log
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
// per-field details (typically from fieldErrorsFromBindErr).
func ValidationErr(details ...FieldError) *AppError {
	return newAppError(http.StatusBadRequest, errCodeValidation, "One or more fields are invalid", details...)
}

// BadRequestErr builds a 400 response with a custom code/message, for
// business-rule violations that aren't tied to a single field (e.g.
// "from and to account must differ").
func BadRequestErr(code, message string) *AppError {
	return newAppError(http.StatusBadRequest, code, message)
}

func NotFoundErr(message string) *AppError {
	return newAppError(http.StatusNotFound, errCodeNotFound, message)
}

// UnauthorizedErr builds a 401 response for authentication failures (missing,
// malformed, or invalid/expired token) — see errCodeUnauthorized.
func UnauthorizedErr(message string) *AppError {
	return newAppError(http.StatusUnauthorized, errCodeUnauthorized, message)
}

// ForbiddenErr builds a 403 response for a valid but insufficiently
// privileged caller (e.g. accessing another user's resource).
func ForbiddenErr(message string) *AppError {
	return newAppError(http.StatusForbidden, errCodeForbidden, message)
}

func ConflictErr(code, message string) *AppError {
	return newAppError(http.StatusConflict, code, message)
}

// InternalErr builds a sanitized 500 response for an unexpected failure.
// err is the real underlying error (a DB failure, a token-signing error,
// ...) — it's never sent to the client, but errorHandlerMiddleware logs it
// so a 500 is actually debuggable instead of just "internal server error"
// with no further trace. Always pass the error that triggered this branch,
// never nil.
func InternalErr(err error) *AppError {
	appErr := newAppError(http.StatusInternalServerError, errCodeInternal, "internal server error")
	appErr.Cause = err
	return appErr
}

// fail records the error on the gin context and stops the handler chain.
// errorHandlerMiddleware renders the actual response, so every endpoint
// reports failures the same way regardless of what kind of error it is.
func fail(ctx *gin.Context, err *AppError) {
	ctx.Error(err) //nolint:errcheck // gin.Context.Error only returns for chaining
	ctx.Abort()
}

// errorHandlerMiddleware centralizes error-to-response rendering on top of
// Gin's own error-collection mechanism (ctx.Error/ctx.Errors) rather than a
// bespoke one, so it works for any handler registered after it. It renders
// whatever fail(ctx, ...) recorded, and defaults to a sanitized 500 for
// anything that isn't an *AppError — a handler can never accidentally leak
// raw internal error text just by forgetting to wrap an error.
//
// Before rendering, it logs the failure via log (request_id/method/path plus
// whatever the error actually was) — a 500's AppError.Cause included — so a
// client-visible "internal server error" always has a matching log line to
// debug from, instead of vanishing the moment the sanitized response is
// written.
func errorHandlerMiddleware(log *slog.Logger) gin.HandlerFunc {
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
			Code:    errCodeInternal,
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

// transferAppError maps a transferTx/depositTx failure to an AppError. This
// is the only piece of these endpoints that's specific to money movement —
// the actual response rendering is shared with every other endpoint via
// fail/AppError.
func transferAppError(err error) *AppError {
	var pqErr *pq.Error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return NotFoundErr("account not found")
	case errors.Is(err, store.ErrSameAccount):
		return BadRequestErr(errCodeValidation, err.Error())
	case errors.Is(err, store.ErrInvalidAmount):
		return ValidationErr(FieldError{Field: "amount", Message: err.Error()})
	case errors.Is(err, store.ErrCurrencyMismatch):
		return BadRequestErr(errCodeCurrencyMismatch, err.Error())
	case errors.Is(err, store.ErrInsufficientFunds):
		return ConflictErr(errCodeInsufficientFunds, err.Error())
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
		return ConflictErr(errCodeInsufficientFunds, "insufficient funds")
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
		return BadRequestErr(errCodeNotFound, "account not found")
	default:
		return InternalErr(err)
	}
}
