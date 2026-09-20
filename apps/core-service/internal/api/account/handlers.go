package account

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/api/core"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/util"
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
// @Success      200      {object}  core.successResponse{data=accountResponse}
// @Failure      400      {object}  core.errorResponse
// @Failure      401      {object}  core.errorResponse
// @Failure      500      {object}  core.errorResponse
// @Router       /accounts [post]
func (h *Handler) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := core.GetAuthPayload(ctx)

	account, err := h.Store.CreateAccountTx(ctx, store.CreateAccountTxParams{
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
		Name:        req.Name,
		Description: req.Description,
		IsMain:      false,
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, toAccountResponse(account), "Account created successfully")
}

type listmeAccountsQuery struct {
	core.PaginationQuery
	Name string `form:"name" binding:"omitempty"`
}

// listmeAccounts godoc
// @Summary      List accounts
// @Description  List accounts owned by the authenticated user, paginated
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        name    query     string  false  "Account name"
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  core.successResponse{data=[]accountuserResponse,meta=core.Meta}
// @Failure      400    {object}  core.errorResponse
// @Failure      401    {object}  core.errorResponse
// @Failure      500    {object}  core.errorResponse
// @Router       /accounts/me [get]
func (h *Handler) listmeAccounts(ctx *gin.Context) {
	var query listmeAccountsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := core.GetAuthPayload(ctx)

	name := sql.NullString{String: util.EscapeLikePattern(query.Name), Valid: true}

	accounts, err := h.Store.ListMeAccountsByUserId(ctx, db.ListMeAccountsByUserIdParams{
		Name: name,
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
		LimitCount:  query.Limit,
		OffsetCount: query.Offset(),
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	total, err := h.Store.ListMeAccountsByUserIdCount(ctx, db.ListMeAccountsByUserIdCountParams{
		Name: name,
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	responses := util.MapSlice(accounts, func(row db.ListMeAccountsByUserIdRow) accountuserResponse {
		return toAccountuserResponse(row.AccountUserDetailsView)
	})

	core.SucceedWithMeta(ctx, http.StatusOK, responses, "Accounts retrieved successfully", core.Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}

type searchAccountByNumberQuery struct {
	core.PaginationQuery
	Number string `form:"number" binding:"required"`
}

// searchAccountByNumber godoc
// @Summary      Search an account by number
// @Description  Look up a destination account by its account number, for picking a transfer destination
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        number  query     string  true  "Account number"
// @Param        page    query     int  false  "Page number"     default(1)
// @Param        limit   query     int  false  "Items per page"  default(10)
// @Success      200     {object}  core.successResponse{data=[]accountSearchResponse,meta=core.Meta}
// @Failure      400     {object}  core.errorResponse
// @Failure      401     {object}  core.errorResponse
// @Failure      404     {object}  core.errorResponse
// @Failure      500     {object}  core.errorResponse
// @Router       /accounts/search-by-number [get]
func (h *Handler) searchAccountByNumber(ctx *gin.Context) {
	var query searchAccountByNumberQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	number := sql.NullString{String: util.EscapeLikePattern(query.Number), Valid: true}

	accountList, err := h.Store.ListAccountsSearchByUserNumber(ctx, db.ListAccountsSearchByUserNumberParams{
		Number:      number,
		OffsetCount: query.Offset(),
		LimitCount:  query.Limit,
	})

	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	total, err := h.Store.CountAccountsSearchByUserNumber(ctx, number)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	responses := util.MapSlice(accountList, func(row db.ListAccountsSearchByUserNumberRow) accountSearchResponse {
		return toAccountSearchResponse(row.AccountUserDetailsView)
	})

	core.SucceedWithMeta(ctx, http.StatusOK, responses, "Accounts retrieved successfully", core.Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})

}
