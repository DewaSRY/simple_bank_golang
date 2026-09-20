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

// router is implemented by every domain module's Handler. public and
// authorized are the two v1 route groups (see bindRouters); a Handler with no
// endpoints on one of them simply ignores that argument.
type router interface {
	RegisterRoutes(public, authorized *gin.RouterGroup)
}

// bindRouters is the composition root: it builds the route groups and hands
// each one to the owning domain module's Handler. Adding a new endpoint to
// an existing module never touches this function — only that module's own
// RegisterRoutes. Adding a new module means one Handler construction plus
// one entry in the routers slice below.
func (server *Server) bindRouters(engine *gin.Engine) {

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := engine.Group("/api/v1")
	v1.GET("/health", server.health)

	// Authorized routes
	authorized := v1.Group("/")
	authorized.Use(core.AuthMiddleware(server.tokenMaker))

	routers := []router{
		&auth.Handler{
			Store:               server.store,
			TokenMaker:          server.tokenMaker,
			AccessTokenDuration: server.config.JWTAccessTokenDuration,
		},
		&account.Handler{Store: server.store},
		&transfer.Handler{Store: server.store},
	}

	for _, r := range routers {
		r.RegisterRoutes(v1, authorized)
	}
}

// health godoc
// @Summary      Health check
// @Description  Report service health status
// @Tags         health
// @Produce      json
// @Success      200  {object}  core.successResponse
// @Router       /health [get]
func (server *Server) health(ctx *gin.Context) {
	core.Succeed(ctx, http.StatusOK, gin.H{"status": "ok"}, "Service is healthy")
}
