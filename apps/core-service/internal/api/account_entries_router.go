package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	db "github.com/DewaSRY/core-service/db/sqlc"
)

type paramsAccountEntries struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type queryAccountEntries struct {
	Limit  int32 `form:"limit" binding:"required"`
	Offset int32 `form:"offset" binding:"required"`
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
// @Success      200    {object}  successResponse{data=accountEntriesViewResponse,meta=Meta}
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      403    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /accounts/{id}/entries [get]
func (server *Server) listAccountEntriesByAccountId(ctx *gin.Context) {
	var params paramsAccountEntries
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
