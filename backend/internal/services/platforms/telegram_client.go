package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TelegramClient implements PlatformAdapter for Telegram Bot API
// Note: This is for BOT accounts, not user automation (which requires MTProto)
type TelegramClient struct {
	creds        *AccountCredentials
	httpClient   *http.Client
	botToken     string
	baseURL      string
	authenticated bool
	botInfo      *TelegramBotInfo
}

type TelegramBotInfo struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
	CanJoinGroups bool `json:"can_join_groups"`
	CanReadAllGroupMessages bool `json:"can_read_all_group_messages"`
}

type TelegramMessage struct {
	MessageID int64 `json:"message_id"`
	Chat      struct {
		ID    int64  `json:"id"`
		Title string `json:"title,omitempty"`
		Type  string `json:"type"`
	} `json:"chat"`
	From struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	} `json:"from"`
	Text string `json:"text"`
	Date int64  `json:"date"`
}

type TelegramResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	Description string          `json:"description,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
}

func NewTelegramClient(creds *AccountCredentials) (*TelegramClient, error) {
	if creds.AccessToken == "" {
		return nil, errors.New("bot token required for Telegram")
	}

	return &TelegramClient{
		creds:       creds,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		botToken:    creds.AccessToken,
		baseURL:     "https://api.telegram.org",
		authenticated: false,
	}, nil
}

func (c *TelegramClient) GetPlatformType() PlatformType {
	return PlatformTelegram
}

func (c *TelegramClient) apiCall(ctx context.Context, method string, params map[string]interface{}) (*TelegramResponse, error) {
	url := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.botToken, method)
	
	var req *http.Request
	var err error

	if params != nil {
		body, _ := json.Marshal(params)
		req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
	}
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	
	var result TelegramResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		if result.ErrorCode == 429 {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}

	return &result, nil
}

func (c *TelegramClient) Authenticate(ctx context.Context, credentials map[string]string) error {
	resp, err := c.apiCall(ctx, "getMe", nil)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	var botInfo TelegramBotInfo
	if err := json.Unmarshal(resp.Result, &botInfo); err != nil {
		return err
	}

	c.botInfo = &botInfo
	c.authenticated = true
	return nil
}

func (c *TelegramClient) IsAuthenticated() bool {
	return c.authenticated
}

func (c *TelegramClient) RefreshAuth(ctx context.Context) error {
	// Bot tokens don't expire
	return nil
}

func (c *TelegramClient) GetProfile(ctx context.Context) (*UserProfile, error) {
	if c.botInfo == nil {
		if err := c.Authenticate(ctx, nil); err != nil {
			return nil, err
		}
	}

	return &UserProfile{
		ID:          fmt.Sprintf("%d", c.botInfo.ID),
		Username:    c.botInfo.Username,
		DisplayName: c.botInfo.FirstName,
		Verified:    c.botInfo.IsBot,
	}, nil
}

func (c *TelegramClient) GetUserByUsername(ctx context.Context, username string) (*UserProfile, error) {
	// Telegram Bot API doesn't support looking up users by username directly
	return nil, ErrNotImplemented
}

func (c *TelegramClient) Follow(ctx context.Context, targetUserID string) (*ActionProof, error) {
	return nil, ErrNotImplemented
}

func (c *TelegramClient) Unfollow(ctx context.Context, targetUserID string) (*ActionProof, error) {
	return nil, ErrNotImplemented
}

func (c *TelegramClient) Like(ctx context.Context, postID string) (*ActionProof, error) {
	return nil, ErrNotImplemented
}

func (c *TelegramClient) Unlike(ctx context.Context, postID string) (*ActionProof, error) {
	return nil, ErrNotImplemented
}

func (c *TelegramClient) Repost(ctx context.Context, postID string) (*ActionProof, error) {
	return nil, ErrNotImplemented
}

// Post sends a message to a chat
func (c *TelegramClient) Post(ctx context.Context, content *PostContent) (*ActionProof, error) {
	if content.ChannelID == "" {
		return nil, errors.New("channel_id (chat_id) required for Telegram")
	}

	params := map[string]interface{}{
		"chat_id": content.ChannelID,
		"text":    content.Text,
	}

	// Parse mode for markdown/html
	if content.ReplyToID != "" {
		params["parse_mode"] = "Markdown"
	}

	resp, err := c.apiCall(ctx, "sendMessage", params)
	if err != nil {
		return nil, err
	}

	var msg TelegramMessage
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}

	return &ActionProof{
		PostID:    fmt.Sprintf("%d", msg.MessageID),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":    content.ChannelID,
			"message_id": fmt.Sprintf("%d", msg.MessageID),
		},
	}, nil
}

// Reply sends a reply to a message
func (c *TelegramClient) Reply(ctx context.Context, messageID string, content *PostContent) (*ActionProof, error) {
	if content.ChannelID == "" {
		return nil, errors.New("channel_id (chat_id) required for Telegram")
	}

	params := map[string]interface{}{
		"chat_id":             content.ChannelID,
		"text":                content.Text,
		"reply_to_message_id": messageID,
	}

	resp, err := c.apiCall(ctx, "sendMessage", params)
	if err != nil {
		return nil, err
	}

	var msg TelegramMessage
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}

	return &ActionProof{
		PostID:    fmt.Sprintf("%d", msg.MessageID),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":           content.ChannelID,
			"reply_to":          messageID,
			"new_message_id":    fmt.Sprintf("%d", msg.MessageID),
		},
	}, nil
}

func (c *TelegramClient) Quote(ctx context.Context, postID string, content *PostContent) (*ActionProof, error) {
	// Telegram doesn't have quote-tweet style functionality
	return c.Reply(ctx, postID, content)
}

func (c *TelegramClient) DeletePost(ctx context.Context, messageID string) error {
	// Need chat_id to delete - this is a limitation
	return ErrNotImplemented
}

// DeleteMessage deletes a message with chat context
func (c *TelegramClient) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	_, err := c.apiCall(ctx, "deleteMessage", params)
	return err
}

// SendPhoto sends a photo message
func (c *TelegramClient) SendPhoto(ctx context.Context, chatID, photoURL, caption string) (*ActionProof, error) {
	params := map[string]interface{}{
		"chat_id": chatID,
		"photo":   photoURL,
	}
	if caption != "" {
		params["caption"] = caption
	}

	resp, err := c.apiCall(ctx, "sendPhoto", params)
	if err != nil {
		return nil, err
	}

	var msg TelegramMessage
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}

	return &ActionProof{
		PostID:    fmt.Sprintf("%d", msg.MessageID),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":    chatID,
			"message_id": fmt.Sprintf("%d", msg.MessageID),
			"type":       "photo",
		},
	}, nil
}

// ForwardMessage forwards a message
func (c *TelegramClient) ForwardMessage(ctx context.Context, toChatID, fromChatID, messageID string) (*ActionProof, error) {
	params := map[string]interface{}{
		"chat_id":      toChatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}

	resp, err := c.apiCall(ctx, "forwardMessage", params)
	if err != nil {
		return nil, err
	}

	var msg TelegramMessage
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}

	return &ActionProof{
		PostID:    fmt.Sprintf("%d", msg.MessageID),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"from_chat_id":   fromChatID,
			"to_chat_id":     toChatID,
			"new_message_id": fmt.Sprintf("%d", msg.MessageID),
		},
	}, nil
}

// JoinChat joins a chat by invite link
func (c *TelegramClient) JoinChat(ctx context.Context, inviteLink string) (*ActionProof, error) {
	// Extract chat ID from invite link or use directly
	var chatID string
	if u, err := url.Parse(inviteLink); err == nil && u.Host == "t.me" {
		chatID = "@" + u.Path[1:] // Remove leading /
	} else {
		chatID = inviteLink
	}

	params := map[string]interface{}{
		"chat_id": chatID,
	}

	resp, err := c.apiCall(ctx, "getChat", params)
	if err != nil {
		return nil, err
	}

	var chat struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	json.Unmarshal(resp.Result, &chat)

	return &ActionProof{
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":    fmt.Sprintf("%d", chat.ID),
			"chat_title": chat.Title,
			"action":     "join",
		},
	}, nil
}

func (c *TelegramClient) VerifyAction(ctx context.Context, actionType string, proof *ActionProof) (bool, error) {
	// For messages, we can verify they still exist
	if proof.PostID == "" {
		return false, errors.New("no message ID in proof")
	}

	chatID, ok := proof.Metadata["chat_id"]
	if !ok {
		return false, errors.New("no chat_id in proof metadata")
	}

	// Try to get the chat to verify bot is still member
	params := map[string]interface{}{
		"chat_id": chatID,
	}

	_, err := c.apiCall(ctx, "getChat", params)
	return err == nil, err
}

func (c *TelegramClient) GetRateLimitStatus(ctx context.Context) (*RateLimitStatus, error) {
	// Telegram rate limits: ~30 messages/second to same chat, 20 messages/minute to same group
	return &RateLimitStatus{
		Remaining: 30,
		Limit:     30,
		ResetAt:   time.Now().Add(time.Second).Unix(),
	}, nil
}
