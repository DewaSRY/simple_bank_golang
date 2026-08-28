package store

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

const testDBSource = "postgresql://simple_bank:password@localhost:5433/simple_bank?sslmode=disable"

func TestMain(m *testing.M) {
	con, err := sql.Open("postgres", testDBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	testDB = con

	os.Exit(m.Run())
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(testDB)
}
