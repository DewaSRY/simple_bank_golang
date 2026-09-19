package account

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/api/core"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
)

// errCodeMainAccount is account-specific (only deleteAccountAppError uses
// it), so unlike the generic codes it doesn't live in core.
const errCodeMainAccount = "MAIN_ACCOUNT"

// detailAccount godoc
// @Summary      Get account details
// @Description  Retrieve the details of an account owned by the authenticated user
// @Tags         accounts-manage
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  successResponse{data=accountuserResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      409  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /accounts/{id} [get]
func (h *Handler) detailAccount(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	account, err := h.Store.GetAccountViewById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("account not found"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	authPayload := core.GetAuthPayload(ctx)
	if account.AccountUserDetailsView.UserID.Int64 != authPayload.ID {
		core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	core.Succeed(ctx, http.StatusOK, toAccountuserResponse(account.AccountUserDetailsView), "Account retrieved successfully")
}

type updateAccountRequest struct {
	Name        string `json:"name" binding:"omitempty"`
	Description string `json:"description" binding:"omitempty"`
}

// updateAccount godoc
// @Summary      Update an account
// @Description  Update the name and/or description of an account owned by the authenticated user
// @Tags         accounts-manage
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                    true  "Account ID"
// @Param        request  body      updateAccountRequest   true  "Account update payload"
// @Success      200      {object}  successResponse{data=accountResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      403      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /accounts/{id} [put]
func (h *Handler) updateAccount(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	var req updateAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	account, err := h.Store.GetAccountById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("account not found"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	authPayload := core.GetAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	name := req.Name
	if name == "" {
		name = account.Name.String
	}
	description := req.Description
	if description == "" {
		description = account.Description.String
	}

	updated, err := h.Store.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:          params.ID,
		Name:        sql.NullString{String: name, Valid: name != ""},
		Description: sql.NullString{String: description, Valid: description != ""},
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, toAccountResponse(updated), "Account updated successfully")
}

type deleteAccountResponse struct {
	Account          accountResponse `json:"account"`
	BalanceSweptToID *int64          `json:"balance_swept_to_account_id,omitempty"`
}

// deleteAccount godoc
// @Summary      Delete an account
// @Description  Soft-delete an account owned by the authenticated user, sweeping any remaining balance to the user's main account
// @Tags         accounts-manage
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  successResponse{data=deleteAccountResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      409  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /accounts/{id} [delete]
func (h *Handler) deleteAccount(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	account, err := h.Store.GetAccountById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("account not found"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	authPayload := core.GetAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	result, err := h.Store.DeleteAccountTx(ctx, store.DeleteAccountTxParams{
		AccountID: params.ID,
		UserID:    authPayload.ID,
	})
	if err != nil {
		core.Fail(ctx, deleteAccountAppError(err))
		return
	}

	resp := deleteAccountResponse{Account: toAccountResponse(result.Account)}
	if result.SweepTransfer != nil {
		resp.BalanceSweptToID = &result.SweepTransfer.ToAccountID
	}

	core.Succeed(ctx, http.StatusOK, resp, "Account deleted successfully")
}

// deleteAccountAppError maps a DeleteAccountTx failure to an AppError.
func deleteAccountAppError(err error) *core.AppError {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return core.NotFoundErr("account not found")
	case errors.Is(err, store.ErrCannotDeleteMainAccount):
		return core.ConflictErr(errCodeMainAccount, err.Error())
	default:
		return core.InternalErr(err)
	}
}
