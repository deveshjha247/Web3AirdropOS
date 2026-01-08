package locks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Common errors
var (
	ErrLockNotAcquired = errors.New("could not acquire lock")
	ErrLockExpired     = errors.New("lock expired")
	ErrLockNotOwned    = errors.New("lock not owned by this client")
)

// ResourceType represents different types of lockable resources
type ResourceType string

const (
	ResourceAccount  ResourceType = "account"  // Lock per platform account (for rate limiting)
	ResourceWallet   ResourceType = "wallet"   // Lock per wallet (for transactions)
	ResourceTask     ResourceType = "task"     // Lock per task execution
	ResourceBrowser  ResourceType = "browser"  // Lock per browser session
	ResourceCampaign ResourceType = "campaign" // Lock per campaign execution
)

// DistributedLock represents a distributed lock backed by Redis
type DistributedLock struct {
	client    *redis.Client
	key       string
	token     string
	expiresAt time.Time
}

// LockManager manages distributed locks
type LockManager struct {
	redis     *redis.Client
	keyPrefix string
}

// NewLockManager creates a new lock manager
func NewLockManager(redisClient *redis.Client) *LockManager {
	return &LockManager{
		redis:     redisClient,
		keyPrefix: "web3airdropos:lock:",
	}
}

// lockKey generates a Redis key for a lock
func (m *LockManager) lockKey(resourceType ResourceType, resourceID string) string {
	return fmt.Sprintf("%s%s:%s", m.keyPrefix, resourceType, resourceID)
}

// Acquire tries to acquire a lock
func (m *LockManager) Acquire(ctx context.Context, resourceType ResourceType, resourceID string, ttl time.Duration) (*DistributedLock, error) {
	key := m.lockKey(resourceType, resourceID)
	token := uuid.New().String()

	// Use SET NX EX for atomic lock acquisition
	ok, err := m.redis.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	if !ok {
		return nil, ErrLockNotAcquired
	}

	return &DistributedLock{
		client:    m.redis,
		key:       key,
		token:     token,
		expiresAt: time.Now().Add(ttl),
	}, nil
}

// AcquireWithRetry tries to acquire a lock with retries and exponential backoff
func (m *LockManager) AcquireWithRetry(ctx context.Context, resourceType ResourceType, resourceID string, ttl time.Duration, maxWait time.Duration) (*DistributedLock, error) {
	deadline := time.Now().Add(maxWait)
	retryInterval := 50 * time.Millisecond
	maxRetryInterval := 500 * time.Millisecond

	for {
		lock, err := m.Acquire(ctx, resourceType, resourceID, ttl)
		if err == nil {
			return lock, nil
		}
		if err != ErrLockNotAcquired {
			return nil, err
		}

		if time.Now().After(deadline) {
			return nil, ErrLockNotAcquired
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
			// Exponential backoff with jitter
			retryInterval = retryInterval * 2
			if retryInterval > maxRetryInterval {
				retryInterval = maxRetryInterval
			}
		}
	}
}

// IsLocked checks if a resource is currently locked
func (m *LockManager) IsLocked(ctx context.Context, resourceType ResourceType, resourceID string) (bool, error) {
	key := m.lockKey(resourceType, resourceID)
	exists, err := m.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetLockTTL returns the remaining TTL of a lock
func (m *LockManager) GetLockTTL(ctx context.Context, resourceType ResourceType, resourceID string) (time.Duration, error) {
	key := m.lockKey(resourceType, resourceID)
	ttl, err := m.redis.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		return 0, nil // Lock doesn't exist or has expired
	}
	return ttl, nil
}

// Release releases the lock (only if we own it)
func (l *DistributedLock) Release(ctx context.Context) error {
	// Lua script to atomically check and delete
	// This ensures we only delete our own lock
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.key}, l.token).Result()
	if err != nil {
		return err
	}
	if result.(int64) == 0 {
		return ErrLockNotOwned
	}
	return nil
}

// Extend extends the lock TTL (only if we own it)
func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	// Lua script to atomically check and extend
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.key}, l.token, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		return err
	}
	if result.(int64) == 0 {
		return ErrLockExpired
	}
	l.expiresAt = time.Now().Add(ttl)
	return nil
}

// IsExpired checks if the lock has expired (locally)
func (l *DistributedLock) IsExpired() bool {
	return time.Now().After(l.expiresAt)
}

// Token returns the lock token
func (l *DistributedLock) Token() string {
	return l.token
}

// ExpiresAt returns when the lock expires
func (l *DistributedLock) ExpiresAt() time.Time {
	return l.expiresAt
}

// AccountLock provides a convenient wrapper for locking platform accounts
type AccountLock struct {
	manager *LockManager
}

// NewAccountLock creates a new account lock helper
func NewAccountLock(manager *LockManager) *AccountLock {
	return &AccountLock{manager: manager}
}

// Lock acquires a lock for a platform account
func (a *AccountLock) Lock(ctx context.Context, accountID string, ttl time.Duration) (*DistributedLock, error) {
	return a.manager.Acquire(ctx, ResourceAccount, accountID, ttl)
}

// LockWithRetry acquires a lock for a platform account with retries
func (a *AccountLock) LockWithRetry(ctx context.Context, accountID string, ttl time.Duration, maxWait time.Duration) (*DistributedLock, error) {
	return a.manager.AcquireWithRetry(ctx, ResourceAccount, accountID, ttl, maxWait)
}

// WalletLock provides a convenient wrapper for locking wallets
type WalletLock struct {
	manager *LockManager
}

// NewWalletLock creates a new wallet lock helper
func NewWalletLock(manager *LockManager) *WalletLock {
	return &WalletLock{manager: manager}
}

// Lock acquires a lock for a wallet (prevents concurrent transactions)
func (w *WalletLock) Lock(ctx context.Context, walletID string, ttl time.Duration) (*DistributedLock, error) {
	return w.manager.Acquire(ctx, ResourceWallet, walletID, ttl)
}

// LockWithRetry acquires a lock for a wallet with retries
func (w *WalletLock) LockWithRetry(ctx context.Context, walletID string, ttl time.Duration, maxWait time.Duration) (*DistributedLock, error) {
	return w.manager.AcquireWithRetry(ctx, ResourceWallet, walletID, ttl, maxWait)
}

// WithLock executes a function while holding a lock
func WithLock(ctx context.Context, manager *LockManager, resourceType ResourceType, resourceID string, ttl time.Duration, fn func() error) error {
	lock, err := manager.Acquire(ctx, resourceType, resourceID, ttl)
	if err != nil {
		return err
	}
	defer lock.Release(ctx)

	return fn()
}

// WithLockRetry executes a function while holding a lock (with retry)
func WithLockRetry(ctx context.Context, manager *LockManager, resourceType ResourceType, resourceID string, ttl, maxWait time.Duration, fn func() error) error {
	lock, err := manager.AcquireWithRetry(ctx, resourceType, resourceID, ttl, maxWait)
	if err != nil {
		return err
	}
	defer lock.Release(ctx)

	return fn()
}
