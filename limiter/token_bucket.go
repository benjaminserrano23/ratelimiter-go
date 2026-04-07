package limiter

import (
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/store"
)

// TokenBucket implements the token bucket algorithm.
type TokenBucket struct {
	store store.Store
}

func NewTokenBucket(s store.Store) *TokenBucket {
	return &TokenBucket{store: s}
}

// Allow checks if a request is allowed under the token bucket algorithm.
// limit: max tokens (burst size), window: refill period for all tokens.
func (tb *TokenBucket) Allow(key string, limit int, window time.Duration) Result {
	now := time.Now()
	refillRate := float64(limit) / window.Seconds()

	result := tb.store.ConsumeToken(key, limit, refillRate)
	tb.store.IncrMetrics(key, !result.Allowed)

	if !result.Allowed {
		return Result{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   now.Add(time.Duration(float64(time.Second) / refillRate)),
		}
	}

	return Result{
		Allowed:   true,
		Remaining: int(result.Remaining),
		ResetAt:   now.Add(window),
	}
}
