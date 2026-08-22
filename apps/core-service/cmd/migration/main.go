package main

import (
	"fmt"
	"os/exec"

	"os"
)

func createMigrationFile(migrationName string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("migrate create -ext sql -dir db/migrations -seq %s", migrationName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

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

	default:
		fmt.Println("Unknown command:", args[0])
	}

}
