package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/models"
	"github.com/web3airdropos/backend/internal/services/platforms"
)

// AuditService handles audit logging
type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// LogEntry represents a simplified entry for logging
type LogEntry struct {
	UserID     uuid.UUID
	AccountID  *uuid.UUID
	WalletID   *uuid.UUID
	ProfileID  *uuid.UUID
	
	Action     models.AuditLogAction
	Platform   string
	TargetType string
	TargetID   string
	
	TaskID      *uuid.UUID
	ExecutionID *uuid.UUID
	JobID       *uuid.UUID
	CampaignID  *uuid.UUID
	SessionID   *uuid.UUID
	
	Result       models.AuditLogResult
	ErrorCode    string
	ErrorMessage string
	
	Proof        *platforms.ActionProof
	RequestData  interface{}
	ResponseData interface{}
	
	Duration     time.Duration
	IPAddress    string
	UserAgent    string
	
	IdempotencyKey string
}

// Log creates an audit log entry
func (s *AuditService) Log(ctx context.Context, entry *LogEntry) (*models.AuditLog, error) {
	log := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       entry.UserID,
		AccountID:    entry.AccountID,
		WalletID:     entry.WalletID,
		ProfileID:    entry.ProfileID,
		Action:       entry.Action,
		Platform:     entry.Platform,
		TargetType:   entry.TargetType,
		TargetID:     entry.TargetID,
		TaskID:       entry.TaskID,
		ExecutionID:  entry.ExecutionID,
		JobID:        entry.JobID,
		CampaignID:   entry.CampaignID,
		SessionID:    entry.SessionID,
		Result:       entry.Result,
		ErrorCode:    entry.ErrorCode,
		ErrorMessage: entry.ErrorMessage,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		Duration:     entry.Duration.Milliseconds(),
		IdempotencyKey: entry.IdempotencyKey,
		CreatedAt:    time.Now(),
	}

	// Serialize proof
	if entry.Proof != nil {
		log.ProofType = getProofType(entry.Proof)
		log.ProofValue = getProofValue(entry.Proof)
		if proofData, err := json.Marshal(entry.Proof); err == nil {
			log.ProofData = string(proofData)
		}
	}

	// Serialize request/response
	if entry.RequestData != nil {
		if data, err := json.Marshal(entry.RequestData); err == nil {
			log.RequestData = string(data)
		}
	}
	if entry.ResponseData != nil {
		if data, err := json.Marshal(entry.ResponseData); err == nil {
			log.ResponseData = string(data)
		}
	}

	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}

// LogSuccess is a convenience method for successful actions
func (s *AuditService) LogSuccess(ctx context.Context, userID uuid.UUID, action models.AuditLogAction, platform string, proof *platforms.ActionProof) (*models.AuditLog, error) {
	return s.Log(ctx, &LogEntry{
		UserID:   userID,
		Action:   action,
		Platform: platform,
		Result:   models.ResultSuccess,
		Proof:    proof,
	})
}

// LogFailure is a convenience method for failed actions
func (s *AuditService) LogFailure(ctx context.Context, userID uuid.UUID, action models.AuditLogAction, platform string, err error) (*models.AuditLog, error) {
	return s.Log(ctx, &LogEntry{
		UserID:       userID,
		Action:       action,
		Platform:     platform,
		Result:       models.ResultFailed,
		ErrorMessage: err.Error(),
	})
}

// LogTaskExecution logs a task execution event
func (s *AuditService) LogTaskExecution(ctx context.Context, exec *models.TaskExecution, task *models.CampaignTask, result models.AuditLogResult, proof *platforms.ActionProof, err error) (*models.AuditLog, error) {
	entry := &LogEntry{
		TaskID:      &exec.TaskID,
		ExecutionID: &exec.ID,
		Action:      taskTypeToAction(task.Type),
		Platform:    task.TargetPlatform,
		TargetType:  string(task.Type),
		TargetID:    task.TargetAccount,
		Result:      result,
		Proof:       proof,
		IdempotencyKey: exec.IdempotencyKey,
	}

	if exec.AccountID != nil {
		entry.AccountID = exec.AccountID
	}
	if exec.WalletID != nil {
		entry.WalletID = exec.WalletID
	}

	// Get user ID from campaign
	var campaign models.Campaign
	if err := s.db.Joins("JOIN campaign_tasks ON campaigns.id = campaign_tasks.campaign_id").
		Where("campaign_tasks.id = ?", task.ID).First(&campaign).Error; err == nil {
		entry.UserID = campaign.UserID
		entry.CampaignID = &campaign.ID
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
	}

	return s.Log(ctx, entry)
}

