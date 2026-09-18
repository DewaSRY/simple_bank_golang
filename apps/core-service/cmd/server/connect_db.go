package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
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
func connectDB(cfg config.Config, logger *slog.Logger) *sql.DB {
	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		logger.Error("cannot open db", "error", err)
		os.Exit(1)
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

		logger.Warn("cannot ping db", "attempt", attempt, "max_attempts", DBConnectRetries, "error", pingErr)
		if attempt < DBConnectRetries {
			time.Sleep(DBConnectRetryInterval)
		}
	}

	conn.Close()
	logger.Error("cannot connect to db, giving up", "attempts", DBConnectRetries, "error", pingErr)
	os.Exit(1)
	return nil
}
