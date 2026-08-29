# Why Viper — Configuration Loading

## The decision

Configuration (`DB_DRIVER`, `DB_SOURCE`, `SERVER_ADDRESS`, `JWT_SECRET_KEY`, `JWT_ACCESS_TOKEN_DURATION`) is loaded through [spf13/viper](https://github.com/spf13/viper) into a single typed `Config` struct — see [internal/config/config.go](../internal/config/config.go) — rather than scattering `os.Getenv("DB_SOURCE")` calls across `main.go` and every package that needs a setting.

## Why not the alternatives

**Why not raw `os.Getenv`.** `os.Getenv` gives you a string, always, whether the variable is set or not — a typo in the variable name (`JWT_SECRET_KYE`) or a missing entry silently becomes `""`, and you find out at runtime when JWT signing fails with a confusing error, not at startup with a clear one. It also gives you no type: `JWT_ACCESS_TOKEN_DURATION=15m` has to be parsed into a `time.Duration` by hand, every place it's read. And it doesn't compose with a `.env`-style file for local development — you'd need a second, different mechanism for that.

**Why not a hand-rolled `.env` reader** (e.g. `godotenv`). That solves the file-loading half of the problem, but you still need something to turn the raw string map into a typed struct, and you still need a story for "read from environment in production, read from a file in local dev" without two code paths. Viper does both: it reads a config file into memory *and* unifies it with real environment variables *and* unmarshals the result into a struct — one dependency instead of two, and one code path instead of a `if os.Getenv("ENV") == "local"` branch.

**The trade-off you're accepting.** Viper is a large, somewhat "magic" library for what is conceptually a small job — the alternative of `os.Getenv` + `strconv` + a small hand-written struct-builder is genuinely simpler to read for five settings. The payoff shows up as the settings list grows past the current five, or when you add config sources (flags, remote config) that would otherwise mean rewriting the loader.

## How it's wired in this codebase

[internal/config/config.go](../internal/config/config.go):

```go
type Config struct {
	DBDriver               string        `mapstructure:"DB_DRIVER"`
	DBSource               string        `mapstructure:"DB_SOURCE"`
	ServerAddress          string        `mapstructure:"SERVER_ADDRESS"`
	JWTSecretKey           string        `mapstructure:"JWT_SECRET_KEY"`
	JWTAccessTokenDuration time.Duration `mapstructure:"JWT_ACCESS_TOKEN_DURATION"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)   // directory to look for the config file in
	viper.SetConfigName("app")  // filename without extension -> "app.env"
	viper.SetConfigType("env")  // parse it as KEY=VALUE, not YAML/JSON/TOML

	viper.AutomaticEnv()        // real environment variables override the file

	if err = viper.ReadInConfig(); err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
```

Three mechanics worth understanding, because they're easy to misuse:

1. **`mapstructure`, not `json` or `viper` tags.** Viper unmarshals through the [`mapstructure`](https://github.com/go-viper/mapstructure) library internally, so the struct tag that maps `DB_SOURCE` to `DBSource` is `mapstructure:"DB_SOURCE"` — a `json:"DB_SOURCE"` tag here does nothing. This is the single most common mistake when adding a new config field: add the field, forget the right tag name, and get a zero-value with no error.

2. **File + environment are unified, and environment wins.** `viper.AutomaticEnv()` means any real environment variable with a matching name (`DB_SOURCE=...` set by the shell, Docker, or a deploy platform) overrides whatever [app.env](../app.env.example) says, without any extra code. In practice this means: `app.env` (git-ignored; [app.env.example](../app.env.example) documents the required keys) is the convenience path for local development, and production sets real env vars and never ships a config file at all. There's exactly one loader for both cases.

3. **Config is loaded once, at process start, and passed down explicitly.** `cmd/server/main.go` calls `config.LoadConfig(".")` exactly once and threads the resulting `Config` value into `api.NewServer(store, cfg)` (see [cmd/server/main.go](../cmd/server/main.go)) — nothing downstream calls `viper.Get(...)` directly or re-reads config later. This matters for two reasons: it keeps `Config` a plain, mockable value for tests instead of a global you have to reset between them, and it means config is fully resolved and validated (via `LoadConfig`'s returned `err`) before the server binds a port — a missing `JWT_SECRET_KEY` fails fast at startup (`log.Fatal("cannot load config:", err)`), not on the first login request.

## Adding a new setting — the actual workflow

1. Add the field to `Config` in [internal/config/config.go](../internal/config/config.go) with a `mapstructure:"YOUR_ENV_NAME"` tag matching the environment variable name exactly.
2. Add the same key (empty or with a sane default) to [app.env.example](../app.env.example) so it's documented for anyone setting up the project locally.
3. Set the real value in your local `app.env` (git-ignored) and in whatever mechanism sets environment variables in each deployed environment.
4. Read it off the `Config` value that's already threaded through (`cfg.YourNewField`) — never call `viper` directly outside of `LoadConfig`.

If the value needs a non-string type (duration, int, bool), give the field that type directly (as `JWTAccessTokenDuration time.Duration` already does) — viper/mapstructure parses `"15m"` into a `time.Duration` for you; you don't need a manual `time.ParseDuration` call anywhere.
