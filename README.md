# Rate Limiter as a Service

HTTP microservice in Go that exposes rate limiting as an external API. Supports Token Bucket and Sliding Window Log algorithms with in-memory storage.

## Features

- **Token Bucket** — burst-friendly algorithm with configurable refill rate
- **Sliding Window Log** — precise request counting within a time window
- **REST API** — `POST /check` to verify rate limits, `GET /metrics` for stats
- **Per-key limiting** — each key (user, IP, API key) has independent limits
- **YAML config** — server port and storage backend
- **In-memory store** — thread-safe with `sync.RWMutex`

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
# Build and run
go build -o ratelimiter .
./ratelimiter

# Or run directly
go run .

# Test a request
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"key":"user-1","limit":5,"window":"1m"}'

# Check metrics
curl http://localhost:8080/metrics
```

## Configuration

Edit `config.yaml`:

```yaml
server:
  port: "8080"

store:
  type: "memory"
```

## Development

```bash
# Run tests
go test ./...

# Run with verbose output
go test ./... -v
```

## Tech stack

- Go (standard library `net/http`)
- `gopkg.in/yaml.v3` (config parsing)
