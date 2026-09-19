// Package transfer handles every endpoint that touches the ledger: deposits,
// cross-account transfers, and reading account entries/history.
package transfer

import (
	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/db/store"
)

// Handler holds everything the transfer endpoints need from the rest of the
// service.
type Handler struct {
	Store store.Storer
}

// idPathParam binds the ":id" path parameter shared by every single-account
// endpoint.
type idPathParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// RegisterRoutes registers deposit/transfer/entry/history endpoints on rg
// (the authorized API group).
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/accounts/:id/entries", h.listAccountEntriesByAccountId)
	rg.GET("/accounts/:id/recent-destinations", h.listRecentTransferDestinations)
	rg.GET("/accounts/:id/transactions", h.listAccountTransactionHistory)
	rg.POST("/accounts/:id/deposit", h.deposit)

	rg.POST("/transactions/transfer", h.transactionTransfer)
}
