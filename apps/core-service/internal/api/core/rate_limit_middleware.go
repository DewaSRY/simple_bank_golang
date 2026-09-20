package core

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	config "github.com/DewaSRY/core-service/internal/config"
)

// ipLimiterEntry is one client IP's token bucket plus when it was last used,
// so idle entries can be swept out of the map instead of accumulating
// forever.
type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipLimiterStore holds one token-bucket limiter per client IP, in process
// memory. There is no shared/external store (e.g. Redis) — counters reset on
// every restart and are not shared across instances, which matches this
// service's current single-instance deployment.
type ipLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiterEntry
	rps      rate.Limit
	burst    int
}

func newIPLimiterStore(requestsPerSecond float64, burst int) *ipLimiterStore {
	return &ipLimiterStore{
		limiters: make(map[string]*ipLimiterEntry),
		rps:      rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// allow reports whether ip may make a request right now, creating a fresh
// token bucket for ip on its first request.
func (s *ipLimiterStore) allow(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.limiters[ip]
	if !ok {
		entry = &ipLimiterEntry{limiter: rate.NewLimiter(s.rps, s.burst)}
		s.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()

	return entry.limiter.Allow()
}

// sweep removes entries idle longer than idleTTL, so a burst of one-off
// unique client IPs doesn't grow the map without bound.
func (s *ipLimiterStore) sweep(idleTTL time.Duration) {
	cutoff := time.Now().Add(-idleTTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	for ip, entry := range s.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(s.limiters, ip)
		}
	}
}

// sweepLoop runs sweep every idleTTL until stop is closed. It's started as a
// goroutine that lives for the process's lifetime, same as the server.
func (s *ipLimiterStore) sweepLoop(idleTTL time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(idleTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sweep(idleTTL)
		case <-stop:
			return
		}
	}
}

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
	store := newIPLimiterStore(cfg.RateLimitRequestsPerSecond, cfg.RateLimitBurst)
	go store.sweepLoop(rateLimiterIdleTTL, stop)

	return func(ctx *gin.Context) {
		if !store.allow(ctx.ClientIP()) {
			Fail(ctx, TooManyRequestsErr("rate limit exceeded, try again later"))
			return
		}
		ctx.Next()
	}
}
