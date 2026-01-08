package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cryptoautomation/backend/internal/services"
)

type DashboardHandler struct {
	services *services.Container
}

func NewDashboardHandler(s *services.Container) *DashboardHandler {
	return &DashboardHandler{services: s}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	userID := getUserID(c)

	stats, err := h.services.Dashboard.GetStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) GetRecentActivity(c *gin.Context) {
	userID := getUserID(c)

	activities, err := h.services.Dashboard.GetRecentActivity(userID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

func (h *DashboardHandler) GetActiveCampaigns(c *gin.Context) {
	userID := getUserID(c)

	campaigns, err := h.services.Dashboard.GetActiveCampaigns(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"campaigns": campaigns})
}
