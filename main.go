package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	case "redis":
		rs, err := store.NewRedisStore(cfg.Store.RedisURL)
		if err != nil {
			log.Fatalf("failed to connect to redis: %v", err)
		}
		s = rs
		fmt.Println("using redis store:", cfg.Store.RedisURL)
	default:
		s = store.NewMemoryStore()
		fmt.Println("using in-memory store")
	}
	defer s.Close()

	h := handler.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/check", h.Check)
	mux.HandleFunc("/metrics", h.Metrics)
	mux.HandleFunc("/health", h.Health)

	addr := ":" + cfg.Server.Port
	server := &http.Server{Addr: addr, Handler: mux}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("ratelimiter-go listening on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nshutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	fmt.Println("server stopped")
}
