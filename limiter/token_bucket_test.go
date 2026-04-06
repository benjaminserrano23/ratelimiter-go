package limiter_test

import (
	"testing"
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/limiter"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

func TestTokenBucket_AllowsUpToLimit(t *testing.T) {
	s := store.NewMemoryStore()
	tb := limiter.NewTokenBucket(s)

	limit := 5
	window := time.Minute

	for i := 0; i < limit; i++ {
		result := tb.Allow("user1", limit, window)
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	result := tb.Allow("user1", limit, window)
	if result.Allowed {
		t.Fatal("request beyond limit should be denied")
	}
	if result.Remaining != 0 {
		t.Fatalf("remaining should be 0, got %d", result.Remaining)
	}
}

func TestTokenBucket_DifferentKeys(t *testing.T) {
	s := store.NewMemoryStore()
	tb := limiter.NewTokenBucket(s)

	r1 := tb.Allow("key-a", 1, time.Minute)
	r2 := tb.Allow("key-b", 1, time.Minute)

	if !r1.Allowed || !r2.Allowed {
		t.Fatal("different keys should be independent")
	}

	r3 := tb.Allow("key-a", 1, time.Minute)
	if r3.Allowed {
		t.Fatal("key-a should be exhausted")
	}

	r4 := tb.Allow("key-b", 1, time.Minute)
	if r4.Allowed {
		t.Fatal("key-b should be exhausted")
	}
}
