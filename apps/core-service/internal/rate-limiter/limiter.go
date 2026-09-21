// Package ratelimiter implements an in-process, per-key token-bucket rate
// limiter. It has no HTTP dependency — internal/api/core/rate_limit_middleware.go
// is the Gin adapter that keys it by client IP.
package ratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// entry is one key's token bucket plus when it was last used, so idle
// entries can be swept out of the map instead of accumulating forever.
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter holds one token-bucket limiter per key, in process memory. There
// is no shared/external store (e.g. Redis) — counters reset on every restart
// and are not shared across instances.
type Limiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rps      rate.Limit
	burst    int
}

// New builds a Limiter whose buckets refill at requestsPerSecond and hold up
// to burst tokens.
func New(requestsPerSecond float64, burst int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*entry),
		rps:      rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// Allow reports whether key may act right now, creating a fresh token bucket
// for key on its first use.
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

// Sweep removes entries idle longer than idleTTL, so a burst of one-off
// unique keys doesn't grow the map without bound.
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

// SweepLoop runs Sweep every idleTTL until stop is closed. It's started as a
// goroutine that lives for the process's lifetime, same as the server.
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
