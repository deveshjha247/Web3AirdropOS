package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// JobStatus represents the status of a queued job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
)

// JobPriority represents job priority levels
type JobPriority int

const (
	PriorityLow    JobPriority = 1
	PriorityNormal JobPriority = 5
	PriorityHigh   JobPriority = 10
)

// Job represents a queued job
type Job struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Priority    JobPriority     `json:"priority"`
	Status      JobStatus       `json:"status"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	Error       string          `json:"error,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
	
	// Locking
	LockedBy    string          `json:"locked_by,omitempty"`
	LockedUntil *time.Time      `json:"locked_until,omitempty"`
	
	// Deduplication
	DedupeKey   string          `json:"dedupe_key,omitempty"`
}

// Queue represents a Redis-backed job queue
type Queue struct {
	redis     *redis.Client
	name      string
	keyPrefix string
}

// NewQueue creates a new Redis queue
func NewQueue(redisClient *redis.Client, name string) *Queue {
	return &Queue{
		redis:     redisClient,
		name:      name,
		keyPrefix: fmt.Sprintf("web3airdropos:queue:%s:", name),
	}
}

// Key prefixes
func (q *Queue) pendingKey() string    { return q.keyPrefix + "pending" }
func (q *Queue) processingKey() string { return q.keyPrefix + "processing" }
func (q *Queue) completedKey() string  { return q.keyPrefix + "completed" }
func (q *Queue) failedKey() string     { return q.keyPrefix + "failed" }
func (q *Queue) scheduledKey() string  { return q.keyPrefix + "scheduled" }
func (q *Queue) jobKey(id string) string { return q.keyPrefix + "job:" + id }
func (q *Queue) dedupeKey(key string) string { return q.keyPrefix + "dedupe:" + key }

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(ctx context.Context, jobType string, payload interface{}, opts ...JobOption) (*Job, error) {
	job := &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Priority:   PriorityNormal,
		Status:     JobStatusPending,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(job)
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	job.Payload = payloadBytes

	// Check for duplicate if dedupe key is set
	if job.DedupeKey != "" {
		exists, err := q.redis.Exists(ctx, q.dedupeKey(job.DedupeKey)).Result()
		if err != nil {
			return nil, err
		}
		if exists > 0 {
			return nil, ErrDuplicateJob
		}
		// Set dedupe key with TTL
		q.redis.SetEX(ctx, q.dedupeKey(job.DedupeKey), job.ID, 24*time.Hour)
	}

	// Store job data
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	pipe := q.redis.Pipeline()
	pipe.Set(ctx, q.jobKey(job.ID), jobBytes, 7*24*time.Hour) // 7 day TTL

	// Add to appropriate queue
	if job.ScheduledAt != nil && job.ScheduledAt.After(time.Now()) {
		// Scheduled job
		pipe.ZAdd(ctx, q.scheduledKey(), &redis.Z{
			Score:  float64(job.ScheduledAt.Unix()),
			Member: job.ID,
		})
	} else {
		// Immediate job - use priority as score
		pipe.ZAdd(ctx, q.pendingKey(), &redis.Z{
			Score:  float64(job.Priority),
			Member: job.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	return job, err
}

// Dequeue retrieves and locks the next job for processing
func (q *Queue) Dequeue(ctx context.Context, workerID string, lockDuration time.Duration) (*Job, error) {
	// First, move any scheduled jobs that are due
	now := time.Now()
	scheduledJobs, err := q.redis.ZRangeByScore(ctx, q.scheduledKey(), &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", now.Unix()),
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, jobID := range scheduledJobs {
		// Move from scheduled to pending
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}
		pipe := q.redis.Pipeline()
		pipe.ZRem(ctx, q.scheduledKey(), jobID)
		pipe.ZAdd(ctx, q.pendingKey(), &redis.Z{
			Score:  float64(job.Priority),
			Member: jobID,
		})
		pipe.Exec(ctx)
	}

	// Get highest priority job
	result, err := q.redis.ZRevRangeWithScores(ctx, q.pendingKey(), 0, 0).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil // No jobs available
	}

	jobID := result[0].Member.(string)

	// Try to lock the job
	lockKey := q.keyPrefix + "lock:" + jobID
	locked, err := q.redis.SetNX(ctx, lockKey, workerID, lockDuration).Result()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, nil // Job was locked by another worker
	}

	// Move to processing queue
	pipe := q.redis.Pipeline()
	pipe.ZRem(ctx, q.pendingKey(), jobID)
	pipe.SAdd(ctx, q.processingKey(), jobID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		q.redis.Del(ctx, lockKey)
		return nil, err
	}

	// Get and update job
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	now = time.Now()
	job.Status = JobStatusProcessing
	job.StartedAt = &now
	job.LockedBy = workerID
	lockedUntil := now.Add(lockDuration)
	job.LockedUntil = &lockedUntil

	if err := q.updateJob(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// Complete marks a job as completed
func (q *Queue) Complete(ctx context.Context, jobID string, result interface{}) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	now := time.Now()
	job.Status = JobStatusCompleted
	job.CompletedAt = &now

	if result != nil {
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return err
		}
		job.Result = resultBytes
	}

	// Remove dedupe key
	if job.DedupeKey != "" {
		q.redis.Del(ctx, q.dedupeKey(job.DedupeKey))
	}

	pipe := q.redis.Pipeline()
	pipe.SRem(ctx, q.processingKey(), jobID)
	pipe.ZAdd(ctx, q.completedKey(), &redis.Z{
		Score:  float64(now.Unix()),
		Member: jobID,
	})
	pipe.Del(ctx, q.keyPrefix+"lock:"+jobID)
	
	// Update job
	jobBytes, _ := json.Marshal(job)
	pipe.Set(ctx, q.jobKey(jobID), jobBytes, 7*24*time.Hour)
	
	_, err = pipe.Exec(ctx)
	return err
}

// Fail marks a job as failed
func (q *Queue) Fail(ctx context.Context, jobID string, jobErr error) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Error = jobErr.Error()
	job.RetryCount++

	// Check if should retry
	if job.RetryCount < job.MaxRetries {
		// Calculate exponential backoff
		backoff := time.Duration(1<<uint(job.RetryCount)) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		
		job.Status = JobStatusRetrying
		scheduledAt := time.Now().Add(backoff)
		job.ScheduledAt = &scheduledAt
		job.LockedBy = ""
		job.LockedUntil = nil

		pipe := q.redis.Pipeline()
		pipe.SRem(ctx, q.processingKey(), jobID)
		pipe.ZAdd(ctx, q.scheduledKey(), &redis.Z{
			Score:  float64(scheduledAt.Unix()),
			Member: jobID,
		})
		pipe.Del(ctx, q.keyPrefix+"lock:"+jobID)
		
		jobBytes, _ := json.Marshal(job)
		pipe.Set(ctx, q.jobKey(jobID), jobBytes, 7*24*time.Hour)
		
		_, err = pipe.Exec(ctx)
		return err
	}

	// Max retries exceeded - move to failed queue
	now := time.Now()
	job.Status = JobStatusFailed
	job.CompletedAt = &now

	pipe := q.redis.Pipeline()
	pipe.SRem(ctx, q.processingKey(), jobID)
	pipe.ZAdd(ctx, q.failedKey(), &redis.Z{
		Score:  float64(now.Unix()),
		Member: jobID,
	})
	pipe.Del(ctx, q.keyPrefix+"lock:"+jobID)
	
	jobBytes, _ := json.Marshal(job)
	pipe.Set(ctx, q.jobKey(jobID), jobBytes, 7*24*time.Hour)
	
	_, err = pipe.Exec(ctx)
	return err
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	data, err := q.redis.Get(ctx, q.jobKey(jobID)).Result()
	if err == redis.Nil {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}

	var job Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// updateJob updates a job in Redis
func (q *Queue) updateJob(ctx context.Context, job *Job) error {
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.redis.Set(ctx, q.jobKey(job.ID), jobBytes, 7*24*time.Hour).Err()
}

// Stats returns queue statistics
func (q *Queue) Stats(ctx context.Context) (*QueueStats, error) {
	pipe := q.redis.Pipeline()
	pendingCmd := pipe.ZCard(ctx, q.pendingKey())
	processingCmd := pipe.SCard(ctx, q.processingKey())
	completedCmd := pipe.ZCard(ctx, q.completedKey())
	failedCmd := pipe.ZCard(ctx, q.failedKey())
	scheduledCmd := pipe.ZCard(ctx, q.scheduledKey())
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &QueueStats{
		Pending:    pendingCmd.Val(),
		Processing: processingCmd.Val(),
		Completed:  completedCmd.Val(),
		Failed:     failedCmd.Val(),
		Scheduled:  scheduledCmd.Val(),
	}, nil
}

// QueueStats represents queue statistics
type QueueStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Scheduled  int64 `json:"scheduled"`
}

// JobOption is a function that configures a job
type JobOption func(*Job)

// WithPriority sets the job priority
func WithPriority(priority JobPriority) JobOption {
	return func(j *Job) {
		j.Priority = priority
	}
}

// WithMaxRetries sets the maximum retry count
func WithMaxRetries(maxRetries int) JobOption {
	return func(j *Job) {
		j.MaxRetries = maxRetries
	}
}

// WithDelay schedules the job for later
func WithDelay(delay time.Duration) JobOption {
	return func(j *Job) {
		scheduledAt := time.Now().Add(delay)
		j.ScheduledAt = &scheduledAt
	}
}

// WithScheduledAt schedules the job for a specific time
func WithScheduledAt(t time.Time) JobOption {
	return func(j *Job) {
		j.ScheduledAt = &t
	}
}

// WithDeduplication sets a deduplication key
func WithDeduplication(key string) JobOption {
	return func(j *Job) {
		j.DedupeKey = key
	}
}

// Common errors
var (
	ErrJobNotFound  = errors.New("job not found")
	ErrDuplicateJob = errors.New("duplicate job already exists")
)
