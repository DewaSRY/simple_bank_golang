package api

import (
	"database/sql"
	"errors"
	"net/http"

	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type createTransactionTransferRequest struct {
	FromAccountID int64           `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64           `json:"to_account_id" binding:"required,min=1"`
	Amount        decimal.Decimal `json:"amount"`
}

func (server *Server) transactionTransfer(ctx *gin.Context) {
	var req createTransactionTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	if !req.Amount.IsPositive() {
		fail(ctx, ValidationErr(FieldError{Field: "amount", Message: "amount must be greater than zero"}))
		return
	}

	arg := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount.StringFixed(2),
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		fail(ctx, transferAppError(err))
		return
	}

	succeed(ctx, http.StatusOK, result, "Transfer completed successfully")
}

// transferAppError maps a transferTx failure to an AppError. This is the
// only piece of this endpoint that's specific to transfers — the actual
// response rendering is shared with every other endpoint via fail/AppError.
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
		return InternalErr()
	}
}
