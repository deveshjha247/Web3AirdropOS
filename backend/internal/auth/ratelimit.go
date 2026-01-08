package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimitConfig defines rate limit parameters
type RateLimitConfig struct {
	Requests int           // Maximum requests
	Window   time.Duration // Time window
	BurstSize int          // Additional burst capacity
}

// Default rate limit configurations
var (
	// API rate limits
	RateLimitDefault = RateLimitConfig{Requests: 100, Window: time.Minute, BurstSize: 20}
	RateLimitAuth    = RateLimitConfig{Requests: 5, Window: time.Minute, BurstSize: 0}      // Strict for auth
	RateLimitWrite   = RateLimitConfig{Requests: 30, Window: time.Minute, BurstSize: 5}
	RateLimitRead    = RateLimitConfig{Requests: 200, Window: time.Minute, BurstSize: 50}

	// Platform-specific rate limits
	PlatformRateLimits = map[string]RateLimitConfig{
		"farcaster": {Requests: 20, Window: time.Minute, BurstSize: 5},
		"twitter":   {Requests: 15, Window: 15 * time.Minute, BurstSize: 0},
		"telegram":  {Requests: 25, Window: time.Second, BurstSize: 5},
		"discord":   {Requests: 50, Window: time.Minute, BurstSize: 10},
	}
)

// RateLimiter implements sliding window rate limiting with Redis
type RateLimiter struct {
	redis     *redis.Client
	keyPrefix string
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:     redisClient,
		keyPrefix: "web3airdropos:ratelimit:",
	}
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed     bool          `json:"allowed"`
	Remaining   int           `json:"remaining"`
	ResetAfter  time.Duration `json:"reset_after"`
	RetryAfter  time.Duration `json:"retry_after,omitempty"`
	Limit       int           `json:"limit"`
	Window      time.Duration `json:"window"`
}

// Check performs a rate limit check using sliding window algorithm
func (r *RateLimiter) Check(ctx context.Context, identifier string, config RateLimitConfig) (*RateLimitResult, error) {
	key := r.keyPrefix + identifier
	now := time.Now()
	windowStart := now.Add(-config.Window)
	
	// Lua script for atomic sliding window rate limiting
	// Removes old entries, adds current request, and counts
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local max_requests = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])
		local burst = tonumber(ARGV[5])
		
		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
		
		-- Count current requests in window
		local current_count = redis.call('ZCARD', key)
		local total_allowed = max_requests + burst
		
		if current_count < total_allowed then
			-- Add the new request
			redis.call('ZADD', key, now, now .. '-' .. math.random(100000))
			redis.call('PEXPIRE', key, window_ms)
			return {1, total_allowed - current_count - 1, 0}
		else
			-- Rate limited - find oldest entry to calculate retry time
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local retry_after = 0
			if #oldest >= 2 then
				retry_after = tonumber(oldest[2]) + window_ms - now
			end
			return {0, 0, retry_after}
		end
	`)

	result, err := script.Run(ctx, r.redis, []string{key},
		now.UnixMilli(),
		windowStart.UnixMilli(),
		config.Requests,
		config.Window.Milliseconds(),
		config.BurstSize,
	).Result()

	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	retryAfterMs := values[2].(int64)

	// Calculate reset time
	resetAfter := config.Window

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAfter: resetAfter,
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
		Limit:      config.Requests + config.BurstSize,
		Window:     config.Window,
	}, nil
}

// CheckIP rate limits by IP address
func (r *RateLimiter) CheckIP(ctx context.Context, ip string, config RateLimitConfig) (*RateLimitResult, error) {
	return r.Check(ctx, "ip:"+ip, config)
}

// CheckUser rate limits by user ID
func (r *RateLimiter) CheckUser(ctx context.Context, userID string, config RateLimitConfig) (*RateLimitResult, error) {
	return r.Check(ctx, "user:"+userID, config)
}

// CheckEndpoint rate limits by user+endpoint combination
func (r *RateLimiter) CheckEndpoint(ctx context.Context, userID, endpoint string, config RateLimitConfig) (*RateLimitResult, error) {
	return r.Check(ctx, "endpoint:"+userID+":"+endpoint, config)
}

// CheckPlatform rate limits by platform account
func (r *RateLimiter) CheckPlatform(ctx context.Context, platform, accountID string) (*RateLimitResult, error) {
	config, ok := PlatformRateLimits[platform]
	if !ok {
		config = RateLimitDefault
	}
	return r.Check(ctx, "platform:"+platform+":"+accountID, config)
}

// SetRateLimitHeaders adds rate limit headers to HTTP response
func (r *RateLimiter) SetRateLimitHeaders(w http.ResponseWriter, result *RateLimitResult) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(result.ResetAfter).Unix(), 10))
	
	if !result.Allowed {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	}
}

// Reset clears rate limit for an identifier
func (r *RateLimiter) Reset(ctx context.Context, identifier string) error {
	return r.redis.Del(ctx, r.keyPrefix+identifier).Err()
}

// GetStats returns rate limit statistics for an identifier
func (r *RateLimiter) GetStats(ctx context.Context, identifier string) (int64, error) {
	key := r.keyPrefix + identifier
	now := time.Now()
	// Count entries in the last minute
	return r.redis.ZCount(ctx, key, 
		strconv.FormatInt(now.Add(-time.Minute).UnixMilli(), 10),
		strconv.FormatInt(now.UnixMilli(), 10),
	).Result()
}

// IPRateLimitMiddleware provides a pre-configured middleware
type IPRateLimitMiddleware struct {
	limiter *RateLimiter
	config  RateLimitConfig
}

// NewIPRateLimitMiddleware creates a new IP-based rate limit middleware
func NewIPRateLimitMiddleware(limiter *RateLimiter, config RateLimitConfig) *IPRateLimitMiddleware {
	return &IPRateLimitMiddleware{
		limiter: limiter,
		config:  config,
	}
}

// Handler returns an HTTP middleware
func (m *IPRateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		
		result, err := m.limiter.CheckIP(r.Context(), ip, m.config)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		m.limiter.SetRateLimitHeaders(w, result)

		if !result.Allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from request headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (client IP)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
