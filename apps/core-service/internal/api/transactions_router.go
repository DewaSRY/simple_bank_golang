package api

import (
	"database/sql"
	"errors"

	"github.com/DewaSRY/core-service/db/store"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type createTransactionTransferRequest struct {
	FromAccountID int64           `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64           `json:"to_account_id" binding:"required,min=1"`
	Amount        decimal.Decimal `json:"amount"`
}

// transactionTransfer godoc
// @Summary      Transfer funds
// @Description  Transfer money from an account owned by the authenticated user to another account
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createTransactionTransferRequest  true  "Transfer payload"
// @Success      200      {object}  successResponse{data=store.TransferTxResult}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      403      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      409      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /transactions/transfer [post]
// func (server *Server) transactionTransfer(ctx *gin.Context) {
// 	var req createTransactionTransferRequest
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
// 		return
// 	}

// 	if !req.Amount.IsPositive() {
// 		fail(ctx, ValidationErr(FieldError{Field: "amount", Message: "amount must be greater than zero"}))
// 		return
// 	}

// 	fromAccount, err := server.store.GetAccountById(ctx, req.FromAccountID)
// 	if err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			fail(ctx, NotFoundErr("account not found"))
// 			return
// 		}
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	authPayload := getAuthPayload(ctx)
// 	if fromAccount.Owner != authPayload.Username {
// 		fail(ctx, ForbiddenErr("from_account does not belong to the authenticated user"))
// 		return
// 	}

// 	arg := db.CreateTransferParams{
// 		FromAccountID: req.FromAccountID,
// 		ToAccountID:   req.ToAccountID,
// 		Amount:        req.Amount.StringFixed(2),
// 	}

// 	result, err := server.store.TransferTx(ctx, arg)
// 	if err != nil {
// 		fail(ctx, transferAppError(err))
// 		return
// 	}

// 	succeed(ctx, http.StatusOK, result, "Transfer completed successfully")
// }

type getTransactionParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getTransaction godoc
// @Summary      Get transfer by ID
// @Description  Retrieve a single transfer involving an account owned by the authenticated user
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Transfer ID"
// @Success      200  {object}  successResponse{data=db.Transfer}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /transactions/{id} [get]
// func (server *Server) getTransaction(ctx *gin.Context) {
// 	var params getTransactionParams
// 	if err := ctx.ShouldBindUri(&params); err != nil {
// 		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
// 		return
// 	}

// 	transfer, err := server.store.GetTransferById(ctx, params.ID)
// 	if err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			fail(ctx, NotFoundErr("transfer not found"))
// 			return
// 		}
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	fromAccount, err := server.store.GetAccountById(ctx, transfer.FromAccountID)
// 	if err != nil {
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	toAccount, err := server.store.GetAccountById(ctx, transfer.ToAccountID)
// 	if err != nil {
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	authPayload := getAuthPayload(ctx)
// 	if fromAccount.Owner != authPayload.Username && toAccount.Owner != authPayload.Username {
// 		fail(ctx, ForbiddenErr("transfer does not belong to the authenticated user"))
// 		return
// 	}

// 	succeed(ctx, http.StatusOK, transfer, "Transfer retrieved successfully")
// }

// listTransactions godoc
// @Summary      List transfers
// @Description  List transfers involving accounts owned by the authenticated user, paginated
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]db.Transfer,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /transactions [get]
// func (server *Server) listTransactions(ctx *gin.Context) {
// 	var query paginationQuery
// 	if err := ctx.ShouldBindQuery(&query); err != nil {
// 		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
// 		return
// 	}

// 	authPayload := getAuthPayload(ctx)

// 	transfers, err := server.store.ListTransfersByOwner(ctx, db.ListTransfersByOwnerParams{
// 		Owner:       authPayload.Username,
// 		LimitCount:  query.Limit,
// 		OffsetCount: query.offset(),
// 	})
// 	if err != nil {
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	total, err := server.store.CountTransfersByOwner(ctx, authPayload.Username)
// 	if err != nil {
// 		fail(ctx, InternalErr())
// 		return
// 	}

// 	succeedWithMeta(ctx, http.StatusOK, transfers, "Transfer history retrieved successfully", Meta{
// 		Page: query.Page, Limit: query.Limit, Total: total,
// 	})
// }

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
