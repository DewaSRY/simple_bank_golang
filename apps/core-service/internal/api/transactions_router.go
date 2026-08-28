package api

import (
	"fmt"

	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/gin-gonic/gin"
)

type createTransactionTransferRequest struct {
	FromAccountID int64   `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64   `json:"to_account_id" binding:"required,min=1"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

func (server *Server) transactionTransfer(ctx *gin.Context) {
	var req createTransactionTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	arg := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        fmt.Sprintf("%.2f", req.Amount),
	}

	if req.FromAccountID == req.ToAccountID {
		ctx.JSON(400, gin.H{"error": "from_account_id and to_account_id cannot be the same"})
		return
	}

	fromAccountExists, err := server.store.CheckIsAccountWithIdExist(ctx, req.FromAccountID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if !fromAccountExists {
		ctx.JSON(400, gin.H{"error": "from_account_id does not exist"})
		return
	}

	toAccountExists, err := server.store.CheckIsAccountWithIdExist(ctx, req.ToAccountID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if !toAccountExists {
		ctx.JSON(400, gin.H{"error": "to_account_id does not exist"})
		return
	}

	transaction, err := server.store.CreateTransfer(ctx, arg)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, transaction)

}
