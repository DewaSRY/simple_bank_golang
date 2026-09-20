package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/DewaSRY/core-service/internal/api/core"

	db "github.com/DewaSRY/core-service/internal/db/sqlc"
	"github.com/DewaSRY/core-service/internal/db/store"
)

type loginUserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// loginUser godoc
// @Summary      Login
// @Description  Authenticate by email. There is no password: the caller's
// @Description  browser/device fingerprint (derived from request headers)
// @Description  must match the one this account was registered/last bound
// @Description  with, or the request is rejected with 403.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      loginUserRequest  true  "Login credentials"
// @Success      200      {object}  core.successResponse{data=AuthResponse}
// @Failure      400      {object}  core.errorResponse
// @Failure      401      {object}  core.errorResponse
// @Failure      403      {object}  core.errorResponse
// @Failure      500      {object}  core.errorResponse
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
			core.Fail(ctx, core.UnauthorizedErr("invalid credentials"))
			return
		}
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	fingerprint := h.requestDeviceFingerprint(ctx)

	switch user.DeviceFingerprintHash {
	case fingerprint:
		// already bound to this device, nothing to do
	case "":
		// legacy/unbound row (pre-dates device binding) — bind it now
		bound, err := h.Store.BindUserDeviceFingerprint(ctx, db.BindUserDeviceFingerprintParams{
			ID:                    user.ID,
			DeviceFingerprintHash: fingerprint,
		})
		if err != nil {
			core.Fail(ctx, core.InternalErr(err))
			return
		}
		user.ID, user.Username, user.Email = bound.ID, bound.Username, bound.Email
	default:
		core.Fail(ctx, core.ForbiddenErr("this account is bound to a different device"))
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
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// registerUser godoc
// @Summary      Register
// @Description  Register a new user and return an access token. There is no
// @Description  password: the account is bound to the caller's current
// @Description  browser/device fingerprint, and future logins are only
// @Description  accepted from that same device.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      registerUserRequest  true  "Registration details"
// @Success      200      {object}  core.successResponse{data=AuthResponse}
// @Failure      400      {object}  core.errorResponse
// @Failure      401      {object}  core.errorResponse
// @Failure      500      {object}  core.errorResponse
// @Router       /auth/register [post]
func (h *Handler) registerUser(ctx *gin.Context) {
	var req registerUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.Fail(ctx, core.ValidationErr(core.FieldErrorsFromBindErr(err)...))
		return
	}

	_, err := h.Store.GetUserByEmail(ctx, req.Email)
	if err == nil {
		core.Fail(ctx, core.BadRequestErr("email_exists", "email already exists"))
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		core.Fail(ctx, core.InternalErr(err))
		return
	}

	usernameExists, err := h.Store.CheckIsUsernameExist(ctx, req.Username)
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
		return
	}
	if usernameExists {
		core.Fail(ctx, core.BadRequestErr("username_exists", "username already exists"))
		return
	}

	arg := db.CreateUserParams{
		Username:              req.Username,
		Email:                 req.Email,
		DeviceFingerprintHash: h.requestDeviceFingerprint(ctx),
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

	_, err = h.Store.CreateAccountTx(ctx, store.CreateAccountTxParams{
		UserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Name:   "Main Account",
		IsMain: true,
	})
	if err != nil {
		core.Fail(ctx, core.InternalErr(err))
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
// @Success      200  {object}  core.successResponse{data=profileResponse}
// @Failure      401  {object}  core.errorResponse
// @Failure      404  {object}  core.errorResponse
// @Failure      500  {object}  core.errorResponse
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
