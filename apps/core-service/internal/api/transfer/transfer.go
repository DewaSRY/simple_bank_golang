// Package transfer handles every endpoint that touches the ledger: deposits,
// cross-account transfers, and reading account entries/history.
package transfer

import (
	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/db/store"
)

type Handler struct {
	Store store.Storer
}

type idPathParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// account transfer
	rg.GET("/accounts/:id/entries", h.listAccountEntriesByAccountId)
	rg.GET("/accounts/:id/recent-destinations", h.listRecentTransferDestinations)
	rg.GET("/accounts/:id/transactions", h.listAccountTransactionHistory)
	rg.POST("/accounts/:id/deposit", h.deposit)

	rg.POST("/transactions/transfer", h.transactionTransfer)
}
