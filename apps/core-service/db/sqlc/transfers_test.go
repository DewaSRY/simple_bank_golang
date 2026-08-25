package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTransfer(t *testing.T) {
	q := createTestQueries(t)

	arg := transferParams()

	// create account first
	accountFrom, err := q.CreateAccount(context.Background(), accountParams())
	require.NoError(t, err)

	accountTo, err := q.CreateAccount(context.Background(), accountParams())
	require.NoError(t, err)

	// set account ids to transfer params
	arg.FromAccountID = accountFrom.ID
	arg.ToAccountID = accountTo.ID

	transfer, err := q.CreateTransfer(context.Background(), arg)

	require.NoError(t, err)
	require.NotNil(t, transfer)
}
