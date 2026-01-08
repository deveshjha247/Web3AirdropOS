package services

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/models"
)

type DashboardService struct {
	container *Container
}

func NewDashboardService(c *Container) *DashboardService {
	return &DashboardService{container: c}
}

type DashboardStats struct {
	// Wallet stats
	TotalWallets    int     `json:"total_wallets"`
	EVMWallets      int     `json:"evm_wallets"`
	SolanaWallets   int     `json:"solana_wallets"`
	TotalBalanceUSD float64 `json:"total_balance_usd"`

	// Account stats
	TotalAccounts     int `json:"total_accounts"`
	FarcasterAccounts int `json:"farcaster_accounts"`
	TwitterAccounts   int `json:"twitter_accounts"`
	DiscordAccounts   int `json:"discord_accounts"`
	TelegramAccounts  int `json:"telegram_accounts"`

	// Campaign stats
	ActiveCampaigns    int `json:"active_campaigns"`
	CompletedCampaigns int `json:"completed_campaigns"`
	TotalTasks         int `json:"total_tasks"`
	CompletedTasks     int `json:"completed_tasks"`
	PendingTasks       int `json:"pending_tasks"`

	// Job stats
	ActiveJobs  int `json:"active_jobs"`
	RunningJobs int `json:"running_jobs"`

	// Browser stats
	ActiveSessions int `json:"active_sessions"`

	// Content stats
	PendingDrafts  int `json:"pending_drafts"`
	ScheduledPosts int `json:"scheduled_posts"`

	// Activity
	TodayTransactions int   `json:"today_transactions"`
	TodayPosts        int   `json:"today_posts"`
	WeeklyActivity    []int `json:"weekly_activity"` // Last 7 days activity count
}

