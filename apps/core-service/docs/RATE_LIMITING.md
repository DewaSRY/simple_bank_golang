# Global Rate Limiting — As Implemented

## Who this doc is for

You're assumed to be comfortable with Go and HTTP middleware, but you haven't necessarily reasoned about *why* a rate limiter is usually a token bucket rather than a fixed counter. If you have, skip Section 0 — it's a primer, not load-bearing for the rest of the doc.

Everything below is verified against the source as it exists today, not the idealized behavior "add rate limiting" implies. In particular: the limiter is a small, hand-rolled, in-process token bucket ([internal/rate-limiter/limiter.go](../internal/rate-limiter/limiter.go)) — there is no Redis/shared store, no distributed coordination, and (as Section 4 covers) its IP key is trusted from request headers by default in a way that's worth understanding before you rely on it for anything security-critical.

## Section 0 — Background Primer: token bucket vs. the alternatives

| Approach | Bursts allowed? | Memory cost | Precision | Typical use |
|---|---|---|---|---|
| Fixed window counter (reset every N seconds) | Yes, at window boundaries (2x burst possible right at the edge) | One counter per key | Coarse — boundary effect | Simple, cheap, "good enough" limits |
| Sliding window log (timestamp per request) | No | Grows with request volume | Exact | Low-volume, precision-critical limits |
| **Token bucket (this project)** | Yes, up to `burst` tokens, then throttled to the refill rate | One bucket per key, constant size | Exact, smooth | Sustained-rate limits that still allow short bursts |

A token bucket starts with `burst` tokens. Each allowed request consumes one. Tokens refill continuously at `requestsPerSecond` per second, capped at `burst`. This is why a burst of `burst` requests can land instantly, then subsequent requests are throttled to the refill rate — not a hard "N requests per window" ceiling with the wraparound gap that implies. This project uses [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate) (`rate.Limiter`/`rate.NewLimiter`) as the underlying primitive, not a hand-written token counter — this repo only owns the per-key map wrapped around it.

**Gotchas that surprise newcomers** (each is the subject of its own section further down):

