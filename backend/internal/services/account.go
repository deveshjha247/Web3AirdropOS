package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/models"
	"github.com/web3airdropos/backend/internal/websocket"
)

type AccountService struct {
	container *Container
}

func NewAccountService(c *Container) *AccountService {
	return &AccountService{container: c}
}

type CreateAccountRequest struct {
	Platform        models.PlatformType `json:"platform" binding:"required"`
	Username        string              `json:"username" binding:"required"`
	DisplayName     string              `json:"display_name"`
	ProfileURL      string              `json:"profile_url"`
	AvatarURL       string              `json:"avatar_url"`
	WalletID        *uuid.UUID          `json:"wallet_id"`
	BrowserProfileID *uuid.UUID         `json:"browser_profile_id"`
	ProxyID         *uuid.UUID          `json:"proxy_id"`
	AccessToken     string              `json:"access_token"`
	RefreshToken    string              `json:"refresh_token"`
}

type UpdateAccountRequest struct {
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	WalletID     *uuid.UUID `json:"wallet_id"`
	ProxyID      *uuid.UUID `json:"proxy_id"`
	IsActive     *bool      `json:"is_active"`
}

func (s *AccountService) List(userID uuid.UUID, platform string) ([]models.PlatformAccount, error) {
	var accounts []models.PlatformAccount
	query := s.container.DB.Where("user_id = ?", userID)
	
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	
	if err := query.Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *AccountService) Get(userID, accountID uuid.UUID) (*models.PlatformAccount, error) {
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).
		Preload("Activities", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(50)
		}).
		First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *AccountService) Create(userID uuid.UUID, req *CreateAccountRequest) (*models.PlatformAccount, error) {
	account := &models.PlatformAccount{
		ID:               uuid.New(),
		UserID:           userID,
		Platform:         req.Platform,
		Username:         req.Username,
		DisplayName:      req.DisplayName,
		ProfileURL:       req.ProfileURL,
		AvatarURL:        req.AvatarURL,
		WalletID:         req.WalletID,
		BrowserProfileID: req.BrowserProfileID,
		ProxyID:          req.ProxyID,
		AccessToken:      req.AccessToken,
		RefreshToken:     req.RefreshToken,
		IsActive:         true,
		LastLoginAt:      time.Now(),
	}

	if err := s.container.DB.Create(account).Error; err != nil {
		return nil, err
	}

	// Broadcast event
	s.container.WSHub.BroadcastToUser(userID.String(), "account:created", account)

	return account, nil
}

func (s *AccountService) Update(userID, accountID uuid.UUID, req *UpdateAccountRequest) (*models.PlatformAccount, error) {
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.WalletID != nil {
		updates["wallet_id"] = req.WalletID
	}
	if req.ProxyID != nil {
		updates["proxy_id"] = req.ProxyID
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.container.DB.Model(&account).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "account:updated", account)
	return &account, nil
}

func (s *AccountService) Delete(userID, accountID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).Delete(&models.PlatformAccount{})
	if result.RowsAffected == 0 {
		return errors.New("account not found")
	}
	s.container.WSHub.BroadcastToUser(userID.String(), "account:deleted", map[string]string{"id": accountID.String()})
	return nil
}

func (s *AccountService) GetActivities(userID, accountID uuid.UUID, limit int, offset int) ([]models.AccountActivity, int64, error) {
	var activities []models.AccountActivity
	var total int64

	// Verify ownership
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		return nil, 0, err
	}

	s.container.DB.Model(&models.AccountActivity{}).Where("account_id = ?", accountID).Count(&total)

	if err := s.container.DB.Where("account_id = ?", accountID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&activities).Error; err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}

func (s *AccountService) LinkWallet(userID, accountID, walletID uuid.UUID) error {
	// Verify account ownership
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		return errors.New("account not found")
	}

	// Verify wallet ownership
	var wallet models.Wallet
	if err := s.container.DB.Where("id = ? AND user_id = ?", walletID, userID).First(&wallet).Error; err != nil {
		return errors.New("wallet not found")
	}

	// Link them
	if err := s.container.DB.Model(&account).Update("wallet_id", walletID).Error; err != nil {
		return err
	}

	s.container.WSHub.BroadcastToUser(userID.String(), "account:wallet_linked", map[string]string{
		"account_id": accountID.String(),
		"wallet_id":  walletID.String(),
	})

	return nil
}

