# Rate Limiter as a Service

[![CI](https://github.com/benjaminserrano23/ratelimiter-go/actions/workflows/ci.yml/badge.svg)](https://github.com/benjaminserrano23/ratelimiter-go/actions/workflows/ci.yml)

HTTP microservice in Go that exposes rate limiting as an external API. Supports Token Bucket and Sliding Window Log algorithms with pluggable storage backends (in-memory or Redis).

## Features

- **Token Bucket** — burst-friendly algorithm with configurable refill rate
- **Sliding Window Log** — precise request counting within a time window
- **REST API** — `POST /check` to verify rate limits, `GET /metrics` for stats
- **Per-key limiting** — each key (user, IP, API key) has independent limits
- **Redis backend** — Lua scripts for atomic operations, auto-expiring keys
- **In-memory backend** — thread-safe with atomic operations, zero dependencies
- **Docker ready** — multi-stage Dockerfile, works standalone or with Docker Compose
- **YAML + env config** — `config.yaml` with `STORE_TYPE`, `REDIS_URL`, `PORT` overrides
- **Automatic cleanup** — background goroutine sweeps expired keys (memory store)
- **Graceful shutdown** — handles SIGINT/SIGTERM with in-flight request draining

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/check` | Check if a request is allowed |
| `GET` | `/metrics` | Get per-key request statistics |
| `GET` | `/health` | Health check |

### POST /check

```json
{
  "key": "user-123",
  "limit": 10,
  "window": "1m",
  "algorithm": "token_bucket"
}
```

Response (200 or 429):
```json
{
  "allowed": true,
  "remaining": 9,
  "reset_at": "2026-04-06T22:00:00Z"
}
```

`algorithm` is optional — defaults to `token_bucket`. Use `"sliding_window"` for exact counting.

## Usage

```bash
# Build and run (in-memory store)
go build -o ratelimiter .
./ratelimiter

# Run with Redis
STORE_TYPE=redis REDIS_URL=localhost:6379 ./ratelimiter

# Test a request
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"key":"user-1","limit":5,"window":"1m"}'

# Check metrics
curl http://localhost:8080/metrics
```

## Docker

```bash
# Standalone
docker build -t ratelimiter .
docker run -p 8080:8080 ratelimiter

# With Redis (via goproxy's docker-compose)
# See github.com/benjaminserrano23/goproxy for the full stack
```

## Configuration

Edit `config.yaml`:

```yaml
server:
  port: "8080"

store:
  type: "memory"    # or "redis"
  redis_url: "localhost:6379"
```

All values can be overridden with environment variables:

| Env var | Description |
|---------|-------------|
| `PORT` | Server port |
| `STORE_TYPE` | `memory` or `redis` |
| `REDIS_URL` | Redis address (host:port) |

## Architecture

```
POST /check → handler → limiter (token_bucket / sliding_window) → store (memory / redis)
                ↓
            GET /metrics → store.GetMetrics()
```

The `Store` interface abstracts the storage backend:
- **Memory**: `sync.Mutex` + Go maps, background cleanup goroutine
- **Redis**: Lua scripts for atomic check-and-consume, sorted sets for sliding window, auto-TTL

## Development

```bash
go test ./...        # Run tests
go test ./... -v     # Verbose
go test ./... -race  # Race detector (Linux/macOS)
```

## Tech stack

- Go (standard library `net/http`)
- `github.com/redis/go-redis/v9` (Redis client)
- `gopkg.in/yaml.v3` (config parsing)
