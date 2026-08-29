package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Error codes returned in the "error.code" field of a normalized error
// response. Keep these stable — API consumers switch on them.
const (
	errCodeValidation        = "VALIDATION_ERROR"
	errCodeNotFound          = "NOT_FOUND"
	errCodeCurrencyMismatch  = "CURRENCY_MISMATCH"
	errCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	errCodeInternal          = "INTERNAL_ERROR"
	errCodeUnauthorized      = "UNAUTHORIZED"
	errCodeForbidden         = "FORBIDDEN"
	errCodeConflict          = "CONFLICT"
)

// FieldError describes a single invalid request field, per docs/NORMALIZE_RESPONSE.md.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type successResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Meta    any    `json:"meta,omitempty"`
}

// succeed writes a normalized success envelope. Every endpoint should call
// this (or succeedWithMeta for paginated lists) instead of ctx.JSON, so the
// {data, message} shape stays consistent without each handler re-deriving it.
func succeed(ctx *gin.Context, status int, data any, message string) {
	ctx.JSON(status, successResponse{Data: data, Message: message})
}

// succeedWithMeta is succeed plus a "meta" block, for paginated list endpoints
// (see the Pagination Response section of docs/NORMALIZE_RESPONSE.md).
func succeedWithMeta(ctx *gin.Context, status int, data any, message string, meta any) {
	ctx.JSON(status, successResponse{Data: data, Message: message, Meta: meta})
}

// fieldErrorsFromBindErr extracts per-field details from a ShouldBindJSON/
// ShouldBindUri error, using the validator library's own error introspection
// (github.com/go-playground/validator/v10, already a transitive dependency
// via Gin's default binding engine — used here directly instead of
// hand-parsing error strings). Returns nil for errors that aren't field
// validation failures (e.g. malformed JSON), so callers fall back to a plain
// message.
func fieldErrorsFromBindErr(err error) []FieldError {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}

	details := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		details = append(details, FieldError{Field: fe.Field(), Message: validationMessage(fe)})
	}
	return details
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " must be at least " + fe.Param()
	case "gt":
		return fe.Field() + " must be greater than " + fe.Param()
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	default:
		return fe.Error()
	}
}
