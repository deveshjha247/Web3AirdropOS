package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/cryptoautomation/backend/internal/models"
	ws "github.com/cryptoautomation/backend/internal/websocket"
)

type BrowserService struct {
	container    *Container
	dockerClient *client.Client
	sessions     map[uuid.UUID]*BrowserSession
}

type BrowserSession struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ProfileID    uuid.UUID
	ContainerID  string
	VNCURL       string
	DebuggerURL  string
	WebSocketURL string
	wsConn       *websocket.Conn
	cancel       context.CancelFunc
}

func NewBrowserService(c *Container) *BrowserService {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("⚠️ Docker client not available: %v\n", err)
	}

	return &BrowserService{
		container:    c,
		dockerClient: dockerClient,
		sessions:     make(map[uuid.UUID]*BrowserSession),
	}
}

type CreateProfileRequest struct {
	Name         string     `json:"name" binding:"required"`
	UserAgent    string     `json:"user_agent"`
	ScreenWidth  int        `json:"screen_width"`
	ScreenHeight int        `json:"screen_height"`
	Language     string     `json:"language"`
	Timezone     string     `json:"timezone"`
	Platform     string     `json:"platform"`
	ProxyID      *uuid.UUID `json:"proxy_id"`
}

func (s *BrowserService) ListProfiles(userID uuid.UUID) ([]models.BrowserProfile, error) {
	var profiles []models.BrowserProfile
	if err := s.container.DB.Where("user_id = ?", userID).Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *BrowserService) CreateProfile(userID uuid.UUID, req *CreateProfileRequest) (*models.BrowserProfile, error) {
	profile := &models.BrowserProfile{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		UserAgent:    req.UserAgent,
		ScreenWidth:  req.ScreenWidth,
		ScreenHeight: req.ScreenHeight,
		Language:     req.Language,
		Timezone:     req.Timezone,
		Platform:     req.Platform,
		ProxyID:      req.ProxyID,
		IsActive:     true,
	}

	// Set defaults
	if profile.UserAgent == "" {
		profile.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	if profile.ScreenWidth == 0 {
		profile.ScreenWidth = 1920
	}
	if profile.ScreenHeight == 0 {
		profile.ScreenHeight = 1080
	}
	if profile.Language == "" {
		profile.Language = "en-US"
	}
	if profile.Platform == "" {
		profile.Platform = "Windows"
	}

	if err := s.container.DB.Create(profile).Error; err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *BrowserService) DeleteProfile(userID, profileID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", profileID, userID).Delete(&models.BrowserProfile{})
	if result.RowsAffected == 0 {
		return errors.New("profile not found")
	}
	return nil
}

type StartSessionRequest struct {
	ProfileID       uuid.UUID  `json:"profile_id" binding:"required"`
	TaskExecutionID *uuid.UUID `json:"task_execution_id"`
	StartURL        string     `json:"start_url"`
}

func (s *BrowserService) StartSession(userID uuid.UUID, req *StartSessionRequest) (*models.BrowserSession, error) {
	// Get profile
	var profile models.BrowserProfile
	if err := s.container.DB.Where("id = ? AND user_id = ?", req.ProfileID, userID).First(&profile).Error; err != nil {
		return nil, errors.New("profile not found")
	}

	// Get proxy if configured
	var proxyConfig string
	if profile.ProxyID != nil {
		var proxy models.Proxy
		if err := s.container.DB.First(&proxy, profile.ProxyID).Error; err == nil {
			if proxy.Username != "" {
				proxyConfig = fmt.Sprintf("%s:%s@%s:%d", proxy.Username, proxy.Password, proxy.Host, proxy.Port)
			} else {
				proxyConfig = fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)
			}
		}
	}

	// Create browser session record
	sessionID := uuid.New()
	session := &models.BrowserSession{
		ID:              sessionID,
		UserID:          userID,
		ProfileID:       req.ProfileID,
		Status:          "starting",
		TaskExecutionID: req.TaskExecutionID,
		StartedAt:       time.Now(),
		LastActivityAt:  time.Now(),
	}

	if err := s.container.DB.Create(session).Error; err != nil {
		return nil, err
	}

	// Broadcast terminal message
	s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
		Level:   "info",
		Source:  "browser",
		Message: "Starting browser session with profile: " + profile.Name,
	})

	// Start browser container
	go s.startBrowserContainer(userID, session, &profile, proxyConfig, req.StartURL)

	return session, nil
}

