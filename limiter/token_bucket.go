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
	tokens, lastRefill, exists := tb.store.GetTokenBucket(key)

	if !exists {
		// First request — initialize with limit-1 tokens (this request uses one)
		tb.store.SetTokenBucket(key, float64(limit-1), now)
		tb.store.IncrMetrics(key, false)
		return Result{
			Allowed:   true,
			Remaining: limit - 1,
			ResetAt:   now.Add(window),
		}
	}

	// Calculate tokens to add based on elapsed time
	elapsed := now.Sub(lastRefill)
	refillRate := float64(limit) / window.Seconds() // tokens per second
	tokens += elapsed.Seconds() * refillRate

	// Cap at max
	if tokens > float64(limit) {
		tokens = float64(limit)
	}

	if tokens < 1 {
		// Denied
		tb.store.SetTokenBucket(key, tokens, now)
		tb.store.IncrMetrics(key, true)
		return Result{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   now.Add(time.Duration(float64(time.Second) / refillRate)),
		}
	}

	tokens--
	tb.store.SetTokenBucket(key, tokens, now)
	tb.store.IncrMetrics(key, false)

	return Result{
		Allowed:   true,
		Remaining: int(tokens),
		ResetAt:   now.Add(window),
	}
}
