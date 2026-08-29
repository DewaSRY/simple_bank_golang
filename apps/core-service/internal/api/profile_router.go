package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type profileResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// GetProfile godoc
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
func (server *Server) GetProfile(ctx *gin.Context) {
	authPayload := getAuthPayload(ctx)

	user, err := server.store.GetUserById(ctx, authPayload.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(ctx, NotFoundErr("user not found"))
			return
		}
		fail(ctx, InternalErr())
		return
	}

	succeed(ctx, http.StatusOK, profileResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, "Profile retrieved successfully")
}
