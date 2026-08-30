package main

import (
	"fmt"
	"log"
	"os/exec"

	"os"

	config "github.com/DewaSRY/core-service/internal/config"
)

func createMigrationFile(migrationName string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate create -ext sql -dir db/migrations -seq %s", migrationName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Output:", string(output))
		return
	}

	fmt.Println(string(output))
}

func upMigration(dbURI string) {

	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate -path db/migrations -database %s up", dbURI))
	output, err := cmd.CombinedOutput()
	fmt.Println("Running migration up...")
	fmt.Println(string(output))
	if err != nil {
		// Note: the postgres driver already rolled back the failing migration
		// file's own transaction (see db/migrations/README.md). We must still
		// exit non-zero so callers (Makefile/CI) don't treat this as success.
		fmt.Println("Migration up failed.")
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Migration up completed.")
}

func downMigration(dbURI string) {
	cmd := exec.Command("migrate", "-path", "db/migrations", "-database", dbURI, "force", "1")
	output, err := cmd.CombinedOutput()
	fmt.Println("Running migration down...")
	fmt.Println(string(output))
	if err != nil {
		// Note: the postgres driver already rolled back the failing migration
		// file's own transaction (see db/migrations/README.md). We must still
		// exit non-zero so callers (Makefile/CI) don't treat this as success.
		fmt.Println("Migration down failed.")
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Migration down completed.")
}

func main() {

	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Please provide an argument.")
		return
	}

	switch args[0] {
	case "create":
		if len(args) < 2 {
			fmt.Println("Please provide a name for the migration.")
			return
		}

		createMigrationFile(args[1])
	case "up":
		cfg, err := config.LoadConfig(".")
		if err != nil {
			log.Fatal("cannot load config:", err)
		}
		upMigration(cfg.DBSource)
	case "down":
		cfg, err := config.LoadConfig(".")
		if err != nil {
			log.Fatal("cannot load config:", err)
		}
		downMigration(cfg.DBSource)
	default:
		fmt.Println("Unknown command:", args[0])
	}

}
