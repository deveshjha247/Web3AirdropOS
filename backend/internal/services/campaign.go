package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cryptoautomation/backend/internal/models"
	"github.com/cryptoautomation/backend/internal/websocket"
)

type CampaignService struct {
	container *Container
}

func NewCampaignService(c *Container) *CampaignService {
	return &CampaignService{container: c}
}

type CreateCampaignRequest struct {
	Name            string              `json:"name" binding:"required"`
	Description     string              `json:"description"`
	Type            models.CampaignType `json:"type" binding:"required"`
	URL             string              `json:"url"`
	ImageURL        string              `json:"image_url"`
	StartDate       time.Time           `json:"start_date"`
	EndDate         time.Time           `json:"end_date"`
	Deadline        *time.Time          `json:"deadline"`
	EstimatedReward string              `json:"estimated_reward"`
	RewardType      string              `json:"reward_type"`
	WalletGroupIDs  []uuid.UUID         `json:"wallet_group_ids"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type UpdateCampaignRequest struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	URL             string              `json:"url"`
	Status          string              `json:"status"`
	Priority        *int                `json:"priority"`
	EndDate         *time.Time          `json:"end_date"`
	Deadline        *time.Time          `json:"deadline"`
	EstimatedReward string              `json:"estimated_reward"`
}

type CampaignProgress struct {
	CampaignID      uuid.UUID              `json:"campaign_id"`
	TotalTasks      int                    `json:"total_tasks"`
	CompletedTasks  int                    `json:"completed_tasks"`
	PendingTasks    int                    `json:"pending_tasks"`
	FailedTasks     int                    `json:"failed_tasks"`
	ProgressPercent float64                `json:"progress_percent"`
	WalletProgress  []WalletTaskProgress   `json:"wallet_progress"`
	AccountProgress []AccountTaskProgress  `json:"account_progress"`
}

type WalletTaskProgress struct {
	WalletID       uuid.UUID `json:"wallet_id"`
	WalletAddress  string    `json:"wallet_address"`
	CompletedTasks int       `json:"completed_tasks"`
	TotalTasks     int       `json:"total_tasks"`
}

type AccountTaskProgress struct {
	AccountID      uuid.UUID `json:"account_id"`
	Username       string    `json:"username"`
	Platform       string    `json:"platform"`
	CompletedTasks int       `json:"completed_tasks"`
	TotalTasks     int       `json:"total_tasks"`
}

func (s *CampaignService) List(userID uuid.UUID, status string, campaignType string) ([]models.Campaign, error) {
	var campaigns []models.Campaign
	query := s.container.DB.Where("user_id = ?", userID).
		Preload("WalletGroups").
		Preload("Tasks")
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if campaignType != "" {
		query = query.Where("type = ?", campaignType)
	}
	
	if err := query.Order("priority DESC, created_at DESC").Find(&campaigns).Error; err != nil {
		return nil, err
	}

	// Calculate progress for each campaign
	for i := range campaigns {
		s.calculateProgress(&campaigns[i])
	}

	return campaigns, nil
}

func (s *CampaignService) Get(userID, campaignID uuid.UUID) (*models.Campaign, error) {
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).
		Preload("WalletGroups").
		Preload("WalletGroups.Wallets").
		Preload("Tasks").
		Preload("Tasks.Executions").
		First(&campaign).Error; err != nil {
		return nil, err
	}
	
	s.calculateProgress(&campaign)
	return &campaign, nil
}

func (s *CampaignService) Create(userID uuid.UUID, req *CreateCampaignRequest) (*models.Campaign, error) {
	metadataJSON, _ := json.Marshal(req.Metadata)
	
	campaign := &models.Campaign{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		URL:             req.URL,
		ImageURL:        req.ImageURL,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Deadline:        req.Deadline,
		Status:          "active",
		EstimatedReward: req.EstimatedReward,
		RewardType:      req.RewardType,
		Metadata:        string(metadataJSON),
	}

	if err := s.container.DB.Create(campaign).Error; err != nil {
		return nil, err
	}

	// Link wallet groups
	if len(req.WalletGroupIDs) > 0 {
		var groups []models.WalletGroup
		s.container.DB.Where("id IN ? AND user_id = ?", req.WalletGroupIDs, userID).Find(&groups)
		s.container.DB.Model(campaign).Association("WalletGroups").Append(&groups)
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "campaign:created", campaign)
	return campaign, nil
}

func (s *CampaignService) Update(userID, campaignID uuid.UUID, req *UpdateCampaignRequest) (*models.Campaign, error) {
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).First(&campaign).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.URL != "" {
		updates["url"] = req.URL
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}
	if req.Deadline != nil {
		updates["deadline"] = *req.Deadline
	}
	if req.EstimatedReward != "" {
		updates["estimated_reward"] = req.EstimatedReward
	}

	if err := s.container.DB.Model(&campaign).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "campaign:updated", campaign)
	return &campaign, nil
}

func (s *CampaignService) Delete(userID, campaignID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).Delete(&models.Campaign{})
	if result.RowsAffected == 0 {
		return errors.New("campaign not found")
	}
	s.container.WSHub.BroadcastToUser(userID.String(), "campaign:deleted", map[string]string{"id": campaignID.String()})
	return nil
}

func (s *CampaignService) GetTasks(userID, campaignID uuid.UUID) ([]models.CampaignTask, error) {
	// Verify ownership
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).First(&campaign).Error; err != nil {
		return nil, err
	}

	var tasks []models.CampaignTask
	if err := s.container.DB.Where("campaign_id = ?", campaignID).
		Preload("Executions").
		Order("order ASC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

type AddTaskRequest struct {
	Name             string         `json:"name" binding:"required"`
	Description      string         `json:"description"`
	Type             models.TaskType `json:"type" binding:"required"`
	TargetURL        string         `json:"target_url"`
	TargetPlatform   string         `json:"target_platform"`
	TargetAccount    string         `json:"target_account"`
	RequiredAction   string         `json:"required_action"`
	IsAutomatable    bool           `json:"is_automatable"`
	RequiresManual   bool           `json:"requires_manual"`
	Points           int            `json:"points"`
	Order            int            `json:"order"`
	DependsOn        *uuid.UUID     `json:"depends_on"`
}

func (s *CampaignService) AddTask(userID, campaignID uuid.UUID, req *AddTaskRequest) (*models.CampaignTask, error) {
	// Verify ownership
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).First(&campaign).Error; err != nil {
		return nil, err
	}

	task := &models.CampaignTask{
		ID:              uuid.New(),
		CampaignID:      campaignID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		TargetURL:       req.TargetURL,
		TargetPlatform:  req.TargetPlatform,
		TargetAccount:   req.TargetAccount,
		RequiredAction:  req.RequiredAction,
		IsAutomatable:   req.IsAutomatable,
		RequiresManual:  req.RequiresManual,
		Points:          req.Points,
		Order:           req.Order,
		DependsOn:       req.DependsOn,
	}

	if err := s.container.DB.Create(task).Error; err != nil {
		return nil, err
	}

	// Update campaign task count
	s.container.DB.Model(&campaign).Update("total_tasks", gorm.Expr("total_tasks + 1"))

	s.container.WSHub.BroadcastToUser(userID.String(), "task:created", task)
	return task, nil
}

type BulkExecuteRequest struct {
	WalletIDs   []uuid.UUID `json:"wallet_ids"`
	AccountIDs  []uuid.UUID `json:"account_ids"`
	TaskIDs     []uuid.UUID `json:"task_ids"`
	Parallel    bool        `json:"parallel"`
	MaxParallel int         `json:"max_parallel"`
}

func (s *CampaignService) ExecuteBulk(userID, campaignID uuid.UUID, req *BulkExecuteRequest) error {
	// Verify ownership
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).First(&campaign).Error; err != nil {
		return err
	}

	// Create automation job for bulk execution
	config := map[string]interface{}{
		"campaign_id": campaignID.String(),
		"wallet_ids":  req.WalletIDs,
		"account_ids": req.AccountIDs,
		"task_ids":    req.TaskIDs,
		"parallel":    req.Parallel,
		"max_parallel": req.MaxParallel,
	}
	configJSON, _ := json.Marshal(config)

	job := &models.AutomationJob{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        models.JobTypeBulkExecute,
		Name:        "Bulk Execute: " + campaign.Name,
		Description: "Bulk task execution for campaign",
		Config:      string(configJSON),
		CampaignID:  &campaignID,
		IsActive:    true,
		Status:      "pending",
	}

	if err := s.container.DB.Create(job).Error; err != nil {
		return err
	}

	// Notify terminal
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "campaign",
		Message: "Starting bulk execution for " + campaign.Name,
		Details: map[string]interface{}{
			"wallets":  len(req.WalletIDs),
			"accounts": len(req.AccountIDs),
			"tasks":    len(req.TaskIDs),
		},
	})

	// TODO: Enqueue the job for processing
	// s.container.Scheduler.EnqueueJob(job.ID)

	return nil
}

func (s *CampaignService) GetProgress(userID, campaignID uuid.UUID) (*CampaignProgress, error) {
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", campaignID, userID).
		Preload("Tasks").
		Preload("Tasks.Executions").
		Preload("WalletGroups.Wallets").
		First(&campaign).Error; err != nil {
		return nil, err
	}

	progress := &CampaignProgress{
		CampaignID: campaignID,
		TotalTasks: len(campaign.Tasks),
	}

	// Count task statuses
	var completedCount, pendingCount, failedCount int
	for _, task := range campaign.Tasks {
		for _, exec := range task.Executions {
			switch exec.Status {
			case "completed":
				completedCount++
			case "failed":
				failedCount++
			default:
				pendingCount++
			}
		}
	}

	progress.CompletedTasks = completedCount
	progress.PendingTasks = pendingCount
	progress.FailedTasks = failedCount

	if progress.TotalTasks > 0 {
		progress.ProgressPercent = float64(completedCount) / float64(progress.TotalTasks) * 100
	}

	// Calculate per-wallet progress
	walletProgressMap := make(map[uuid.UUID]*WalletTaskProgress)
	for _, group := range campaign.WalletGroups {
		for _, wallet := range group.Wallets {
			if _, ok := walletProgressMap[wallet.ID]; !ok {
				walletProgressMap[wallet.ID] = &WalletTaskProgress{
					WalletID:      wallet.ID,
					WalletAddress: wallet.Address,
					TotalTasks:    len(campaign.Tasks),
				}
			}
		}
	}

	for _, task := range campaign.Tasks {
		for _, exec := range task.Executions {
			if exec.WalletID != nil {
				if wp, ok := walletProgressMap[*exec.WalletID]; ok && exec.Status == "completed" {
					wp.CompletedTasks++
				}
			}
		}
	}

	for _, wp := range walletProgressMap {
		progress.WalletProgress = append(progress.WalletProgress, *wp)
	}

	return progress, nil
}

func (s *CampaignService) calculateProgress(campaign *models.Campaign) {
	var completed int64
	s.container.DB.Model(&models.TaskExecution{}).
		Joins("JOIN campaign_tasks ON campaign_tasks.id = task_executions.task_id").
		Where("campaign_tasks.campaign_id = ? AND task_executions.status = ?", campaign.ID, "completed").
		Count(&completed)

	campaign.CompletedTasks = int(completed)
	if campaign.TotalTasks > 0 {
		campaign.ProgressPercent = float64(completed) / float64(campaign.TotalTasks) * 100
	}
}
