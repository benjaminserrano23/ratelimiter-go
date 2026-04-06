package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/benjaminserrano23/ratelimiter-go/limiter"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

type checkRequest struct {
	Key       string `json:"key"`
	Limit     int    `json:"limit"`
	Window    string `json:"window"` // e.g. "60s", "1m", "1h"
	Algorithm string `json:"algorithm,omitempty"` // "token_bucket" or "sliding_window"
}

type checkResponse struct {
	Allowed   bool   `json:"allowed"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"reset_at"`
}

type metricsEntry struct {
	Key     string `json:"key"`
	Total   int64  `json:"total"`
	Denied  int64  `json:"denied"`
	Allowed int64  `json:"allowed"`
}

type Handler struct {
	tokenBucket   limiter.Limiter
	slidingWindow limiter.Limiter
	store         store.Store
}

func New(s store.Store) *Handler {
	return &Handler{
		tokenBucket:   limiter.NewTokenBucket(s),
		slidingWindow: limiter.NewSlidingWindow(s),
		store:         s,
	}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Key == "" || req.Limit <= 0 || req.Window == "" {
		http.Error(w, "key, limit (>0), and window are required", http.StatusBadRequest)
		return
	}

	window, err := time.ParseDuration(req.Window)
	if err != nil {
		http.Error(w, "invalid window duration (e.g. 60s, 1m, 1h)", http.StatusBadRequest)
		return
	}

	// Choose algorithm
	var lim limiter.Limiter
	switch req.Algorithm {
	case "sliding_window":
		lim = h.slidingWindow
	default:
		lim = h.tokenBucket
	}

	result := lim.Allow(req.Key, req.Limit, window)

	resp := checkResponse{
		Allowed:   result.Allowed,
		Remaining: result.Remaining,
		ResetAt:   result.ResetAt.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if !result.Allowed {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := h.store.GetMetrics()
	entries := make([]metricsEntry, 0, len(metrics))
	for key, counts := range metrics {
		entries = append(entries, metricsEntry{
			Key:     key,
			Total:   counts[0],
			Denied:  counts[1],
			Allowed: counts[0] - counts[1],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
