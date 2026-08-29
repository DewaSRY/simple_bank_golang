package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "secret123"

	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)
	require.NotEqual(t, password, hashedPassword)

	err = CheckPassword(password, hashedPassword)
	require.NoError(t, err)

	err = CheckPassword("wrong-password", hashedPassword)
	require.Error(t, err)
}

func TestHashPasswordProducesDifferentHashes(t *testing.T) {
	password := "secret123"

	hashedPassword1, err := HashPassword(password)
	require.NoError(t, err)

	hashedPassword2, err := HashPassword(password)
	require.NoError(t, err)

	require.NotEqual(t, hashedPassword1, hashedPassword2)
}
