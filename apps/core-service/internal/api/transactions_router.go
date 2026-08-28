package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/DewaSRY/core-service/db/store"
	db "github.com/DewaSRY/core-service/db/sqlc"
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Amount.IsPositive() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than zero"})
		return
	}

	arg := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount.StringFixed(2),
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		respondTransferError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// respondTransferError maps a transferTx failure to the appropriate HTTP
// status instead of collapsing every error into a 500 with raw driver text.
func respondTransferError(ctx *gin.Context, err error) {
	var pqErr *pq.Error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
	case errors.Is(err, store.ErrSameAccount),
		errors.Is(err, store.ErrCurrencyMismatch),
		errors.Is(err, store.ErrInvalidAmount):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, store.ErrInsufficientFunds):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "check_violation":
		ctx.JSON(http.StatusConflict, gin.H{"error": "insufficient funds"})
	case errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation":
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "account not found"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