func (s *BrowserService) startBrowserContainer(userID uuid.UUID, session *models.BrowserSession, profile *models.BrowserProfile, proxyConfig string, startURL string) {
	ctx := context.Background()

	if s.dockerClient == nil {
		s.container.DB.Model(session).Updates(map[string]interface{}{
			"status":         "failed",
			"manual_message": "Docker not available",
		})
		return
	}

	// Container configuration
	env := []string{
		fmt.Sprintf("SCREEN_WIDTH=%d", profile.ScreenWidth),
		fmt.Sprintf("SCREEN_HEIGHT=%d", profile.ScreenHeight),
		fmt.Sprintf("VNC_PASSWORD=%s", uuid.New().String()[:8]),
		fmt.Sprintf("USER_AGENT=%s", profile.UserAgent),
		fmt.Sprintf("LANGUAGE=%s", profile.Language),
	}

	if proxyConfig != "" {
		env = append(env, fmt.Sprintf("PROXY=%s", proxyConfig))
	}

	if startURL != "" {
		env = append(env, fmt.Sprintf("START_URL=%s", startURL))
	}

	resp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "browser-automation:latest",
			Env:   env,
			ExposedPorts: map[string]struct{}{
				"5900/tcp": {},
				"9222/tcp": {},
				"8080/tcp": {},
			},
		},
		&container.HostConfig{
			AutoRemove: true,
			Resources: container.Resources{
				Memory:   1024 * 1024 * 1024, // 1GB RAM
				NanoCPUs: 1000000000,         // 1 CPU
			},
		},
		nil,
		nil,
		fmt.Sprintf("browser-%s", session.ID.String()[:8]),
	)

	if err != nil {
		s.container.DB.Model(session).Updates(map[string]interface{}{
			"status":         "failed",
			"manual_message": err.Error(),
		})
		s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
			Level:   "error",
			Source:  "browser",
			Message: "Failed to create browser container: " + err.Error(),
		})
		return
	}

	// Start container
	if err := s.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		s.container.DB.Model(session).Updates(map[string]interface{}{
			"status":         "failed",
			"manual_message": err.Error(),
		})
		return
	}

	// Get container info for ports
	info, err := s.dockerClient.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return
	}

	// Update session with container info
	vncPort := info.NetworkSettings.Ports["5900/tcp"][0].HostPort
	debugPort := info.NetworkSettings.Ports["9222/tcp"][0].HostPort
	wsPort := info.NetworkSettings.Ports["8080/tcp"][0].HostPort

	s.container.DB.Model(session).Updates(map[string]interface{}{
		"container_id":   resp.ID,
		"vnc_url":        fmt.Sprintf("vnc://localhost:%s", vncPort),
		"debugger_url":   fmt.Sprintf("http://localhost:%s", debugPort),
		"websocket_url":  fmt.Sprintf("ws://localhost:%s", wsPort),
		"status":         "ready",
	})

	// Store in memory
	s.sessions[session.ID] = &BrowserSession{
		ID:           session.ID,
		UserID:       userID,
		ProfileID:    session.ProfileID,
		ContainerID:  resp.ID,
		VNCURL:       fmt.Sprintf("vnc://localhost:%s", vncPort),
		DebuggerURL:  fmt.Sprintf("http://localhost:%s", debugPort),
		WebSocketURL: fmt.Sprintf("ws://localhost:%s", wsPort),
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
		Level:   "success",
		Source:  "browser",
		Message: "Browser session ready",
		Details: map[string]interface{}{
			"session_id": session.ID.String(),
			"vnc_url":    fmt.Sprintf("vnc://localhost:%s", vncPort),
		},
	})

	s.container.WSHub.BroadcastToUser(userID.String(), "browser:ready", map[string]interface{}{
		"session_id":    session.ID.String(),
		"vnc_url":       fmt.Sprintf("vnc://localhost:%s", vncPort),
		"debugger_url":  fmt.Sprintf("http://localhost:%s", debugPort),
		"websocket_url": fmt.Sprintf("ws://localhost:%s", wsPort),
	})
}

