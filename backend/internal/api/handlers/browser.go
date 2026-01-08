package handlers

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/services"
)

type BrowserHandler struct {
	services *services.Container
}

func NewBrowserHandler(s *services.Container) *BrowserHandler {
	return &BrowserHandler{services: s}
}

func (h *BrowserHandler) ListProfiles(c *gin.Context) {
	userID := getUserID(c)

	profiles, err := h.services.Browser.ListProfiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (h *BrowserHandler) CreateProfile(c *gin.Context) {
	userID := getUserID(c)

	var req services.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.services.Browser.CreateProfile(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *BrowserHandler) DeleteProfile(c *gin.Context) {
	userID := getUserID(c)
	profileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile ID"})
		return
	}

	if err := h.services.Browser.DeleteProfile(userID, profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile deleted"})
}

func (h *BrowserHandler) StartSession(c *gin.Context) {
	userID := getUserID(c)

	var req services.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.services.Browser.StartSession(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *BrowserHandler) ListSessions(c *gin.Context) {
	userID := getUserID(c)

	sessions, err := h.services.Browser.ListSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *BrowserHandler) GetSession(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := h.services.Browser.GetSession(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *BrowserHandler) StopSession(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if err := h.services.Browser.StopSession(userID, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session stopped"})
}

func (h *BrowserHandler) ExecuteAction(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req services.BrowserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, err := h.services.Browser.ExecuteAction(userID, sessionID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, action)
}

func (h *BrowserHandler) ContinueTask(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Browser.ContinueTask(userID, sessionID, req.Result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task continued"})
}

func (h *BrowserHandler) GetScreenshot(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	screenshot, err := h.services.Browser.GetScreenshot(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return as base64 or binary
	if c.Query("format") == "base64" {
		c.JSON(http.StatusOK, gin.H{"screenshot": base64.StdEncoding.EncodeToString(screenshot)})
	} else {
		c.Data(http.StatusOK, "image/png", screenshot)
	}
}
