package token

import (
	"errors"
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
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}
}