func (s *BrowserService) ListSessions(userID uuid.UUID) ([]models.BrowserSession, error) {
	var sessions []models.BrowserSession
	if err := s.container.DB.Where("user_id = ? AND status != ?", userID, "stopped").
		Order("started_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *BrowserService) GetSession(userID, sessionID uuid.UUID) (*models.BrowserSession, error) {
	var session models.BrowserSession
	if err := s.container.DB.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *BrowserService) StopSession(userID, sessionID uuid.UUID) error {
	session, err := s.GetSession(userID, sessionID)
	if err != nil {
		return err
	}

	// Stop container
	if s.dockerClient != nil && session.ContainerID != "" {
		ctx := context.Background()
		timeout := 10
		s.dockerClient.ContainerStop(ctx, session.ContainerID, container.StopOptions{Timeout: &timeout})
	}

	// Update status
	s.container.DB.Model(session).Update("status", "stopped")

	// Remove from memory
	delete(s.sessions, sessionID)

	s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
		Level:   "info",
		Source:  "browser",
		Message: "Browser session stopped",
	})

	return nil
}

type BrowserActionRequest struct {
	Type   string `json:"type" binding:"required"` // navigate, click, type, scroll, screenshot, evaluate
	Target string `json:"target"`                  // CSS selector or URL
	Value  string `json:"value"`                   // Text to type, JS to evaluate, etc.
}

func (s *BrowserService) ExecuteAction(userID, sessionID uuid.UUID, req *BrowserActionRequest) (*models.BrowserAction, error) {
	session, err := s.GetSession(userID, sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != "ready" && session.Status != "busy" {
		return nil, errors.New("session not ready")
	}

	// Create action record
	action := &models.BrowserAction{
		ID:        uuid.New(),
		SessionID: sessionID,
		Type:      req.Type,
		Target:    req.Target,
		Value:     req.Value,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := s.container.DB.Create(action).Error; err != nil {
		return nil, err
	}

	// Update session status
	s.container.DB.Model(session).Update("status", "busy")

	// Broadcast terminal message
	s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
		Level:   "debug",
		Source:  "browser",
		Message: fmt.Sprintf("Executing action: %s", req.Type),
		Details: map[string]interface{}{
			"target": req.Target,
			"value":  req.Value,
		},
	})

	// Execute action via CDP
	go s.executeActionViaCDP(userID, session, action, req)

	return action, nil
}

func (s *BrowserService) executeActionViaCDP(userID uuid.UUID, session *models.BrowserSession, action *models.BrowserAction, req *BrowserActionRequest) {
	var result interface{}
	var err error

	switch req.Type {
	case "navigate":
		result, err = s.cdpNavigate(session.DebuggerURL, req.Target)
	case "click":
		result, err = s.cdpClick(session.DebuggerURL, req.Target)
	case "type":
		result, err = s.cdpType(session.DebuggerURL, req.Target, req.Value)
	case "screenshot":
		result, err = s.cdpScreenshot(session.DebuggerURL)
	case "evaluate":
		result, err = s.cdpEvaluate(session.DebuggerURL, req.Value)
	default:
		err = errors.New("unsupported action type")
	}

	if err != nil {
		action.Status = "failed"
		action.Error = err.Error()
	} else {
		action.Status = "success"
		resultJSON, _ := json.Marshal(result)
		action.Result = string(resultJSON)
	}

	s.container.DB.Save(action)
	s.container.DB.Model(session).Update("status", "ready")

	if err != nil {
		s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
			Level:   "error",
			Source:  "browser",
			Message: "Action failed: " + err.Error(),
		})
	} else {
		s.container.WSHub.BroadcastTerminal(userID.String(), ws.TerminalMessage{
			Level:   "success",
			Source:  "browser",
			Message: "Action completed: " + req.Type,
		})
	}
}

