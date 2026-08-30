# Environment Variable Configuration — As Implemented

## Who this doc is for

You're assumed to be comfortable with Go, but you haven't necessarily used [spf13/viper](https://github.com/spf13/viper) or its underlying [`mapstructure`](https://github.com/go-viper/mapstructure) decoder before. If you have, skip Section 0 — it's a primer, not load-bearing for the rest of the doc.

Everything below is verified against the source as it exists today, not the idealized behavior the library's README promises. Two real production incidents are documented in detail, because both were *silent* — no error, no log line, just a wrong value flowing downstream — and the fix in each case is easy to accidentally undo the next time this file is touched.

## Section 0 — Background Primer: why Viper, not something simpler

| Approach | Reads real env vars | Reads a local file | Produces a typed struct | Typical use here |
|---|---|---|---|---|
| `os.Getenv` | Yes | No | No — you parse by hand | Fine for 1-2 flags, doesn't scale |
| `godotenv` | Indirectly (loads file *into* the environment) | Yes | No — still just strings | Solves half the problem |
| `viper` + `mapstructure` | Yes | Yes | Yes, via struct tags | What this project uses |

`os.Getenv` always returns a string, whether the variable is set or not — a typo in the name or a missing entry silently becomes `""`. `godotenv` fixes file-loading but still leaves you to hand-parse every non-string value and hand-write a "file in dev, real env in prod" branch. Viper does all three jobs — read a file, unify it with real environment variables (environment wins), and unmarshal the result into a typed `Config` struct — through one dependency and one code path.

The trade-off: Viper is a large, somewhat "magic" library for what is conceptually a small job. That cost shows up directly in the two pitfalls below — both are cases where Viper's implicit behavior didn't match what the code at the call site looked like it was doing.

**Gotchas that surprise newcomers** (each is the subject of its own section further down):

1. Viper's `AutomaticEnv()` sounds like "check the environment for anything my struct asks for." It isn't — it only checks the environment for keys Viper *already knows about* from some other source. See [Section 3](#section-3-loadconfig--merging-a-file-with-the-environment).
2. Passing your own `viper.DecodeHook(...)` to `Unmarshal` doesn't add a hook to Viper's defaults — it throws the defaults away and replaces them outright. See [Section 3](#section-3-loadconfig--merging-a-file-with-the-environment).
3. The struct tag Viper reads is `mapstructure:"..."`, not `json:"..."` — a natural thing to reach for out of habit, and it fails silently (zero value, no error) if you get it wrong. See [Section 2](#section-2-the-config-struct--the-schema).

## Section 1 — Architecture at a Glance

There are two composition roots, and both are thin: they call `config.LoadConfig(".")` once and hand the resulting value to whatever needs it. Neither implements any parsing or merging logic itself.

- [cmd/server/main.go:26](../cmd/server/main.go#L26) — loads config, then threads it into `connectDB(cfg)` ([cmd/server/main.go:32](../cmd/server/main.go#L32)) and `api.NewServer(store, cfg)` ([cmd/server/main.go:43](../cmd/server/main.go#L43)).
- [cmd/migration/main.go](../cmd/migration/main.go) — loads config independently (its own `LoadConfig(".")` calls, once for `up` and once for `down`), and reads only `cfg.DBSource` out of it.

| Concern | Owner | Analogy |
|---|---|---|
| Struct/schema definition | [internal/config/config.go:12-19](../internal/config/config.go#L12-L19) (`Config`) | The blank intake form's field list |
| Loading & merging file + env | [internal/config/config.go:23-67](../internal/config/config.go#L23-L67) (`LoadConfig`) | The clerk who merges a paper form with a phoned-in override, env winning |
| Token signing/verification | [internal/token/jwt_maker.go](../internal/token/jwt_maker.go) | The notary who signs with whatever secret the clerk handed over |
| CORS gate | [internal/api/server.go:88-100](../internal/api/server.go#L88-L100) (`corsMiddleware`) | The bouncer checking today's allowlist |
| Local dev convenience file | [app.env](../app.env) (git-ignored) / [app.env.example](../app.env.example) | The paper form — used only when nobody phones in a completed override |

It's split this way so `Config` stays a plain, mockable value instead of a global you'd have to reset between tests — [internal/api/auth_router_test.go:41-42](../internal/api/auth_router_test.go#L41-L42) constructs a bare `config.Config{...}` literal directly, never touching `LoadConfig` at all. That split is also exactly why the two production incidents below were invisible in tests: nothing that exercises `LoadConfig`'s actual file/env-merging logic runs in CI, so both bugs only reproduced in a real container. See "Cross-Feature Coupling" below.

## Section 2 — The `Config` struct — the schema

**The problem it solves.** Every setting the app needs — five strings plus one duration plus one string slice — needs one place they're all declared, typed, and named, instead of a `DB_SOURCE` string literal scattered across `main.go`, `connect_db.go`, and `cmd/migration/main.go`.

**How it's implemented.** [internal/config/config.go:12-19](../internal/config/config.go#L12-L19):

```go
type Config struct {
	DBDriver               string        `mapstructure:"DB_DRIVER"`
	DBSource               string        `mapstructure:"DB_SOURCE"`
	ServerAddress          string        `mapstructure:"SERVER_ADDRESS"`
	JWTSecretKey           string        `mapstructure:"JWT_SECRET_KEY"`
	JWTAccessTokenDuration time.Duration `mapstructure:"JWT_ACCESS_TOKEN_DURATION"`
	CORSAllowedOrigins     []string      `mapstructure:"CORS_ALLOWED_ORIGINS"`
}
```

| Field | Env var | Go type | Example (from [app.env.example](../app.env.example)) |
|---|---|---|---|
| `DBDriver` | `DB_DRIVER` | `string` | *(blank — set per environment)* |
| `DBSource` | `DB_SOURCE` | `string` | *(blank — set per environment)* |
| `ServerAddress` | `SERVER_ADDRESS` | `string` | *(blank — set per environment)* |
| `JWTSecretKey` | `JWT_SECRET_KEY` | `string` | *(blank — set per environment)* |
| `JWTAccessTokenDuration` | `JWT_ACCESS_TOKEN_DURATION` | `time.Duration` | `15m` |
| `CORSAllowedOrigins` | `CORS_ALLOWED_ORIGINS` | `[]string` | *(blank; comma-separated when set, e.g. `http://localhost:3000,http://localhost:5173`)* |

**If you're new to `mapstructure`:** it's the library Viper unmarshals *through* — the tag name is a `mapstructure` requirement, not a Viper one, and it has nothing to do with `encoding/json`. Writing `json:"DB_SOURCE"` here compiles fine and does nothing; `viper.Unmarshal` never looks at it. This is the single most common mistake when adding a field: add it, tag it wrong (or not at all), and you get a zero value with `err == nil` — no crash, no warning, the field is just always empty.

**Rough edge worth flagging:** the struct itself does zero validation. There's no way to tell, just from `Config`, which fields are actually required for the app to run versus cosmetic. That validation — where it exists at all — lives entirely in the consumers (Section 4), not here.

## Section 3 — `LoadConfig` — merging a file with the environment

**The problem it solves.** Local development wants a convenience file (`app.env`) you can edit without touching your shell profile. Production (Docker, or any real deploy target) wants to set real environment variables and ship no file at all. Without Viper, that's two different code paths with an `if os.Getenv("ENV") == "local"` branch somewhere. `LoadConfig` is one function that serves both.

**How it's implemented**, in full, because every line here has been the site of a real bug at some point — [internal/config/config.go:23-67](../internal/config/config.go#L23-L67):

```go
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)   // directory to look for the config file in
	viper.SetConfigName("app")  // filename without extension -> "app.env"
	viper.SetConfigType("env")  // parse it as KEY=VALUE, not YAML/JSON/TOML

	viper.AutomaticEnv()        // real environment variables override the file

	// app.env is optional. If it doesn't exist, continue and
	// rely on environment variables instead.
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	// Explicitly bind every mapstructure-tagged field to its env var.
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
```

Three mechanics, then the two pitfalls that shaped this exact code:

1. **`AddConfigPath` / `SetConfigName` / `SetConfigType`** point Viper at `<path>/app.env`, parsed as flat `KEY=VALUE` pairs — not YAML/JSON/TOML despite the `.env` extension looking arbitrary.
2. **`ReadInConfig`'s error is deliberately swallowed, but only for one specific error type.** A missing `app.env` (`viper.ConfigFileNotFoundError`) is fine and expected in production. Any other error — a malformed file, a permissions problem — is not swallowed and propagates out of `LoadConfig`.
3. **Config is resolved once, at process start, before anything else happens.** [cmd/server/main.go:26-29](../cmd/server/main.go#L26-L29) calls `LoadConfig` and `log.Fatal`s immediately on error — a config problem fails loudly at startup rather than surfacing later on the first request that needs the broken field. (This holds for genuine errors; the callout below is about the case where `err == nil` but the value is still wrong.)

### Pitfall: `AutomaticEnv()` does not mean `Unmarshal()` sees your env vars

This bit us for real: the `core-services` container in [docker-compose.yaml](../../../docker-compose.yaml) sets `DB_DRIVER`, `DB_SOURCE`, `SERVER_ADDRESS`, etc. as plain environment variables and does **not** mount an `app.env` file. The container started, but `sql.Open(cfg.DBDriver, cfg.DBSource)` in [cmd/server/connect_db.go:31](../cmd/server/connect_db.go#L31) failed because `cfg.DBDriver` was an empty string — even though `DB_DRIVER=postgres` was clearly set in `docker-compose.yaml`.

**Why this happens.** `AutomaticEnv()` only makes Viper check the environment for keys **Viper already knows about** — keys it learned from a config file it successfully read, from a `viper.SetDefault`, or from an explicit `viper.BindEnv(key)` call. It does not inspect the `Config` struct's tags and go hunting through `os.Environ()` on its own. Put together:

1. No `app.env` file exists in the container (by design).
2. `viper.ReadInConfig()` returns `ConfigFileNotFoundError`, correctly treated as "fine, keep going" — no error surfaces yet.
3. But because no file was read, Viper has **zero keys registered**. `AutomaticEnv()` has nothing to match environment variables against.
4. `viper.Unmarshal(&config)` walks its (empty) set of known keys and finds none — every field in `Config` comes back as its Go zero value.
5. `LoadConfig` returns `err == nil` and a struct that looks populated in the source but is actually empty. The failure shows up later and somewhere else — a cryptic `sql: unknown driver ""` from `connect_db.go`, not a clear config error.

**This is a silent failure** — no error, no log line, a zero-value config quietly flowing into the rest of the app. It only reproduces when there is no config file at all, which is exactly the production/Docker setup this project targets, so it's easy to test locally with `app.env` present and never see it.

**The fix**, already in the code above: loop over `reflect.VisibleFields(reflect.TypeFor[Config]())` and call `viper.BindEnv(key)` for every `mapstructure`-tagged field before unmarshalling. `BindEnv` tells Viper "this key exists, go look for an environment variable with this exact name," independent of whether any file was ever read. Driving the loop off `reflect` instead of hand-listing `"DB_DRIVER"`, `"DB_SOURCE"`, ... means it can't silently drift: add a tagged field to `Config`, and it's bound automatically.

### Pitfall: a custom `DecodeHook` replaces Viper's defaults, it doesn't add to them

This one predates the pitfall above, and it's the reason `StringToTimeDurationHookFunc` appears explicitly in the code even though nothing about duration parsing looks special.

**How it happened.** `JWTAccessTokenDuration time.Duration` was added first, with a bare `viper.Unmarshal(&config)` — no `DecodeHook` option anywhere. It worked immediately: `JWT_ACCESS_TOKEN_DURATION=15m` came out as a correct `time.Duration`, no `time.ParseDuration` call written anywhere. That's because Viper's *default* decoder config, used whenever you don't pass a `DecodeHook` option yourself, already composes `mapstructure.StringToTimeDurationHookFunc()` with `mapstructure.StringToSliceHookFunc(",")` — durations and comma-separated slices both parse for free.

Later, `CORSAllowedOrigins []string` was added, and to make `CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173` split into a slice, the natural-looking fix was:

```go
err = viper.Unmarshal(&config, viper.DecodeHook(mapstructure.StringToSliceHookFunc(",")))
```

CORS parsing started working — and JWT duration parsing silently broke. Passing a `DecodeHook` option to `Unmarshal` doesn't *add* a hook to Viper's default set, it **replaces the entire decode hook chain**. Viper's built-in default (duration + slice, composed) was gone, swapped for a decoder that only knew how to split on commas. The struct tag was still correct, `BindEnv` still found the key, `Unmarshal` still returned `err == nil` — the duration value just came back wrong. Nothing about this failure looks like a config problem from the call site; it would have surfaced wherever `cfg.JWTAccessTokenDuration` is used next — token signing and the `expires_in` response field (Section 4).

**The fix**, already in the code above: don't pass a lone hook — recompose the *same* pair Viper would have used by default, explicitly, with `mapstructure.ComposeDecodeHookFunc(StringToTimeDurationHookFunc(), StringToSliceHookFunc(","))`.

**The takeaway:** the moment you add your *first* explicit `viper.DecodeHook(...)` option, you've taken over decoding for every field in the struct, not just the one you're touching. Every non-string field's parsing behavior from then on depends on being listed inside that one `ComposeDecodeHookFunc(...)` call — there's no partial opt-in.

## Section 4 — Consumers: where `Config` values actually get used

`LoadConfig` never gets called a second time and nothing downstream calls `viper.Get(...)` directly — every consumer reads a typed field off the `Config` value that was threaded to it.

| Field | Consumed by | Where |
|---|---|---|
| `DBDriver`, `DBSource` | `sql.Open` | [cmd/server/connect_db.go:31](../cmd/server/connect_db.go#L31) |
| `DBSource` | migration up/down commands | [cmd/migration/main.go](../cmd/migration/main.go) (`upMigration(cfg.DBSource)`, `downMigration(cfg.DBSource)`) |
| `ServerAddress` | port pre-check + HTTP listen | [cmd/server/main.go:49](../cmd/server/main.go#L49) (`net.Listen`), [cmd/server/main.go:54](../cmd/server/main.go#L54) (`server.Start`) |
| `JWTSecretKey` | token maker construction | [internal/api/server.go:40](../internal/api/server.go#L40) (`token.NewJWTMaker(cfg.JWTSecretKey)`) |
| `JWTAccessTokenDuration` | token creation + response payload | [internal/api/auth_router.go:61,70](../internal/api/auth_router.go#L61) and [:154,163](../internal/api/auth_router.go#L154) (login and register) |
| `CORSAllowedOrigins` | CORS middleware | [internal/api/server.go:48,88-100](../internal/api/server.go#L48) (`corsMiddleware`) |

**Token signing key.** `JWTSecretKey` isn't just read — it's validated, but *not* by `LoadConfig`. [internal/token/jwt_maker.go:11,21-23](../internal/token/jwt_maker.go#L11):

```go
const minSecretKeySize = 32
// ...
if len(secretKey) < minSecretKeySize {
	return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
}
```

`api.NewServer` wraps and re-raises this as `"cannot create token maker: %w"` ([internal/api/server.go:41-43](../internal/api/server.go#L41-L43)), and `main.go` still `log.Fatal`s on it — so a too-short secret does fail at startup, just via a different layer than `LoadConfig`'s own returned `err`.

**Rough edge — validation is split across three different places, none of which is `Config` itself:**
- `DBDriver`/`DBSource` are validated implicitly, by `sql.Open` and the retry-bounded ping loop in `connectDB` ([cmd/server/connect_db.go:41-55](../cmd/server/connect_db.go#L41-L55)) failing loudly.
- `JWTSecretKey` is validated explicitly, but inside `internal/token`, not `internal/config`.
- `ServerAddress` and `JWTAccessTokenDuration` have **no validation anywhere**. An empty `SERVER_ADDRESS` reaches `net.Listen` and fails there (still caught, still `log.Fatal`'d — see [cmd/server/main.go:49-51](../cmd/server/main.go#L49-L51)) but a missing `JWT_ACCESS_TOKEN_DURATION` quietly decodes to `0s`: tokens would be issued already-expired, with no error at any layer. This has not been hit in production (the key is always set), but it's a real gap — worth a `SetDefault` or an explicit check if this file is touched again.

**Rough edge — an empty `CORSAllowedOrigins` disables CORS entirely, rather than erroring or defaulting to "allow nothing visibly."** [internal/api/server.go:85-91](../internal/api/server.go#L85-L91):

```go
// corsMiddleware allows browser clients on allowedOrigins to call this API.
// An empty list disables CORS entirely (no Access-Control-* headers are
// sent), which is the safe default for server-to-server deployments.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(ctx *gin.Context) { ctx.Next() }
	}
	...
```

This is intentional per the comment, and matches a server-to-server deployment where no browser client needs CORS headers at all — but it means leaving `CORS_ALLOWED_ORIGINS` unset is indistinguishable, at the config layer, from deliberately disabling browser access. Nothing logs which case you're in.

## Section 5 — Local dev vs. production: `app.env` vs. real environment variables

Any real environment variable with a matching name (`DB_SOURCE=...` set by the shell, Docker, or a deploy platform) overrides whatever `app.env` says, with no extra code — that's `AutomaticEnv()` plus the `BindEnv` loop from Section 3 working together. In practice:

- **Local dev**: [app.env](../app.env) (git-ignored — see [.gitignore:27-30](../../../.gitignore#L27-L30), which excludes `.env`, `app.env`, and `*.env`) holds real values. [app.env.example](../app.env.example) is the checked-in template documenting the required keys, with everything blank except `JWT_ACCESS_TOKEN_DURATION=15m`.
- **Production / Docker**: the `core-services` service in [docker-compose.yaml](../../../docker-compose.yaml) sets all six values as plain `environment:` entries and mounts no file at all.

There is exactly one loader (`LoadConfig`) for both cases — which is also exactly the setup that triggers the `AutomaticEnv()` pitfall above: it only reproduces when there is no config file, i.e. the production path, so it's easy to develop and test entirely against the `app.env` path and never see it.

## Cross-Feature Coupling

- **The migration CLI shares the same loader, independently.** [cmd/migration/main.go](../cmd/migration/main.go) is a second, separate binary that calls `config.LoadConfig(".")` on its own (once per `up`/`down` invocation) — it does not go through `cmd/server/main.go` at all. Any bug in `LoadConfig` (both pitfalls above included) affects migrations and the API server simultaneously, even though they're invoked completely independently and might run in different containers.
- **Tests bypass `LoadConfig` entirely.** [internal/api/auth_router_test.go:40-44](../internal/api/auth_router_test.go#L40-L44) builds a `config.Config{...}` struct literal directly and hands it to `NewServer` — no file, no environment, no `viper` call in the path at all. This is why neither pitfall above was ever caught by a test: the code path that actually merges file and environment has no automated coverage today. Both incidents were only found by running the real Docker container.

## Pure consumers — not part of the config system itself

Two pieces take a `Config` value as a parameter but contain no loading, merging, or parsing logic of their own — worth naming so they don't get mistaken for part of the system described above:

- `connectDB` ([cmd/server/connect_db.go](../cmd/server/connect_db.go)) — reads `cfg.DBDriver`/`cfg.DBSource` once, then spends the rest of its body tuning the connection pool and retrying the initial ping. None of that logic is config-related; `cfg` is just its input.
- `Server.Start` ([internal/api/server.go:81-83](../internal/api/server.go#L81-L83)) — takes the already-resolved `cfg.ServerAddress` string and calls `router.Run(address)`. No parsing happens here either.

## Summary — data flow

**Local dev (`app.env` present):**
```
main() → LoadConfig(".")
  → viper reads app.env into memory (6 keys now known to viper)
  → AutomaticEnv() can now match any of those 6 names against real env vars, which win if set
  → BindEnv loop (redundant here, but harmless — same 6 keys)
  → Unmarshal (duration + slice hooks) → Config{...} fully populated
  → threaded into connectDB(cfg) and api.NewServer(store, cfg)
```

**Production / Docker (no file):**
```
main() → LoadConfig(".")
  → viper.ReadInConfig() → ConfigFileNotFoundError, swallowed
  → viper knows 0 keys — AutomaticEnv() alone would find nothing here
  → BindEnv loop registers all 6 mapstructure-tagged keys explicitly (the fix)
  → AutomaticEnv() can now match those 6 names against docker-compose's environment: entries
  → Unmarshal (duration + slice hooks) → Config{...} fully populated
  → threaded into connectDB(cfg) and api.NewServer(store, cfg)
```

Both flows end at the same `Config{...}` value and the same two call sites — the only difference is *which* mechanism (file-read vs. explicit `BindEnv`) put each key into Viper's known-key set before `Unmarshal` ran.

## Final Reference — every environment variable

| Env var | Go field | Type | Default in `app.env.example` | Required in practice | Consumed at |
|---|---|---|---|---|---|
| `DB_DRIVER` | `DBDriver` | `string` | *(blank)* | Yes — `sql.Open` fails without it | [connect_db.go:31](../cmd/server/connect_db.go#L31) |
| `DB_SOURCE` | `DBSource` | `string` | *(blank)* | Yes — `sql.Open`/migrations fail without it | [connect_db.go:31](../cmd/server/connect_db.go#L31), [cmd/migration/main.go](../cmd/migration/main.go) |
| `SERVER_ADDRESS` | `ServerAddress` | `string` | *(blank)* | Yes — `net.Listen` fails without it | [main.go:49](../cmd/server/main.go#L49) |
| `JWT_SECRET_KEY` | `JWTSecretKey` | `string` | *(blank)* | Yes — rejected below 32 chars | [server.go:40](../internal/api/server.go#L40), [jwt_maker.go:21-23](../internal/token/jwt_maker.go#L21-L23) |
| `JWT_ACCESS_TOKEN_DURATION` | `JWTAccessTokenDuration` | `time.Duration` | `15m` | **Not enforced** — unset silently decodes to `0s` | [auth_router.go:61,70,154,163](../internal/api/auth_router.go#L61) |
| `CORS_ALLOWED_ORIGINS` | `CORSAllowedOrigins` | `[]string` (comma-separated) | *(blank)* | No — blank deliberately disables CORS | [server.go:48,88-100](../internal/api/server.go#L48) |
