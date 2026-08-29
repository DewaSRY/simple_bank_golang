package main

import (
	"log"

	_ "github.com/lib/pq"

	store "github.com/DewaSRY/core-service/db/store"
	api "github.com/DewaSRY/core-service/internal/api"
	config "github.com/DewaSRY/core-service/internal/config"
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

	if err := server.Start(cfg.ServerAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
