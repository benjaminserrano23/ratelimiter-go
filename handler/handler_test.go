package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benjaminserrano23/ratelimiter-go/handler"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

func TestCheck_AllowsRequest(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	body := `{"key":"test","limit":5,"window":"1m"}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Check(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != true {
		t.Fatal("first request should be allowed")
	}
}

func TestCheck_DeniesAfterLimit(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	for i := 0; i < 3; i++ {
		body := `{"key":"limited","limit":3,"window":"1m"}`
		req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.Check(w, req)
	}

	// 4th request should be denied
	body := `{"key":"limited","limit":3,"window":"1m"}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Check(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestCheck_InvalidBody(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.Check(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCheck_SlidingWindow(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	body := `{"key":"sw-test","limit":2,"window":"1m","algorithm":"sliding_window"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.Check(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Check(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestMetrics(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	// Make a request first
	body := `{"key":"m-test","limit":5,"window":"1m"}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Check(w, req)

	// Check metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	h.Metrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var entries []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&entries)
	if len(entries) == 0 {
		t.Fatal("metrics should have at least one entry")
	}
}

func TestHealth(t *testing.T) {
	s := store.NewMemoryStore()
	h := handler.New(s)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
