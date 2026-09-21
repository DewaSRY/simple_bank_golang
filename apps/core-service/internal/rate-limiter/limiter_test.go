package ratelimiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ratelimiter "github.com/DewaSRY/core-service/internal/rate-limiter"
)

func TestLimiter_AllowsUpToBurstThenRejects(t *testing.T) {
	l := ratelimiter.New(1, 3)

	for i := 0; i < 3; i++ {
		require.True(t, l.Allow("key"), "request %d within burst should be allowed", i+1)
	}
	require.False(t, l.Allow("key"))
}

func TestLimiter_TracksKeysIndependently(t *testing.T) {
	l := ratelimiter.New(1, 1)

	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"), "second request for the same key should be throttled")
	require.True(t, l.Allow("b"), "a different key should have its own bucket")
}

func TestLimiter_SweepEvictsIdleKeys(t *testing.T) {
	l := ratelimiter.New(1, 1)

	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"))

	l.Sweep(0) // idleTTL of 0: everything seen before "now" is evicted

	require.True(t, l.Allow("a"), "a swept key should get a fresh bucket")
}

func TestLimiter_SweepLoopStopsOnClose(t *testing.T) {
	l := ratelimiter.New(1, 1)
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		l.SweepLoop(time.Millisecond, stop)
		close(done)
	}()

	close(stop)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SweepLoop did not stop after stop was closed")
	}
}
