package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

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
		fail(ctx, InternalErr(err))
		return
	}

	succeed(ctx, http.StatusOK, toAccountResponse(account), "Account created successfully")
}

type listmeAccountsQuery struct {
	paginationQuery
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
// @Success      200    {object}  successResponse{data=[]accountuserResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/me [get]
func (server *Server) listmeAccounts(ctx *gin.Context) {
	var query listmeAccountsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := getAuthPayload(ctx)

	accounts, err := server.store.ListMeAccountsByUserId(ctx, db.ListMeAccountsByUserIdParams{
		Name: sql.NullString{String: query.Name, Valid: true},
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
		LimitCount:  query.Limit,
		OffsetCount: query.offset(),
	})
	if err != nil {
		fail(ctx, InternalErr(err))
		return
	}

	total, err := server.store.ListMeAccountsByUserIdCount(ctx, db.ListMeAccountsByUserIdCountParams{
		Name: sql.NullString{String: query.Name, Valid: true},
		UserID: sql.NullInt64{
			Int64: authPayload.ID,
			Valid: true,
		},
	})
	if err != nil {
		fail(ctx, InternalErr(err))
		return
	}

	responses := util.MapSlice(accounts, func(row db.ListMeAccountsByUserIdRow) accountuserResponse {
		return toAccountuserResponse(row.AccountUserDetailsView)
	})

	succeedWithMeta(ctx, http.StatusOK, responses, "Accounts retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}

type searchAccountByNumberQuery struct {
	paginationQuery
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
// @Success      200     {object}  successResponse{data=[]accountuserResponse,meta=Meta}
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

	accountList, err := server.store.ListAccountsSearchByUserNumber(ctx, db.ListAccountsSearchByUserNumberParams{
		Number:      sql.NullString{String: query.Number, Valid: true},
		OffsetCount: query.offset(),
		LimitCount:  query.Limit,
	})

	if err != nil {
		fail(ctx, InternalErr(err))
		return
	}

	total, err := server.store.CountAccountsSearchByUserNumber(ctx, sql.NullString{String: query.Number, Valid: true})
	if err != nil {
		fail(ctx, InternalErr(err))
		return
	}

	responses := util.MapSlice(accountList, func(row db.ListAccountsSearchByUserNumberRow) accountuserResponse {
		return toAccountuserResponse(row.AccountUserDetailsView)
	})

	succeedWithMeta(ctx, http.StatusOK, responses, "Accounts retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})

}
