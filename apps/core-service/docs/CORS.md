# Cross-Origin Requests (CORS) — As Implemented

## Who this doc is for

You're assumed to be comfortable with HTTP and Go, but you haven't necessarily had to reason about **why** a browser blocks a request that `curl` handles fine. If you have, skip Section 0 — it's a primer, not load-bearing for the rest of the doc.

Everything below is verified against the source as it exists today, not the idealized behavior "add a CORS middleware" implies. In particular: the middleware here is a thin, mostly-default wrapper around [gin-contrib/cors](https://github.com/gin-contrib/cors) v1.7.7, wired from one config field ([internal/config/config.go:18](../internal/config/config.go#L18)), and most of what makes it correct is the *order* it's registered in, not its own logic.

## Section 0 — Background Primer: what a browser actually does before "just setting headers"

| Approach | Handles preflight `OPTIONS`? | Origin allowlisting | Compatible with `AllowCredentials` | Typical use |
|---|---|---|---|---|
| Hand-written `ctx.Header(...)` in a handler | No, unless you special-case `OPTIONS` yourself | Whatever you hand-code | Easy to get wrong silently | Fine for a single trusted internal caller, brittle otherwise |
| `cors.Default()` (gin-contrib/cors, all origins) | Yes | None — reflects/allows every origin | No — spec forbids `*` + credentials | Public, unauthenticated APIs only |
| `cors.New(cors.Config{AllowOrigins: [...]})` | Yes | Explicit list from config | Yes | What this project uses |

A browser doesn't just attach an `Origin` header and let the response through. For any "non-simple" request — a JSON body, an `Authorization` header, or a method outside `GET`/`HEAD`/`POST`-with-simple-content-type — it first sends a **preflight**: an `OPTIONS` request carrying `Access-Control-Request-Method`/`-Headers`, and it will not send the real request at all unless the preflight response carries matching `Access-Control-Allow-*` headers. This API's login/JSON-body/`Authorization`-header traffic is exactly the "non-simple" case, so every authenticated request from a browser is actually two HTTP round-trips, not one.

**Gotchas that surprise newcomers** (each is the subject of its own section further down):

1. A rejected preflight never reaches this service's application logic or logs — the browser blocks it client-side once the response headers don't match. "It works in Postman/curl but fails in the browser" almost always means this, not a server bug. See [Section 3](#section-3--the-empty-list-branch-cors-off-by-default).
2. `AllowCredentials: true` (needed so the `Authorization` header is honored cross-origin) makes a wildcard origin (`*`) actively illegal per spec — the browser refuses to expose the response, not just discourage it. See [Section 2](#section-2--corsmiddleware--the-gate-itself).
3. The CORS middleware runs *before* every other middleware in the chain, including `AuthMiddleware` — a valid preflight response is sent for a route that would otherwise 401, because the preflight never reaches the route at all. See [Cross-Feature Coupling](#cross-feature-coupling).

## Section 1 — Architecture at a Glance

The composition root is `NewServer` ([internal/api/server.go:40-88](../internal/api/server.go#L40-L88)): it builds `corsMiddleware(cfg.CORSAllowedOrigins)` and registers it on the router with `router.Use(...)`, alongside the rest of the middleware chain. `NewServer` doesn't implement any CORS logic itself — it delegates entirely to `corsMiddleware`, which delegates to `gin-contrib/cors`.

| Concern | Owner | Analogy |
|---|---|---|
| On/off switch + allowlist | [internal/config/config.go:18](../internal/config/config.go#L18) (`Config.CORSAllowedOrigins`) | The building's visitor list, updated per deployment |
| The gate itself | [internal/api/server.go:121-133](../internal/api/server.go#L121-L133) (`corsMiddleware`) | The bouncer checking today's list against whoever's at the door |
| Where the gate stands in line | [internal/api/server.go:68-73](../internal/api/server.go#L68-L73) (`router.Use(...)` chain) | The bouncer stands *before* the ID-checker (`AuthMiddleware`) and the incident-report writer (`ErrorHandlerMiddleware`/`RecoveryMiddleware`) |
| The actual preflight/simple-request logic | `gin-contrib/cors` v1.7.7 (vendored dependency, not this repo) | The rulebook the bouncer follows — this repo only fills in the allowlist and a couple of settings |

It's split this way so the allowlist can change per environment (local/staging/production) without a code change or redeploy of logic — only a config value — and so the gate can sit at a fixed, correct point in the middleware chain regardless of which domain packages (`auth`, `account`, `transfer`) get added later. That chain position is also exactly why a plausible-looking refactor ("move CORS registration after auth, since auth is 'more important'") would silently break authenticated browser traffic — see [Cross-Feature Coupling](#cross-feature-coupling).

## Section 2 — `corsMiddleware` — the gate itself

**The problem it solves.** A hand-written `ctx.Header("Access-Control-Allow-Origin", ...)` doesn't handle preflight `OPTIONS` requests, doesn't validate the origin against a list, and doesn't interact correctly with credentialed requests — getting all three right by hand is exactly the kind of narrow, easy-to-get-subtly-wrong logic a maintained library should own instead.

**How it's implemented.** [internal/api/server.go:121-133](../internal/api/server.go#L121-L133):

```go
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
```

| Field | What actually happens |
|---|---|
| `AllowOrigins` | Exact-match allowlist from `cfg.CORSAllowedOrigins` — scheme+host+port must match exactly what the browser sends; there's no wildcard/subdomain support configured. |
| `AllowMethods` | Fixed at 6 methods, hardcoded in Go — not environment-configurable. `HEAD` is not in this list (gin-contrib/cors's own `DefaultConfig()` includes it; this project's list doesn't). |
| `AllowHeaders` | Fixed allowlist of 5 request headers a browser is permitted to send cross-origin, including `Authorization` and `X-Request-Id` (see [Cross-Feature Coupling](#cross-feature-coupling) for why `X-Request-Id` specifically matters here). |
| `AllowCredentials` | Always `true` — this is what makes the `Authorization` header (and cookies, if ever used) actually reach the handler cross-origin. Per spec, this is incompatible with a wildcard origin — see the gotcha in Section 0. |
| `MaxAge` | `12 * time.Hour` — how long a browser may cache a successful preflight response before re-sending one. |

Under the hood, `gin-contrib/cors`'s `applyCors` ([cors@v1.7.7/config.go:71-93](https://github.com/gin-contrib/cors/blob/v1.7.7/config.go#L71-L93)) does three things worth knowing precisely, since none of them are visible from this repo's own code:

1. **No `Origin` header, or an `Origin` matching the request's own `Host`, is treated as "not a CORS request" and passed through untouched** — same-origin requests and non-browser clients (server-to-server calls, `curl`, mobile apps) never hit any of this logic, allowed or not.
2. **An `Origin` present but not in `AllowOrigins` gets `403 Forbidden` immediately** (`c.AbortWithStatus(http.StatusForbidden)`) — this is a real HTTP response from *this service*, not a browser-side block; it's the one case where a bad origin is visible in this service's own access logs.
3. **A valid preflight (`OPTIONS` with a valid `Origin`) is aborted after headers are set** (`c.AbortWithStatus(cors.optionsResponseStatusCode)`, default `204`) — the middleware chain stops here; nothing registered after `corsMiddleware` runs for a preflight request, ever.

## Section 3 — The empty-list branch: CORS off by default

**The problem it solves.** Not every deployment has a browser frontend — server-to-server calls, mobile clients, and same-origin setups need no `Access-Control-*` headers at all, and sending them anyway (or defaulting to `cors.Default()`'s allow-everything behavior) would be an unnecessary attack surface.

**How it's implemented.** When `cfg.CORSAllowedOrigins` is empty, `corsMiddleware` returns a plain pass-through handler instead of a `cors.New(...)` instance ([internal/api/server.go:122-124](../internal/api/server.go#L122-L124)):

```go
if len(allowedOrigins) == 0 {
    return func(ctx *gin.Context) { ctx.Next() }
}
```

No `Access-Control-*` header is ever sent in this mode — not "empty allowlist" (which `gin-contrib/cors` would treat as "no origin is ever valid," still emitting some response shape), but no CORS layer running at all.

**Rough edge — this is indistinguishable from a misconfiguration.** An operator who forgets to set `CORS_ALLOWED_ORIGINS` in a deployment that *does* have a browser frontend gets exactly this same silent-disable behavior — no error at startup (see `AGENTS.md`'s "No startup config validation" note), no warning beyond one `log.Info` line. `NewServer` does log which branch a given run took, once, at startup ([internal/api/server.go:75-82](../internal/api/server.go#L75-L82)):

```go
if len(cfg.CORSAllowedOrigins) == 0 {
    log.Info("CORS disabled: CORS_ALLOWED_ORIGINS is empty")
} else {
    log.Info("CORS enabled", slog.Any("allowed_origins", cfg.CORSAllowedOrigins))
}
```

— but this is observability, not a fix: nothing stops the service from starting and serving traffic with CORS silently off. See `docs/CONFIG_ENV_VARIABLE.md`'s own callout of this same ambiguity from the config side.

## Cross-Feature Coupling

- **Middleware order in `internal/api/server.go` is a correctness property of this feature, not an implementation detail.** The chain is registered `RequestIDMiddleware` → `LoggingMiddleware` → `corsMiddleware` → `ErrorHandlerMiddleware` → `RecoveryMiddleware` ([internal/api/server.go:68-73](../internal/api/server.go#L68-L73)), and `corsMiddleware` runs *before* the `authorized` route group's `AuthMiddleware` is ever reached, because `AuthMiddleware` is attached per-group in `router.go` ([internal/api/router.go:37-38](../internal/api/router.go#L37-L38)), downstream of every global `router.Use(...)` call. Concretely: a preflight `OPTIONS /api/v1/accounts` with no `Authorization` header gets a `204` from `corsMiddleware` and never reaches `AuthMiddleware` — it does **not** get a `401`. This is correct (a preflight legitimately doesn't carry the app's own auth), but it means you cannot infer anything about an authenticated route's auth behavior from testing its `OPTIONS` response.
- **`corsMiddleware` runs before `ErrorHandlerMiddleware`, so a CORS rejection (`403`, unlisted origin) never goes through this app's normal error-response envelope** (`{"error": {"code", "message", "details"}}`). A CORS-rejected request gets a bare `403` with no JSON body in that shape — API consumers debugging a CORS issue by inspecting the response body will find it empty, not a `core.AppError`.
- **`AllowHeaders` must be kept in sync with `RequestIDMiddleware` and any future custom request header.** `X-Request-Id` is in the allowlist ([internal/api/server.go:129](../internal/api/server.go#L129)) specifically so a browser client can propagate its own request ID (`RequestIDMiddleware` reads `X-Request-Id` if the caller sent one — [internal/api/core/logging_middleware.go:39-48](../internal/api/core/logging_middleware.go#L39-L48)); adding a new header a handler reads (another custom header, a tenant ID, etc.) without also adding it here means browser clients silently can't send it, while `curl`/server-to-server callers work fine — another instance of the "works outside the browser, fails in it" class of bug from Section 0.
- **This doc's "off by default" ambiguity (Section 3) is the same ambiguity `docs/CONFIG_ENV_VARIABLE.md` documents from the config-loading side** — both docs describe the same real gap, from two different layers; see that doc's "Rough edge" callout in its Section 4 for the config-field perspective.

## Pure consumers — not part of the CORS system itself

- **`AuthMiddleware`** ([internal/api/core/auth_middleware.go:27](../internal/api/core/auth_middleware.go#L27)) parses `Authorization: Bearer <token>` on the `authorized` route group. It has no CORS-awareness and doesn't need any — it simply runs later in the chain, after `corsMiddleware` has already let a non-preflight request through.
- **`ErrorHandlerMiddleware`/`RecoveryMiddleware`** ([internal/api/core/apperror.go:122](../internal/api/core/apperror.go#L122), [internal/api/core/logging_middleware.go:316](../internal/api/core/logging_middleware.go#L316)) render this app's normal error envelope and recover panics respectively. As noted above, they sit *after* `corsMiddleware` in the chain, so they never see a CORS-rejected or preflight request at all — don't go looking here to "fix" a CORS response shape.
- **`gin-contrib/cors` itself** is an external dependency, not code owned by this repo — its `applyCors`/preflight-handling logic (Section 2) is worth understanding to reason about behavior, but any real change to *that* logic means either a config change here or an upstream library change, not an edit under `internal/`.

## Summary — data flow

**Same-origin or non-browser request** (`curl`, mobile app, server-to-server): no `Origin` header, or one matching the request's `Host` → `corsMiddleware`'s underlying `applyCors` returns immediately without setting any header → request proceeds normally through `ErrorHandlerMiddleware` → `RecoveryMiddleware` → route handler, exactly as if CORS didn't exist.

**Simple cross-origin browser request** (e.g. a `GET` with no custom headers) from a listed origin: `Origin` present and in `AllowOrigins` → `Access-Control-Allow-*` response headers set → request proceeds through the rest of the chain as normal, response headers let the browser expose it to script.

**Preflighted cross-origin browser request** (JSON body, `Authorization` header, or non-simple method) from a listed origin: browser sends `OPTIONS` first → `corsMiddleware` matches the origin, sets preflight headers, responds `204`, aborts the chain → **nothing else in this app runs** → browser, satisfied, sends the real request → that second request follows the "simple cross-origin" path above, this time actually reaching `AuthMiddleware` and the route handler.

**Cross-origin request from an unlisted origin**: `Origin` present but not in `AllowOrigins` → `403 Forbidden`, no JSON error envelope, chain aborted — visible in this service's access logs (unlike a browser-side block), but opaque to the API consumer.

**CORS disabled** (`CORS_ALLOWED_ORIGINS` empty): `corsMiddleware` is a no-op `ctx.Next()` → every request, cross-origin or not, proceeds straight through — a browser frontend calling this deployment will have every cross-origin request silently blocked *by the browser*, with no signal from this service at all.

## Final Reference — every CORS-relevant surface

| Surface | Location | Purpose |
|---|---|---|
| `CORS_ALLOWED_ORIGINS` env var | `app.env` / real environment | Comma-separated allowlist; blank disables CORS entirely |
| `Config.CORSAllowedOrigins` | [internal/config/config.go:18](../internal/config/config.go#L18) | Typed, decoded (comma-split) form of the env var |
| `corsMiddleware` | [internal/api/server.go:121-133](../internal/api/server.go#L121-L133) | Builds the actual `gin.HandlerFunc`, or a no-op if the list is empty |
| Middleware registration | [internal/api/server.go:68-73](../internal/api/server.go#L68-L73) | Fixes `corsMiddleware`'s position in the chain — before `ErrorHandlerMiddleware`/`RecoveryMiddleware`, and (via group attachment order) before `AuthMiddleware` |
| Startup log line | [internal/api/server.go:75-82](../internal/api/server.go#L75-L82) | Only observability into which branch (enabled/disabled) a given run took |
| `AllowMethods`/`AllowHeaders` | [internal/api/server.go:128-129](../internal/api/server.go#L128-L129) | Fixed per-deployment method/header allowlist — code change + redeploy required, not config-driven |

## Adding or changing allowed origins

1. Set `CORS_ALLOWED_ORIGINS` in `app.env` (local dev) or the real environment variable (staging/production) to a comma-separated list of exact origins — scheme, host, and port must match what the browser sends, e.g. `https://app.example.com` (no trailing slash, no path).
2. Multiple frontends are supported by listing more than one origin: `CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com`.
3. No code change is needed to update the list — `corsMiddleware` is rebuilt from `cfg.CORSAllowedOrigins` on every server start, same as every other config-driven setting (see [CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md)).
4. If a new method or request header is needed by the frontend (e.g. a custom header), add it to `AllowMethods`/`AllowHeaders` in `corsMiddleware` ([internal/api/server.go](../internal/api/server.go)) — these are fixed per-deployment, not configurable via environment variables, since they describe what this API's handlers actually accept, not something that varies by environment.
