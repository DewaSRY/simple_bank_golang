package core

import (
	"time"

	"github.com/gin-gonic/gin"

	config "github.com/DewaSRY/core-service/internal/config"
	ratelimiter "github.com/DewaSRY/core-service/internal/rate-limiter"
)

const rateLimiterIdleTTL = 10 * time.Minute

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