func (s *DashboardService) GetStats(userID uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Wallet stats
	var totalWallets, evmWallets, solanaWallets int64
	s.container.DB.Model(&models.Wallet{}).Where("user_id = ?", userID).Count(&totalWallets)
	s.container.DB.Model(&models.Wallet{}).Where("user_id = ? AND type = ?", userID, models.WalletTypeEVM).Count(&evmWallets)
	s.container.DB.Model(&models.Wallet{}).Where("user_id = ? AND type = ?", userID, models.WalletTypeSolana).Count(&solanaWallets)
	stats.TotalWallets = int(totalWallets)
	stats.EVMWallets = int(evmWallets)
	stats.SolanaWallets = int(solanaWallets)

	// Account stats
	var totalAccounts, farcasterAccounts, twitterAccounts, discordAccounts, telegramAccounts int64
	s.container.DB.Model(&models.PlatformAccount{}).Where("user_id = ?", userID).Count(&totalAccounts)
	s.container.DB.Model(&models.PlatformAccount{}).Where("user_id = ? AND platform = ?", userID, models.PlatformFarcaster).Count(&farcasterAccounts)
	s.container.DB.Model(&models.PlatformAccount{}).Where("user_id = ? AND platform = ?", userID, models.PlatformTwitter).Count(&twitterAccounts)
	s.container.DB.Model(&models.PlatformAccount{}).Where("user_id = ? AND platform = ?", userID, models.PlatformDiscord).Count(&discordAccounts)
	s.container.DB.Model(&models.PlatformAccount{}).Where("user_id = ? AND platform = ?", userID, models.PlatformTelegram).Count(&telegramAccounts)
	stats.TotalAccounts = int(totalAccounts)
	stats.FarcasterAccounts = int(farcasterAccounts)
	stats.TwitterAccounts = int(twitterAccounts)
	stats.DiscordAccounts = int(discordAccounts)
	stats.TelegramAccounts = int(telegramAccounts)

	// Campaign stats
	var activeCampaigns, completedCampaigns int64
	s.container.DB.Model(&models.Campaign{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeCampaigns)
	s.container.DB.Model(&models.Campaign{}).Where("user_id = ? AND status = ?", userID, "completed").Count(&completedCampaigns)
	stats.ActiveCampaigns = int(activeCampaigns)
	stats.CompletedCampaigns = int(completedCampaigns)

	// Task stats from campaigns
	var totalTasks, completedTasks, pendingTasks int64
	s.container.DB.Model(&models.CampaignTask{}).
		Joins("JOIN campaigns ON campaign_tasks.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Count(&totalTasks)
	stats.TotalTasks = int(totalTasks)

	s.container.DB.Model(&models.TaskExecution{}).
		Joins("JOIN campaign_tasks ON task_executions.task_id = campaign_tasks.id").
		Joins("JOIN campaigns ON campaign_tasks.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND task_executions.status = ?", userID, "completed").
		Count(&completedTasks)
	stats.CompletedTasks = int(completedTasks)

	s.container.DB.Model(&models.TaskExecution{}).
		Joins("JOIN campaign_tasks ON task_executions.task_id = campaign_tasks.id").
		Joins("JOIN campaigns ON campaign_tasks.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND task_executions.status IN ?", userID, []string{"pending", "in_progress", "waiting_manual"}).
		Count(&pendingTasks)
	stats.PendingTasks = int(pendingTasks)

	// Job stats
	var activeJobs, runningJobs int64
	s.container.DB.Model(&models.AutomationJob{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&activeJobs)
	s.container.DB.Model(&models.AutomationJob{}).Where("user_id = ? AND status = ?", userID, "running").Count(&runningJobs)
	stats.ActiveJobs = int(activeJobs)
	stats.RunningJobs = int(runningJobs)

	// Browser sessions
	var activeSessions int64
	s.container.DB.Model(&models.BrowserSession{}).Where("user_id = ? AND status != ?", userID, "stopped").Count(&activeSessions)
	stats.ActiveSessions = int(activeSessions)

	// Content stats
	var pendingDrafts, scheduledPosts int64
	s.container.DB.Model(&models.ContentDraft{}).Where("user_id = ? AND status = ?", userID, "draft").Count(&pendingDrafts)
	s.container.DB.Model(&models.ScheduledPost{}).Where("user_id = ? AND status = ?", userID, "pending").Count(&scheduledPosts)
	stats.PendingDrafts = int(pendingDrafts)
	stats.ScheduledPosts = int(scheduledPosts)

	// Today's activity
	today := time.Now().Truncate(24 * time.Hour)
	var todayTransactions, todayPosts int64
	s.container.DB.Model(&models.Transaction{}).
		Joins("JOIN wallets ON transactions.wallet_id = wallets.id").
		Where("wallets.user_id = ? AND transactions.created_at >= ?", userID, today).
		Count(&todayTransactions)
	stats.TodayTransactions = int(todayTransactions)

	s.container.DB.Model(&models.AccountActivity{}).
		Joins("JOIN platform_accounts ON account_activities.account_id = platform_accounts.id").
		Where("platform_accounts.user_id = ? AND account_activities.created_at >= ? AND account_activities.type = ?", userID, today, "post").
		Count(&todayPosts)
	stats.TodayPosts = int(todayPosts)

	// Weekly activity
	stats.WeeklyActivity = make([]int, 7)
	for i := 6; i >= 0; i-- {
		dayStart := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		var count int64
		s.container.DB.Model(&models.AccountActivity{}).
			Joins("JOIN platform_accounts ON account_activities.account_id = platform_accounts.id").
			Where("platform_accounts.user_id = ? AND account_activities.created_at >= ? AND account_activities.created_at < ?", userID, dayStart, dayEnd).
			Count(&count)

		stats.WeeklyActivity[6-i] = int(count)
	}

	return stats, nil
}

type RecentActivity struct {
	ID          uuid.UUID   `json:"id"`
	Type        string      `json:"type"` // transaction, post, task, login, etc.
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Platform    string      `json:"platform,omitempty"`
	Status      string      `json:"status"`
	Timestamp   time.Time   `json:"timestamp"`
	Metadata    interface{} `json:"metadata,omitempty"`
}

func (s *DashboardService) GetRecentActivity(userID uuid.UUID, limit int) ([]RecentActivity, error) {
	if limit == 0 {
		limit = 20
	}

	var activities []RecentActivity

	// Get recent account activities
	var accountActivities []models.AccountActivity
	s.container.DB.Model(&models.AccountActivity{}).
		Joins("JOIN platform_accounts ON account_activities.account_id = platform_accounts.id").
		Where("platform_accounts.user_id = ?", userID).
		Order("account_activities.created_at DESC").
		Limit(limit / 2).
		Find(&accountActivities)

	for _, a := range accountActivities {
		activities = append(activities, RecentActivity{
			ID:          a.ID,
			Type:        "activity",
			Title:       a.Type + " activity",
			Description: a.Content,
			Status:      a.Status,
			Timestamp:   a.CreatedAt,
		})
	}

	// Get recent task executions
	var taskExecutions []models.TaskExecution
	s.container.DB.Model(&models.TaskExecution{}).
		Joins("JOIN campaign_tasks ON task_executions.task_id = campaign_tasks.id").
		Joins("JOIN campaigns ON campaign_tasks.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Order("task_executions.created_at DESC").
		Limit(limit / 2).
		Preload("Task").
		Find(&taskExecutions)

	// Convert task executions to activities
	for _, te := range taskExecutions {
		status := string(te.Status)
		title := "Task execution"
		description := "task"
		if te.Task != nil {
			if te.Task.Name != "" {
				title = te.Task.Name
			}
			description = string(te.Task.Type) + " task on " + te.Task.TargetPlatform
		}
		activities = append(activities, RecentActivity{
			ID:          te.ID,
			Type:        "task",
			Title:       title,
			Description: description,
			Status:      status,
			Timestamp:   te.CreatedAt,
		})
	}

	// Sort merged activities by timestamp descending
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

	// Limit to requested count
	if len(activities) > limit {
		activities = activities[:limit]
	}

	return activities, nil
}

type ActiveCampaignInfo struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	ProgressPercent float64    `json:"progress_percent"`
	TotalTasks      int        `json:"total_tasks"`
	CompletedTasks  int        `json:"completed_tasks"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	EstimatedReward string     `json:"estimated_reward,omitempty"`
}

func (s *DashboardService) GetActiveCampaigns(userID uuid.UUID) ([]ActiveCampaignInfo, error) {
	var campaigns []models.Campaign
	if err := s.container.DB.Where("user_id = ? AND status = ?", userID, "active").
		Preload("Tasks").
		Order("priority DESC, deadline ASC").
		Limit(10).
		Find(&campaigns).Error; err != nil {
		return nil, err
	}

	var result []ActiveCampaignInfo
	for _, c := range campaigns {
		info := ActiveCampaignInfo{
			ID:              c.ID,
			Name:            c.Name,
			Type:            string(c.Type),
			TotalTasks:      len(c.Tasks),
			Deadline:        c.Deadline,
			EstimatedReward: c.EstimatedReward,
		}

		// Count completed executions
		var completed int64
		s.container.DB.Model(&models.TaskExecution{}).
			Joins("JOIN campaign_tasks ON task_executions.task_id = campaign_tasks.id").
			Where("campaign_tasks.campaign_id = ? AND task_executions.status = ?", c.ID, "completed").
			Count(&completed)

		info.CompletedTasks = int(completed)
		if info.TotalTasks > 0 {
			info.ProgressPercent = float64(completed) / float64(info.TotalTasks) * 100
		}

		result = append(result, info)
	}

	return result, nil
}
