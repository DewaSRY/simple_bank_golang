# Logging — As Implemented

## Who this doc is for

You're comfortable with Go, but you haven't necessarily used `log/slog` (stdlib since Go 1.21) before. If you have, skip Section 0. Everything after that is verified against the source as it exists today — file:line cited throughout — not an idealized design. This system replaced a previous state with no logging at all beyond the stdlib `log` package in `main.go`/`connect_db.go` and Gin's own unstructured default logger; the motivating problem was that a client-visible `500 internal server error` had **no corresponding log line anywhere** — the real error was discarded the moment a handler called `InternalErr()`. That gap, and how it's closed, is covered in §3.

## Section 0 — Background primer: why `log/slog`, not a third-party logger

| Approach | Structured output | Levels | Extra dependency | Used here |
|---|---|---|---|---|
| stdlib `log` | No — plain strings | No | No | What this replaced |
| `log/slog` (stdlib) | Yes — key/value attrs, JSON or text | Yes | No | **What this uses** |
| `zap` / `zerolog` / `logrus` | Yes | Yes | Yes | Not used |

`log/slog` was chosen purely because it's already in the standard library (Go 1.25 per `go.mod`) and does everything this service needs — leveled, structured (JSON or text), with a handler interface for swapping output format — without adding a dependency for a job the stdlib already does. Nothing here rules out `zap`/`zerolog` later if throughput ever becomes a concern; `internal/logger.New` is the one place that would change.

**Gotchas that surprise newcomers:**

