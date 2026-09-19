package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/DewaSRY/core-service/internal/api/account"
	"github.com/DewaSRY/core-service/internal/api/auth"
	"github.com/DewaSRY/core-service/internal/api/core"
	"github.com/DewaSRY/core-service/internal/api/transfer"
)

// bindRouters is the composition root: it builds the route groups and hands
// each one to the owning domain module's Handler. Adding a new endpoint to
// an existing module never touches this function — only that module's own
// RegisterRoutes. Adding a new module means one Handler construction plus
// one RegisterRoutes call here.
func (server *Server) bindRouters(router *gin.Engine) {
	router.GET("/health", server.health)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/api/v1")

	authHandler := &auth.Handler{
		Store:               server.store,
		TokenMaker:          server.tokenMaker,
		AccessTokenDuration: server.config.JWTAccessTokenDuration,
	}
	authHandler.RegisterPublicRoutes(v1)

	// Authorized routes
	authorized := v1.Group("/")
	authorized.Use(core.AuthMiddleware(server.tokenMaker))

	authHandler.RegisterAuthorizedRoutes(authorized)

	accountHandler := &account.Handler{Store: server.store}
	accountHandler.RegisterRoutes(authorized)

	transferHandler := &transfer.Handler{Store: server.store}
	transferHandler.RegisterRoutes(authorized)
}

// health godoc
// @Summary      Health check
// @Description  Report service health status
// @Tags         health
// @Produce      json
// @Success      200  {object}  successResponse
// @Router       /health [get]
func (server *Server) health(ctx *gin.Context) {
	core.Succeed(ctx, http.StatusOK, gin.H{"status": "ok"}, "Service is healthy")
}
