package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeySize = 32

// JWTMaker is a JWT-backed implementation of Maker, signing tokens with HS256.
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker builds a JWTMaker. secretKey must be at least minSecretKeySize
// bytes long so tokens can't be brute-forced.
func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey}, nil
}

func (maker *JWTMaker) CreateToken(userID int64, username, email string, duration time.Duration) (string, *Payload, error) {
	payload := NewPayload(userID, username, email, duration)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := jwtToken.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", payload, err
	}

	return signedToken, payload, nil
}

func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, err
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return payload, nil
}
