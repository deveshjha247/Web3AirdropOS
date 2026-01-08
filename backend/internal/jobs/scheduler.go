package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/models"
	"github.com/web3airdropos/backend/internal/websocket"
)

// Scheduler manages all background jobs
type Scheduler struct {
	db       *gorm.DB
	redis    *redis.Client
	wsHub    *websocket.Hub
	cron     *cron.Cron
	workers  map[string]*Worker
	jobQueue chan *JobContext
	stopChan chan struct{}
	mu       sync.RWMutex
}

// JobContext contains all context for a job execution
type JobContext struct {
	Job         *models.AutomationJob
	UserID      uuid.UUID
	ExecutionID uuid.UUID
	Cancel      context.CancelFunc
}

// Worker processes jobs from the queue
type Worker struct {
	id       int
	queue    chan *JobContext
	stop     chan struct{}
	handlers map[models.JobType]JobHandler
}

// JobHandler is a function that processes a specific job type
type JobHandler func(ctx context.Context, jctx *JobContext, s *Scheduler) error

// NewScheduler creates a new job scheduler
func NewScheduler(db *gorm.DB, redis *redis.Client, wsHub *websocket.Hub) *Scheduler {
	return &Scheduler{
		db:       db,
		redis:    redis,
		wsHub:    wsHub,
		cron:     cron.New(cron.WithSeconds()),
		workers:  make(map[string]*Worker),
		jobQueue: make(chan *JobContext, 100),
		stopChan: make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	log.Println("🚀 Starting job scheduler...")

	// Start cron scheduler
	s.cron.Start()

	// Load scheduled jobs from database
	s.loadScheduledJobs()

	// Start worker pool
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			id:       i,
			queue:    s.jobQueue,
			stop:     make(chan struct{}),
			handlers: s.getJobHandlers(),
		}
		s.workers[uuid.New().String()] = worker
		go worker.run(s)
	}

	// Start job checker (checks for pending jobs every minute)
	go s.jobChecker()

	// Start Redis queue listener
	go s.redisQueueListener()

	log.Println("✅ Job scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("🛑 Stopping job scheduler...")
	close(s.stopChan)
	s.cron.Stop()
	for _, worker := range s.workers {
		close(worker.stop)
	}
}

func (s *Scheduler) loadScheduledJobs() {
	var jobs []models.AutomationJob
	s.db.Where("is_active = ? AND cron_expression != ''", true).Find(&jobs)

	for _, job := range jobs {
		s.scheduleJob(&job)
	}

	log.Printf("📅 Loaded %d scheduled jobs", len(jobs))
}

func (s *Scheduler) scheduleJob(job *models.AutomationJob) {
	if job.CronExpression == "" {
		return
	}

	_, err := s.cron.AddFunc(job.CronExpression, func() {
		s.EnqueueJob(job.ID)
	})

	if err != nil {
		log.Printf("❌ Failed to schedule job %s: %v", job.ID, err)
	}
}

