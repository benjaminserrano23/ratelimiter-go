package store

import "time"

// Entry holds rate limit state for a single key.
type Entry struct {
	Tokens    float64
	LastRefill time.Time
	Requests  []time.Time // for sliding window
}

// Store defines the interface for rate limit state storage.
type Store interface {
	// GetTokenBucket returns the current token count and last refill time.
	GetTokenBucket(key string) (tokens float64, lastRefill time.Time, exists bool)
	// SetTokenBucket updates the token bucket state.
	SetTokenBucket(key string, tokens float64, lastRefill time.Time)

	// GetSlidingWindow returns request timestamps within the window.
	GetSlidingWindow(key string, windowStart time.Time) []time.Time
	// AddSlidingWindow records a new request timestamp.
	AddSlidingWindow(key string, now time.Time)

	// GetMetrics returns total requests and denied requests per key.
	GetMetrics() map[string][2]int64
	// IncrMetrics increments total and optionally denied count.
	IncrMetrics(key string, denied bool)
}
