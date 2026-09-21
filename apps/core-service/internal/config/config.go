package config

import (
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config holds all configuration values, loaded from a file or environment variables.
type Config struct {
	DBDriver               string        `mapstructure:"DB_DRIVER"`
	DBSource               string        `mapstructure:"DB_SOURCE"`
	ServerAddress          string        `mapstructure:"SERVER_ADDRESS"`
	JWTSecretKey           string        `mapstructure:"JWT_SECRET_KEY"`
	JWTAccessTokenDuration time.Duration `mapstructure:"JWT_ACCESS_TOKEN_DURATION"`
	CORSAllowedOrigins     []string      `mapstructure:"CORS_ALLOWED_ORIGINS"`

	// LogLevel controls which log records are emitted: "debug", "info",
	// "warn", or "error" (case-insensitive). Empty/unrecognized defaults to
	// "info" — see internal/logger.New.
	LogLevel string `mapstructure:"LOG_LEVEL"`
	// LogFormat selects the log encoding: "json" (default, machine-parseable)
	// or "text" (human-readable, handy for local dev).
	LogFormat string `mapstructure:"LOG_FORMAT"`
	// LogPrettyJSON indents every JSON log line for readability instead of
	// emitting the usual one-line-per-record form. Only meaningful when
	// LogFormat is "json" (the default); intended for local development,
	// not a log aggregator, which expects one line per record. See
	// internal/logger.New.
	LogPrettyJSON bool `mapstructure:"LOG_PRETTY_JSON"`
	// LogRequestBody/LogResponseBody enable capturing request/response
	// bodies on the per-request access log line. See
	// internal/api/logging_middleware.go.
	LogRequestBody  bool `mapstructure:"LOG_REQUEST_BODY"`
	LogResponseBody bool `mapstructure:"LOG_RESPONSE_BODY"`
	// LogMaxBodySize caps how many bytes of a request/response body are
	// captured for logging, in bytes. Zero/unset falls back to 1MiB — see
	// internal/api/logging_middleware.go.
	LogMaxBodySize int64 `mapstructure:"LOG_MAX_BODY_SIZE"`

	// RateLimitEnabled turns the global per-IP rate limiter on/off without a
	// code change (e.g. disable it for load tests). See
	// internal/api/core/rate_limit_middleware.go.
	RateLimitEnabled bool `mapstructure:"RATE_LIMIT_ENABLED"`
	// RateLimitRequestsPerSecond is the sustained per-IP request rate the
	// token bucket refills at.
	RateLimitRequestsPerSecond float64 `mapstructure:"RATE_LIMIT_REQUESTS_PER_SECOND"`
	// RateLimitBurst is the token bucket's capacity: how many requests a
	// single IP can make in a burst before being throttled to
	// RateLimitRequestsPerSecond.
	RateLimitBurst int `mapstructure:"RATE_LIMIT_BURST"`

	// DeviceFingerprintTrustedIPs is a comma-separated list of client IPs
	// (matched against gin's ctx.ClientIP()) that are exempt from the
	// device-fingerprint binding check in internal/api/auth: a request from
	// one of these IPs always gets a fixed, well-known fingerprint instead
	// of one derived from headers. This exists purely so local tooling
	// (Swagger UI, curl) can register/login repeatedly without being
	// rejected as "bound to a different device" every time headers change.
	// Empty/unset (the default) disables the bypass entirely — never set
	// this in a production environment.
	DeviceFingerprintTrustedIPs []string `mapstructure:"DEVICE_FINGERPRINT_TRUSTED_IPS"`
}

// LoadConfig reads configuration from an optional app.env file
// and/or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	// Enable reading from environment variables.
	viper.AutomaticEnv()

	// app.env is optional. If it doesn't exist, continue and
	// rely on environment variables instead.
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	// AutomaticEnv only checks the environment for keys viper already
	// knows about (from the config file). When app.env isn't present,
	// viper knows no keys at all, so env vars would otherwise be
	// silently ignored. Bind every mapstructure-tagged key explicitly
	// so plain environment variables (e.g. from docker-compose) work
	// even without an app.env file.
	fields := reflect.VisibleFields(reflect.TypeFor[Config]())
	for _, field := range fields {
		key := field.Tag.Get("mapstructure")
		if key == "" {
			continue
		}
		if err = viper.BindEnv(key); err != nil {
			return config, err
		}
	}

	err = viper.Unmarshal(
		&config,
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
			),
		),
	)

	return config, err
}
