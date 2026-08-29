package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	store "github.com/DewaSRY/core-service/db/store"
	api "github.com/DewaSRY/core-service/internal/api"
	config "github.com/DewaSRY/core-service/internal/config"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
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
