package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/db/store"
)

type depositRequest struct {
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
}

// deposit godoc
// @Summary      Deposit into an account
// @Description  Deposit money into an account owned by the authenticated user
// @Tags         accounts-transaction
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
	var params manageAccountParams
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

// listAccountEntries godoc
// @Summary      List account entries
// @Description  List ledger entries for an account owned by the authenticated user, paginated
// @Tags         accounts-transaction
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int  true   "Account ID"
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]accountEntriesViewResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      403    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/{id}/entries [get]
func (server *Server) listAccountEntriesByAccountId(ctx *gin.Context) {
	var params manageAccountParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	var query paginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	accountEntries, err := server.store.ListAccountEntriesByAccountId(ctx, db.ListAccountEntriesByAccountIdParams{
		AccountID: params.ID,
		Limit:     query.Limit,
		Offset:    query.offset(),
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := server.store.CountAccountEntriesByAccountId(ctx, params.ID)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeedWithMeta(ctx, http.StatusOK, toListAccountEntriesViewResponse(accountEntries), "Account entries retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}

// listRecentTransferDestinations godoc
// @Summary      List recent transfer destinations
// @Description  List accounts recently transferred to from an account owned by the authenticated user
// @Tags         accounts-transaction
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

type listAccountTransactionHistoryQuery struct {
	paginationQuery
	Month int32 `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int32 `form:"year" binding:"omitempty,min=1"`
}

// listAccountTransactionHistory godoc
// @Summary      List account transaction history
// @Description  List an account's deposit and transfer entries for a given month, defaulting to the current month
// @Tags         accounts-transaction
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
	var params manageAccountParams
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
