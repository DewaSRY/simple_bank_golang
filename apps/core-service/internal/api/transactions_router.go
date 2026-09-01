package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"

	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
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
// @Success      200      {object}  successResponse{data=store.TransferTxResult}
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

	succeed(ctx, http.StatusOK, result, "Transfer completed successfully")
}

// transferAppError maps a transferTx/depositTx failure to an AppError. This
// is the only piece of these endpoints that's specific to money movement —
// the actual response rendering is shared with every other endpoint via
// fail/AppError.
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

type searchAccountByNumberQuery struct {
	Number string `form:"number" binding:"required"`
}

// searchAccountByNumber godoc
// @Summary      Search an account by number
// @Description  Look up a destination account by its account number, for picking a transfer destination
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        number  query     string  true  "Account number"
// @Success      200     {object}  successResponse{data=publicAccountResponse}
// @Failure      400     {object}  errorResponse
// @Failure      401     {object}  errorResponse
// @Failure      404     {object}  errorResponse
// @Failure      500     {object}  errorResponse
// @Router       /accounts/search-by-number [get]
func (server *Server) searchAccountByNumber(ctx *gin.Context) {
	var query searchAccountByNumberQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	account, err := server.store.FindAccountByNumber(ctx, sql.NullString{String: query.Number, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("account not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, publicAccountResponse{
		ID:     account.ID,
		Name:   account.Name.String,
		Number: account.Number.String,
	}, "Account found")
}

type listRecentTransferDestinationsParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// listRecentTransferDestinations godoc
// @Summary      List recent transfer destinations
// @Description  List accounts recently transferred to from an account owned by the authenticated user
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Source account ID"
// @Success      200  {object}  successResponse{data=[]publicAccountResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /accounts/{id}/recent-destinations [get]
func (server *Server) listRecentTransferDestinations(ctx *gin.Context) {
	var params listRecentTransferDestinationsParams
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

	const recentDestinationsLimit = 5
	destinations, err := server.store.ListRecentTransferDestinations(ctx, db.ListRecentTransferDestinationsParams{
		FromAccountID: params.ID,
		LimitCount:    recentDestinationsLimit,
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	responses := make([]publicAccountResponse, len(destinations))
	for i, d := range destinations {
		responses[i] = publicAccountResponse{ID: d.ID, Name: d.Name.String, Number: d.Number.String}
	}

	succeed(ctx, http.StatusOK, responses, "Recent transfer destinations retrieved successfully")
}

type listAccountTransactionHistoryParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type listAccountTransactionHistoryQuery struct {
	paginationQuery
	Month int32 `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int32 `form:"year" binding:"omitempty,min=1"`
}

type transactionHistoryCounterparty struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}

type transactionHistoryItem struct {
	ID           int64                           `json:"id"`
	Label        string                          `json:"label"`
	Amount       string                          `json:"amount"`
	Currency     string                          `json:"currency"`
	Description  string                          `json:"description"`
	CreatedAt    string                          `json:"created_at"`
	Counterparty *transactionHistoryCounterparty `json:"counterparty"`
}

func toTransactionHistoryItem(row db.ListAccountTransactionHistoryRow) transactionHistoryItem {
	item := transactionHistoryItem{
		ID:          row.ID,
		Amount:      row.Amount,
		Currency:    "IDR",
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	switch row.Type {
	case "SEND":
		item.Label = "Transfer Out"
	case "RECEIVED":
		item.Label = "Transfer In"
	default:
		item.Label = "Deposit"
	}

	if row.CounterpartyAccountID.Valid {
		item.Counterparty = &transactionHistoryCounterparty{
			ID:     row.CounterpartyAccountID.Int64,
			Name:   row.CounterpartyAccountName.String,
			Number: row.CounterpartyAccountNumber.String,
		}
	}

	return item
}

// listAccountTransactionHistory godoc
// @Summary      List account transaction history
// @Description  List an account's deposit and transfer entries for a given month, defaulting to the current month
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int  true   "Account ID"
// @Param        month  query     int  false  "Month (1-12), defaults to the current month"
// @Param        year   query     int  false  "Year, defaults to the current year"
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]transactionHistoryItem,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      403    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/{id}/transactions [get]
func (server *Server) listAccountTransactionHistory(ctx *gin.Context) {
	var params listAccountTransactionHistoryParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	var query listAccountTransactionHistoryQuery
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

	now := time.Now().UTC()
	month := query.Month
	if month == 0 {
		month = int32(now.Month())
	}
	year := query.Year
	if year == 0 {
		year = int32(now.Year())
	}

	periodStart := time.Date(int(year), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	rows, err := server.store.ListAccountTransactionHistory(ctx, db.ListAccountTransactionHistoryParams{
		AccountID:   params.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		LimitCount:  query.Limit,
		OffsetCount: query.offset(),
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	total, err := server.store.CountAccountTransactionHistory(ctx, db.CountAccountTransactionHistoryParams{
		AccountID:   params.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	items := make([]transactionHistoryItem, len(rows))
	for i, row := range rows {
		items[i] = toTransactionHistoryItem(row)
	}

	succeedWithMeta(ctx, http.StatusOK, items, "Transaction history retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}
