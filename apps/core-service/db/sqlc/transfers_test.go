package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTransfer(t *testing.T) {
	q := createTestQueries(t)

	arg := transferParams()

	transfer, err := q.CreateTransfer(context.Background(), arg)

	require.NoError(t, err)
	require.NotNil(t, transfer)
}
