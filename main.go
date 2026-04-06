package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/benjaminserrano23/ratelimiter-go/config"
	"github.com/benjaminserrano23/ratelimiter-go/handler"
	"github.com/benjaminserrano23/ratelimiter-go/store"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var s store.Store
	switch cfg.Store.Type {
	case "memory":
		s = store.NewMemoryStore()
	default:
		s = store.NewMemoryStore()
	}

	h := handler.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/check", h.Check)
	mux.HandleFunc("/metrics", h.Metrics)
	mux.HandleFunc("/health", h.Health)

	addr := ":" + cfg.Server.Port
	fmt.Printf("ratelimiter-go listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
