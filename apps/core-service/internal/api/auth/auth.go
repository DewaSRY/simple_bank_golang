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

func (h *Handler) RegisterRoutes(public, authorized *gin.RouterGroup) {
	public.POST("/auth/login", h.loginUser)
	public.POST("/auth/register", h.registerUser)

	authorized.GET("/auth/profile", h.getProfile)
}
