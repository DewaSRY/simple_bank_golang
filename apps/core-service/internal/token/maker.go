package token

import "time"

// Maker manages the lifecycle of access tokens without exposing how the
// token is actually implemented (JWT, PASETO, ...) to its callers.
type Maker interface {
	CreateToken(userID int64, username, email string, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}
