package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a Redis-backed atomic implementation of Store for distributed rate limiting.
type Redis struct {
	client *redis.Client
	prefix string

	tbScript  *redis.Script
	fwScript  *redis.Script
	swcScript *redis.Script
	swlScript *redis.Script
}

// RedisConfig holds the configuration for a Redis store.
type RedisConfig struct {
	// Client is the Redis client instance.
	Client *redis.Client

	// Prefix is the key prefix for all rate limit keys (default: "rateshield:").
	Prefix string
}

// NewRedis creates a new Redis-backed store.
func NewRedis(cfg RedisConfig) (*Redis, error) {
	if cfg.Client == nil {
		return nil, errors.New("redis client is required")
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "rateshield:"
	}

	tbScript := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		local ttlMs = tonumber(ARGV[5])

		local data = redis.call('HMGET', key, 'tokens', 'last_update')
		local current_tokens = tonumber(data[1])
		local last_update = tonumber(data[2])

		if current_tokens == nil then
			current_tokens = capacity
			last_update = now
		else
			local elapsed = (now - last_update) / 1000
			if elapsed > 0 then
				current_tokens = current_tokens + (elapsed * rate)
				if current_tokens > capacity then
					current_tokens = capacity
				end
				last_update = now
			end
		end

		local allowed = 0
		if current_tokens >= requested then
			current_tokens = current_tokens - requested
			allowed = 1
		end

		redis.call('HSET', key, 'tokens', current_tokens, 'last_update', last_update)
		redis.call('PEXPIRE', key, ttlMs)

		local remaining = math.floor(current_tokens)
		if remaining < 0 then remaining = 0 end

		local retry_after_ms = 0
		if allowed == 0 and rate > 0 then
			local needed = requested - current_tokens
			retry_after_ms = math.ceil((needed / rate) * 1000)
		end

		local reset_after_ms = 0
		if rate > 0 then
			local missing_tokens = capacity - current_tokens
			if missing_tokens < 0 then missing_tokens = 0 end
			reset_after_ms = math.ceil((missing_tokens / rate) * 1000)
		end

		return {allowed, remaining, reset_after_ms, retry_after_ms}
	`)

	fwScript := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local windowMs = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		local windowNum = math.floor(now / windowMs)
		local windowKey = key .. ":fw:" .. windowNum
		local windowEndMs = (windowNum + 1) * windowMs

		local count = redis.call('INCRBY', windowKey, requested)
		if count == requested then
			redis.call('PEXPIRE', windowKey, windowMs * 2)
		end

		local allowed = 0
		if count <= limit then
			allowed = 1
		else
			redis.call('DECRBY', windowKey, requested)
			count = count - requested
		end

		local remaining = limit - count
		if remaining < 0 then remaining = 0 end

		local retry_after_ms = 0
		if allowed == 0 then
			retry_after_ms = windowEndMs - now
			if retry_after_ms < 0 then retry_after_ms = 0 end
		end

		return {allowed, remaining, windowEndMs - now, retry_after_ms}
	`)

	swcScript := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local windowMs = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		local currWinNum = math.floor(now / windowMs)
		local prevWinNum = currWinNum - 1
		local currKey = key .. ":swc:" .. currWinNum
		local prevKey = key .. ":swc:" .. prevWinNum

		local currCount = tonumber(redis.call('GET', currKey) or "0")
		local prevCount = tonumber(redis.call('GET', prevKey) or "0")

		local winStartMs = currWinNum * windowMs
		local timeInWinMs = now - winStartMs
		local weight = 1.0 - (timeInWinMs / windowMs)
		if weight < 0 then weight = 0 end

		local estimated = math.floor(prevCount * weight + currCount)
		local allowed = 0
		if (estimated + requested) <= limit then
			allowed = 1
			currCount = redis.call('INCRBY', currKey, requested)
			if currCount == requested then
				redis.call('PEXPIRE', currKey, windowMs * 2)
			end
			estimated = estimated + requested
		end

		local remaining = limit - estimated
		if remaining < 0 then remaining = 0 end

		local windowEndMs = (currWinNum + 1) * windowMs
		local retry_after_ms = 0
		if allowed == 0 then
			retry_after_ms = windowEndMs - now
			if retry_after_ms < 0 then retry_after_ms = 0 end
		end

		return {allowed, remaining, windowEndMs - now, retry_after_ms}
	`)

	swlScript := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local windowMs = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		local ttlMs = tonumber(ARGV[5])

		local logKey = key .. ":swl"
		local minScore = now - windowMs

		redis.call('ZREMRANGEBYSCORE', logKey, '-inf', minScore)

		local count = redis.call('ZCARD', logKey)
		local allowed = 0
		if (count + requested) <= limit then
			allowed = 1
			local seqKey = logKey .. ':seq'
			for i = 1, requested do
				local seq = redis.call('INCR', seqKey)
				redis.call('ZADD', logKey, now, now .. '-' .. i .. '-' .. seq)
			end
			redis.call('PEXPIRE', logKey, ttlMs)
			redis.call('PEXPIRE', seqKey, ttlMs)
			count = count + requested
		end

		local remaining = limit - count
		if remaining < 0 then remaining = 0 end

		local retry_after_ms = 0
		if allowed == 0 then
			local needed_to_expire = (count + requested) - limit
			if needed_to_expire < 1 then needed_to_expire = 1 end
			local oldest = redis.call('ZRANGE', logKey, needed_to_expire - 1, needed_to_expire - 1, 'WITHSCORES')
			if #oldest > 1 then
				local oldestTime = tonumber(oldest[2])
				retry_after_ms = (oldestTime + windowMs) - now
				if retry_after_ms < 0 then retry_after_ms = 0 end
			else
				retry_after_ms = windowMs
			end
		end

		local reset_after_ms = 0
		local newest = redis.call('ZRANGE', logKey, -1, -1, 'WITHSCORES')
		if #newest > 1 then
			local newestTime = tonumber(newest[2])
			reset_after_ms = (newestTime + windowMs) - now
			if reset_after_ms < 0 then reset_after_ms = 0 end
		end

		return {allowed, remaining, reset_after_ms, retry_after_ms}
	`)

	return &Redis{
		client:    cfg.Client,
		prefix:    prefix,
		tbScript:  tbScript,
		fwScript:  fwScript,
		swcScript: swcScript,
		swlScript: swlScript,
	}, nil
}

