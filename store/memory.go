package store

import (
	"sync"
	"time"
)

type memoryStore struct {
	mu       sync.RWMutex
	buckets  map[string]*tokenBucketEntry
	windows  map[string][]time.Time
	metrics  map[string][2]int64 // [total, denied]
	stopCh   chan struct{}
}

type tokenBucketEntry struct {
	tokens     float64
	lastRefill time.Time
}

// NewMemoryStore creates an in-memory store with a background cleanup goroutine
// that removes stale keys every 60 seconds.
func NewMemoryStore() *memoryStore {
	s := &memoryStore{
		buckets: make(map[string]*tokenBucketEntry),
		windows: make(map[string][]time.Time),
		metrics: make(map[string][2]int64),
		stopCh:  make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Close stops the background cleanup goroutine.
func (m *memoryStore) Close() {
	close(m.stopCh)
}

// cleanup removes token bucket entries idle for >5 minutes and
// sliding window entries with no timestamps in the current window.
func (m *memoryStore) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case now := <-ticker.C:
			m.mu.Lock()
			cutoff := now.Add(-5 * time.Minute)
			for key, entry := range m.buckets {
				if entry.lastRefill.Before(cutoff) {
					delete(m.buckets, key)
				}
			}
			for key, timestamps := range m.windows {
				valid := make([]time.Time, 0, len(timestamps))
				for _, t := range timestamps {
					if !t.Before(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(m.windows, key)
				} else {
					m.windows[key] = valid
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *memoryStore) GetTokenBucket(key string) (float64, time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	e, ok := m.buckets[key]
	if !ok {
		return 0, time.Time{}, false
	}
	return e.tokens, e.lastRefill, true
}

func (m *memoryStore) SetTokenBucket(key string, tokens float64, lastRefill time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets[key] = &tokenBucketEntry{tokens: tokens, lastRefill: lastRefill}
}

// ConsumeToken atomically refills and consumes a token.
func (m *memoryStore) ConsumeToken(key string, limit int, refillRate float64) TokenBucketResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, exists := m.buckets[key]
	if !exists {
		m.buckets[key] = &tokenBucketEntry{tokens: float64(limit - 1), lastRefill: now}
		return TokenBucketResult{Allowed: true, Remaining: float64(limit - 1)}
	}

	elapsed := now.Sub(e.lastRefill)
	e.tokens += elapsed.Seconds() * refillRate
	if e.tokens > float64(limit) {
		e.tokens = float64(limit)
	}
	e.lastRefill = now

	if e.tokens < 1 {
		return TokenBucketResult{Allowed: false, Remaining: e.tokens}
	}

	e.tokens--
	return TokenBucketResult{Allowed: true, Remaining: e.tokens}
}

// AddSlidingWindowIfAllowed atomically checks and adds a request if under limit.
func (m *memoryStore) AddSlidingWindowIfAllowed(key string, windowStart time.Time, now time.Time, limit int) (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	timestamps := m.windows[key]
	valid := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if !t.Before(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		m.windows[key] = valid
		return false, len(valid)
	}

	valid = append(valid, now)
	m.windows[key] = valid
	return true, len(valid)
}

func (m *memoryStore) GetSlidingWindow(key string, windowStart time.Time) []time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	timestamps := m.windows[key]
	valid := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if !t.Before(windowStart) {
			valid = append(valid, t)
		}
	}
	m.windows[key] = valid
	return valid
}

func (m *memoryStore) AddSlidingWindow(key string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.windows[key] = append(m.windows[key], now)
}

func (m *memoryStore) GetMetrics() map[string][2]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][2]int64, len(m.metrics))
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

func (m *memoryStore) IncrMetrics(key string, denied bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := m.metrics[key]
	entry[0]++
	if denied {
		entry[1]++
	}
	m.metrics[key] = entry
}
