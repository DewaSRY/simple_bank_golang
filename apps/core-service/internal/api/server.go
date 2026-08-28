package api

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	store "github.com/DewaSRY/core-service/db/store"
)

// Server wires HTTP handlers to the underlying store.
type Server struct {
	store  *store.Store
	router *gin.Engine
}

func NewServer(store *store.Store) *Server {
	registerValidatorFieldNames()

	server := &Server{store: store}

	router := gin.Default()
	router.Use(errorHandlerMiddleware())

	// binding gin controller to the router
	server.bindRouters(router)
	server.router = router
	return server
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
		return ""
	})
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
