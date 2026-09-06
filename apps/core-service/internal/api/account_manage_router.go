package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
)

type manageAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

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
func (server *Server) detailAccount(ctx *gin.Context) {
	var params manageAccountParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	account, err := server.store.GetAccountViewById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("account not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	authPayload := getAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	succeed(ctx, http.StatusOK, toPublicAccountResponse(account), "Account retrieved successfully")
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
func (server *Server) updateAccount(ctx *gin.Context) {
	var params manageAccountParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	var req updateAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	account, err := server.store.GetAccountById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("account not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	authPayload := getAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
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

	updated, err := server.store.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:          params.ID,
		Name:        sql.NullString{String: name, Valid: name != ""},
		Description: sql.NullString{String: description, Valid: description != ""},
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, toAccountResponse(mapper.UpdateAccountRowToAccount(updated)), "Account updated successfully")
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
func (server *Server) deleteAccount(ctx *gin.Context) {
	var params manageAccountParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	account, err := server.store.GetAccountById(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("account not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	authPayload := getAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	result, err := server.store.DeleteAccountTx(ctx, store.DeleteAccountTxParams{
		AccountID: params.ID,
		UserID:    authPayload.ID,
	})
	if err != nil {
		fail(ctx, deleteAccountAppError(err))
		return
	}

	resp := deleteAccountResponse{Account: toAccountResponse(result.Account)}
	if result.SweepTransfer != nil {
		resp.BalanceSweptToID = &result.SweepTransfer.ToAccountID
	}

	succeed(ctx, http.StatusOK, resp, "Account deleted successfully")
}

// deleteAccountAppError maps a DeleteAccountTx failure to an AppError.
func deleteAccountAppError(err error) *AppError {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return NotFoundErr("account not found")
	case errors.Is(err, store.ErrCannotDeleteMainAccount):
		return ConflictErr(errCodeMainAccount, err.Error())
	default:
		return InternalErr()
	}
}
