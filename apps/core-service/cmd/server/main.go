package main

import (
	"log"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	api "github.com/DewaSRY/core-service/internal/api"
	config "github.com/DewaSRY/core-service/internal/config"
	store "github.com/DewaSRY/core-service/internal/db/store"
	_ "github.com/DewaSRY/core-service/internal/docs" // swagger docs
)

// @title Core Simple bank application
// @version 1.0
// @description API of simple bank application.

// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT access token.
func main() {
	// Run in Gin's release mode by default so production doesn't pay for the
	// debug logger/warnings. Set GIN_MODE=debug locally to opt back in.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// create dabase connection pool and verify connectivity
	conn := connectDB(cfg)
	defer conn.Close()

	// ping the database to verify connectivity
	if conn.Ping() != nil {
		log.Fatal("cannot ping db:", err)
	} else {
		log.Println("Successfully connected to the database")
	}

	store := store.NewStore(conn)
	server, err := api.NewServer(store, cfg)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// verify the port is free before starting the server
	listener, err := net.Listen("tcp", cfg.ServerAddress)
	if err != nil {
		log.Fatal("port already in use:", err)
	}
	listener.Close()

	if err := server.Start(cfg.ServerAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
