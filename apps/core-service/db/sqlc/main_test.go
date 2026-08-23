package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

var dbDriver = "postgres"
var dbSource = "postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable"

func TestMain(m *testing.M) {

	con, err := sql.Open(dbDriver, dbSource)

	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testDB = con

	os.Exit(m.Run())

}

func createTestQueries(t *testing.T) *Queries {
	tx, err := testDB.Begin()

	require.NoError(t, err)

	t.Cleanup(func() {
		tx.Rollback()
	})

	return New(tx)
}
