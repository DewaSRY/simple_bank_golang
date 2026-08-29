package api

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	db "github.com/DewaSRY/core-service/db/sqlc"
	store "github.com/DewaSRY/core-service/db/store"
	config "github.com/DewaSRY/core-service/internal/config"
	"github.com/DewaSRY/core-service/internal/token"
)

// Storer is everything a Server needs from the persistence layer: every
// sqlc query plus the store's hand-written transactions. *store.Store
// satisfies this automatically, and tests can swap in a mock instead.
type Storer interface {
	db.Querier
	TransferTx(ctx context.Context, arg db.CreateTransferParams) (store.TransferTxResult, error)
}

// Server wires HTTP handlers to the underlying store.
type Server struct {
	store      Storer
	config     config.Config
	tokenMaker token.Maker
	router     *gin.Engine
}

func NewServer(store Storer, cfg config.Config) (*Server, error) {
	registerValidatorFieldNames()

	tokenMaker, err := token.NewJWTMaker(cfg.JWTSecretKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{store: store, config: cfg, tokenMaker: tokenMaker}

	router := gin.Default()
	router.Use(errorHandlerMiddleware())

	// binding gin controller to the router
	server.bindRouters(router)
	server.router = router
	return server, nil
}

// registerValidatorFieldNames makes binding validation errors report the
// request's json/uri field name (e.g. "from_account_id") instead of the Go
// struct field name (e.g. "FromAccountID"), so normalized error responses
// point API consumers at the field they actually sent.
func registerValidatorFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		if name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]; name != "" && name != "-" {
			return name
		}
		if name := strings.SplitN(fld.Tag.Get("uri"), ",", 2)[0]; name != "" && name != "-" {
			return name
		}
		if name := strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]; name != "" && name != "-" {
			return name
		}
		return ""
	})
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
