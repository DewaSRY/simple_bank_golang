package core

import (
	"time"

	"github.com/gin-gonic/gin"

	config "github.com/DewaSRY/core-service/internal/config"
	ratelimiter "github.com/DewaSRY/core-service/internal/rate-limiter"
)

// rateLimiterIdleTTL is how long a client IP's bucket is kept around with no
// requests before it's evicted from memory.
const rateLimiterIdleTTL = 10 * time.Minute

// RateLimitMiddleware rejects a request with 429 once its client IP has
// exceeded cfg.RateLimitRequestsPerSecond/cfg.RateLimitBurst. It applies
// globally, before any per-route auth check, so it limits every request
// (public and authorized alike) by IP alone.
//
// stop, if non-nil, is closed to stop the background sweep goroutine (tests
// use this to avoid leaking goroutines); production callers can pass nil.
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
