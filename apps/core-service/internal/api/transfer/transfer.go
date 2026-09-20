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

func (h *Handler) RegisterRoutes(_, authorized *gin.RouterGroup) {
	authorized.GET("/accounts/:id/entries", h.listAccountEntriesByAccountId)
	authorized.GET("/accounts/:id/recent-destinations", h.listRecentTransferDestinations)
	authorized.GET("/accounts/:id/transactions", h.listAccountTransactionHistory)
	authorized.POST("/accounts/:id/deposit", h.deposit)

	authorized.POST("/transactions/transfer", h.transactionTransfer)
}
