package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	q := createTestQueries(t)

	arg := accountParams()

	account, err := q.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotNil(t, account)
}

func TestGetAccountById(t *testing.T) {
	q := createTestQueries(t)

	arg := accountParams()

	// create account first
	account, err := q.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotNil(t, account)

	accountById, err := q.GetAccountById(context.Background(), account.ID)
	require.NoError(t, err)
	require.NotNil(t, accountById)
}

func TestIncrementAccountBalance(t *testing.T) {
	q := createTestQueries(t)

	arg := accountParams()

	// create account first
	account, err := q.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotNil(t, account)

	// increment balance: starting balance (1000, from accountParams) + 2000 = 3000
	incrementBy := "2000"
	updateAccount, err := q.IncrementAccountBalance(context.Background(), IncrementAccountBalanceParams{
		ID:      account.ID,
		Balance: incrementBy,
	})
	require.NoError(t, err)
	require.NotNil(t, updateAccount)

	expectedBalance := convertToStringDecimal(t, "3000")
	resultConvertedNewBalance := convertToStringDecimal(t, updateAccount.Balance)

	require.Equal(t, expectedBalance, resultConvertedNewBalance)

}
