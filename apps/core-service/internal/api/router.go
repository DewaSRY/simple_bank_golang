package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (server *Server) bindRouters(router *gin.Engine) {
	router.GET("/health", server.health)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// auth routes (public)
	router.POST("/users", server.createUser)

	router.POST("/auth/login", server.loginUser)
	router.POST("/auth/register", server.registerUser)

	authorized := router.Group("/")
	authorized.Use(authMiddleware(server.tokenMaker))

	// account routes
	authorized.POST("/accounts", server.createAccount)
	authorized.GET("/accounts/:id", server.getAccount)
	authorized.GET("/accounts", server.listAccounts)
	authorized.GET("/accounts/:id/entries", server.listAccountEntries)

	//transaction routes
	authorized.POST("/transactions/transfer", server.transactionTransfer)
	authorized.GET("/transactions/:id", server.getTransaction)
	authorized.GET("/transactions", server.listTransactions)
}

// health godoc
// @Summary      Health check
// @Description  Report service health status
// @Tags         health
// @Produce      json
// @Success      200  {object}  successResponse
// @Router       /health [get]
func (server *Server) health(ctx *gin.Context) {
	succeed(ctx, http.StatusOK, gin.H{"status": "ok"}, "Service is healthy")
}
