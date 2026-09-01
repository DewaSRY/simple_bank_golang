package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	constant "github.com/DewaSRY/core-service/domain/constant"
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

func convertToStringDecimal(t *testing.T, value string) string {
	result, err := decimal.NewFromString(value)
	require.NoError(t, err)
	return result.String()
}

func createTestQueries(t *testing.T) *Queries {
	tx, err := testDB.Begin()

	require.NoError(t, err)

	t.Cleanup(func() {
		tx.Rollback()
	})

	return New(tx)
}

func accountParams() CreateAccountParams {
	return CreateAccountParams{
		Balance:  "1000",
		Currency: "USD",
	}
}

func entriesParamsTypeSend() CreateEntriesParams {
	return CreateEntriesParams{
		AccountID: 1,
		Type:      constant.ENTRY_TYPE_SEND,
		Amount:    "100",
	}
}

func transferParams() CreateTransferParams {
	return CreateTransferParams{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        "100",
	}
}
