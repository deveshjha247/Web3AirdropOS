package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Checker manages health checks
type Checker struct {
	db          *gorm.DB
	redis       *redis.Client
	isReady     bool
	readyMu     sync.RWMutex
	startupTime time.Time
}

// NewChecker creates a new health checker
func NewChecker(db *gorm.DB, redis *redis.Client) *Checker {
	return &Checker{
		db:          db,
		redis:       redis,
		isReady:     false,
		startupTime: time.Now(),
	}
}

// SetReady marks the service as ready
func (c *Checker) SetReady(ready bool) {
	c.readyMu.Lock()
	defer c.readyMu.Unlock()
	c.isReady = ready
}

// IsReady returns whether the service is ready
func (c *Checker) IsReady() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.isReady
}

// CheckStatus contains detailed health status
type CheckStatus struct {
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	Version   string           `json:"version"`
	Checks    map[string]Check `json:"checks,omitempty"`
}

// Check represents a single health check
type Check struct {
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// Healthz handles liveness probe - is the process alive?
func (c *Checker) Healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	})
}

// Readyz handles readiness probe - can the service accept traffic?
func (c *Checker) Readyz(ctx *gin.Context) {
	if !c.IsReady() {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not_ready",
			"timestamp": time.Now().UTC(),
			"message":   "service is starting up",
		})
		return
	}

	// Perform quick checks
	checks := make(map[string]Check)
	allHealthy := true

	// Check database
	dbCheck := c.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		allHealthy = false
	}

	// Check Redis
	redisCheck := c.checkRedis()
	checks["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		allHealthy = false
	}

	status := CheckStatus{
		Status:    "ready",
		Timestamp: time.Now().UTC(),
		Uptime:    time.Since(c.startupTime).Round(time.Second).String(),
		Version:   "1.0.0",
		Checks:    checks,
	}

	if !allHealthy {
		status.Status = "degraded"
		ctx.JSON(http.StatusServiceUnavailable, status)
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// Health provides detailed health status
func (c *Checker) Health(ctx *gin.Context) {
	checks := make(map[string]Check)

	// Database check
	checks["database"] = c.checkDatabase()

	// Redis check
	checks["redis"] = c.checkRedis()

	allHealthy := true
	for _, check := range checks {
		if check.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	status := CheckStatus{
		Timestamp: time.Now().UTC(),
		Uptime:    time.Since(c.startupTime).Round(time.Second).String(),
		Version:   "1.0.0",
		Checks:    checks,
	}

	if allHealthy {
		status.Status = "healthy"
		ctx.JSON(http.StatusOK, status)
	} else {
		status.Status = "unhealthy"
		ctx.JSON(http.StatusServiceUnavailable, status)
	}
}

func (c *Checker) checkDatabase() Check {
	if c.db == nil {
		return Check{Status: "unhealthy", Message: "database not configured"}
	}

	start := time.Now()
	sqlDB, err := c.db.DB()
	if err != nil {
		return Check{Status: "unhealthy", Message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return Check{Status: "unhealthy", Message: err.Error()}
	}

	return Check{
		Status:   "healthy",
		Duration: time.Since(start).String(),
	}
}

func (c *Checker) checkRedis() Check {
	if c.redis == nil {
		return Check{Status: "healthy", Message: "redis not configured (optional)"}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.redis.Ping(ctx).Err(); err != nil {
		return Check{Status: "unhealthy", Message: err.Error()}
	}

	return Check{
		Status:   "healthy",
		Duration: time.Since(start).String(),
	}
}

// RegisterRoutes registers health check routes
func (c *Checker) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", c.Healthz)
	r.GET("/readyz", c.Readyz)
	r.GET("/health", c.Health)
}
