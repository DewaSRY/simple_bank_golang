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

func (h *Handler) RegisterRoutes(_, authorized *gin.RouterGroup) {
	authorized.POST("/accounts", h.createAccount)
	authorized.GET("/accounts/search-by-number", h.searchAccountByNumber)
	authorized.GET("/accounts/me", h.listmeAccounts)

	authorized.GET("/accounts/:id", h.detailAccount)
	authorized.PUT("/accounts/:id", h.updateAccount)
	authorized.DELETE("/accounts/:id", h.deleteAccount)
}
