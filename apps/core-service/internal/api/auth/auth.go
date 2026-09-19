// Package auth handles login, registration, and the authenticated user's
// profile.
package auth

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/token"
)

// Handler holds everything the auth endpoints need from the rest of the
// service.
type Handler struct {
	Store               store.Storer
	TokenMaker          token.Maker
	AccessTokenDuration time.Duration
}

// RegisterPublicRoutes registers the unauthenticated login/register endpoints
// on rg (the public API group).
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.loginUser)
	rg.POST("/auth/register", h.registerUser)
}

// RegisterAuthorizedRoutes registers the authenticated profile endpoint on rg
// (the authorized API group).
func (h *Handler) RegisterAuthorizedRoutes(rg *gin.RouterGroup) {
	rg.GET("/auth/profile", h.getProfile)
}
