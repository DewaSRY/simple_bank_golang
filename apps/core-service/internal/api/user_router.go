package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	db "github.com/DewaSRY/core-service/db/sqlc"
	"github.com/DewaSRY/core-service/pkg/utils"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func newUserResponse(user db.CreateUserRow) userResponse {
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

// createUser godoc
// @Summary      Create a new user
// @Description  Register a new user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      createUserRequest  true  "User registration payload"
// @Success      200      {object}  successResponse{data=userResponse}
// @Failure      400      {object}  errorResponse
// @Failure      409      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /users [post]
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, ValidationErr(fieldErrorsFromBindErr(err)...))
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		fail(ctx, InternalErr())
		return
	}

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

	succeed(ctx, http.StatusOK, newUserResponse(user), "User created successfully")
}