// EnqueueJob adds a job to the processing queue
func (s *Scheduler) EnqueueJob(jobID uuid.UUID) error {
	var job models.AutomationJob
	if err := s.db.First(&job, jobID).Error; err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	jctx := &JobContext{
		Job:         &job,
		UserID:      job.UserID,
		ExecutionID: uuid.New(),
		Cancel:      cancel,
	}

	// Update job status
	s.db.Model(&job).Updates(map[string]interface{}{
		"status":      "running",
		"last_run_at": time.Now(),
	})

	// Notify via WebSocket
	s.wsHub.BroadcastToUser(job.UserID.String(), "job:started", map[string]interface{}{
		"job_id": job.ID,
		"name":   job.Name,
		"type":   job.Type,
	})

	// Send to queue
	select {
	case s.jobQueue <- jctx:
		return nil
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

// EnqueueJobFromRedis adds a job from Redis queue
func (s *Scheduler) EnqueueJobFromRedis(data string) error {
	var payload struct {
		JobID  string `json:"job_id"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return err
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return err
	}

	return s.EnqueueJob(jobID)
}

func (s *Scheduler) jobChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for jobs that should be run
			var jobs []models.AutomationJob
			s.db.Where("is_active = ? AND next_run_at <= ? AND status != ?",
				true, time.Now(), "running").Find(&jobs)

			for _, job := range jobs {
				s.EnqueueJob(job.ID)
			}

		case <-s.stopChan:
			return
		}
	}
}

func (s *Scheduler) redisQueueListener() {
	ctx := context.Background()
	pubsub := s.redis.Subscribe(ctx, "jobs:queue")
	defer pubsub.Close()

	for {
		select {
		case msg := <-pubsub.Channel():
			if err := s.EnqueueJobFromRedis(msg.Payload); err != nil {
				log.Printf("❌ Failed to enqueue job from Redis: %v", err)
			}
		case <-s.stopChan:
			return
		}
	}
}

func (w *Worker) run(s *Scheduler) {
	log.Printf("👷 Worker %d started", w.id)

	for {
		select {
		case jctx := <-w.queue:
			w.processJob(jctx, s)
		case <-w.stop:
			log.Printf("👷 Worker %d stopped", w.id)
			return
		}
	}
}

func (w *Worker) processJob(jctx *JobContext, s *Scheduler) {
	startTime := time.Now()
	log.Printf("⚙️ Worker %d processing job: %s (%s)", w.id, jctx.Job.Name, jctx.Job.Type)

	// Create log entry
	jobLog := &models.JobLog{
		ID:        uuid.New(),
		JobID:     jctx.Job.ID,
		Level:     "info",
		Message:   "Job started",
		CreatedAt: time.Now(),
	}
	s.db.Create(jobLog)

	// Send terminal message
	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "job",
		Message: "Starting job: " + jctx.Job.Name,
		Details: map[string]interface{}{
			"job_id": jctx.Job.ID,
			"type":   jctx.Job.Type,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Get handler for job type
	handler, ok := w.handlers[jctx.Job.Type]
	if !ok {
		s.completeJob(jctx, "failed", "Unknown job type", startTime)
		return
	}

	// Execute job
	err := handler(ctx, jctx, s)
	if err != nil {
		s.completeJob(jctx, "failed", err.Error(), startTime)
		return
	}

	s.completeJob(jctx, "completed", "Job completed successfully", startTime)
}

func (s *Scheduler) completeJob(jctx *JobContext, status, message string, startTime time.Time) {
	duration := time.Since(startTime)

	// Update job status
	updates := map[string]interface{}{
		"status":     "idle",
		"total_runs": gorm.Expr("total_runs + 1"),
	}

	if status == "completed" {
		updates["success_runs"] = gorm.Expr("success_runs + 1")
	} else {
		updates["failed_runs"] = gorm.Expr("failed_runs + 1")
	}

	s.db.Model(&jctx.Job).Updates(updates)

	// Log completion
	level := "success"
	if status == "failed" {
		level = "error"
	}

	s.db.Create(&models.JobLog{
		ID:        uuid.New(),
		JobID:     jctx.Job.ID,
		Level:     level,
		Message:   message,
		Details:   `{"duration_ms": ` + string(rune(duration.Milliseconds())) + `}`,
		CreatedAt: time.Now(),
	})

	// Notify via WebSocket
	s.wsHub.BroadcastToUser(jctx.UserID.String(), "job:completed", map[string]interface{}{
		"job_id":   jctx.Job.ID,
		"status":   status,
		"message":  message,
		"duration": duration.String(),
	})

	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   level,
		Source:  "job",
		Message: message,
		Details: map[string]interface{}{
			"job_id":   jctx.Job.ID,
			"duration": duration.String(),
		},
	})
}

func (s *Scheduler) getJobHandlers() map[models.JobType]JobHandler {
	return map[models.JobType]JobHandler{
		models.JobTypeScheduledPost:   s.handleScheduledPost,
		models.JobTypeCampaignTask:    s.handleCampaignTask,
		models.JobTypeBalanceSync:     s.handleBalanceSync,
		models.JobTypePlatformSync:    s.handlePlatformSync,
		models.JobTypeEngagement:      s.handleEngagement,
		models.JobTypeContentGenerate: s.handleContentGenerate,
		models.JobTypeBulkExecute:     s.handleBulkExecute,
	}
}

// Job Handlers

func (s *Scheduler) handleScheduledPost(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "post",
		Message: "Processing scheduled posts...",
	})

	// Get pending scheduled posts
	var posts []models.ScheduledPost
	if err := s.db.Where("user_id = ? AND status = ? AND scheduled_for <= ?",
		jctx.UserID, "pending", time.Now()).Find(&posts).Error; err != nil {
		return err
	}

	for _, post := range posts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Process each post
			s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
				Level:     "info",
				Source:    "post",
				Message:   "Publishing post to " + post.Platform,
				AccountID: post.AccountID.String(),
			})

			// Mark as processing
			s.db.Model(&post).Update("status", "processing")

			// Get the account to publish from
			var account models.PlatformAccount
			if err := s.db.First(&account, post.AccountID).Error; err != nil {
				s.db.Model(&post).Updates(map[string]interface{}{
					"status":        "failed",
					"error_message": "account not found",
				})
				continue
			}

			// Publish via platform adapter
			var postURL string
			var pubErr error

			switch account.Platform {
			case models.PlatformFarcaster:
				postURL, pubErr = s.publishToFarcaster(&account, post.Content)
			case models.PlatformTelegram:
				postURL, pubErr = s.publishToTelegram(&account, post.Content)
			default:
				pubErr = fmt.Errorf("platform %s not supported for automated publishing", account.Platform)
			}

			if pubErr != nil {
				s.db.Model(&post).Updates(map[string]interface{}{
					"status":        "failed",
					"error_message": pubErr.Error(),
				})
				s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
					Level:     "error",
					Source:    "post",
					Message:   "Failed to publish: " + pubErr.Error(),
					AccountID: post.AccountID.String(),
				})
				continue
			}

			s.db.Model(&post).Updates(map[string]interface{}{
				"status":    "posted",
				"posted_at": time.Now(),
				"post_url":  postURL,
			})

			// Add random delay between posts (human-like behavior)
			time.Sleep(time.Duration(2+jctx.ExecutionID.ID()%5) * time.Second)
		}
	}

	return nil
}

func (s *Scheduler) handleCampaignTask(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	var config struct {
		CampaignID string   `json:"campaign_id"`
		TaskIDs    []string `json:"task_ids"`
	}

	if err := json.Unmarshal([]byte(jctx.Job.Config), &config); err != nil {
		return err
	}

	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "campaign",
		Message: "Processing campaign tasks...",
	})

	// Get tasks
	for _, taskIDStr := range config.TaskIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			taskID, _ := uuid.Parse(taskIDStr)
			var task models.CampaignTask
			if err := s.db.First(&task, taskID).Error; err != nil {
				continue
			}

			s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
				Level:   "info",
				Source:  "task",
				Message: "Executing task: " + task.Name,
				TaskID:  taskID.String(),
			})

			// Create execution record
			execution := &models.TaskExecution{
				ID:        uuid.New(),
				TaskID:    taskID,
				Status:    "in_progress",
				StartedAt: time.Now(),
			}
			s.db.Create(execution)

			// Check if task requires manual intervention
			if task.RequiresManual {
				s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
					Level:   "warn",
					Source:  "task",
					Message: "⚠️ Manual action required: " + task.Name,
					TaskID:  taskID.String(),
				})

				s.wsHub.BroadcastTaskUpdate(jctx.UserID.String(), websocket.TaskStatusUpdate{
					TaskID:         taskID.String(),
					Status:         "waiting_manual",
					Message:        task.RequiredAction,
					RequiresManual: true,
				})

				// Mark as waiting
				s.db.Model(execution).Update("status", "waiting_manual")
				continue
			}

			// Execute task based on type
			var execErr error
			switch task.ActionType {
			case "social_action":
				execErr = s.executeSocialAction(ctx, jctx.UserID, &task, execution)
			case "transaction":
				execErr = s.executeTransaction(ctx, jctx.UserID, &task, execution)
			default:
				execErr = fmt.Errorf("unknown action type: %s", task.ActionType)
			}

			if execErr != nil {
				s.db.Model(execution).Updates(map[string]interface{}{
					"status":        "failed",
					"error_message": execErr.Error(),
					"completed_at":  time.Now(),
				})
				s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
					Level:   "error",
					Source:  "task",
					Message: "Task failed: " + execErr.Error(),
					TaskID:  taskID.String(),
				})
				continue
			}

			s.db.Model(execution).Updates(map[string]interface{}{
				"status":       "completed",
				"completed_at": time.Now(),
			})
		}
	}

	return nil
}

func (s *Scheduler) handleBalanceSync(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "wallet",
		Message: "Syncing wallet balances...",
	})

	var wallets []models.Wallet
	if err := s.db.Where("user_id = ?", jctx.UserID).Find(&wallets).Error; err != nil {
		return err
	}

	for _, wallet := range wallets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
				Level:    "debug",
				Source:   "wallet",
				Message:  "Syncing balance for: " + wallet.Address[:10] + "...",
				WalletID: wallet.ID.String(),
			})

			// TODO: Actually fetch balance from RPC
			// For now, just update last sync time
			s.db.Model(&wallet).Update("last_balance_sync", time.Now())

			time.Sleep(500 * time.Millisecond) // Rate limiting
		}
	}

	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "wallet",
		Message: "Balance sync completed for " + string(rune(len(wallets))) + " wallets",
	})

	return nil
}

func (s *Scheduler) handlePlatformSync(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "platform",
		Message: "Syncing platform accounts...",
	})

	var accounts []models.PlatformAccount
	if err := s.db.Where("user_id = ? AND is_active = ?", jctx.UserID, true).Find(&accounts).Error; err != nil {
		return err
	}

	for _, account := range accounts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
				Level:     "debug",
				Source:    "platform",
				Message:   "Syncing " + string(account.Platform) + " account: " + account.Username,
				AccountID: account.ID.String(),
			})

			// TODO: Actually sync account data from platform API
			s.db.Model(&account).Update("last_activity_at", time.Now())

			time.Sleep(1 * time.Second) // Rate limiting
		}
	}

	return nil
}

func (s *Scheduler) handleEngagement(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	var config struct {
		AccountIDs []string `json:"account_ids"`
		Actions    []string `json:"actions"` // like, reply, follow, recast
		MaxActions int      `json:"max_actions"`
	}

	if err := json.Unmarshal([]byte(jctx.Job.Config), &config); err != nil {
		return err
	}

	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "engagement",
		Message: "Starting engagement automation...",
	})

	// TODO: Implement engagement logic
	// This would interact with platform APIs to perform actions

	return nil
}

func (s *Scheduler) handleContentGenerate(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "ai",
		Message: "Generating AI content...",
	})

	// TODO: Call AI microservice to generate content

	return nil
}

func (s *Scheduler) handleBulkExecute(ctx context.Context, jctx *JobContext, scheduler *Scheduler) error {
	var config struct {
		CampaignID  string   `json:"campaign_id"`
		WalletIDs   []string `json:"wallet_ids"`
		AccountIDs  []string `json:"account_ids"`
		TaskIDs     []string `json:"task_ids"`
		Parallel    bool     `json:"parallel"`
		MaxParallel int      `json:"max_parallel"`
	}

	if err := json.Unmarshal([]byte(jctx.Job.Config), &config); err != nil {
		return err
	}

	s.wsHub.BroadcastTerminal(jctx.UserID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "bulk",
		Message: "Starting bulk execution...",
		Details: map[string]interface{}{
			"wallets":  len(config.WalletIDs),
			"accounts": len(config.AccountIDs),
			"tasks":    len(config.TaskIDs),
		},
	})

	// TODO: Implement bulk execution logic with parallelism control

	return nil
}

// PublishToRedis publishes a job to Redis for distributed processing
func (s *Scheduler) PublishToRedis(jobID, userID uuid.UUID) error {
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]string{
		"job_id":  jobID.String(),
		"user_id": userID.String(),
	})
	return s.redis.Publish(ctx, "jobs:queue", string(payload)).Err()
}

// publishToFarcaster publishes content to Farcaster via Neynar
func (s *Scheduler) publishToFarcaster(account *models.PlatformAccount, content string) (string, error) {
	if s.config.NeynarAPIKey == "" {
		return "", fmt.Errorf("NEYNAR_API_KEY not configured")
	}

	// Post via Neynar API
	client := &http.Client{Timeout: 30 * time.Second}

	payload := map[string]interface{}{
		"signer_uuid": account.PlatformUserID,
		"text":        content,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.neynar.com/v2/farcaster/cast", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("api_key", s.config.NeynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("neynar API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("neynar API error: %s", string(body))
	}

	var result struct {
		Cast struct {
			Hash string `json:"hash"`
		} `json:"cast"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://warpcast.com/%s/%s", account.Username, result.Cast.Hash), nil
}

// publishToTelegram publishes content to Telegram
func (s *Scheduler) publishToTelegram(account *models.PlatformAccount, content string) (string, error) {
	if s.config.TelegramBotToken == "" {
		return "", fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	// Send message via Telegram Bot API
	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.config.TelegramBotToken)
	payload := map[string]interface{}{
		"chat_id": account.PlatformUserID,
		"text":    content,
	}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("telegram API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("telegram API error: %s", string(body))
	}

	var result struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://t.me/c/%s/%d", account.PlatformUserID, result.Result.MessageID), nil
}

// executeSocialAction executes a social media action
func (s *Scheduler) executeSocialAction(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Get account for the action
	var config struct {
		AccountID string `json:"account_id"`
		Action    string `json:"action"` // follow, like, recast, reply, post
		Target    string `json:"target"` // target user/cast
		Content   string `json:"content"`
	}

	if task.Config != "" {
		json.Unmarshal([]byte(task.Config), &config)
	}

	if config.AccountID == "" {
		return fmt.Errorf("account_id not specified in task config")
	}

	accountID, err := uuid.Parse(config.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account_id: %w", err)
	}

	var account models.PlatformAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	// Execute based on platform and action
	switch account.Platform {
	case models.PlatformFarcaster:
		return s.executeFarcasterAction(&account, config.Action, config.Target, config.Content, execution)
	case models.PlatformTelegram:
		return s.executeTelegramAction(&account, config.Action, config.Target, config.Content, execution)
	default:
		return fmt.Errorf("platform %s not supported for social actions", account.Platform)
	}
}

// executeFarcasterAction executes a Farcaster action
func (s *Scheduler) executeFarcasterAction(account *models.PlatformAccount, action, target, content string, execution *models.TaskExecution) error {
	if s.config.NeynarAPIKey == "" {
		return fmt.Errorf("NEYNAR_API_KEY not configured")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var endpoint string
	var payload map[string]interface{}

	switch action {
	case "follow":
		endpoint = "https://api.neynar.com/v2/farcaster/user/follow"
		payload = map[string]interface{}{
			"signer_uuid": account.PlatformUserID,
			"target_fids": []string{target},
		}
	case "like":
		endpoint = "https://api.neynar.com/v2/farcaster/reaction"
		payload = map[string]interface{}{
			"signer_uuid":   account.PlatformUserID,
			"reaction_type": "like",
			"target":        target,
		}
	case "recast":
		endpoint = "https://api.neynar.com/v2/farcaster/reaction"
		payload = map[string]interface{}{
			"signer_uuid":   account.PlatformUserID,
			"reaction_type": "recast",
			"target":        target,
		}
	case "reply":
		endpoint = "https://api.neynar.com/v2/farcaster/cast"
		payload = map[string]interface{}{
			"signer_uuid": account.PlatformUserID,
			"text":        content,
			"parent":      target,
		}
	default:
		return fmt.Errorf("unknown farcaster action: %s", action)
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	req.Header.Set("api_key", s.config.NeynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("neynar API error: %s", string(body))
	}

	// Store proof
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	proofData, _ := json.Marshal(result)
	s.db.Model(execution).Update("proof_data", string(proofData))

	return nil
}

// executeTelegramAction executes a Telegram action
func (s *Scheduler) executeTelegramAction(account *models.PlatformAccount, action, target, content string, execution *models.TaskExecution) error {
	if s.config.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s", s.config.TelegramBotToken)

	switch action {
	case "post", "send":
		url := baseURL + "/sendMessage"
		payload := map[string]interface{}{
			"chat_id": target,
			"text":    content,
		}
		payloadBytes, _ := json.Marshal(payload)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("telegram error: %s", string(body))
		}
	default:
		return fmt.Errorf("unknown telegram action: %s", action)
	}

	return nil
}

// executeTransaction executes a blockchain transaction
func (s *Scheduler) executeTransaction(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Transaction execution requires manual approval for security
	// Mark as requiring manual intervention
	s.db.Model(execution).Update("status", "waiting_manual")

	s.wsHub.BroadcastTaskUpdate(userID.String(), websocket.TaskStatusUpdate{
		TaskID:         execution.TaskID.String(),
		Status:         "waiting_manual",
		Message:        "Transaction requires manual approval",
		RequiresManual: true,
	})

	return fmt.Errorf("transaction requires manual approval")
}
