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
