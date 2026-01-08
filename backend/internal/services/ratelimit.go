package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// LockType represents different types of resource locks
type LockType string

const (
	LockTypeAccount  LockType = "account"  // One action per account at a time
	LockTypeWallet   LockType = "wallet"   // One tx per wallet at a time
	LockTypePlatform LockType = "platform" // Rate limit per platform
	LockTypeGlobal   LockType = "global"   // Global concurrency control
)

// Common errors
var (
	ErrLockNotAcquired = errors.New("could not acquire lock")
	ErrLockExpired     = errors.New("lock expired")
	ErrRateLimited     = errors.New("rate limit exceeded")
)

// RateLimiter handles rate limiting and distributed locks using Redis
type RateLimiter struct {
	redis     *redis.Client
	keyPrefix string
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:     redisClient,
		keyPrefix: "web3airdropos:",
	}
}

// Lock represents an acquired lock
type Lock struct {
	key       string
	token     string
	limiter   *RateLimiter
	expiresAt time.Time
}

// AcquireLock tries to acquire a distributed lock
func (r *RateLimiter) AcquireLock(ctx context.Context, lockType LockType, resourceID string, ttl time.Duration) (*Lock, error) {
	key := fmt.Sprintf("%slock:%s:%s", r.keyPrefix, lockType, resourceID)
	token := uuid.New().String()

	// Try to acquire lock with SET NX EX
	ok, err := r.redis.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	if !ok {
		return nil, ErrLockNotAcquired
	}

	return &Lock{
		key:       key,
		token:     token,
		limiter:   r,
		expiresAt: time.Now().Add(ttl),
	}, nil
}

// TryAcquireLock tries to acquire a lock with retries
func (r *RateLimiter) TryAcquireLock(ctx context.Context, lockType LockType, resourceID string, ttl time.Duration, maxWait time.Duration) (*Lock, error) {
	deadline := time.Now().Add(maxWait)
	retryInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		lock, err := r.AcquireLock(ctx, lockType, resourceID, ttl)
		if err == nil {
			return lock, nil
		}
		if err != ErrLockNotAcquired {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
			// Exponential backoff up to 1 second
			if retryInterval < time.Second {
				retryInterval *= 2
			}
		}
	}

	return nil, ErrLockNotAcquired
}

// Release releases the lock
func (l *Lock) Release(ctx context.Context) error {
	// Use Lua script to ensure we only delete our own lock
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	_, err := script.Run(ctx, l.limiter.redis, []string{l.key}, l.token).Result()
	return err
}

// Extend extends the lock TTL
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.limiter.redis, []string{l.key}, l.token, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		return err
	}
	if result.(int64) == 0 {
		return ErrLockExpired
	}
	l.expiresAt = time.Now().Add(ttl)
	return nil
}

// RateLimitConfig defines rate limit parameters
type RateLimitConfig struct {
	Window    time.Duration // Time window for rate limiting
	MaxTokens int           // Maximum requests in window
	BurstSize int           // Burst size above limit
}

// DefaultRateLimits for different platforms
var DefaultRateLimits = map[string]RateLimitConfig{
	"farcaster": {Window: time.Minute, MaxTokens: 20, BurstSize: 5},
	"telegram":  {Window: time.Second, MaxTokens: 25, BurstSize: 5},
	"twitter":   {Window: 15 * time.Minute, MaxTokens: 15, BurstSize: 0},
	"discord":   {Window: time.Minute, MaxTokens: 50, BurstSize: 10},
	"default":   {Window: time.Minute, MaxTokens: 30, BurstSize: 5},
}

// CheckRateLimit checks if an action is within rate limits using sliding window
func (r *RateLimiter) CheckRateLimit(ctx context.Context, platform string, accountID string) (bool, error) {
	config, ok := DefaultRateLimits[platform]
	if !ok {
		config = DefaultRateLimits["default"]
	}

	key := fmt.Sprintf("%sratelimit:%s:%s", r.keyPrefix, platform, accountID)
	now := time.Now().UnixMilli()
	windowStart := now - config.Window.Milliseconds()

	// Use sorted set with timestamps as scores
	pipe := r.redis.Pipeline()
	
	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	
	// Count current entries
	countCmd := pipe.ZCard(ctx, key)
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, err
	}

	count := countCmd.Val()
	maxAllowed := int64(config.MaxTokens + config.BurstSize)

	return count < maxAllowed, nil
}

// RecordAction records an action for rate limiting
func (r *RateLimiter) RecordAction(ctx context.Context, platform string, accountID string) error {
	config, ok := DefaultRateLimits[platform]
	if !ok {
		config = DefaultRateLimits["default"]
	}

	key := fmt.Sprintf("%sratelimit:%s:%s", r.keyPrefix, platform, accountID)
	now := time.Now().UnixMilli()

	pipe := r.redis.Pipeline()
	
	// Add current action
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
	
	// Set expiry on key
	pipe.Expire(ctx, key, config.Window*2)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetRemainingQuota returns remaining actions allowed
func (r *RateLimiter) GetRemainingQuota(ctx context.Context, platform string, accountID string) (int, error) {
	config, ok := DefaultRateLimits[platform]
	if !ok {
		config = DefaultRateLimits["default"]
	}

	key := fmt.Sprintf("%sratelimit:%s:%s", r.keyPrefix, platform, accountID)
	now := time.Now().UnixMilli()
	windowStart := now - config.Window.Milliseconds()

	// Count actions in current window
	count, err := r.redis.ZCount(ctx, key, fmt.Sprintf("%d", windowStart), fmt.Sprintf("%d", now)).Result()
	if err != nil && err != redis.Nil {
		return 0, err
	}

	remaining := config.MaxTokens - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// WaitForQuota waits until quota is available
func (r *RateLimiter) WaitForQuota(ctx context.Context, platform string, accountID string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		allowed, err := r.CheckRateLimit(ctx, platform, accountID)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
			continue
		}
	}

	return ErrRateLimited
}

// AccountLock provides convenient account-level locking
func (r *RateLimiter) AccountLock(ctx context.Context, accountID uuid.UUID, ttl time.Duration) (*Lock, error) {
	return r.AcquireLock(ctx, LockTypeAccount, accountID.String(), ttl)
}

// WalletLock provides convenient wallet-level locking
func (r *RateLimiter) WalletLock(ctx context.Context, walletID uuid.UUID, ttl time.Duration) (*Lock, error) {
	return r.AcquireLock(ctx, LockTypeWallet, walletID.String(), ttl)
}

// GlobalConcurrencyLimit limits total concurrent operations
func (r *RateLimiter) CheckGlobalConcurrency(ctx context.Context, userID uuid.UUID, maxConcurrent int) (bool, error) {
	key := fmt.Sprintf("%sconcurrency:%s", r.keyPrefix, userID.String())
	
	count, err := r.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}

	return count < maxConcurrent, nil
}

// IncrementConcurrency increments the concurrent operation count
func (r *RateLimiter) IncrementConcurrency(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("%sconcurrency:%s", r.keyPrefix, userID.String())
	pipe := r.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 5*time.Minute) // Auto-cleanup
	_, err := pipe.Exec(ctx)
	return err
}

// DecrementConcurrency decrements the concurrent operation count
func (r *RateLimiter) DecrementConcurrency(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("%sconcurrency:%s", r.keyPrefix, userID.String())
	return r.redis.Decr(ctx, key).Err()
}
