package transfer

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/DewaSRY/core-service/internal/api/core"
	"github.com/DewaSRY/core-service/internal/db/store"
)

const (
	errCodeCurrencyMismatch  = "CURRENCY_MISMATCH"
	errCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
)

func transferAppError(err error) *core.AppError {
	var pqErr *pq.Error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return core.NotFoundErr("account not found")
	case errors.Is(err, store.ErrSameAccount):
		return core.BadRequestErr(core.ErrCodeValidation, err.Error())
	case errors.Is(err, store.ErrInvalidAmount):
		return core.ValidationErr(core.FieldError{Field: "amount", Message: err.Error()})
	case errors.Is(err, store.ErrCurrencyMismatch):
		return core.BadRequestErr(errCodeCurrencyMismatch, err.Error())
	case errors.Is(err, store.ErrInsufficientFunds):
		return core.ConflictErr(errCodeInsufficientFunds, err.Error())
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
		return core.ConflictErr(errCodeInsufficientFunds, "insufficient funds")
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
		return core.BadRequestErr(core.ErrCodeNotFound, "account not found")
	default:
		return core.InternalErr(err)
	}
}
