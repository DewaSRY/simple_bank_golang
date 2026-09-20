package transfer

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/DewaSRY/core-service/internal/api/core"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
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
// @Success      200      {object}  core.successResponse{data=accountEntriesViewResponse}
// @Failure      400      {object}  core.errorResponse
// @Failure      401      {object}  core.errorResponse
// @Failure      403      {object}  core.errorResponse
// @Failure      404      {object}  core.errorResponse
// @Failure      409      {object}  core.errorResponse
// @Failure      500      {object}  core.errorResponse
// @Router       /transactions/transfer [post]
func (h *Handler) transactionTransfer(ctx *gin.Context) {
	var req createTransactionTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	if req.FromAccountID == req.ToAccountID {
		core.Fail(ctx, core.ValidationErr(core.FieldError{Field: "to_account_id", Message: "to_account_id must be different from from_account_id"}))
		return
	}

	if !req.Amount.IsPositive() {
		core.Fail(ctx, core.ValidationErr(core.FieldError{Field: "amount", Message: "amount must be greater than zero"}))
		return
	}

	fromAccount, err := h.Store.GetAccountById(ctx, req.FromAccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("account not found"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	authPayload := core.GetAuthPayload(ctx)
	if fromAccount.UserID.Int64 != authPayload.ID {
		core.Fail(ctx, core.ForbiddenErr("from_account does not belong to the authenticated user"))
		return
	}

	arg := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount.StringFixed(2),
		Description:   sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	result, err := h.Store.TransferTx(ctx, arg)
	if err != nil {
		core.Fail(ctx, transferAppError(err))
		return
	}

	accountEntries, err := h.Store.AccountEntriesByAccountId(ctx, result.FromEntry.ID)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, toAccountEntriesViewResponse(accountEntries.AccountEntriesView), "Transfer completed successfully")
}
