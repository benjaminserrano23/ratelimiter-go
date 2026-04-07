package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a Redis-backed store.
// url format: "redis://host:port" or "host:port"
func NewRedisStore(url string) (*redisStore, error) {
	addr := url
	addr = strings.TrimPrefix(addr, "redis://")

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &redisStore{client: client, ctx: ctx}, nil
}

func (r *redisStore) Close() {
	r.client.Close()
}

// --- Token Bucket (Lua script for atomicity) ---

var consumeTokenScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local tokens = tonumber(redis.call('HGET', key, 'tokens') or '-1')
local last_refill = tonumber(redis.call('HGET', key, 'last_refill') or '0')

if tokens == -1 then
    -- First request: initialize bucket
    local remaining = limit - 1
    redis.call('HSET', key, 'tokens', remaining, 'last_refill', now)
    redis.call('EXPIRE', key, 600)
    return {1, tostring(remaining)}
end

-- Refill tokens based on elapsed time
local elapsed = now - last_refill
local refilled = tokens + (elapsed * refill_rate)
if refilled > limit then
    refilled = limit
end

if refilled < 1 then
    redis.call('HSET', key, 'tokens', refilled, 'last_refill', now)
    redis.call('EXPIRE', key, 600)
    return {0, tostring(refilled)}
end

refilled = refilled - 1
redis.call('HSET', key, 'tokens', refilled, 'last_refill', now)
redis.call('EXPIRE', key, 600)
return {1, tostring(refilled)}
`)

func (r *redisStore) ConsumeToken(key string, limit int, refillRate float64) TokenBucketResult {
	rKey := "ratelimit:tb:" + key
	now := float64(time.Now().UnixNano()) / 1e9

	result, err := consumeTokenScript.Run(r.ctx, r.client, []string{rKey},
		limit, refillRate, now,
	).Slice()
	if err != nil {
		return TokenBucketResult{Allowed: false, Remaining: 0}
	}

	allowed := result[0].(int64) == 1
	remaining, _ := strconv.ParseFloat(result[1].(string), 64)
	return TokenBucketResult{Allowed: allowed, Remaining: remaining}
}

// --- Sliding Window (Lua script for atomicity) ---

var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local window_start = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)
if count >= limit then
    return {0, count}
end

redis.call('ZADD', key, now, now .. ':' .. math.random(1000000))
redis.call('EXPIRE', key, 600)
return {1, count + 1}
`)

func (r *redisStore) AddSlidingWindowIfAllowed(key string, windowStart time.Time, now time.Time, limit int) (bool, int) {
	rKey := "ratelimit:sw:" + key
	ws := float64(windowStart.UnixNano()) / 1e9
	n := float64(now.UnixNano()) / 1e9

	result, err := slidingWindowScript.Run(r.ctx, r.client, []string{rKey},
		ws, n, limit,
	).Slice()
	if err != nil {
		return false, 0
	}

	allowed := result[0].(int64) == 1
	count := int(result[1].(int64))
	return allowed, count
}

// --- Legacy methods (needed by interface, not used in atomic path) ---

func (r *redisStore) GetTokenBucket(key string) (float64, time.Time, bool) {
	rKey := "ratelimit:tb:" + key
	vals, err := r.client.HGetAll(r.ctx, rKey).Result()
	if err != nil || len(vals) == 0 {
		return 0, time.Time{}, false
	}
	tokens, _ := strconv.ParseFloat(vals["tokens"], 64)
	lastRefill, _ := strconv.ParseFloat(vals["last_refill"], 64)
	secs := int64(lastRefill)
	nsecs := int64((lastRefill - float64(secs)) * 1e9)
	return tokens, time.Unix(secs, nsecs), true
}

func (r *redisStore) SetTokenBucket(key string, tokens float64, lastRefill time.Time) {
	rKey := "ratelimit:tb:" + key
	now := float64(lastRefill.UnixNano()) / 1e9
	r.client.HSet(r.ctx, rKey, "tokens", tokens, "last_refill", now)
	r.client.Expire(r.ctx, rKey, 10*time.Minute)
}

func (r *redisStore) GetSlidingWindow(key string, windowStart time.Time) []time.Time {
	rKey := "ratelimit:sw:" + key
	ws := float64(windowStart.UnixNano()) / 1e9
	r.client.ZRemRangeByScore(r.ctx, rKey, "-inf", fmt.Sprintf("%f", ws))

	members, err := r.client.ZRangeWithScores(r.ctx, rKey, 0, -1).Result()
	if err != nil {
		return nil
	}
	result := make([]time.Time, 0, len(members))
	for _, m := range members {
		secs := int64(m.Score)
		nsecs := int64((m.Score - float64(secs)) * 1e9)
		result = append(result, time.Unix(secs, nsecs))
	}
	return result
}

// --- Metrics ---

func (r *redisStore) GetMetrics() map[string][2]int64 {
	keys, err := r.client.Keys(r.ctx, "ratelimit:metrics:*").Result()
	if err != nil {
		return nil
	}
	result := make(map[string][2]int64, len(keys))
	for _, k := range keys {
		name := strings.TrimPrefix(k, "ratelimit:metrics:")
		vals, err := r.client.HGetAll(r.ctx, k).Result()
		if err != nil {
			continue
		}
		total, _ := strconv.ParseInt(vals["total"], 10, 64)
		denied, _ := strconv.ParseInt(vals["denied"], 10, 64)
		result[name] = [2]int64{total, denied}
	}
	return result
}

func (r *redisStore) IncrMetrics(key string, denied bool) {
	rKey := "ratelimit:metrics:" + key
	r.client.HIncrBy(r.ctx, rKey, "total", 1)
	if denied {
		r.client.HIncrBy(r.ctx, rKey, "denied", 1)
	}
}
