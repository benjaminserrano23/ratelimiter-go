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
}

type tokenBucketEntry struct {
	tokens     float64
	lastRefill time.Time
}

func NewMemoryStore() Store {
	return &memoryStore{
		buckets: make(map[string]*tokenBucketEntry),
		windows: make(map[string][]time.Time),
		metrics: make(map[string][2]int64),
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

func (m *memoryStore) GetSlidingWindow(key string, windowStart time.Time) []time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	timestamps := m.windows[key]
	// Prune old entries
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
