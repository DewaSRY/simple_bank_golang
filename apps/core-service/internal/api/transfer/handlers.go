package transfer

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/DewaSRY/core-service/internal/api/core"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/util"
)

func (h *Handler) requireOwnedAccount(ctx *gin.Context, accountID int64) (account db.Account, ok bool) {
	account, err := h.Store.GetAccountById(ctx, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("account not found"))
			return db.Account{}, false
		}
		core.Fail(ctx, core.InternalErr(err))
		return db.Account{}, false
	}

	authPayload := core.GetAuthPayload(ctx)
	if account.UserID.Int64 != authPayload.ID {
		core.Fail(ctx, core.ForbiddenErr("account does not belong to the authenticated user"))
		return db.Account{}, false
	}

	return account, true
}

func periodFromMonthYear(month, year int32) (start, end time.Time) {
	now := time.Now().UTC()
	if month == 0 {
		month = int32(now.Month())
	}
	if year == 0 {
		year = int32(now.Year())
	}
	start = time.Date(int(year), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

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
// @Success      200      {object}  successResponse{data=accountEntriesViewResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      403      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /accounts/{id}/deposit [post]
func (h *Handler) deposit(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	var req depositRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	if !req.Amount.IsPositive() {
		core.Fail(ctx, core.ValidationErr(core.FieldError{Field: "amount", Message: "amount must be greater than zero"}))
		return
	}

	if _, ok := h.requireOwnedAccount(ctx, params.ID); !ok {
		return
	}

	result, err := h.Store.DepositTx(ctx, store.DepositTxParams{
		AccountID:   params.ID,
		Amount:      req.Amount.StringFixed(2),
		Description: req.Description,
	})
	if err != nil {
		core.Fail(ctx, transferAppError(err))
		return
	}

	accountEntries, err := h.Store.AccountEntriesByAccountId(ctx, result.Entry.ID)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, toAccountEntriesViewResponse(accountEntries.AccountEntriesView), "Deposit completed successfully")
}

type listAccountEntriesQuery struct {
	core.PaginationQuery
	Month int32 `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int32 `form:"year" binding:"omitempty,min=1"`
}

// listAccountEntries godoc
// @Summary      List account entries
// @Description  List ledger entries for an account owned by the authenticated user, paginated
// @Tags         accounts-transaction
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int  true   "Account ID"
// @Param        month  query     int  false  "Month (1-12), defaults to the current month"
// @Param        year   query     int  false  "Year, defaults to the current year"
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  successResponse{data=[]accountEntriesViewResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      403    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/{id}/entries [get]
func (h *Handler) listAccountEntriesByAccountId(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	var query listAccountEntriesQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	periodStart, periodEnd := periodFromMonthYear(query.Month, query.Year)

	accountEntries, err := h.Store.ListAccountEntriesByAccountId(ctx, db.ListAccountEntriesByAccountIdParams{
		AccountID:   params.ID,
		PeriodStart: sql.NullTime{Time: periodStart, Valid: true},
		PeriodEnd:   sql.NullTime{Time: periodEnd, Valid: true},
		EntryType:   sql.NullString{},
		OffsetCount: query.Offset(),
		LimitCount:  query.Limit,
	})

	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	total, err := h.Store.CountAccountEntriesByAccountId(ctx, db.CountAccountEntriesByAccountIdParams{
		AccountID:   params.ID,
		PeriodStart: sql.NullTime{Time: periodStart, Valid: true},
		PeriodEnd:   sql.NullTime{Time: periodEnd, Valid: true},
		EntryType:   sql.NullString{},
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	entries := util.MapSlice(accountEntries, func(row db.ListAccountEntriesByAccountIdRow) accountEntriesViewResponse {
		return toAccountEntriesViewResponse(row.AccountEntriesView)
	})

	core.SucceedWithMeta(ctx, http.StatusOK, entries, "Account entries retrieved successfully", core.Meta{
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
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200  {object}  successResponse{data=[]publicAccountResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /accounts/{id}/recent-destinations [get]
func (h *Handler) listRecentTransferDestinations(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}
	var query core.PaginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	if _, ok := h.requireOwnedAccount(ctx, params.ID); !ok {
		return
	}

	destinations, err := h.Store.ListRecentTransferDestinations(ctx, db.ListRecentTransferDestinationsParams{
		FromAccountID: params.ID,
		LimitCount:    query.Limit,
		OffsetCount:   query.Offset(),
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	responses := util.MapSlice(destinations, toPublicAccountResponseFromDestination)

	core.Succeed(ctx, http.StatusOK, responses, "Recent transfer destinations retrieved successfully")
}

type listAccountTransactionHistoryQuery struct {
	core.PaginationQuery
	Month int32 `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int32 `form:"year" binding:"omitempty,min=1"`
}

// listAccountTransactionHistory godoc
// @Summary      List account transaction history
// @Description  List labeled transaction history for an account owned by the authenticated user, paginated
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
func (h *Handler) listAccountTransactionHistory(ctx *gin.Context) {
	var params idPathParam
	if err := ctx.ShouldBindUri(&params); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	var query listAccountTransactionHistoryQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	if _, ok := h.requireOwnedAccount(ctx, params.ID); !ok {
		return
	}

	periodStart, periodEnd := periodFromMonthYear(query.Month, query.Year)

	history, err := h.Store.ListAccountTransactionHistory(ctx, db.ListAccountTransactionHistoryParams{
		AccountID:   params.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		OffsetCount: query.Offset(),
		LimitCount:  query.Limit,
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	total, err := h.Store.CountAccountTransactionHistory(ctx, db.CountAccountTransactionHistoryParams{
		AccountID:   params.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	items := util.MapSlice(history, toTransactionHistoryItem)

	core.SucceedWithMeta(ctx, http.StatusOK, items, "Transaction history retrieved successfully", core.Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}
