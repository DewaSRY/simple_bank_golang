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
	corelog "github.com/DewaSRY/core-service/internal/logger"
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
		// No Config yet, so no LOG_LEVEL/LOG_FORMAT to build a structured
		// logger from — the stdlib logger is the only option this early.
		log.Fatal("cannot load config:", err)
	}

	logger := corelog.New(cfg)

	// create dabase connection pool and verify connectivity
	conn := connectDB(cfg, logger)
	defer conn.Close()

	// ping the database to verify connectivity
	if err := conn.Ping(); err != nil {
		logger.Error("cannot ping db", "error", err)
		os.Exit(1)
	}
	logger.Info("successfully connected to the database")

	store := store.NewStore(conn)
	server, err := api.NewServer(store, cfg, logger)
	if err != nil {
		logger.Error("cannot create server", "error", err)
		os.Exit(1)
	}

	// verify the port is free before starting the server
	listener, err := net.Listen("tcp", cfg.ServerAddress)
	if err != nil {
		logger.Error("port already in use", "error", err, "address", cfg.ServerAddress)
		os.Exit(1)
	}
	listener.Close()

	logger.Info("starting server", "address", cfg.ServerAddress)
	if err := server.Start(cfg.ServerAddress); err != nil {
		logger.Error("cannot start server", "error", err)
		os.Exit(1)
	}
}