// CDP helper methods (Chrome DevTools Protocol)
func (s *BrowserService) cdpNavigate(debuggerURL, url string) (interface{}, error) {
	return s.cdpSend(debuggerURL, "Page.navigate", map[string]interface{}{
		"url": url,
	})
}

func (s *BrowserService) cdpClick(debuggerURL, selector string) (interface{}, error) {
	// First, find the element
	_, err := s.cdpSend(debuggerURL, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`document.querySelector('%s').click()`, selector),
	})
	return nil, err
}

func (s *BrowserService) cdpType(debuggerURL, selector, text string) (interface{}, error) {
	// Focus element and insert text
	_, err := s.cdpSend(debuggerURL, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`
			const el = document.querySelector('%s');
			el.focus();
			el.value = '%s';
			el.dispatchEvent(new Event('input', { bubbles: true }));
		`, selector, strings.ReplaceAll(text, "'", "\\'")),
	})
	return nil, err
}

func (s *BrowserService) cdpScreenshot(debuggerURL string) (interface{}, error) {
	return s.cdpSend(debuggerURL, "Page.captureScreenshot", map[string]interface{}{
		"format": "png",
	})
}

func (s *BrowserService) cdpEvaluate(debuggerURL, expression string) (interface{}, error) {
	return s.cdpSend(debuggerURL, "Runtime.evaluate", map[string]interface{}{
		"expression": expression,
	})
}

func (s *BrowserService) cdpSend(debuggerURL, method string, params map[string]interface{}) (interface{}, error) {
	// Get WebSocket debugger URL
	resp, err := http.Get(debuggerURL + "/json/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var versionInfo map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&versionInfo)

	wsURL, ok := versionInfo["webSocketDebuggerUrl"].(string)
	if !ok {
		return nil, errors.New("could not get WebSocket URL")
	}

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send command
	cmd := map[string]interface{}{
		"id":     1,
		"method": method,
		"params": params,
	}

	if err := conn.WriteJSON(cmd); err != nil {
		return nil, err
	}

	// Read response
	var response map[string]interface{}
	if err := conn.ReadJSON(&response); err != nil {
		return nil, err
	}

	if errData, ok := response["error"]; ok {
		return nil, fmt.Errorf("CDP error: %v", errData)
	}

	return response["result"], nil
}

func (s *BrowserService) ContinueTask(userID, sessionID uuid.UUID, result map[string]interface{}) error {
	session, err := s.GetSession(userID, sessionID)
	if err != nil {
		return err
	}

	if session.TaskExecutionID == nil {
		return errors.New("no task associated with this session")
	}

	// Continue the task execution
	return s.container.Task.Continue(userID, *session.TaskExecutionID, *session.TaskExecutionID, result)
}

func (s *BrowserService) GetScreenshot(userID, sessionID uuid.UUID) ([]byte, error) {
	session, err := s.GetSession(userID, sessionID)
	if err != nil {
		return nil, err
	}

	result, err := s.cdpScreenshot(session.DebuggerURL)
	if err != nil {
		return nil, err
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid screenshot result")
	}

	dataStr, ok := resultMap["data"].(string)
	if !ok {
		return nil, errors.New("no screenshot data")
	}

	// Decode base64
	return io.ReadAll(bytes.NewReader([]byte(dataStr)))
}