1. There's one bucket **per key** (here, per client IP), not one global bucket — a single abusive IP is throttled without affecting anyone else. See [Section 2](#section-2--limiter--the-bucket-store).
2. The map of per-key buckets grows forever unless something evicts idle entries — this repo has a background sweep goroutine for exactly that reason. See [Section 3](#section-3--sweep-and-sweeploop--bounding-memory).
3. "Per client IP" is only as trustworthy as `ctx.ClientIP()` — and this repo's gin engine is left at its (permissive) default trusted-proxy config. See [Section 4](#section-4--the-key-ctxclientip-and-why-its-not-fully-trustworthy-here).

## Section 1 — Architecture at a Glance

The composition root is `NewServer` ([internal/api/server.go:72-80](../internal/api/server.go#L72-L80)): it conditionally registers `core.RateLimitMiddleware(cfg, nil)` on the router alongside the rest of the middleware chain, gated on `cfg.RateLimitEnabled`. `NewServer` implements no rate-limiting logic itself — it delegates to `core.RateLimitMiddleware`, which in turn delegates the actual bucket bookkeeping to the standalone `ratelimiter` package.

| Concern | Owner | Analogy |
|---|---|---|
| On/off switch + tuning | [internal/config/config.go:43-64](../internal/config/config.go#L43-L64) (`RateLimitEnabled`/`RateLimitRequestsPerSecond`/`RateLimitBurst`) | The building's posted occupancy limit, set per deployment |
| The bucket store (HTTP-agnostic) | [internal/rate-limiter/limiter.go](../internal/rate-limiter/limiter.go) (`ratelimiter.Limiter`) | The turnstile mechanism itself — has no idea it's guarding an HTTP API specifically |
| The Gin adapter | [internal/api/core/rate_limit_middleware.go](../internal/api/core/rate_limit_middleware.go) (`RateLimitMiddleware`) | The person who points the turnstile at the front door and decides "IP" is the badge it checks |
| Where the turnstile stands in line | [internal/api/server.go:72-80](../internal/api/server.go#L72-L80) (`router.Use(...)` chain) | It stands *after* `ErrorHandlerMiddleware` and *before* `RecoveryMiddleware` — deliberately, see [Cross-Feature Coupling](#cross-feature-coupling) |

It's split into two packages — `ratelimiter` (generic) and `core.RateLimitMiddleware` (HTTP-specific) — so the bucket logic has no Gin/HTTP dependency at all: `ratelimiter.Limiter` is keyed by an arbitrary `string`, and nothing in it says "IP address." `core.RateLimitMiddleware` is the only place that decides the key is `ctx.ClientIP()`. That split is also exactly why Section 4's caveat is a property of the *adapter's choice of key*, not of the bucket algorithm itself — swapping the key source (e.g. to an API key) wouldn't touch `internal/rate-limiter` at all.

## Section 2 — `Limiter` — the bucket store

**The problem it solves.** Tracking "how many requests has this IP made recently" needs a data structure that's safe for concurrent access (every request hits it) and doesn't require a shared external store for a single-instance deployment.

**How it's implemented.** [internal/rate-limiter/limiter.go:20-38](../internal/rate-limiter/limiter.go#L20-L38):

```go
type Limiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rps      rate.Limit
	burst    int
}

func New(requestsPerSecond float64, burst int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*entry),
		rps:      rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}
```

`Allow` ([internal/rate-limiter/limiter.go:42-54](../internal/rate-limiter/limiter.go#L42-L54)) is the entire request path: look up the key's bucket (lazily creating one with the configured `rps`/`burst` on first sight), stamp `lastSeen`, and delegate the actual allow/deny decision to `rate.Limiter.Allow()`.

```go
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.limiters[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.limiters[key] = e
	}
	e.lastSeen = time.Now()

	return e.limiter.Allow()
}
```

| Field | What actually happens |
|---|---|
| `mu` | A single `sync.Mutex` guards the whole map — every request serializes through it. There's no per-key locking or sharding. |
| `limiters` | `map[string]*entry`, one entry per distinct key ever seen (until swept — Section 3). |
| `rps`/`burst` | Fixed at construction time (`New`), shared by every key's bucket — there's no per-key override (e.g. a stricter limit for one specific IP). |
| `entry.lastSeen` | Updated on *every* `Allow` call, allowed or denied — this is what Section 3's sweep uses to find idle keys, not "idle" in the sense of "not being throttled." |

**Worth flagging — one mutex for every request.** Every single request that reaches this middleware (when enabled) takes the same lock, does a map lookup, and releases it. For this service's traffic profile that's fine, but it's a real, single global bottleneck if traffic volume ever grows enough to matter — there's no lock striping or `sync.Map` here.

## Section 3 — `Sweep`/`SweepLoop` — bounding memory

**The problem it solves.** `limiters` never removes an entry on its own — a flood of one-off, never-repeated client IPs (or a spoofed key, see Section 4) would otherwise grow the map forever, unbounded by traffic volume.

**How it's implemented.** [internal/rate-limiter/limiter.go:56-85](../internal/rate-limiter/limiter.go#L56-L85):

```go
func (l *Limiter) Sweep(idleTTL time.Duration) {
	cutoff := time.Now().Add(-idleTTL)

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, e := range l.limiters {
		if e.lastSeen.Before(cutoff) {
			delete(l.limiters, key)
		}
	}
}

func (l *Limiter) SweepLoop(idleTTL time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(idleTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Sweep(idleTTL)
		case <-stop:
			return
		}
	}
}
```

`RateLimitMiddleware` starts `SweepLoop` as a background goroutine with a fixed `rateLimiterIdleTTL` of `10 * time.Minute` ([internal/api/core/rate_limit_middleware.go:12-14,25](../internal/api/core/rate_limit_middleware.go#L12-L14)) — not configurable via `Config`, unlike the rate/burst values. A key idle for 10 minutes is evicted; its next request starts a brand-new bucket at full burst, exactly as if it were a first-time caller.

**`stop`, the parameter that exists only for tests.** `RateLimitMiddleware(cfg config.Config, stop <-chan struct{})` takes a `stop` channel closed to end the sweep goroutine ([internal/api/core/rate_limit_middleware.go:23](../internal/api/core/rate_limit_middleware.go#L23)). Production code (`NewServer`) always passes `nil` ([internal/api/server.go:78](../internal/api/server.go#L78)) — a nil channel never becomes ready to receive, so the goroutine simply runs for the process's lifetime, same as the server itself. Only `rate_limit_middleware_test.go` passes a real channel, to avoid leaking a goroutine per test case.

## Section 4 — The key: `ctx.ClientIP()`, and why it's not fully trustworthy here

**The problem this piece solves.** The bucket store (Section 2) is keyed by an opaque `string` — something has to decide *what* that string is for an HTTP request. `RateLimitMiddleware` decides it's `ctx.ClientIP()` ([internal/api/core/rate_limit_middleware.go:28](../internal/api/core/rate_limit_middleware.go#L28)).

**How it's implemented.** [internal/api/core/rate_limit_middleware.go:23-34](../internal/api/core/rate_limit_middleware.go#L23-L34):

```go
func RateLimitMiddleware(cfg config.Config, stop <-chan struct{}) gin.HandlerFunc {
	limiter := ratelimiter.New(cfg.RateLimitRequestsPerSecond, cfg.RateLimitBurst)
	go limiter.SweepLoop(rateLimiterIdleTTL, stop)

	return func(ctx *gin.Context) {
		if !limiter.Allow(ctx.ClientIP()) {
			Fail(ctx, TooManyRequestsErr("rate limit exceeded, try again later"))
			return
		}
		ctx.Next()
	}
}
```

**Rough edge — verified, not speculative: this app never calls `gin.Engine.SetTrustedProxies`.** Gin's own default (`gin@v1.12.0/gin.go:39-48,214-215`) is `ForwardedByClientIP: true`, `RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"}`, and `trustedCIDRs` covering `0.0.0.0/0`/`::/0` — i.e., **trust every proxy, always**, until `SetTrustedProxies` is called to narrow it. `internal/api/server.go`'s `NewServer` never calls it. The practical effect: `ctx.ClientIP()` ([`gin@v1.12.0/context.go:975`](https://github.com/gin-gonic/gin/blob/v1.12.0/context.go#L975)) will read `X-Forwarded-For`/`X-Real-IP` from *any* caller, not just a real, known reverse proxy. Concretely:

- A direct, unproxied client can set `X-Forwarded-For: <anything>` on its own request and `ClientIP()` returns that value instead of the socket's real remote address.
- Rotating that header on every request gets a brand-new token bucket every time (Section 2's `Allow` treats an unseen key as a fresh, full-burst bucket) — the rate limiter can be bypassed entirely by an attacker willing to send a different `X-Forwarded-For` value per request.
- The reverse is also possible: an attacker can set `X-Forwarded-For` to *someone else's* real IP, exhausting that IP's bucket and causing legitimate traffic from it to be throttled (a targeted denial-of-service on one victim IP).

This is not hypothetical/aspirational — it's the default behavior of the exact gin version this repo depends on (`go.mod`: `github.com/gin-gonic/gin v1.12.0`), with no code anywhere in `internal/` or `cmd/` overriding it. If this service is deployed behind a real reverse proxy/load balancer (see `docs/TERRAFORM_EC2_DEPLOY.md`), fixing this means calling `router.SetTrustedProxies([]string{...})` with that proxy's actual address(es) — narrowing which upstream is allowed to set the forwarded-for headers `ClientIP()` trusts — not a change to this doc's two packages.

## Cross-Feature Coupling

- **Middleware order in `internal/api/server.go` is a correctness property of this feature, not an implementation detail.** The chain is `RequestIDMiddleware` → `LoggingMiddleware` → `corsMiddleware` → `ErrorHandlerMiddleware` → `RateLimitMiddleware` (if enabled) → `RecoveryMiddleware` ([internal/api/server.go:72-80](../internal/api/server.go#L72-L80)). `RateLimitMiddleware` must sit *after* `ErrorHandlerMiddleware`: a throttled request is reported via `Fail(ctx, TooManyRequestsErr(...))` ([internal/api/core/rate_limit_middleware.go:29](../internal/api/core/rate_limit_middleware.go#L29)), and only `ErrorHandlerMiddleware` knows how to render an `*AppError` onto the response — see the detailed comment directly above the `router.Use(...)` block ([internal/api/server.go:69-71](../internal/api/server.go#L69-L71)).
- **A rejected request is logged at `Warn`, not `Error`, and does reach the access log.** `LoggingMiddleware`'s completion handler classifies by status code — `>= 500` is `Error`, `>= 400` (which includes this feature's `429`) is `Warn` ([internal/api/core/logging_middleware.go:144-151](../internal/api/core/logging_middleware.go#L144-L151)). A flood of `429`s is visible in logs as a wave of `Warn`-level `"request completed"` lines, not silence and not an alarming `Error` spike.
- **`ErrCodeRateLimited`/`TooManyRequestsErr` are shared, generic error-response building blocks** ([internal/api/core/apperror.go:27,92-95](../internal/api/core/apperror.go#L27)) — they live in `internal/api/core` (the shared kernel every domain package depends on), not in this feature's own files, so any future 429 elsewhere in the app (e.g. a per-endpoint limit) would reuse the same error code and JSON shape rather than inventing a new one.
- **Same `ctx.ClientIP()` footgun (Section 4) also affects the unrelated `DEVICE_FINGERPRINT_TRUSTED_IPS` auth bypass** ([internal/config/config.go:55-64](../internal/config/config.go#L55-L64), [internal/api/auth/handler_utils.go](../internal/api/auth/handler_utils.go)) — that feature also trusts `ctx.ClientIP()` as a caller's identity, for a much higher-stakes decision (skipping the only login credential this app has). Both features share the exact same underlying risk from the same unconfigured `SetTrustedProxies` gap; fixing it once (calling `SetTrustedProxies` in `NewServer`) fixes it for both, since neither feature does its own IP resolution.
- **This is single-instance, in-memory state — it does not survive a restart or scale across replicas.** Per `docs/TERRAFORM_EC2_DEPLOY.md`, this service targets a single EC2 instance today; if that ever changes to multiple instances behind a load balancer, each instance gets its own independent set of buckets, so the *effective* rate limit per client becomes `RateLimitRequestsPerSecond × number of instances` — nothing in this package or the middleware accounts for that.

## Summary — data flow

**Rate limiting disabled** (`RATE_LIMIT_ENABLED=false`/unset): `RateLimitMiddleware` is never registered on the router ([internal/api/server.go:77-79](../internal/api/server.go#L77-L79)) — every request proceeds straight through as if this feature didn't exist. No log line, no metric, nothing observable marks this state (unlike CORS's analogous "disabled" branch, which logs once at startup — see `docs/CORS.md` §3).

**Allowed request** (bucket has a token): `ctx.ClientIP()` computed → `Limiter.Allow` finds/creates that key's bucket, consumes a token, returns `true` → `ctx.Next()` → request proceeds through `RecoveryMiddleware` into routing/handlers exactly as if rate limiting didn't exist.

**Throttled request** (bucket empty): `Limiter.Allow` returns `false` → `Fail(ctx, TooManyRequestsErr(...))` records a `429` `*AppError` on the Gin context and aborts → `ErrorHandlerMiddleware` (registered earlier in the chain, but Gin's `ctx.Next()` stack unwinds back into it) renders `{"error": {"code": "RATE_LIMITED", "message": "rate limit exceeded, try again later"}}` → `LoggingMiddleware` logs the completed request at `Warn`.

**Idle key eviction** (background, independent of any single request): every `rateLimiterIdleTTL` (10 minutes), `SweepLoop` calls `Sweep`, which deletes any key whose bucket hasn't been touched in that window → that key's next request (if any) starts over with a fresh, full-burst bucket.

## Final Reference — every rate-limiting surface

| Surface | Location | Purpose |
|---|---|---|
| `RATE_LIMIT_ENABLED` / `RATE_LIMIT_REQUESTS_PER_SECOND` / `RATE_LIMIT_BURST` env vars | `app.env` / real environment | On/off switch and token-bucket tuning |
| `Config.RateLimitEnabled`/`RateLimitRequestsPerSecond`/`RateLimitBurst` | [internal/config/config.go:43-53](../internal/config/config.go#L43-L53) | Typed, decoded form of the env vars |
| `ratelimiter.Limiter` | [internal/rate-limiter/limiter.go](../internal/rate-limiter/limiter.go) | The HTTP-agnostic per-key token-bucket store (`New`, `Allow`, `Sweep`, `SweepLoop`) |
| `RateLimitMiddleware` | [internal/api/core/rate_limit_middleware.go](../internal/api/core/rate_limit_middleware.go) | Builds the `gin.HandlerFunc`, keys `Limiter.Allow` by `ctx.ClientIP()`, maps a denial to a 429 `*AppError` |
| Middleware registration | [internal/api/server.go:72-80](../internal/api/server.go#L72-L80) | Fixes the middleware's position — after `ErrorHandlerMiddleware`, before `RecoveryMiddleware` — and its on/off gate |
| `ErrCodeRateLimited` / `TooManyRequestsErr` | [internal/api/core/apperror.go:27,92-95](../internal/api/core/apperror.go#L27) | The shared error code/constructor a 429 response is built from |
| `rate_limit_middleware_test.go` | [internal/api/core/rate_limit_middleware_test.go](../internal/api/core/rate_limit_middleware_test.go) | HTTP-level tests: burst-then-reject, per-IP isolation, refill-over-time |
| `limiter_test.go` | [internal/rate-limiter/limiter_test.go](../internal/rate-limiter/limiter_test.go) | Package-level tests: burst-then-reject, per-key isolation, sweep eviction, sweep-loop shutdown |

## Adjusting the limit

1. Set `RATE_LIMIT_ENABLED=true`, `RATE_LIMIT_REQUESTS_PER_SECOND=<n>`, `RATE_LIMIT_BURST=<n>` in `app.env` (local dev) or the real environment (staging/production) — see `docs/CONFIG_ENV_VARIABLE.md`.
2. No code change is required to retune the limit — `RateLimitMiddleware` is rebuilt from `cfg` on every server start, same as every other config-driven setting.
3. The idle-eviction window (`rateLimiterIdleTTL`, currently a hardcoded `10 * time.Minute`) is **not** environment-configurable — changing it means editing the constant in [internal/api/core/rate_limit_middleware.go](../internal/api/core/rate_limit_middleware.go) and redeploying.
4. If this service is ever placed behind a real reverse proxy/load balancer, call `router.SetTrustedProxies([]string{...})` in `NewServer` with that proxy's actual address(es) before relying on this feature (or the `DEVICE_FINGERPRINT_TRUSTED_IPS` auth bypass) for anything security-sensitive — see Section 4.
