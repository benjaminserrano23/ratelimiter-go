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

	// Get requests within the window
	requests := sw.store.GetSlidingWindow(key, windowStart)

	if len(requests) >= limit {
		// Denied
		sw.store.IncrMetrics(key, true)
		// Reset at = oldest request in window + window duration
		resetAt := now.Add(window)
		if len(requests) > 0 {
			resetAt = requests[0].Add(window)
		}
		return Result{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   resetAt,
		}
	}

	// Allowed — record the request
	sw.store.AddSlidingWindow(key, now)
	sw.store.IncrMetrics(key, false)

	return Result{
		Allowed:   true,
		Remaining: limit - len(requests) - 1,
		ResetAt:   now.Add(window),
	}
}