// Query queries audit logs with filters
func (s *AuditService) Query(ctx context.Context, query *models.AuditLogQuery) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	db := s.db.Model(&models.AuditLog{})

	if query.UserID != nil {
		db = db.Where("user_id = ?", *query.UserID)
	}
	if query.AccountID != nil {
		db = db.Where("account_id = ?", *query.AccountID)
	}
	if query.WalletID != nil {
		db = db.Where("wallet_id = ?", *query.WalletID)
	}
	if query.Action != nil {
		db = db.Where("action = ?", *query.Action)
	}
	if query.Platform != "" {
		db = db.Where("platform = ?", query.Platform)
	}
	if query.Result != nil {
		db = db.Where("result = ?", *query.Result)
	}
	if query.CampaignID != nil {
		db = db.Where("campaign_id = ?", *query.CampaignID)
	}
	if query.TaskID != nil {
		db = db.Where("task_id = ?", *query.TaskID)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", *query.EndTime)
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	} else {
		db = db.Limit(50)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// Order by newest first
	db = db.Order("created_at DESC")

	if err := db.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByIdempotencyKey checks if an action was already performed
func (s *AuditService) GetByIdempotencyKey(ctx context.Context, key string) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := s.db.Where("idempotency_key = ?", key).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetRecentByAccount gets recent actions for an account
func (s *AuditService) GetRecentByAccount(ctx context.Context, accountID uuid.UUID, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := s.db.Where("account_id = ?", accountID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetStats returns statistics for a user
func (s *AuditService) GetStats(ctx context.Context, userID uuid.UUID, since time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total actions
	var totalActions int64
	s.db.Model(&models.AuditLog{}).Where("user_id = ? AND created_at >= ?", userID, since).Count(&totalActions)
	stats["total_actions"] = totalActions

	// Success rate
	var successCount int64
	s.db.Model(&models.AuditLog{}).Where("user_id = ? AND created_at >= ? AND result = ?", userID, since, models.ResultSuccess).Count(&successCount)
	if totalActions > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalActions) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	// Actions by platform
	type PlatformCount struct {
		Platform string
		Count    int64
	}
	var platformCounts []PlatformCount
	s.db.Model(&models.AuditLog{}).
		Select("platform, count(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Group("platform").
		Scan(&platformCounts)
	
	platformStats := make(map[string]int64)
	for _, pc := range platformCounts {
		platformStats[pc.Platform] = pc.Count
	}
	stats["by_platform"] = platformStats

	// Actions by type
	type ActionCount struct {
		Action string
		Count  int64
	}
	var actionCounts []ActionCount
	s.db.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Group("action").
		Scan(&actionCounts)
	
	actionStats := make(map[string]int64)
	for _, ac := range actionCounts {
		actionStats[ac.Action] = ac.Count
	}
	stats["by_action"] = actionStats

	return stats, nil
}

// Helper functions

func getProofType(proof *platforms.ActionProof) string {
	if proof.TxHash != "" {
		return "tx_hash"
	}
	if proof.CastHash != "" {
		return "cast_hash"
	}
	if proof.PostURL != "" {
		return "post_url"
	}
	if proof.ScreenshotPath != "" {
		return "screenshot"
	}
	if proof.PostID != "" {
		return "post_id"
	}
	return "unknown"
}

func getProofValue(proof *platforms.ActionProof) string {
	if proof.TxHash != "" {
		return proof.TxHash
	}
	if proof.CastHash != "" {
		return proof.CastHash
	}
	if proof.PostURL != "" {
		return proof.PostURL
	}
	if proof.ScreenshotPath != "" {
		return proof.ScreenshotPath
	}
	if proof.PostID != "" {
		return proof.PostID
	}
	return ""
}

func taskTypeToAction(taskType models.TaskType) models.AuditLogAction {
	switch taskType {
	case models.TaskTypeFollow:
		return models.ActionFollow
	case models.TaskTypeLike:
		return models.ActionLike
	case models.TaskTypeRecast:
		return models.ActionRepost
	case models.TaskTypePost:
		return models.ActionPost
	case models.TaskTypeReply:
		return models.ActionReply
	case models.TaskTypeTransaction:
		return models.ActionTransaction
	case models.TaskTypeClaim:
		return models.ActionClaim
	case models.TaskTypeConnect:
		return models.ActionAccountLink
	default:
		return models.ActionTaskComplete
	}
}
