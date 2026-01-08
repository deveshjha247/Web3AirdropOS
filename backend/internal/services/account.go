package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/models"
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

	// TODO: Implement platform-specific sync logic
	switch account.Platform {
	case models.PlatformFarcaster:
		return s.syncFarcaster(&account)
	case models.PlatformTwitter:
		return s.syncTwitter(&account)
	case models.PlatformTelegram:
		return s.syncTelegram(&account)
	case models.PlatformDiscord:
		return s.syncDiscord(&account)
	}

	return nil
}

func (s *AccountService) syncFarcaster(account *models.PlatformAccount) error {
	// TODO: Implement Farcaster sync via Warpcast API
	s.container.DB.Model(account).Updates(map[string]interface{}{
		"last_activity_at": time.Now(),
	})
	return nil
}

func (s *AccountService) syncTwitter(account *models.PlatformAccount) error {
	// TODO: Implement Twitter sync via API
	return nil
}

func (s *AccountService) syncTelegram(account *models.PlatformAccount) error {
	// TODO: Implement Telegram sync
	return nil
}

func (s *AccountService) syncDiscord(account *models.PlatformAccount) error {
	// TODO: Implement Discord sync via API
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
