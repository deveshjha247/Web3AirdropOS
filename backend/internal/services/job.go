package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/models"
	"github.com/web3airdropos/backend/internal/websocket"
)

type JobService struct {
	container *Container
}

func NewJobService(c *Container) *JobService {
	return &JobService{container: c}
}

type CreateJobRequest struct {
	Type           models.JobType `json:"type" binding:"required"`
	Name           string         `json:"name" binding:"required"`
	Description    string         `json:"description"`
	CronExpression string         `json:"cron_expression"`
	Config         interface{}    `json:"config"`
	WalletIDs      []uuid.UUID    `json:"wallet_ids"`
	AccountIDs     []uuid.UUID    `json:"account_ids"`
	CampaignID     *uuid.UUID     `json:"campaign_id"`
	IsActive       bool           `json:"is_active"`
}

type UpdateJobRequest struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	CronExpression string      `json:"cron_expression"`
	Config         interface{} `json:"config"`
	IsActive       *bool       `json:"is_active"`
}

func (s *JobService) List(userID uuid.UUID, jobType string, status string) ([]models.AutomationJob, error) {
	var jobs []models.AutomationJob
	query := s.container.DB.Where("user_id = ?", userID)

	if jobType != "" {
		query = query.Where("type = ?", jobType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *JobService) Get(userID, jobID uuid.UUID) (*models.AutomationJob, error) {
	var job models.AutomationJob
	if err := s.container.DB.Where("id = ? AND user_id = ?", jobID, userID).
		Preload("Logs", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(100)
		}).
		First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *JobService) Create(userID uuid.UUID, req *CreateJobRequest) (*models.AutomationJob, error) {
	configJSON, _ := json.Marshal(req.Config)
	walletIDsJSON, _ := json.Marshal(req.WalletIDs)
	accountIDsJSON, _ := json.Marshal(req.AccountIDs)

	job := &models.AutomationJob{
		ID:             uuid.New(),
		UserID:         userID,
		Type:           req.Type,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Config:         string(configJSON),
		WalletIDs:      string(walletIDsJSON),
		AccountIDs:     string(accountIDsJSON),
		CampaignID:     req.CampaignID,
		IsActive:       req.IsActive,
		Status:         "idle",
	}

	// Calculate next run time if cron expression provided
	if req.CronExpression != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(req.CronExpression)
		if err != nil {
			return nil, errors.New("invalid cron expression: " + err.Error())
		}
		nextRun := schedule.Next(time.Now())
		job.NextRunAt = &nextRun
	}

	if err := s.container.DB.Create(job).Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "job:created", job)
	return job, nil
}

func (s *JobService) Update(userID, jobID uuid.UUID, req *UpdateJobRequest) (*models.AutomationJob, error) {
	job, err := s.Get(userID, jobID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CronExpression != "" {
		updates["cron_expression"] = req.CronExpression
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		updates["config"] = string(configJSON)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.container.DB.Model(job).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "job:updated", job)
	return job, nil
}

func (s *JobService) Delete(userID, jobID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", jobID, userID).Delete(&models.AutomationJob{})
	if result.RowsAffected == 0 {
		return errors.New("job not found")
	}
	s.container.WSHub.BroadcastToUser(userID.String(), "job:deleted", map[string]string{"id": jobID.String()})
	return nil
}

func (s *JobService) Start(userID, jobID uuid.UUID) error {
	job, err := s.Get(userID, jobID)
	if err != nil {
		return err
	}

	if job.Status == "running" {
		return errors.New("job is already running")
	}

	// Update status
	s.container.DB.Model(job).Updates(map[string]interface{}{
		"status":      "running",
		"last_run_at": time.Now(),
	})

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "job",
		Message: "Starting job: " + job.Name,
	})

	// Enqueue job for execution via Redis queue
	jobPayload, _ := json.Marshal(map[string]interface{}{
		"job_id":  jobID.String(),
		"user_id": userID.String(),
		"type":    job.Type,
	})
	s.container.Redis.LPush(s.container.Redis.Context(), "job:queue", string(jobPayload))

	s.container.WSHub.BroadcastToUser(userID.String(), "job:started", job)
	return nil
}

func (s *JobService) Stop(userID, jobID uuid.UUID) error {
	job, err := s.Get(userID, jobID)
	if err != nil {
		return err
	}

	if job.Status != "running" {
		return errors.New("job is not running")
	}

	// Update status
	s.container.DB.Model(job).Update("status", "idle")

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "job",
		Message: "Stopping job: " + job.Name,
	})

	// Signal job cancellation via Redis pub/sub
	s.container.Redis.Publish(s.container.Redis.Context(), "job:cancel", jobID.String())

	s.container.WSHub.BroadcastToUser(userID.String(), "job:stopped", job)
	return nil
}

func (s *JobService) GetLogs(userID, jobID uuid.UUID, limit int, offset int, level string) ([]models.JobLog, int64, error) {
	// Verify ownership
	_, err := s.Get(userID, jobID)
	if err != nil {
		return nil, 0, err
	}

	var logs []models.JobLog
	var total int64

	query := s.container.DB.Model(&models.JobLog{}).Where("job_id = ?", jobID)
	if level != "" {
		query = query.Where("level = ?", level)
	}

	query.Count(&total)

	if limit == 0 {
		limit = 100
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// AddLog adds a log entry for a job
func (s *JobService) AddLog(jobID uuid.UUID, level, message string, details interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	log := &models.JobLog{
		ID:        uuid.New(),
		JobID:     jobID,
		Level:     level,
		Message:   message,
		Details:   string(detailsJSON),
		CreatedAt: time.Now(),
	}

	return s.container.DB.Create(log).Error
}