// Get retrieves the state for a key.
func (r *Redis) Get(ctx context.Context, key string) (*State, error) {
	data, err := r.client.Get(ctx, r.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return &State{
			Requests: make([]time.Time, 0),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Set saves the state for a key with a TTL.
func (r *Redis) Set(ctx context.Context, key string, state *State, ttl time.Duration) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, r.prefix+key, data, ttl).Err()
}

// Increment atomically increments the count for a key.
func (r *Redis) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	fullKey := r.prefix + key

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// Reset removes all state for a key, including auxiliary keys created by different algorithms.
func (r *Redis) Reset(ctx context.Context, key string) error {
	fullKey := r.prefix + key
	// Collect all possible auxiliary keys used by the different algorithms.
	keysToDelete := []string{fullKey}

	// Fixed Window auxiliary keys: we don't know the exact window numbers,
	// so scan for matching patterns.
	var cursor uint64
	pattern := fullKey + ":*"
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		keysToDelete = append(keysToDelete, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keysToDelete) > 0 {
		return r.client.Del(ctx, keysToDelete...).Err()
	}
	return nil
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	return r.client.Close()
}

func parseEvalResult(raw interface{}, now time.Time) (EvalResult, error) {
	slice, ok := raw.([]interface{})
	if !ok || len(slice) < 4 {
		return EvalResult{}, fmt.Errorf("invalid redis script result format: %v", raw)
	}

	allowedInt, ok1 := slice[0].(int64)
	remaining, ok2 := slice[1].(int64)
	resetMs, ok3 := slice[2].(int64)
	retryMs, ok4 := slice[3].(int64)

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return EvalResult{}, fmt.Errorf("invalid types in redis script result: %v", raw)
	}

	return EvalResult{
		Allowed:    allowedInt == 1,
		Remaining:  remaining,
		ResetAt:    now.Add(time.Duration(resetMs) * time.Millisecond),
		RetryAfter: time.Duration(retryMs) * time.Millisecond,
	}, nil
}

// AllowTokenBucket performs an atomic token bucket check in Redis using Lua script.
func (r *Redis) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (EvalResult, error) {
	now := time.Now()
	res, err := r.tbScript.Run(ctx, r.client, []string{r.prefix + key},
		rate, capacity, now.UnixMilli(), n, ttl.Milliseconds(),
	).Result()

	if err != nil {
		return EvalResult{}, err
	}

	return parseEvalResult(res, now)
}

// AllowFixedWindow performs an atomic fixed window check in Redis using Lua script.
func (r *Redis) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	now := time.Now()
	res, err := r.fwScript.Run(ctx, r.client, []string{r.prefix + key},
		limit, window.Milliseconds(), now.UnixMilli(), n,
	).Result()

	if err != nil {
		return EvalResult{}, err
	}

	return parseEvalResult(res, now)
}

// AllowSlidingWindowCounter performs an atomic sliding window counter check in Redis using Lua script.
func (r *Redis) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	now := time.Now()
	res, err := r.swcScript.Run(ctx, r.client, []string{r.prefix + key},
		limit, window.Milliseconds(), now.UnixMilli(), n,
	).Result()

	if err != nil {
		return EvalResult{}, err
	}

	return parseEvalResult(res, now)
}

// AllowSlidingWindowLog performs an atomic sliding window log check in Redis using Lua script.
func (r *Redis) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	now := time.Now()
	res, err := r.swlScript.Run(ctx, r.client, []string{r.prefix + key},
		limit, window.Milliseconds(), now.UnixMilli(), n, ttl.Milliseconds(),
	).Result()

	if err != nil {
		return EvalResult{}, err
	}

	return parseEvalResult(res, now)
}
