package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/DewaSRY/core-service/internal/api/core"
	"github.com/DewaSRY/core-service/internal/util"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
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
func (h *Handler) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	user, err := h.Store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.UnauthorizedErr("invalid username or password"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	if err := util.CheckPassword(req.Password, user.HashedPassword); err != nil {
		core.Fail(ctx, core.UnauthorizedErr("invalid username or password"))
		return
	}

	accessToken, _, err := h.TokenMaker.CreateToken(user.ID, user.Username, user.Email, h.AccessTokenDuration)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.AccessTokenDuration.Seconds()),
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
func (h *Handler) registerUser(ctx *gin.Context) {
	var req registerUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	// Check if the password and password confirmation match
	if req.Password != req.PasswordConfirm {
		core.Fail(ctx, core.BadRequestErr("password_mismatch", "password and password confirmation do not match"))
		return
	}

	// Check if the email already exists
	_, err := h.Store.GetUserByEmail(ctx, req.Email)
	if err == nil {
		core.Fail(ctx, core.BadRequestErr("email_exists", "email already exists"))
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	// Check if the username already exists
	usernameExists, err := h.Store.CheckIsUsernameExist(ctx, req.Username)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}
	if usernameExists {
		core.Fail(ctx, core.BadRequestErr("username_exists", "username already exists"))
		return
	}

	// Hash the password
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	// Create the user in the database
	arg := db.CreateUserParams{
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	}

	user, err := h.Store.CreateUser(ctx, arg)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			core.Fail(ctx, core.ConflictErr(core.ErrCodeConflict, "username or email already exists"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	// Create the user's Main Account. This is the only account flagged
	// IsMain: true — it can never be deleted, and receives the remaining
	// balance whenever another of the user's accounts is deleted.
	_, err = h.Store.CreateAccountTx(ctx, store.CreateAccountTxParams{
		UserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Name:   "Main Account",
		IsMain: true,
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	// create access token for the new user
	accessToken, _, err := h.TokenMaker.CreateToken(user.ID, user.Username, user.Email, h.AccessTokenDuration)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.AccessTokenDuration.Seconds()),
	}, "Registration successful")

}

type profileResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// getProfile godoc
// @Summary      Get profile
// @Description  Retrieve the profile of the authenticated user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  successResponse{data=profileResponse}
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /auth/profile [get]
func (h *Handler) getProfile(ctx *gin.Context) {
	authPayload := core.GetAuthPayload(ctx)

	user, err := h.Store.GetUserById(ctx, authPayload.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.Fail(ctx, core.NotFoundErr("user not found"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	core.Succeed(ctx, http.StatusOK, profileResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, "Profile retrieved successfully")
}
