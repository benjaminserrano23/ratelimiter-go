package limiter_test

import (
	"testing"
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/limiter"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

func TestSlidingWindow_AllowsUpToLimit(t *testing.T) {
	s := store.NewMemoryStore()
	sw := limiter.NewSlidingWindow(s)

	limit := 3
	window := time.Minute

	for i := 0; i < limit; i++ {
		result := sw.Allow("user1", limit, window)
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	result := sw.Allow("user1", limit, window)
	if result.Allowed {
		t.Fatal("request beyond limit should be denied")
	}
}

func TestSlidingWindow_DifferentKeys(t *testing.T) {
	s := store.NewMemoryStore()
	sw := limiter.NewSlidingWindow(s)

	r1 := sw.Allow("a", 1, time.Minute)
	r2 := sw.Allow("b", 1, time.Minute)

	if !r1.Allowed || !r2.Allowed {
		t.Fatal("different keys should be independent")
	}
}
