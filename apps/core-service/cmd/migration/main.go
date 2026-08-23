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
	if err != nil {
		fmt.Println("Migration up failed.")
		fmt.Println("Error:", err)
		fmt.Println("Output:", string(output))
		return
	}

	fmt.Println("Migration up completed.")

	fmt.Println(string(output))
}

func downMigration(dbURI string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate -path db/migrations -database %s down", dbURI))
	output, err := cmd.CombinedOutput()
	fmt.Println("Running migration down...")
	if err != nil {
		fmt.Println("Migration down failed.")
		fmt.Println("Error:", err)
		fmt.Println("Output:", string(output))
		return
	}

	fmt.Println("Migration down completed.")

	fmt.Println(string(output))
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
