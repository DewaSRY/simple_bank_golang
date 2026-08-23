package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testQueries *Queries

var dbDriver = "postgres"
var dbSource = "postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable"

func TestMain(m *testing.M) {

	con, err := sql.Open(dbDriver, dbSource)

	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(con)

	os.Exit(m.Run())

}
