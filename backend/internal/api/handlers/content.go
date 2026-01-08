package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cryptoautomation/backend/internal/services"
)

type ContentHandler struct {
	services *services.Container
}

func NewContentHandler(s *services.Container) *ContentHandler {
	return &ContentHandler{services: s}
}

func (h *ContentHandler) Generate(c *gin.Context) {
	userID := getUserID(c)

	var req services.GenerateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	drafts, err := h.services.Content.Generate(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drafts": drafts})
}

func (h *ContentHandler) ListDrafts(c *gin.Context) {
	userID := getUserID(c)
	platform := c.Query("platform")
	status := c.Query("status")

	drafts, err := h.services.Content.ListDrafts(userID, platform, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drafts": drafts})
}

func (h *ContentHandler) GetDraft(c *gin.Context) {
	userID := getUserID(c)
	draftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}

	draft, err := h.services.Content.GetDraft(userID, draftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	c.JSON(http.StatusOK, draft)
}

func (h *ContentHandler) UpdateDraft(c *gin.Context) {
	userID := getUserID(c)
	draftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}

	var req services.UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	draft, err := h.services.Content.UpdateDraft(userID, draftID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, draft)
}

func (h *ContentHandler) DeleteDraft(c *gin.Context) {
	userID := getUserID(c)
	draftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}

	if err := h.services.Content.DeleteDraft(userID, draftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "draft deleted"})
}

func (h *ContentHandler) ApproveDraft(c *gin.Context) {
	userID := getUserID(c)
	draftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}

	draft, err := h.services.Content.ApproveDraft(userID, draftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, draft)
}

func (h *ContentHandler) Schedule(c *gin.Context) {
	userID := getUserID(c)

	var req services.SchedulePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.services.Content.Schedule(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, post)
}

func (h *ContentHandler) ListScheduled(c *gin.Context) {
	userID := getUserID(c)
	platform := c.Query("platform")
	status := c.Query("status")

	posts, err := h.services.Content.ListScheduled(userID, platform, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"scheduled_posts": posts})
}

func (h *ContentHandler) CancelScheduled(c *gin.Context) {
	userID := getUserID(c)
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	if err := h.services.Content.CancelScheduled(userID, postID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "scheduled post cancelled"})
}
