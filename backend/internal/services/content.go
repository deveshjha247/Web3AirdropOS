package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cryptoautomation/backend/internal/models"
	"github.com/cryptoautomation/backend/internal/websocket"
)

type ContentService struct {
	container *Container
}

func NewContentService(c *Container) *ContentService {
	return &ContentService{container: c}
}

type GenerateContentRequest struct {
	Platform    string            `json:"platform" binding:"required"` // farcaster, twitter, telegram
	Type        string            `json:"type" binding:"required"`     // post, reply, thread
	Prompt      string            `json:"prompt"`
	Tone        string            `json:"tone"`        // casual, professional, funny, informative
	Context     string            `json:"context"`     // Additional context
	ReplyTo     string            `json:"reply_to"`    // Post/tweet being replied to
	MaxLength   int               `json:"max_length"`
	NumOptions  int               `json:"num_options"` // Number of variants to generate
	Keywords    []string          `json:"keywords"`
	Hashtags    bool              `json:"hashtags"`
	CampaignID  *uuid.UUID        `json:"campaign_id"`
}

type GeneratedContent struct {
	Content         string   `json:"content"`
	Tone            string   `json:"tone"`
	Platform        string   `json:"platform"`
	Hashtags        []string `json:"hashtags,omitempty"`
	PredictedMetrics struct {
		EngagementScore float64 `json:"engagement_score"`
		ViralPotential  float64 `json:"viral_potential"`
	} `json:"predicted_metrics"`
}

type AIServiceResponse struct {
	Contents []GeneratedContent `json:"contents"`
	Error    string             `json:"error,omitempty"`
}

func (s *ContentService) Generate(userID uuid.UUID, req *GenerateContentRequest) ([]models.ContentDraft, error) {
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "ai",
		Message: fmt.Sprintf("Generating %s content for %s...", req.Type, req.Platform),
	})

	if req.NumOptions == 0 {
		req.NumOptions = 3
	}

	// Call AI microservice
	aiReq := map[string]interface{}{
		"platform":    req.Platform,
		"type":        req.Type,
		"prompt":      req.Prompt,
		"tone":        req.Tone,
		"context":     req.Context,
		"reply_to":    req.ReplyTo,
		"max_length":  req.MaxLength,
		"num_options": req.NumOptions,
		"keywords":    req.Keywords,
		"hashtags":    req.Hashtags,
	}

	body, _ := json.Marshal(aiReq)
	resp, err := http.Post(
		s.container.Config.AIServiceURL+"/generate",
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "error",
			Source:  "ai",
			Message: "Failed to connect to AI service: " + err.Error(),
		})
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var aiResp AIServiceResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return nil, err
	}

	if aiResp.Error != "" {
		return nil, errors.New(aiResp.Error)
	}

	// Create drafts from generated content
	var drafts []models.ContentDraft
	for _, content := range aiResp.Contents {
		metricsJSON, _ := json.Marshal(content.PredictedMetrics)
		
		draft := models.ContentDraft{
			ID:                  uuid.New(),
			UserID:              userID,
			Platform:            req.Platform,
			Type:                req.Type,
			Content:             content.Content,
			Prompt:              req.Prompt,
			AIModel:             "gpt-4",
			Tone:                content.Tone,
			Status:              "draft",
			PredictedEngagement: string(metricsJSON),
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err := s.container.DB.Create(&draft).Error; err != nil {
			continue
		}
		drafts = append(drafts, draft)
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "ai",
		Message: fmt.Sprintf("Generated %d content options", len(drafts)),
	})

	return drafts, nil
}

func (s *ContentService) ListDrafts(userID uuid.UUID, platform string, status string) ([]models.ContentDraft, error) {
	var drafts []models.ContentDraft
	query := s.container.DB.Where("user_id = ?", userID)

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&drafts).Error; err != nil {
		return nil, err
	}
	return drafts, nil
}

func (s *ContentService) GetDraft(userID, draftID uuid.UUID) (*models.ContentDraft, error) {
	var draft models.ContentDraft
	if err := s.container.DB.Where("id = ? AND user_id = ?", draftID, userID).First(&draft).Error; err != nil {
		return nil, err
	}
	return &draft, nil
}

type UpdateDraftRequest struct {
	Content string `json:"content"`
	Tone    string `json:"tone"`
	Status  string `json:"status"`
}

