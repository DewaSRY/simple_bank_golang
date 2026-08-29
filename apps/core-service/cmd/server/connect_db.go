package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"

	config "github.com/DewaSRY/core-service/internal/config"
	_ "github.com/DewaSRY/core-service/internal/docs" // swagger docs
)

const (
	DBMaxOpenConns         = 25
	DBMaxIdleConns         = 25
	DBConnMaxLifetime      = 5 * time.Minute
	DBConnMaxIdleTime      = 5 * time.Minute
	DBConnectTimeout       = 5 * time.Second
	DBConnectRetries       = 5
	DBConnectRetryInterval = 2 * time.Second
)

// connectDB opens the database connection pool, tunes it, and verifies
// connectivity with a bounded number of retries before giving up. This lets
// the service tolerate the DB not being ready yet (e.g. container startup
// order) while still failing fast on a genuinely bad/unreachable database
// instead of only discovering it on the first request.
func connectDB(cfg config.Config) *sql.DB {
	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		log.Fatal("cannot open db:", err)
	}

	conn.SetMaxOpenConns(DBMaxOpenConns)
	conn.SetMaxIdleConns(DBMaxIdleConns)
	conn.SetConnMaxLifetime(DBConnMaxLifetime)
	conn.SetConnMaxIdleTime(DBConnMaxIdleTime)

	var pingErr error
	for attempt := 1; attempt <= DBConnectRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), DBConnectTimeout)
		pingErr = conn.PingContext(ctx)
		cancel()

		if pingErr == nil {
			return conn
		}

		log.Printf("cannot ping db (attempt %d/%d): %v", attempt, DBConnectRetries, pingErr)
		if attempt < DBConnectRetries {
			time.Sleep(DBConnectRetryInterval)
		}
	}

	conn.Close()
	log.Fatalf("cannot connect to db after %d attempts: %v", DBConnectRetries, pingErr)
	return nil
}
