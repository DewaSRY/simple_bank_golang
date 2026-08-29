package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const testSecretKey = "12345678901234567890123456789012"

func TestJWTMakerCreateAndVerifyToken(t *testing.T) {
	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	userID := int64(42)
	username := "dewa"
	email := "dewa@example.com"
	duration := time.Minute

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(userID, username, email, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, payload)

	verifiedPayload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotNil(t, verifiedPayload)

	require.Equal(t, userID, verifiedPayload.ID)
	require.Equal(t, username, verifiedPayload.Username)
	require.Equal(t, email, verifiedPayload.Email)
	require.WithinDuration(t, issuedAt, verifiedPayload.IssuedAt.Time, time.Second)
	require.WithinDuration(t, expiresAt, verifiedPayload.ExpiresAt.Time, time.Second)
}

func TestJWTMakerExpiredToken(t *testing.T) {
	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	token, payload, err := maker.CreateToken(1, "dewa", "dewa@example.com", -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, payload)

	verifiedPayload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrExpiredToken)
	require.Nil(t, verifiedPayload)
}

func TestJWTMakerInvalidAlgNone(t *testing.T) {
	payload := NewPayload(1, "dewa", "dewa@example.com", time.Minute)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	verifiedPayload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.Nil(t, verifiedPayload)
}

func TestJWTMakerMalformedToken(t *testing.T) {
	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	verifiedPayload, err := maker.VerifyToken("this-is-not-a-jwt")
	require.Error(t, err)
	require.Nil(t, verifiedPayload)
}

func TestJWTMakerModifiedToken(t *testing.T) {
	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	token, _, err := maker.CreateToken(1, "dewa", "dewa@example.com", time.Minute)
	require.NoError(t, err)

	verifiedPayload, err := maker.VerifyToken(token + "tampered")
	require.Error(t, err)
	require.Nil(t, verifiedPayload)
}

func TestJWTMakerWrongSecretKey(t *testing.T) {
	maker, err := NewJWTMaker(testSecretKey)
	require.NoError(t, err)

	token, _, err := maker.CreateToken(1, "dewa", "dewa@example.com", time.Minute)
	require.NoError(t, err)

	otherMaker, err := NewJWTMaker("09876543210987654321098765432109")
	require.NoError(t, err)

	verifiedPayload, err := otherMaker.VerifyToken(token)
	require.Error(t, err)
	require.Nil(t, verifiedPayload)
}

func TestNewJWTMakerRejectsShortSecret(t *testing.T) {
	maker, err := NewJWTMaker("too-short")
	require.Error(t, err)
	require.Nil(t, maker)
}