func (s *ContentService) UpdateDraft(userID, draftID uuid.UUID, req *UpdateDraftRequest) (*models.ContentDraft, error) {
	draft, err := s.GetDraft(userID, draftID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Tone != "" {
		updates["tone"] = req.Tone
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if err := s.container.DB.Model(draft).Updates(updates).Error; err != nil {
		return nil, err
	}

	return draft, nil
}

func (s *ContentService) DeleteDraft(userID, draftID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", draftID, userID).Delete(&models.ContentDraft{})
	if result.RowsAffected == 0 {
		return errors.New("draft not found")
	}
	return nil
}

func (s *ContentService) ApproveDraft(userID, draftID uuid.UUID) (*models.ContentDraft, error) {
	draft, err := s.GetDraft(userID, draftID)
	if err != nil {
		return nil, err
	}

	if err := s.container.DB.Model(draft).Update("status", "approved").Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "content:approved", draft)
	return draft, nil
}

type SchedulePostRequest struct {
	DraftID     *uuid.UUID `json:"draft_id"`
	AccountID   uuid.UUID  `json:"account_id" binding:"required"`
	Content     string     `json:"content"`
	Platform    string     `json:"platform" binding:"required"`
	ScheduledAt time.Time  `json:"scheduled_at" binding:"required"`
	MediaURLs   []string   `json:"media_urls"`
}

func (s *ContentService) Schedule(userID uuid.UUID, req *SchedulePostRequest) (*models.ScheduledPost, error) {
	content := req.Content

	// If draft ID provided, get content from draft
	if req.DraftID != nil {
		draft, err := s.GetDraft(userID, *req.DraftID)
		if err != nil {
			return nil, err
		}
		content = draft.Content
	}

	if content == "" {
		return nil, errors.New("content is required")
	}

	// Verify account ownership
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", req.AccountID, userID).First(&account).Error; err != nil {
		return nil, errors.New("account not found")
	}

	mediaJSON, _ := json.Marshal(req.MediaURLs)

	post := &models.ScheduledPost{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   req.AccountID,
		Platform:    req.Platform,
		Content:     content,
		MediaURLs:   string(mediaJSON),
		ScheduledAt: req.ScheduledAt,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.container.DB.Create(post).Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "post:scheduled", post)
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "content",
		Message: fmt.Sprintf("Post scheduled for %s at %s", req.Platform, req.ScheduledAt.Format(time.RFC3339)),
	})

	return post, nil
}

func (s *ContentService) ListScheduled(userID uuid.UUID, platform string, status string) ([]models.ScheduledPost, error) {
	var posts []models.ScheduledPost
	query := s.container.DB.Where("user_id = ?", userID)

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("scheduled_at ASC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (s *ContentService) CancelScheduled(userID, postID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ? AND status = ?", postID, userID, "pending").
		Update("status", "cancelled")
	if result.RowsAffected == 0 {
		return errors.New("scheduled post not found or already processed")
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "post:cancelled", map[string]string{"id": postID.String()})
	return nil
}

// GenerateEngagementPlan generates a weekly engagement plan
type EngagementPlanRequest struct {
	AccountID     uuid.UUID   `json:"account_id" binding:"required"`
	Platform      string      `json:"platform" binding:"required"`
	GoalType      string      `json:"goal_type"`      // engagement, followers, visibility
	DaysToGenerate int        `json:"days_to_generate"`
	Topics        []string    `json:"topics"`
}

type EngagementPlan struct {
	Days []DailyPlan `json:"days"`
}

type DailyPlan struct {
	Date     string   `json:"date"`
	Actions  []Action `json:"actions"`
}

type Action struct {
	Time     string `json:"time"`
	Type     string `json:"type"` // post, reply, like, recast
	Content  string `json:"content,omitempty"`
	Target   string `json:"target,omitempty"`
	Reason   string `json:"reason"`
}

func (s *ContentService) GenerateEngagementPlan(userID uuid.UUID, req *EngagementPlanRequest) (*EngagementPlan, error) {
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "ai",
		Message: "Generating engagement plan...",
	})

	if req.DaysToGenerate == 0 {
		req.DaysToGenerate = 7
	}

	// Call AI microservice
	aiReq := map[string]interface{}{
		"platform":   req.Platform,
		"goal_type":  req.GoalType,
		"days":       req.DaysToGenerate,
		"topics":     req.Topics,
	}

	body, _ := json.Marshal(aiReq)
	resp, err := http.Post(
		s.container.Config.AIServiceURL+"/engagement-plan",
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var plan EngagementPlan
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "ai",
		Message: fmt.Sprintf("Generated %d-day engagement plan", len(plan.Days)),
	})

	return &plan, nil
}
