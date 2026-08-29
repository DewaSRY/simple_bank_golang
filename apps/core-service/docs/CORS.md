# Why gin-contrib/cors — Cross-Origin Requests

## The decision

Browser clients (e.g. a separate frontend app running on its own origin) can call this API's `/api/v1/*` routes only if the response carries the right `Access-Control-*` headers. That's handled by [gin-contrib/cors](https://github.com/gin-contrib/cors), wired as a single middleware in [internal/api/server.go](../internal/api/server.go), configured from a new `CORS_ALLOWED_ORIGINS` setting in [internal/config/config.go](../internal/config/config.go) — not hand-written header-setting code, and not a wildcard `*`.

## Why not the alternatives

**Why not hand-write the headers.** CORS looks like "just set a few response headers," but it isn't: the browser also sends a **preflight** `OPTIONS` request ahead of most non-trivial requests (anything with a JSON body, an `Authorization` header, or a method other than `GET`/`POST` with a simple content type), and that preflight expects `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`, and `Access-Control-Max-Age` back, with no body, before the browser will even send the real request. Getting this wrong is a classic source of "works with curl, fails in the browser" bugs. `gin-contrib/cors` handles the preflight short-circuit and header set correctly; a hand-rolled `ctx.Header(...)` call in a handler does not.

**Why not `AllowAllOrigins: true` / `*`.** This API issues JWTs and the login/register endpoints are meant to work with `Authorization` headers and (potentially) cookies from a trusted frontend only. Reflecting or allowing every origin means any website can script a logged-in user's browser into hitting this API. `AllowOrigins` is set explicitly from configuration instead, so only origins this deployment's operator names are allowed — and `AllowCredentials: true` (needed for the `Authorization` header to be honored cross-origin) is disallowed by browsers when the origin list is a wildcard anyway, so `*` wasn't actually compatible with what this API needs.

**The trade-off you're accepting.** Every environment (local dev, staging, production) now has one more setting to keep correct: the frontend's real origin(s) must be listed, exactly (scheme + host + port), or the browser silently rejects the response with a CORS error that never even reaches this service's logs. This is a one-time cost paid at deploy/config time, not a recurring one.

## How it's wired in this codebase

[internal/config/config.go](../internal/config/config.go):

```go
type Config struct {
	// ...
	CORSAllowedOrigins []string `mapstructure:"CORS_ALLOWED_ORIGINS"`
}
```

`CORS_ALLOWED_ORIGINS` is a comma-separated list (e.g. `http://localhost:3000,https://app.example.com`). Viper/mapstructure doesn't split a plain string into a slice on its own, so `LoadConfig` passes a decode hook to do it:

```go
err = viper.Unmarshal(&config, viper.DecodeHook(mapstructure.StringToSliceHookFunc(",")))
```

[internal/api/server.go](../internal/api/server.go) turns that list into a middleware and registers it first, before `errorHandlerMiddleware`, so a rejected preflight never goes through error handling:

```go
router := gin.Default()
router.Use(corsMiddleware(cfg.CORSAllowedOrigins))
router.Use(errorHandlerMiddleware())
```

```go
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(ctx *gin.Context) { ctx.Next() }
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
```

Two mechanics worth understanding:

1. **An empty `CORS_ALLOWED_ORIGINS` disables CORS entirely, on purpose.** A no-op middleware is used instead of `cors.Default()` (which allows all origins) — the safe default for a deployment with no browser frontend (server-to-server calls, mobile clients, same-origin setups) is to send no `Access-Control-*` headers at all, not to silently open the API to any origin because the setting was left blank.

2. **`AllowCredentials: true` is why the origin list can't be a wildcard.** This app expects the `Authorization: Bearer <token>` header on authorized routes; per the CORS spec, browsers refuse to expose a credentialed response to script when `Access-Control-Allow-Origin` is `*`. Listing explicit origins in `CORS_ALLOWED_ORIGINS` is what makes credentialed cross-origin requests possible at all, not just safer.

## Adding or changing allowed origins

1. Set `CORS_ALLOWED_ORIGINS` in `app.env` (local dev) or the real environment variable (staging/production) to a comma-separated list of exact origins — scheme, host, and port must match what the browser sends, e.g. `https://app.example.com` (no trailing slash, no path).
2. Multiple frontends are supported by listing more than one origin: `CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com`.
3. No code change is needed to update the list — `corsMiddleware` is rebuilt from `cfg.CORSAllowedOrigins` on every server start, same as every other config-driven setting (see [CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md)).
4. If a new method or request header is needed by the frontend (e.g. a custom header), add it to `AllowMethods`/`AllowHeaders` in `corsMiddleware` ([internal/api/server.go](../internal/api/server.go)) — these are fixed per-deployment, not configurable via environment variables, since they describe what this API's handlers actually accept, not something that varies by environment.
