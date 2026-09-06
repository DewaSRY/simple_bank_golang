package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	db "github.com/DewaSRY/core-service/db/sqlc"
)

type createTransactionTransferRequest struct {
	FromAccountID int64           `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64           `json:"to_account_id" binding:"required,min=1"`
	Amount        decimal.Decimal `json:"amount"`
	Description   string          `json:"description"`
}

// transactionTransfer godoc
// @Summary      Transfer funds
// @Description  Transfer money from an account owned by the authenticated user to another account
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createTransactionTransferRequest  true  "Transfer payload"
// @Success      200      {object}  successResponse{data=accountEntriesViewResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      403      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      409      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /transactions/transfer [post]
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

	fromAccount, err := server.store.GetAccountById(ctx, req.FromAccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("account not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	authPayload := getAuthPayload(ctx)
	if fromAccount.UserID.Int64 != authPayload.ID {
		fail(ctx, ForbiddenErr("from_account does not belong to the authenticated user"))
		return
	}

	arg := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount.StringFixed(2),
		Description:   sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		fail(ctx, transferAppError(err))
		return
	}

	accountEntries, err := server.store.AccountEntriesByAccountId(ctx, result.FromEntry.ID)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, toAccountEntriesViewResponse(accountEntries), "Transfer completed successfully")
}
