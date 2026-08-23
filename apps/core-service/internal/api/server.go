package api

import (
	"github.com/gin-gonic/gin"

	store "github.com/DewaSRY/core-service/db/store"
)

// Server wires HTTP handlers to the underlying store.
type Server struct {
	store  *store.Store
	router *gin.Engine
}

func NewServer(store *store.Store) *Server {
	server := &Server{store: store}

	router := gin.Default()
	router.GET("/health", server.health)

	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func (server *Server) health(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"status": "ok"})
}
