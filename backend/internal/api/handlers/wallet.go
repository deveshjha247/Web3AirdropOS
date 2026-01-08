package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cryptoautomation/backend/internal/services"
)

type WalletHandler struct {
	services *services.Container
}

func NewWalletHandler(s *services.Container) *WalletHandler {
	return &WalletHandler{services: s}
}

func getUserID(c *gin.Context) uuid.UUID {
	userID, _ := c.Get("user_id")
	return userID.(uuid.UUID)
}

func (h *WalletHandler) List(c *gin.Context) {
	userID := getUserID(c)
	walletType := c.Query("type")
	
	var groupID *uuid.UUID
	if gid := c.Query("group_id"); gid != "" {
		if parsed, err := uuid.Parse(gid); err == nil {
			groupID = &parsed
		}
	}

	wallets, err := h.services.Wallet.List(userID, walletType, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (h *WalletHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	
	var req services.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.services.Wallet.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (h *WalletHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	wallet, err := h.services.Wallet.Get(userID, walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	var req services.UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.services.Wallet.Update(userID, walletID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	if err := h.services.Wallet.Delete(userID, walletID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "wallet deleted"})
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	balance, err := h.services.Wallet.GetBalance(userID, walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	transactions, err := h.services.Wallet.GetTransactions(userID, walletID, 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

func (h *WalletHandler) PrepareTransaction(c *gin.Context) {
	userID := getUserID(c)
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet ID"})
		return
	}

	var req services.PrepareTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prepared, err := h.services.Wallet.PrepareTransaction(userID, walletID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prepared)
}

func (h *WalletHandler) Import(c *gin.Context) {
	userID := getUserID(c)
	
	var req services.ImportWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.services.Wallet.Import(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (h *WalletHandler) BulkCreate(c *gin.Context) {
	userID := getUserID(c)
	
	var req struct {
		Count    int               `json:"count" binding:"required,min=1,max=50"`
		Type     string            `json:"type" binding:"required"`
		GroupID  *uuid.UUID        `json:"group_id"`
		Prefix   string            `json:"prefix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallets, err := h.services.Wallet.BulkCreate(userID, req.Count, req.Type, req.GroupID, req.Prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallets": wallets, "count": len(wallets)})
}

// Wallet Group Handler
type WalletGroupHandler struct {
	services *services.Container
}

func NewWalletGroupHandler(s *services.Container) *WalletGroupHandler {
	return &WalletGroupHandler{services: s}
}

func (h *WalletGroupHandler) List(c *gin.Context) {
	userID := getUserID(c)

	groups, err := h.services.Wallet.ListGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (h *WalletGroupHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	
	var req services.CreateWalletGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.services.Wallet.CreateGroup(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *WalletGroupHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req services.UpdateWalletGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.services.Wallet.UpdateGroup(userID, groupID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *WalletGroupHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.services.Wallet.DeleteGroup(userID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

func (h *WalletGroupHandler) AddWallets(c *gin.Context) {
	userID := getUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req struct {
		WalletIDs []uuid.UUID `json:"wallet_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Wallet.AddWalletsToGroup(userID, groupID, req.WalletIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "wallets added"})
}

func (h *WalletGroupHandler) RemoveWallets(c *gin.Context) {
	userID := getUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req struct {
		WalletIDs []uuid.UUID `json:"wallet_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Wallet.RemoveWalletsFromGroup(userID, groupID, req.WalletIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "wallets removed"})
}
