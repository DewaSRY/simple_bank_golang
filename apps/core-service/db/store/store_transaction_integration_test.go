package store

import (
	"context"
	"sync"
	"testing"

	sqlc "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestTransferTxConcurrent runs many real transfers between two accounts
// concurrently against a live Postgres instance and checks the resulting
// balances against exact arithmetic. This is the test called for in
// NEED_TO_IMPOROVE.md #1/#3: table-driven mock tests can verify call
// sequencing, but only a real DB run exercises actual row locking and
// balance arithmetic under concurrency.
func TestTransferTxConcurrent(t *testing.T) {
	testStore := newTestStore(t)
	ctx := context.Background()

	fromAccount, err := testStore.CreateAccount(ctx, sqlc.CreateAccountParams{Owner: "concurrency-test-from", Balance: "1000.00", Currency: "USD"})
	require.NoError(t, err)
	toAccount, err := testStore.CreateAccount(ctx, sqlc.CreateAccountParams{Owner: "concurrency-test-to", Balance: "1000.00", Currency: "USD"})
	require.NoError(t, err)

	t.Cleanup(func() {
		testDB.ExecContext(ctx, "DELETE FROM entries WHERE account_id IN ($1, $2)", fromAccount.ID, toAccount.ID)
		testDB.ExecContext(ctx, "DELETE FROM transfers WHERE from_account_id IN ($1, $2) OR to_account_id IN ($1, $2)", fromAccount.ID, toAccount.ID)
		testDB.ExecContext(ctx, "DELETE FROM accounts WHERE id IN ($1, $2)", fromAccount.ID, toAccount.ID)
	})

	const numTransfers = 10
	const transferAmount = "50.00"

	errs := make(chan error, numTransfers)
	var wg sync.WaitGroup
	for i := 0; i < numTransfers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, txErr := testStore.TransferTx(ctx, sqlc.CreateTransferParams{
				FromAccountID: fromAccount.ID,
				ToAccountID:   toAccount.ID,
				Amount:        transferAmount,
			})
			errs <- txErr
		}()
	}
	wg.Wait()
	close(errs)

	for txErr := range errs {
		require.NoError(t, txErr)
	}

	finalFrom, err := testStore.GetAccountById(ctx, fromAccount.ID)
	require.NoError(t, err)
	finalTo, err := testStore.GetAccountById(ctx, toAccount.ID)
	require.NoError(t, err)

	fromBalance, err := decimal.NewFromString(finalFrom.Balance)
	require.NoError(t, err)
	toBalance, err := decimal.NewFromString(finalTo.Balance)
	require.NoError(t, err)

	expectedDelta := decimal.RequireFromString(transferAmount).Mul(decimal.NewFromInt(numTransfers))
	startingBalance := decimal.RequireFromString("1000.00")

	require.True(t, startingBalance.Sub(expectedDelta).Equal(fromBalance), "from balance: expected %s, got %s", startingBalance.Sub(expectedDelta), fromBalance)
	require.True(t, startingBalance.Add(expectedDelta).Equal(toBalance), "to balance: expected %s, got %s", startingBalance.Add(expectedDelta), toBalance)
}

// TestTransferTxConcurrentReverse fires transfers in both directions between
// the same two accounts at once. Without the fixed ascending-ID lock order
// in transferTx, this is the classic shape that deadlocks two transactions
// each holding one row and waiting on the other's.
func TestTransferTxConcurrentReverse(t *testing.T) {
	testStore := newTestStore(t)
	ctx := context.Background()

	accountA, err := testStore.CreateAccount(ctx, sqlc.CreateAccountParams{Owner: "concurrency-test-a", Balance: "1000.00", Currency: "USD"})
	require.NoError(t, err)
	accountB, err := testStore.CreateAccount(ctx, sqlc.CreateAccountParams{Owner: "concurrency-test-b", Balance: "1000.00", Currency: "USD"})
	require.NoError(t, err)

	t.Cleanup(func() {
		testDB.ExecContext(ctx, "DELETE FROM entries WHERE account_id IN ($1, $2)", accountA.ID, accountB.ID)
		testDB.ExecContext(ctx, "DELETE FROM transfers WHERE from_account_id IN ($1, $2) OR to_account_id IN ($1, $2)", accountA.ID, accountB.ID)
		testDB.ExecContext(ctx, "DELETE FROM accounts WHERE id IN ($1, $2)", accountA.ID, accountB.ID)
	})

	const numTransfersEachDirection = 10
	const transferAmount = "50.00"

	errs := make(chan error, numTransfersEachDirection*2)
	var wg sync.WaitGroup
	for i := 0; i < numTransfersEachDirection; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, txErr := testStore.TransferTx(ctx, sqlc.CreateTransferParams{FromAccountID: accountA.ID, ToAccountID: accountB.ID, Amount: transferAmount})
			errs <- txErr
		}()
		go func() {
			defer wg.Done()
			_, txErr := testStore.TransferTx(ctx, sqlc.CreateTransferParams{FromAccountID: accountB.ID, ToAccountID: accountA.ID, Amount: transferAmount})
			errs <- txErr
		}()
	}
	wg.Wait()
	close(errs)

	for txErr := range errs {
		require.NoError(t, txErr)
	}

	finalA, err := testStore.GetAccountById(ctx, accountA.ID)
	require.NoError(t, err)
	finalB, err := testStore.GetAccountById(ctx, accountB.ID)
	require.NoError(t, err)

	// Equal transfer counts in both directions should net out to the
	// starting balance on both sides.
	require.Equal(t, "1000.00", finalA.Balance)
	require.Equal(t, "1000.00", finalB.Balance)
}