1. `slog.Logger.With(...)` returns a *new* logger carrying the extra attributes — it doesn't mutate the receiver. `loggerFromContext` ([internal/api/logging_middleware.go:57-62](../internal/api/logging_middleware.go#L57-L62)) relies on this: every call produces an independent logger scoped to one request's `request_id`, and there's no shared mutable state between concurrent requests to worry about.
2. A `slog.HandlerOptions.Level` filters at the handler, not the call site — calling `log.Debug(...)` when the configured level is `info` is a real function call that does real attribute formatting before being dropped inside the handler. This is why `internal/logger.New` also gates `AddSource` on the same level check ([internal/logger/logger.go:31](../internal/logger/logger.go#L31)) — turning on source-line tracking has a real per-line cost, so it's opt-in via `LOG_LEVEL=debug`, not always-on.

## Section 1 — Architecture at a glance

The composition root is `cmd/server/main.go`: it builds one `*slog.Logger` via `internal/logger.New(cfg)` ([cmd/server/main.go:42](../cmd/server/main.go#L42)) and threads it into `connectDB` and `api.NewServer` — no package below that constructs its own logger or falls back to a package-level global. `internal/logger` itself does nothing but build the logger from config; it contains no logging call sites of its own.

| Concern | Owner | Analogy |
|---|---|---|
| Building the configured `*slog.Logger` | [internal/logger/logger.go](../internal/logger/logger.go) (`New`) | The print shop that sets up the press once, to a spec |
| Assigning/propagating a request ID | [internal/api/logging_middleware.go:27-55](../internal/api/logging_middleware.go#L27-L55) (`requestIDMiddleware`) | The claim-ticket stub every request gets stapled to |
| One access-log line per request | [internal/api/logging_middleware.go:70-102](../internal/api/logging_middleware.go#L70-L102) (`loggingMiddleware`) | The security log book everyone signs on the way out |
| Logging + rendering an `*AppError` | [internal/api/apperror.go:108-155](../internal/api/apperror.go#L108-L155) (`errorHandlerMiddleware`, `logAppError`) | The incident report filed before the sanitized public statement goes out |
| Catching panics, logging them, converting to a 500 | [internal/api/logging_middleware.go:124-139](../internal/api/logging_middleware.go#L124-L139) (`recoveryMiddleware`) | The emergency responder who both writes the report and calms the room |

It's split this way so the *what to log* (access lines, panics, errors) stays separate from *how it's formatted* (`internal/logger`) — swapping JSON for text, or the whole handler for a different library, touches one file, not five.

## Section 2 — Configuring what gets logged: `LOG_LEVEL` / `LOG_FORMAT`

**The problem it solves.** Different environments want different amounts of noise — a local dev loop wants `debug` with readable text; production wants `info` (or higher) as JSON for a log aggregator to parse. Instead of a hardcoded logger, both are `Config` fields read the same way every other setting is (`docs/CONFIG_ENV_VARIABLE.md`).

**How it's implemented** — [internal/logger/logger.go](../internal/logger/logger.go), in full:

```go
func New(cfg config.Config) *slog.Logger {
	level := parseLevel(cfg.LogLevel)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level <= slog.LevelDebug,
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.LogFormat, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
```

| `LOG_LEVEL` | Effect |
|---|---|
| `debug` | Everything, plus `source: {file, line, function}` on every record |
| `info` (default; also anything empty/unrecognized) | Access lines, warnings, errors — no per-call-site debug noise |
| `warn` | 4xx/5xx only — access lines for 2xx/3xx are dropped |
| `error` | 5xx and panics only |

`LOG_FORMAT` is `json` (default; also anything other than exactly `"text"`, case-insensitively) or `text`. Neither variable is validated against config load the way `JWTSecretKey` is (`docs/CONFIG_ENV_VARIABLE.md` §4) — a typo like `LOG_LEVEL=infp` silently falls back to `info` rather than erroring, by design (`parseLevel`'s `default` case, [internal/logger/logger.go:44-56](../internal/logger/logger.go#L44-L56)): a misconfigured log level shouldn't be able to take the service down.

**Rough edge worth flagging:** there's no per-package or per-request level override — one `LOG_LEVEL` governs the whole process for its lifetime. Turning on `debug` to trace one noisy endpoint also turns on `debug` (and its `AddSource` cost) for every other request the process is handling concurrently.

## Section 3 — Why every failure now has a matching log line

This is the change that actually motivated this doc. Before it, `InternalErr()` took no arguments:

```go
// old shape
func InternalErr() *AppError {
	return newAppError(http.StatusInternalServerError, errCodeInternal, "internal server error")
}
```

Every call site looked like `fail(ctx, InternalErr())` after an `if err != nil` check — meaning `err`, the actual DB failure/token error/whatever went wrong, was in scope and then thrown away, never logged, never stored anywhere. A `500` in production had exactly as much debuggable information as the string `"internal server error"`.

`AppError` now carries a `Cause error` field ([internal/api/apperror.go:20-33](../internal/api/apperror.go#L20-L33)), and `InternalErr(err error)` requires the caller to pass it:

```go
func InternalErr(err error) *AppError {
	appErr := newAppError(http.StatusInternalServerError, errCodeInternal, "internal server error")
	appErr.Cause = err
	return appErr
}
```

`Cause` is never serialized into the response — `errorBody` ([internal/api/response.go:30-34](../internal/api/response.go#L30-L34)) has no field for it, so the client-facing contract documented in `docs/NORMALIZE_RESPONSE.md` is unchanged. It exists purely for `logAppError` ([internal/api/apperror.go:144-155](../internal/api/apperror.go#L144-L155)) to log:

```go
func logAppError(log *slog.Logger, appErr *AppError) {
	if appErr.Status < http.StatusInternalServerError {
		log.Warn("request failed", slog.Int("status", appErr.Status), slog.String("code", appErr.Code), slog.String("message", appErr.Message))
		return
	}

	attrs := []any{slog.Int("status", appErr.Status), slog.String("code", appErr.Code)}
	if appErr.Cause != nil {
		attrs = append(attrs, slog.String("cause", appErr.Cause.Error()))
	}
	log.Error("request failed", attrs...)
}
```

`errorHandlerMiddleware` calls this — with a logger already carrying `request_id`/`method`/`path` via `loggerFromContext` — right before rendering the response ([internal/api/apperror.go:108-138](../internal/api/apperror.go#L108-L138)). So a `500` now always has a matching `level=ERROR msg="request failed" ... cause="..."` line, and a `4xx` gets a `level=WARN` line — visible, but not something that pages anyone.

**Mechanical consequence:** every one of the ~26 existing `fail(ctx, InternalErr())` call sites across the handler files, plus the two `InternalErr()` uses inside `transferAppError`'s `default` branch, became `InternalErr(err)`. This was purely mechanical — every call site already had `err` in scope from the preceding `if err != nil` check — but it's worth knowing if you're grepping for the old zero-arg signature and not finding it.

## Section 4 — Request IDs and the access log

**The problem it solves.** A client reports "I got a 500" — you need the exact log lines for *that* request, not just "some 500 happened sometime today."

`requestIDMiddleware` ([internal/api/logging_middleware.go:27-38](../internal/api/logging_middleware.go#L27-L38)) reuses an inbound `X-Request-Id` header if the caller already set one (useful if there's an upstream proxy/load balancer assigning them), or generates a 24-hex-character one via `crypto/rand` otherwise, and echoes it back on the response via the same header. Every subsequent log line for that request — the access log, `logAppError`, `recoveryMiddleware`'s panic log — includes it.

`loggingMiddleware` ([internal/api/logging_middleware.go:70-102](../internal/api/logging_middleware.go#L70-L102)) logs exactly one line per request, after it completes, with `request_id`, `method`, `path` (query string included), `status`, `latency`, `client_ip`, and — if `authMiddleware` set one — `user_id`. Level scales with the response: `Error` for 5xx, `Warn` for 4xx, `Info` otherwise.

**Worth flagging:** `authenticatedUserID` ([internal/api/logging_middleware.go:104-118](../internal/api/logging_middleware.go#L104-L118)) exists as a *separate*, panic-safe read of the same context key `getAuthPayload` ([internal/api/auth_middleware.go:64-70](../internal/api/auth_middleware.go#L64-L70)) uses — `getAuthPayload` calls `ctx.MustGet`, which panics if the key was never set, and is only safe because every handler that calls it is registered behind `authMiddleware`. `loggingMiddleware` runs for *every* route, including `/health` and the unauthenticated auth endpoints, so it uses `ctx.Get` (the non-panicking form) instead. Reusing `getAuthPayload` here would panic on every unauthenticated request.

## Section 5 — Panic recovery, and the ordering bug this system already hit once

**The problem it solves.** An unhandled panic in a handler shouldn't crash the process, and — same motivation as §3 — shouldn't vanish without a trace either.

`recoveryMiddleware` ([internal/api/logging_middleware.go:124-139](../internal/api/logging_middleware.go#L124-L139)) replaces Gin's own `gin.Recovery()` (which `gin.Default()` — no longer used, see §6 — would have bundled in). On `recover()`, it logs the panic value, method, path, request ID, and a full stack trace at `Error` level, then calls `fail(ctx, InternalErr(fmt.Errorf("panic: %v", r)))` so the panic gets rendered through the exact same sanitized-500 path as any other internal error.

**This only works if `recoveryMiddleware` is registered *after* `errorHandlerMiddleware`** in `NewServer` ([internal/api/server.go:67-72](../internal/api/server.go#L67-L72)) — getting this backwards was a real bug caught while building this system (not shipped, but worth documenting so it isn't reintroduced). Gin middleware nests through `ctx.Next()`: calling it invokes the next middleware in the chain, and code written *after* that call only runs once the nested call returns. A panic doesn't return normally — it unwinds past every `ctx.Next()` call, skipping all post-`ctx.Next()` code, until something's `recover()` catches it. Concretely:

- **Correct order (`errorHandlerMiddleware`, then `recoveryMiddleware`, then routes):** a panic unwinds past the handler and `recoveryMiddleware`'s own `ctx.Next()` call, gets caught by `recoveryMiddleware`'s `defer/recover`, which calls `fail(ctx, ...)` and returns normally. Control passes back up through `errorHandlerMiddleware`'s `ctx.Next()` call as if nothing unusual happened, so its post-`ctx.Next()` rendering code runs and actually writes the 500.
- **Backwards (`recoveryMiddleware` before `errorHandlerMiddleware`):** the panic still gets caught and `fail(ctx, ...)` still runs, but `errorHandlerMiddleware`'s rendering code is now *below* the recovery point in the call stack and its own `ctx.Next()` call already panicked and unwound — its post-`ctx.Next()` code never runs at all for this request. `fail`'s `ctx.Error(err)` sets state nothing ever reads, `ctx.Abort()` stops handlers that have already stopped, and the client receives an empty `200 OK` for what was actually an unhandled panic.

There's a regression test for exactly this — `TestRecoveryMiddleware_RendersSanitized500` ([internal/api/logging_middleware_test.go](../internal/api/logging_middleware_test.go)) builds the same middleware chain (in the same order) `NewServer` does and asserts a panicking handler still produces a `500` with the sanitized body, not a `200`.

## Section 6 — What replaced `gin.Default()`

`gin.Default()` bundles Gin's own `Logger()` (unstructured, `[GIN] 2024/...` lines to stdout) and `Recovery()` (plain-text stack trace on panic, not leveled or request-scoped). `NewServer` now builds a bare `gin.New()` and registers, in order ([internal/api/server.go:67-72](../internal/api/server.go#L67-L72)):

```go
router := gin.New()
router.Use(requestIDMiddleware())
router.Use(loggingMiddleware(log))
router.Use(corsMiddleware(cfg.CORSAllowedOrigins))
router.Use(errorHandlerMiddleware(log))
router.Use(recoveryMiddleware(log))
```

`corsMiddleware`'s position between `loggingMiddleware` and `errorHandlerMiddleware` isn't load-bearing — it doesn't read or write `ctx.Errors` and doesn't panic — it's grouped there because it, too, needs to run before every route. `requestIDMiddleware` must be outermost (or at least before `loggingMiddleware`) so the access log line has a `request_id` to attach; `loggingMiddleware` must wrap everything else so `ctx.Writer.Status()` reflects the final rendered status, including one written by `errorHandlerMiddleware` or a recovered panic (§5).

**Startup also logs, once, which CORS mode a given run is in** ([internal/api/server.go:73-79](../internal/api/server.go#L73-L79)) — `"CORS disabled: ..."` or `"CORS enabled"` with the origin list — closing a gap `docs/CONFIG_ENV_VARIABLE.md` (§4) used to call out explicitly ("nothing logs which case you're in"). This is observability only; `Config` still can't distinguish "unset" from "deliberately empty" at the type level.

## Cross-feature coupling

- **`main.go`/`connect_db.go` no longer use the stdlib `log` package**, except for the one config-load failure that happens *before* a `Config` (and therefore a configured logger) exists ([cmd/server/main.go:35-40](../cmd/server/main.go#L35-L40)). Every other startup failure — DB open/ping, port already in use, server start — now goes through the same `*slog.Logger` as request handling, followed by `os.Exit(1)` (slog has no `Fatal`).
- **Tests that build a router directly** (`internal/api/auth_middleware_test.go`, `internal/api/logging_middleware_test.go`) pass a discard-writing `*slog.Logger` rather than `nil` — `NewServer` treats `nil` as "use `slog.Default()`" ([internal/api/server.go:42-44](../internal/api/server.go#L42-L44)) for convenience, but that writes to the process's actual stdout, which is noisy in test output; tests construct their own to keep it silent.
- **`docs/NORMALIZE_RESPONSE.md` and `docs/CONFIG_ENV_VARIABLE.md` both cross-reference this doc** for the parts of their own subject that changed here (`InternalErr`'s signature, `errorHandlerMiddleware`'s signature and behavior, the CORS startup log) rather than duplicating the explanation — read this doc first if either of those sections looks inconsistent with the current source.

## Known gaps / not done here

- **No log sampling or rate limiting.** A misbehaving client hammering an endpoint that fails produces one `Error`-level line per request, with no backoff — fine at this service's current scale, a real cost if that changes.
- **No per-request/per-package level override**, per §2's rough edge.
- **`Cause.Error()` is logged as a flat string**, not wrapped/unwrapped further — if the real error is itself a wrapped chain (`fmt.Errorf("...: %w", ...)`), the log line shows the full formatted chain as one string rather than structured fields per layer.
- **No log shipping/aggregation is configured in this repo** — `internal/logger.New` writes to `os.Stdout` only; whatever collects that (Docker's log driver, a sidecar, CloudWatch, etc.) is deploy-environment configuration outside this codebase.
