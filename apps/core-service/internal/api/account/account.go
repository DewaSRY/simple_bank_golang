// Package account handles account creation, listing/search, and
// single-account get/update/delete.
package account

import (
	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/db/store"
)

// Handler holds everything the account endpoints need from the rest of the
// service.
type Handler struct {
	Store store.Storer
}

// idPathParam binds the ":id" path parameter shared by every single-account
// endpoint.
type idPathParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// RegisterRoutes registers account creation/listing/search/get/update/delete
// endpoints on rg (the authorized API group).
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/accounts", h.createAccount)
	rg.GET("/accounts/search-by-number", h.searchAccountByNumber)
	rg.GET("/accounts/me", h.listmeAccounts)

	rg.GET("/accounts/:id", h.detailAccount)
	rg.PUT("/accounts/:id", h.updateAccount)
	rg.DELETE("/accounts/:id", h.deleteAccount)
}
