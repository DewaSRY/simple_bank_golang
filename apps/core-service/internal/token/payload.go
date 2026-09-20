package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrExpiredToken = errors.New("token has expired")

// Payload is the JWT claims carried by every access token issued for a user.
// It stays small and free of sensitive data since JWTs are encoded, not encrypted.
type Payload struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// NewPayload builds a Payload for userID/username/email that expires after duration.
func NewPayload(userID int64, username, email string, duration time.Duration) *Payload {
	now := time.Now()
	return &Payload{
		ID:       userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        generateTokenID(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}
}

// generateTokenID returns a random hex string for the token's jti claim, so
// a specific issued token can be named later (a prerequisite for any future
// revocation mechanism — there is none today). crypto/rand failing means the
// OS entropy source is broken; a timestamp-based fallback still gives every
// token a distinct, merely non-random, ID instead of failing token issuance.
func generateTokenID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
