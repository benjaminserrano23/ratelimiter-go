package limiter

import "time"

// Result represents the outcome of a rate limit check.
type Result struct {
	Allowed   bool      `json:"allowed"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

// Limiter is the interface for rate limiting algorithms.
type Limiter interface {
	Allow(key string, limit int, window time.Duration) Result
}
