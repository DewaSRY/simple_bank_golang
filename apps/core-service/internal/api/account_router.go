package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	mapper "github.com/DewaSRY/core-service/db/mapper"
	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
)

type createAccountRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// createAccount godoc
// @Summary      Create a new account
// @Description  Create a bank account for the authenticated user
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createAccountRequest  true  "Account creation payload"
// @Success      200      {object}  successResponse{data=accountResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /accounts [post]
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := getAuthPayload(ctx)

	account, err := server.store.CreateAccountTx(ctx, store.CreateAccountTxParams{
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
		Name:        req.Name,
		Description: req.Description,
		IsMain:      false,
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, toAccountResponse(account), "Account created successfully")
}

type getAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=0"`
}

// getAccount godoc
// @Summary      Get account by ID
// @Description  Retrieve a single account owned by the authenticated user
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  successResponse{data=accountResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /accounts/{id} [get]
func (server *Server) getAccount(ctx *gin.Context) {
	var params getAccountParams
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

	succeed(ctx, http.StatusOK, toAccountResponse(mapper.GetAccountByIdRowToAccount(account)), "Account retrieved successfully")
}

// listAccounts godoc
// @Summary      List accounts
// @Description  List accounts owned by the authenticated user, paginated
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]accountResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts [get]
func (server *Server) listAccounts(ctx *gin.Context) {
	var query paginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := getAuthPayload(ctx)

	accounts, err := server.store.ListAccountsByUserId(ctx, db.ListAccountsByUserIdParams{
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
		Limit:  query.Limit,
		Offset: query.offset(),
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	total, err := server.store.CountAccountsByUserId(ctx, sql.NullInt64{
		Int64: authPayload.ID,
		Valid: true,
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeedWithMeta(ctx, http.StatusOK, toListAccountResponse(accounts), "Accounts retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}

type updateAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateAccountRequest struct {
	Name        string `json:"name" binding:"omitempty"`
	Description string `json:"description" binding:"omitempty"`
}

// updateAccount godoc
// @Summary      Update an account
// @Description  Update the name and/or description of an account owned by the authenticated user
// @Tags         accounts
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
	var params updateAccountParams
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

type deleteAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type deleteAccountResponse struct {
	Account          accountResponse `json:"account"`
	BalanceSweptToID *int64          `json:"balance_swept_to_account_id,omitempty"`
}

// deleteAccount godoc
// @Summary      Delete an account
// @Description  Soft-delete an account owned by the authenticated user, sweeping any remaining balance to the user's main account
// @Tags         accounts
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
	var params deleteAccountParams
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

type depositParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type depositRequest struct {
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
}

// deposit godoc
// @Summary      Deposit into an account
// @Description  Deposit money into an account owned by the authenticated user
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int             true  "Account ID"
// @Param        request  body      depositRequest  true  "Deposit payload"
// @Success      200      {object}  successResponse{data=store.DepositTxResult}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      403      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /accounts/{id}/deposit [post]
func (server *Server) deposit(ctx *gin.Context) {
	var params depositParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	var req depositRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	if !req.Amount.IsPositive() {
		fail(ctx, ValidationErr(FieldError{Field: "amount", Message: "amount must be greater than zero"}))
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

	result, err := server.store.DepositTx(ctx, store.DepositTxParams{
		AccountID:   params.ID,
		Amount:      req.Amount.StringFixed(2),
		Description: req.Description,
	})
	if err != nil {
		fail(ctx, transferAppError(err))
		return
	}

	succeed(ctx, http.StatusOK, result, "Deposit completed successfully")
}

type listAccountEntriesParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// listAccountEntries godoc
// @Summary      List account entries
// @Description  List ledger entries for an account owned by the authenticated user, paginated
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int  true   "Account ID"
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]accountResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      403    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/{id}/entries [get]
func (server *Server) listAccountEntries(ctx *gin.Context) {
	var params listAccountEntriesParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	var query paginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
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

	entries, err := server.store.ListEntriesByAccount(ctx, db.ListEntriesByAccountParams{
		AccountID:   params.ID,
		LimitCount:  query.Limit,
		OffsetCount: query.offset(),
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	total, err := server.store.CountEntriesByAccount(ctx, params.ID)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeedWithMeta(ctx, http.StatusOK, entries, "Account entries retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}
