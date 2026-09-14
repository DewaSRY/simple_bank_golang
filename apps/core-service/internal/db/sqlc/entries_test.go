package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateEntry(t *testing.T) {
	q := createTestQueries(t)

	arg := entriesParamsTypeSend()

	// create account first
	account, err := q.CreateAccount(context.Background(), accountParams())
	require.NoError(t, err)
	require.NotNil(t, account)

	// set account id to entry params
	arg.AccountID = account.ID

	entry, err := q.CreateEntries(context.Background(), arg)

	require.NoError(t, err)
	require.NotNil(t, entry)
}
