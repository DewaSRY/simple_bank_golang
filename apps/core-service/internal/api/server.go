package api

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	config "github.com/DewaSRY/core-service/internal/config"
	store "github.com/DewaSRY/core-service/internal/db/store"
	"github.com/DewaSRY/core-service/internal/token"
)

// Storer is everything a Server needs from the persistence layer: every
// sqlc query plus the store's hand-written transactions. *store.Store
// satisfies this automatically, and tests can swap in a mock instead.

// Server wires HTTP handlers to the underlying store.
type Server struct {
	store      store.Storer
	config     config.Config
	tokenMaker token.Maker
	logger     *slog.Logger
	router     *gin.Engine
}

// NewServer wires a Server. log receives every request/error log line the
// server produces (see logging_middleware.go and errorHandlerMiddleware) —
// build it with internal/logger.New(cfg) so it's configured the same way
// (LOG_LEVEL/LOG_FORMAT) as the rest of the service. A nil log falls back to
// slog.Default(), which is convenient for tests that don't care about log
// output but shouldn't be relied on in production.
func NewServer(store store.Storer, cfg config.Config, log *slog.Logger) (*Server, error) {
	registerValidatorFieldNames()

	if log == nil {
		log = slog.Default()
	}

	tokenMaker, err := token.NewJWTMaker(cfg.JWTSecretKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{store: store, config: cfg, tokenMaker: tokenMaker, logger: log}

	// gin.New(), not gin.Default(): the default bundles gin's own
	// unstructured Logger()/Recovery(), which this replaces with
	// requestIDMiddleware/loggingMiddleware/recoveryMiddleware so every log
	// line (access log, panic, 500) is structured and carries request_id.
	//
	// recoveryMiddleware must be registered *after* errorHandlerMiddleware.
	// gin middleware nests via ctx.Next(), so a panic unwinds past every
	// ctx.Next() call up to the first recover() above it in the chain —
	// anything after ctx.Next() in a middleware registered *before*
	// recoveryMiddleware. errorHandlerMiddleware's response-rendering code
	// runs after its own ctx.Next() call, so it must sit above recovery in
	// the chain (registered first) to still run once recovery's recover()
	// hands control back to it; the other way around, a panic recovers into
	// a response that's never written and the client sees an empty 200.
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.Use(loggingMiddleware(log, cfg))
	router.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	router.Use(errorHandlerMiddleware(log))
	router.Use(recoveryMiddleware(log))

	// An unset CORS_ALLOWED_ORIGINS is indistinguishable from deliberately
	// disabling CORS at the config layer alone (docs/CONFIG_ENV_VARIABLE.md,
	// §4) — log which case this run is in, once, at startup.
	if len(cfg.CORSAllowedOrigins) == 0 {
		log.Info("CORS disabled: CORS_ALLOWED_ORIGINS is empty")
	} else {
		log.Info("CORS enabled", slog.Any("allowed_origins", cfg.CORSAllowedOrigins))
	}

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

// corsMiddleware allows browser clients on allowedOrigins to call this API.
// An empty list disables CORS entirely (no Access-Control-* headers are
// sent), which is the safe default for server-to-server deployments.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(ctx *gin.Context) { ctx.Next() }
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Timezone", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
