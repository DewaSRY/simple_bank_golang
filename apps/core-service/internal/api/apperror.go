package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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

func InternalErr() *AppError {
	return newAppError(http.StatusInternalServerError, errCodeInternal, "internal server error")
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
func errorHandlerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		var appErr *AppError
		if errors.As(ctx.Errors.Last().Err, &appErr) {
			ctx.JSON(appErr.Status, errorResponse{Error: errorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			}})
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse{Error: errorBody{
			Code:    errCodeInternal,
			Message: "internal server error",
		}})
	}
}
