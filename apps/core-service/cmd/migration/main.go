package main

import (
	"fmt"
	"os/exec"

	"os"
)

var DB_URI = "postgres://simple_bank:password@localhost:5433/simple_bank?sslmode=disable"

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

func upMigration() {

	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate -path db/migrations -database %s up", DB_URI))
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

func downMigration() {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate -path db/migrations -database %s down", DB_URI))
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
		upMigration()
	case "down":
		downMigration()
	default:
		fmt.Println("Unknown command:", args[0])
	}

}
