package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (server *Server) bindRouters(router *gin.Engine) {
	router.GET("/health", server.health)

	// auth routes (public)
	router.POST("/users", server.createUser)
	router.POST("/auth/login", server.loginUser)

	authorized := router.Group("/")
	authorized.Use(authMiddleware(server.tokenMaker))

	// account routes
	authorized.POST("/accounts", server.createAccount)
	authorized.GET("/accounts/:id", server.getAccount)

	//transaction routes
	authorized.POST("/transactions/transfer", server.transactionTransfer)
	// authorized.POST("/transactions", server.createTransaction)
	// authorized.GET("/transactions/:id", server.getTransaction)
	// authorized.GET("/transactions", server.listTransactions)

}

func (server *Server) health(ctx *gin.Context) {
	succeed(ctx, http.StatusOK, gin.H{"status": "ok"}, "Service is healthy")
}
