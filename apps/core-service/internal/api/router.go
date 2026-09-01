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

	// API v1 routes
	v1 := router.Group("/api/v1")

	v1.POST("/auth/login", server.loginUser)
	v1.POST("/auth/register", server.registerUser)

	// Authorized routes
	authorized := v1.Group("/")
	authorized.Use(authMiddleware(server.tokenMaker))

	// Profile routes
	authorized.GET("/auth/profile", server.GetProfile)

	// Account routes
	authorized.POST("/accounts", server.createAccount)
	authorized.GET("/accounts/search-by-number", server.searchAccountByNumber)
	authorized.PUT("/accounts/:id", server.updateAccount)
	authorized.DELETE("/accounts/:id", server.deleteAccount)
	authorized.GET("/accounts", server.listAccounts)
	authorized.GET("/accounts/:id/entries", server.listAccountEntries)
	authorized.GET("/accounts/:id/transactions", server.listAccountTransactionHistory)
	authorized.GET("/accounts/:id/recent-destinations", server.listRecentTransferDestinations)
	authorized.POST("/accounts/:id/deposit", server.deposit)

	// Transaction routes
	authorized.POST("/transactions/transfer", server.transactionTransfer)
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
