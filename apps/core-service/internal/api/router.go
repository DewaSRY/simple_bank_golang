package api

import "github.com/gin-gonic/gin"

func (server *Server) bindRouters(router *gin.Engine) {
	router.GET("/health", server.health)

	// account routes
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)

	//transaction routes
	router.POST("/transactions/transfer", server.transactionTransfer)
	// router.POST("/transactions", server.createTransaction)
	// router.GET("/transactions/:id", server.getTransaction)
	// router.GET("/transactions", server.listTransactions)

}

func (server *Server) health(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"status": "ok"})
}
