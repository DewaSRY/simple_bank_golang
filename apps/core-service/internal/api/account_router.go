package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	db "github.com/DewaSRY/core-service/db/sqlc"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,oneof=USD EUR GBP IDR"`
}

func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := getAuthPayload(ctx)

	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  "0",
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, account, "Account created successfully")
}

type getAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

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
	if account.Owner != authPayload.Username {
		fail(ctx, ForbiddenErr("account does not belong to the authenticated user"))
		return
	}

	succeed(ctx, http.StatusOK, account, "Account retrieved successfully")
}

func (server *Server) listAccounts(ctx *gin.Context) {
	var query paginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	authPayload := getAuthPayload(ctx)

	accounts, err := server.store.ListAccountsByOwner(ctx, db.ListAccountsByOwnerParams{
		Owner:       authPayload.Username,
		LimitCount:  query.Limit,
		OffsetCount: query.offset(),
	})
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	total, err := server.store.CountAccountsByOwner(ctx, authPayload.Username)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeedWithMeta(ctx, http.StatusOK, accounts, "Accounts retrieved successfully", Meta{
		Page: query.Page, Limit: query.Limit, Total: total,
	})
}

type listAccountEntriesParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

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
	if account.Owner != authPayload.Username {
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
