package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/services"
)

type AccountHandler struct {
	services *services.Container
}

func NewAccountHandler(s *services.Container) *AccountHandler {
	return &AccountHandler{services: s}
}

func (h *AccountHandler) List(c *gin.Context) {
	userID := getUserID(c)
	platform := c.Query("platform")

	accounts, err := h.services.Account.List(userID, platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func (h *AccountHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req services.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.services.Account.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *AccountHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	account, err := h.services.Account.Get(userID, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *AccountHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	var req services.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.services.Account.Update(userID, accountID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *AccountHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	if err := h.services.Account.Delete(userID, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account deleted"})
}

func (h *AccountHandler) GetActivities(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	activities, total, err := h.services.Account.GetActivities(userID, accountID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

func (h *AccountHandler) LinkWallet(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	var req struct {
		WalletID uuid.UUID `json:"wallet_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Account.LinkWallet(userID, accountID, req.WalletID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "wallet linked"})
}

func (h *AccountHandler) Sync(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
		return
	}

	if err := h.services.Account.Sync(userID, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sync started"})
}
