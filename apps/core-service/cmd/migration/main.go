package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"

	"os"

	config "github.com/DewaSRY/core-service/internal/config"
	corelog "github.com/DewaSRY/core-service/internal/logger"
)

func createMigrationFile(logger *slog.Logger, migrationName string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate create -ext sql -dir internal/db/migrations -seq %s", migrationName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("migration create failed", "error", err, "output", string(output))
		return
	}

	logger.Info("migration create completed", "output", string(output))
}

func upMigration(logger *slog.Logger, dbURI string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate -path internal/db/migrations -database %s up", dbURI))
	output, err := cmd.CombinedOutput()
	logger.Info("running migration up", "output", string(output))
	if err != nil {
		// Note: the postgres driver already rolled back the failing migration
		// file's own transaction (see internal/db/migrations/README.md). We must still
		// exit non-zero so callers (Makefile/CI) don't treat this as success.
		logger.Error("migration up failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migration up completed")
}

func downMigration(logger *slog.Logger, dbURI string) {
	cmd := exec.Command(
		"migrate",
		"-path", "internal/db/migrations",
		"-database", dbURI,
		"down", "1",
	)

	output, err := cmd.CombinedOutput()

	logger.Info("running migration down", "output", string(output))

	if err != nil {
		logger.Error("migration down failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migration down completed")
}

func forceMigration(logger *slog.Logger, dbURI string, version int) {
	cmd := exec.Command("migrate", "-path", "internal/db/migrations", "-database", dbURI, "force", fmt.Sprintf("%d", version))
	output, err := cmd.CombinedOutput()
	logger.Info("running migration force", "output", string(output))
	if err != nil {
		logger.Error("migration force failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migration force completed")
}

func migrateGotoversion(logger *slog.Logger, dbURI string, version int) {
	cmd := exec.Command("migrate", "-path", "internal/db/migrations", "-database", dbURI, "goto", fmt.Sprintf("%d", version))
	output, err := cmd.CombinedOutput()
	logger.Info("running migration goto version", "output", string(output))
	if err != nil {
		logger.Error("migration goto version failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migration goto version completed")
}

func main() {
	args := os.Args[1:]

	// No Config yet at this point for any command, so build the logger from
	// zero-value config.Config (defaults to info/JSON, same fallback
	// internal/logger.New already applies for an empty LOG_LEVEL/LOG_FORMAT).
	logger := corelog.New(config.Config{})

	if len(args) == 0 {
		logger.Error("please provide an argument")
		return
	}

	switch args[0] {
	case "create":
		if len(args) < 2 {
			logger.Error("please provide a name for the migration")
			return
		}

		createMigrationFile(logger, args[1])
	case "up":
		cfg, err := config.LoadConfig(".")
		if err != nil {
			logger.Error("cannot load config", "error", err)
			os.Exit(1)
		}
		upMigration(logger, cfg.DBSource)
	case "down":
		cfg, err := config.LoadConfig(".")
		if err != nil {
			logger.Error("cannot load config", "error", err)
			os.Exit(1)
		}
		downMigration(logger, cfg.DBSource)
	case "force":
		if len(args) < 2 {
			logger.Error("please provide a version for the force migration")
			return
		}
		version, err := strconv.Atoi(args[1])
		if err != nil {
			logger.Error("invalid version", "version", args[1])
			return
		}
		cfg, err := config.LoadConfig(".")
		if err != nil {
			logger.Error("cannot load config", "error", err)
			os.Exit(1)
		}
		forceMigration(logger, cfg.DBSource, version)

	case "goto":
		if len(args) < 2 {
			logger.Error("please provide a version for the goto migration")
			return
		}
		version, err := strconv.Atoi(args[1])
		if err != nil {
			logger.Error("invalid version", "version", args[1])
			return
		}
		cfg, err := config.LoadConfig(".")
		if err != nil {
			logger.Error("cannot load config", "error", err)
			os.Exit(1)
		}
		migrateGotoversion(logger, cfg.DBSource, version)
	default:
		logger.Error("unknown command", "command", args[0])
	}
}
