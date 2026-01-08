package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/auth"
)

// AuthMiddleware returns a Gin middleware for JWT authentication
func AuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authService.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("session_id", claims.SessionID)
		c.Set("claims", claims)

		c.Next()
	}
}

// RateLimitMiddleware returns a Gin middleware for rate limiting
func RateLimitMiddleware(limiter *auth.RateLimiter, config auth.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		result, err := limiter.CheckIP(c.Request.Context(), ip, config)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(result.Limit)))
		c.Header("X-RateLimit-Remaining", string(rune(result.Remaining)))

		if !result.Allowed {
			c.Header("Retry-After", string(rune(int(result.RetryAfter.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": result.RetryAfter.Seconds(),
			})
			return
		}

		c.Next()
	}
}

// UserRateLimitMiddleware rate limits by authenticated user
func UserRateLimitMiddleware(limiter *auth.RateLimiter, config auth.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// Fall back to IP-based rate limiting
			RateLimitMiddleware(limiter, config)(c)
			return
		}

		result, err := limiter.CheckUser(c.Request.Context(), userID.(uuid.UUID).String(), config)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			return
		}

		if !result.Allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": result.RetryAfter.Seconds(),
			})
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// GetSessionID extracts session ID from gin context
func GetSessionID(c *gin.Context) (uuid.UUID, bool) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return uuid.Nil, false
	}
	return sessionID.(uuid.UUID), true
}

// GetClaims extracts claims from gin context
func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	return claims.(*auth.Claims), true
}

// RequireRole middleware (for future role-based access control)
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement role checking when role system is added
		// For now, just pass through if authenticated
		_, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}
		c.Next()
	}
}

// AuditMiddleware logs all requests for auditing
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store request info for auditing
		c.Set("request_ip", c.ClientIP())
		c.Set("request_user_agent", c.Request.UserAgent())
		c.Set("request_path", c.Request.URL.Path)
		c.Set("request_method", c.Request.Method)

		c.Next()

		// Response info is available after Next()
		c.Set("response_status", c.Writer.Status())
	}
}
