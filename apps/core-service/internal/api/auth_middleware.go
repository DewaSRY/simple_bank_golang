package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/internal/token"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

var (
	errAuthHeaderMissing   = errors.New("authorization header is not provided")
	errAuthHeaderInvalid   = errors.New("authorization header format is invalid")
	errAuthTypeUnsupported = errors.New("unsupported authorization type")
)

// authMiddleware extracts and verifies the "Authorization: Bearer <token>"
// header, then stores the resulting token.Payload in the Gin context so
// downstream handlers can identify the caller via getAuthPayload(ctx).
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := parseAuthHeader(ctx.GetHeader(authorizationHeaderKey), tokenMaker)
		if err != nil {
			fail(ctx, UnauthorizedErr(err.Error()))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

func parseAuthHeader(authorizationHeader string, tokenMaker token.Maker) (*token.Payload, error) {
	if len(authorizationHeader) == 0 {
		return nil, errAuthHeaderMissing
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) != 2 {
		return nil, errAuthHeaderInvalid
	}

	if strings.ToLower(fields[0]) != authorizationTypeBearer {
		return nil, errAuthTypeUnsupported
	}

	payload, err := tokenMaker.VerifyToken(fields[1])
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// getAuthPayload returns the authenticated caller's token payload. It must
// only be called from handlers registered behind authMiddleware.
func getAuthPayload(ctx *gin.Context) *token.Payload {
	payload, ok := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if !ok {
		return nil
	}
	return payload
}
