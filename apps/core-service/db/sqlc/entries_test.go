package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateEntry(t *testing.T) {
	q := createTestQueries(t)

	arg := entriesParamsTypeSend()

	entry, err := q.CreateEntries(context.Background(), arg)

	require.NoError(t, err)
	require.NotNil(t, entry)
}
