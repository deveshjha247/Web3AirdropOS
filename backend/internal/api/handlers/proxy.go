package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cryptoautomation/backend/internal/services"
)

type ProxyHandler struct {
	services *services.Container
}

func NewProxyHandler(s *services.Container) *ProxyHandler {
	return &ProxyHandler{services: s}
}

func (h *ProxyHandler) List(c *gin.Context) {
	userID := getUserID(c)

	proxies, err := h.services.Proxy.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"proxies": proxies})
}

func (h *ProxyHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req services.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxy, err := h.services.Proxy.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, proxy)
}

func (h *ProxyHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy ID"})
		return
	}

	var req services.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxy, err := h.services.Proxy.Update(userID, proxyID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

func (h *ProxyHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy ID"})
		return
	}

	if err := h.services.Proxy.Delete(userID, proxyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "proxy deleted"})
}

func (h *ProxyHandler) Test(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy ID"})
		return
	}

	result, err := h.services.Proxy.Test(userID, proxyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) BulkCreate(c *gin.Context) {
	userID := getUserID(c)

	var req services.BulkCreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxies, err := h.services.Proxy.BulkCreate(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"proxies": proxies, "count": len(proxies)})
}