func (s *AccountService) Sync(userID, accountID uuid.UUID) error {
	var account models.PlatformAccount
	if err := s.container.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		return err
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:     "info",
		Source:    "platform",
		Message:   "Syncing account: " + account.Username,
		AccountID: accountID.String(),
	})

	// Platform-specific sync
	var err error
	switch account.Platform {
	case models.PlatformFarcaster:
		err = s.syncFarcaster(&account)
	case models.PlatformTwitter:
		err = s.syncTwitter(&account)
	case models.PlatformTelegram:
		err = s.syncTelegram(&account)
	case models.PlatformDiscord:
		err = s.syncDiscord(&account)
	default:
		err = fmt.Errorf("unsupported platform: %s", account.Platform)
	}

	if err != nil {
		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:     "error",
			Source:    "platform",
			Message:   fmt.Sprintf("Sync failed for %s: %v", account.Username, err),
			AccountID: accountID.String(),
		})
		return err
	}

	// Update last synced timestamp
	s.container.DB.Model(&account).Update("last_synced_at", time.Now())

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:     "success",
		Source:    "platform",
		Message:   fmt.Sprintf("Account %s synced successfully", account.Username),
		AccountID: accountID.String(),
	})

	return nil
}

func (s *AccountService) syncFarcaster(account *models.PlatformAccount) error {
	// Use Neynar API to fetch user data
	if s.container.Config.NeynarAPIKey == "" {
		return fmt.Errorf("NEYNAR_API_KEY not configured")
	}

	// Fetch user profile from Neynar
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.neynar.com/v2/farcaster/user?fid=%s", account.PlatformUserID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api_key", s.container.Config.NeynarAPIKey)
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("neynar API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("neynar API returned status %d", resp.StatusCode)
	}

	var result struct {
		Users []struct {
			Fid            int    `json:"fid"`
			Username       string `json:"username"`
			DisplayName    string `json:"display_name"`
			FollowerCount  int    `json:"follower_count"`
			FollowingCount int    `json:"following_count"`
		} `json:"users"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode neynar response: %w", err)
	}

	if len(result.Users) > 0 {
		user := result.Users[0]
		metadata := map[string]interface{}{
			"follower_count":  user.FollowerCount,
			"following_count": user.FollowingCount,
			"display_name":    user.DisplayName,
		}
		metadataJSON, _ := json.Marshal(metadata)
		
		s.container.DB.Model(account).Updates(map[string]interface{}{
			"display_name": user.DisplayName,
			"metadata":     string(metadataJSON),
		})
	}

	return nil
}

func (s *AccountService) syncTwitter(account *models.PlatformAccount) error {
	// Twitter API v2 sync - requires bearer token
	if s.container.Config.TwitterBearer == "" {
		return fmt.Errorf("TWITTER_BEARER_TOKEN not configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s?user.fields=public_metrics,description", account.PlatformUserID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.container.Config.TwitterBearer)
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("twitter API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("twitter rate limit exceeded")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("twitter API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			ID            string `json:"id"`
			Username      string `json:"username"`
			Name          string `json:"name"`
			PublicMetrics struct {
				Followers int `json:"followers_count"`
				Following int `json:"following_count"`
				Tweets    int `json:"tweet_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode twitter response: %w", err)
	}

	metadata := map[string]interface{}{
		"follower_count":  result.Data.PublicMetrics.Followers,
		"following_count": result.Data.PublicMetrics.Following,
		"tweet_count":     result.Data.PublicMetrics.Tweets,
	}
	metadataJSON, _ := json.Marshal(metadata)
	
	s.container.DB.Model(account).Updates(map[string]interface{}{
		"display_name": result.Data.Name,
		"metadata":     string(metadataJSON),
	})

	return nil
}

func (s *AccountService) syncTelegram(account *models.PlatformAccount) error {
	// Telegram Bot API - get chat/channel info
	if s.container.Config.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	// For Telegram, we can get updates or chat info depending on account type
	// Update the last activity timestamp
	s.container.DB.Model(account).Update("last_synced_at", time.Now())
	
	return nil
}

func (s *AccountService) syncDiscord(account *models.PlatformAccount) error {
	// Discord sync requires OAuth token
	// For now, just update the timestamp
	s.container.DB.Model(account).Update("last_synced_at", time.Now())
	return nil
}

// LogActivity creates an activity record for an account
func (s *AccountService) LogActivity(accountID uuid.UUID, activityType string, content string, metadata map[string]interface{}, campaignID *uuid.UUID, automatedBy string) error {
	metadataJSON, _ := json.Marshal(metadata)
	
	activity := &models.AccountActivity{
		ID:          uuid.New(),
		AccountID:   accountID,
		Type:        activityType,
		Content:     content,
		Metadata:    string(metadataJSON),
		Status:      "success",
		CampaignID:  campaignID,
		AutomatedBy: automatedBy,
		CreatedAt:   time.Now(),
	}

	return s.container.DB.Create(activity).Error
}
