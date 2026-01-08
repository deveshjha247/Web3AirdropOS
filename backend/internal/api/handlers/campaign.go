package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/services"
)

type CampaignHandler struct {
	services *services.Container
}

func NewCampaignHandler(s *services.Container) *CampaignHandler {
	return &CampaignHandler{services: s}
}

func (h *CampaignHandler) List(c *gin.Context) {
	userID := getUserID(c)
	status := c.Query("status")
	campaignType := c.Query("type")

	campaigns, err := h.services.Campaign.List(userID, status, campaignType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"campaigns": campaigns})
}

func (h *CampaignHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req services.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	campaign, err := h.services.Campaign.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

func (h *CampaignHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	campaign, err := h.services.Campaign.Get(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

func (h *CampaignHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	var req services.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	campaign, err := h.services.Campaign.Update(userID, campaignID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

func (h *CampaignHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	if err := h.services.Campaign.Delete(userID, campaignID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "campaign deleted"})
}

func (h *CampaignHandler) GetTasks(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	tasks, err := h.services.Campaign.GetTasks(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *CampaignHandler) AddTask(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	var req services.AddTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.services.Campaign.AddTask(userID, campaignID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *CampaignHandler) ExecuteBulk(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	var req services.BulkExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Campaign.ExecuteBulk(userID, campaignID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bulk execution started"})
}

func (h *CampaignHandler) GetProgress(c *gin.Context) {
	userID := getUserID(c)
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign ID"})
		return
	}

	progress, err := h.services.Campaign.GetProgress(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}
