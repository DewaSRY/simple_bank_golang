package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DewaSRY/core-service/pkg/utils"
)

type loginUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginUserResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// loginUser godoc
// @Summary      Login
// @Description  Authenticate a user and return an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      loginUserRequest  true  "Login credentials"
// @Success      200      {object}  successResponse{data=loginUserResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /auth/login [post]
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	user, err := server.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, UnauthorizedErr("invalid username or password"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	if err := utils.CheckPassword(req.Password, user.HashedPassword); err != nil {
		fail(ctx, UnauthorizedErr("invalid username or password"))
		return
	}

	accessToken, _, err := server.tokenMaker.CreateToken(user.ID, user.Username, user.Email, server.config.JWTAccessTokenDuration)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, loginUserResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(server.config.JWTAccessTokenDuration.Seconds()),
	}, "Login successful")
}
