package limiter_test

import (
	"sync"
	"testing"
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/limiter"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

func TestTokenBucket_Concurrent(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	tb := limiter.NewTokenBucket(s)

	limit := 100
	window := time.Minute
	goroutines := 50
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	allowed := make(chan bool, goroutines*requestsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				result := tb.Allow("concurrent-key", limit, window)
				allowed <- result.Allowed
			}
		}()
	}

	wg.Wait()
	close(allowed)

	totalAllowed := 0
	for a := range allowed {
		if a {
			totalAllowed++
		}
	}

	// Should allow at most `limit` requests
	if totalAllowed > limit {
		t.Fatalf("allowed %d requests, but limit is %d", totalAllowed, limit)
	}

	// Should allow at least 1 request
	if totalAllowed == 0 {
		t.Fatal("no requests were allowed")
	}
}

func TestSlidingWindow_Concurrent(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	sw := limiter.NewSlidingWindow(s)

	limit := 50
	window := time.Minute
	goroutines := 20
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	allowed := make(chan bool, goroutines*requestsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				result := sw.Allow("concurrent-sw", limit, window)
				allowed <- result.Allowed
			}
		}()
	}

	wg.Wait()
	close(allowed)

	totalAllowed := 0
	for a := range allowed {
		if a {
			totalAllowed++
		}
	}

	if totalAllowed > limit {
		t.Fatalf("allowed %d requests, but limit is %d", totalAllowed, limit)
	}
	if totalAllowed == 0 {
		t.Fatal("no requests were allowed")
	}
}
