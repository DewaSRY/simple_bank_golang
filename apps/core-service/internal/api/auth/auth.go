package auth

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/token"
)

type Handler struct {
	Store               store.Storer
	TokenMaker          token.Maker
	AccessTokenDuration time.Duration

	TrustedIPs []string
}

func (h *Handler) RegisterRoutes(public, authorized *gin.RouterGroup) {
	public.POST("/auth/login", h.loginUser)
	public.POST("/auth/register", h.registerUser)

	authorized.GET("/auth/profile", h.getProfile)
}
