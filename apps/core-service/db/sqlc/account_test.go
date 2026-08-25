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

func TestUpdateAccountBalance(t *testing.T) {
	q := createTestQueries(t)

	arg := accountParams()

	// create account first
	account, err := q.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotNil(t, account)

	// update balance
	newBalance := "2000"
	updateAccount, err := q.UpdateAccountBalance(context.Background(), UpdateAccountBalanceParams{
		ID:      account.ID,
		Balance: newBalance,
	})
	require.NoError(t, err)
	require.NotNil(t, updateAccount)
	convertedNewBalance := convertToStringDecimal(t, newBalance)
	resultConvertedNewBalance := convertToStringDecimal(t, updateAccount.Balance)

	require.Equal(t, convertedNewBalance, resultConvertedNewBalance)

}
