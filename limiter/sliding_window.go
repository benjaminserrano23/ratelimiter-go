package limiter

import (
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/store"
)

// SlidingWindow implements the sliding window log algorithm.
type SlidingWindow struct {
	store store.Store
}

func NewSlidingWindow(s store.Store) *SlidingWindow {
	return &SlidingWindow{store: s}
}

// Allow checks if a request is allowed under the sliding window algorithm.
func (sw *SlidingWindow) Allow(key string, limit int, window time.Duration) Result {
	now := time.Now()
	windowStart := now.Add(-window)

	allowed, count := sw.store.AddSlidingWindowIfAllowed(key, windowStart, now, limit)
	sw.store.IncrMetrics(key, !allowed)

	if !allowed {
		return Result{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   now.Add(window),
		}
	}

	return Result{
		Allowed:   true,
		Remaining: limit - count,
		ResetAt:   now.Add(window),
	}
}
