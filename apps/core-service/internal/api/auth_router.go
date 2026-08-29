package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/DewaSRY/core-service/pkg/utils"

	db "github.com/DewaSRY/core-service/db/sqlc"
)

type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
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
// @Success      200      {object}  successResponse{data=AuthResponse}
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

	user, err := server.store.GetUserByEmail(ctx, req.Email)
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

	succeed(ctx, http.StatusOK, AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(server.config.JWTAccessTokenDuration.Seconds()),
	}, "Login successful")
}

type registerUserRequest struct {
	Username        string `json:"username" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	PasswordConfirm string `json:"password_confirm" binding:"required,eqfield=Password"`
}

// registerUser godoc
// @Summary      Register
// @Description  Register a new user and return an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      registerUserRequest  true  "Registration details"
// @Success      200      {object}  successResponse{data=AuthResponse}
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /auth/register [post]
func (server *Server) registerUser(ctx *gin.Context) {
	var req registerUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	// Check if the password and password confirmation match
	if req.Password != req.PasswordConfirm {
		fail(ctx, BadRequestErr("password_mismatch", "password and password confirmation do not match"))
		return
	}

	// Check if the email already exists
	_, err := server.store.GetUserByEmail(ctx, req.Email)
	if err == nil {
		fail(ctx, BadRequestErr("email_exists", "email already exists"))
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		fail(ctx, InternalErr())
		return
	}

	// Check if the username already exists
	usernameExists, err := server.store.CheckIsUsernameExist(ctx, req.Username)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}
	if usernameExists {
		fail(ctx, BadRequestErr("username_exists", "username already exists"))
		return
	}

	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	// Create the user in the database
	arg := db.CreateUserParams{
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			fail(ctx, ConflictErr(errCodeConflict, "username or email already exists"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	// create access token for the new user
	accessToken, _, err := server.tokenMaker.CreateToken(user.ID, user.Username, user.Email, server.config.JWTAccessTokenDuration)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(server.config.JWTAccessTokenDuration.Seconds()),
	}, "Registration successful")

}
