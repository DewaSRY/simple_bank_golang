package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	q := createTestQueries(t)

	arg := CreateAccountParams{
		Owner:    "John Doe new",
		Balance:  1000,
		Currency: "USD",
	}

	account, err := q.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotNil(t, account)
}
