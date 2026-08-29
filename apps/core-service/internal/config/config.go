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
