package services

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"

	"github.com/web3airdropos/backend/internal/models"
)

type ProxyService struct {
	container *Container
}

func NewProxyService(c *Container) *ProxyService {
	return &ProxyService{container: c}
}

type CreateProxyRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type" binding:"required"` // http, socks5, residential
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password"`
	Country  string `json:"country"`
}

type UpdateProxyRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     *int   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Country  string `json:"country"`
	IsActive *bool  `json:"is_active"`
}

type ProxyTestResult struct {
	Success   bool   `json:"success"`
	Latency   int    `json:"latency"` // in milliseconds
	ExternalIP string `json:"external_ip"`
	Country   string `json:"country"`
	Error     string `json:"error,omitempty"`
}

func (s *ProxyService) List(userID uuid.UUID) ([]models.Proxy, error) {
	var proxies []models.Proxy
	if err := s.container.DB.Where("user_id = ?", userID).Find(&proxies).Error; err != nil {
		return nil, err
	}
	return proxies, nil
}

func (s *ProxyService) Create(userID uuid.UUID, req *CreateProxyRequest) (*models.Proxy, error) {
	proxy := &models.Proxy{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     req.Name,
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Country:  req.Country,
		IsActive: true,
	}

	if err := s.container.DB.Create(proxy).Error; err != nil {
		return nil, err
	}

	return proxy, nil
}

func (s *ProxyService) Update(userID, proxyID uuid.UUID, req *UpdateProxyRequest) (*models.Proxy, error) {
	var proxyRecord models.Proxy
	if err := s.container.DB.Where("id = ? AND user_id = ?", proxyID, userID).First(&proxyRecord).Error; err != nil {
		return nil, errors.New("proxy not found")
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Host != "" {
		updates["host"] = req.Host
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.Country != "" {
		updates["country"] = req.Country
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.container.DB.Model(&proxyRecord).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &proxyRecord, nil
}

func (s *ProxyService) Delete(userID, proxyID uuid.UUID) error {
	result := s.container.DB.Where("id = ? AND user_id = ?", proxyID, userID).Delete(&models.Proxy{})
	if result.RowsAffected == 0 {
		return errors.New("proxy not found")
	}
	return nil
}

func (s *ProxyService) Test(userID, proxyID uuid.UUID) (*ProxyTestResult, error) {
	var proxyRecord models.Proxy
	if err := s.container.DB.Where("id = ? AND user_id = ?", proxyID, userID).First(&proxyRecord).Error; err != nil {
		return nil, errors.New("proxy not found")
	}

	return s.testProxy(&proxyRecord)
}

func (s *ProxyService) testProxy(proxyRecord *models.Proxy) (*ProxyTestResult, error) {
	result := &ProxyTestResult{}
	startTime := time.Now()

	var httpClient *http.Client

	switch proxyRecord.Type {
	case "socks5":
		auth := &proxy.Auth{}
		if proxyRecord.Username != "" {
			auth.User = proxyRecord.Username
			auth.Password = proxyRecord.Password
		} else {
			auth = nil
		}

		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxyRecord.Host, proxyRecord.Port), auth, proxy.Direct)
		if err != nil {
			result.Error = err.Error()
			return result, nil
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: dialer.Dial,
			},
			Timeout: 10 * time.Second,
		}

	case "http":
		proxyURL := fmt.Sprintf("http://%s:%d", proxyRecord.Host, proxyRecord.Port)
		if proxyRecord.Username != "" {
			proxyURL = fmt.Sprintf("http://%s:%s@%s:%d", proxyRecord.Username, proxyRecord.Password, proxyRecord.Host, proxyRecord.Port)
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: 10 * time.Second,
		}
		_ = proxyURL // TODO: Set proxy URL properly

	default:
		result.Error = "unsupported proxy type"
		return result, nil
	}

	// Test connectivity
	resp, err := httpClient.Get("https://api.ipify.org?format=json")
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	result.Latency = int(time.Since(startTime).Milliseconds())
	result.Success = true

	// Parse IP response
	var ipResp struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err == nil {
		result.ExternalIP = ipResp.IP
	}

	// Update proxy record with latency
	s.container.DB.Model(proxyRecord).Updates(map[string]interface{}{
		"last_check": time.Now(),
		"latency":    result.Latency,
	})

	return result, nil
}

type BulkCreateProxyRequest struct {
	Proxies []CreateProxyRequest `json:"proxies" binding:"required"`
}

func (s *ProxyService) BulkCreate(userID uuid.UUID, req *BulkCreateProxyRequest) ([]models.Proxy, error) {
	var proxies []models.Proxy

	for _, proxyReq := range req.Proxies {
		proxy := models.Proxy{
			ID:       uuid.New(),
			UserID:   userID,
			Name:     proxyReq.Name,
			Type:     proxyReq.Type,
			Host:     proxyReq.Host,
			Port:     proxyReq.Port,
			Username: proxyReq.Username,
			Password: proxyReq.Password,
			Country:  proxyReq.Country,
			IsActive: true,
		}

		if err := s.container.DB.Create(&proxy).Error; err != nil {
			continue
		}
		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

// GetHTTPClient returns an HTTP client configured with the specified proxy
func (s *ProxyService) GetHTTPClient(proxyID uuid.UUID) (*http.Client, error) {
	var proxyRecord models.Proxy
	if err := s.container.DB.First(&proxyRecord, proxyID).Error; err != nil {
		return nil, err
	}

	switch proxyRecord.Type {
	case "socks5":
		auth := &proxy.Auth{}
		if proxyRecord.Username != "" {
			auth.User = proxyRecord.Username
			auth.Password = proxyRecord.Password
		} else {
			auth = nil
		}

		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxyRecord.Host, proxyRecord.Port), auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Client{
			Transport: &http.Transport{
				Dial: dialer.Dial,
			},
			Timeout: 30 * time.Second,
		}, nil

	default:
		return &http.Client{Timeout: 30 * time.Second}, nil
	}
}

// GetDialer returns a net.Dialer configured with the specified proxy
func (s *ProxyService) GetDialer(proxyID uuid.UUID) (func(network, addr string) (net.Conn, error), error) {
	var proxyRecord models.Proxy
	if err := s.container.DB.First(&proxyRecord, proxyID).Error; err != nil {
		return nil, err
	}

	if proxyRecord.Type == "socks5" {
		auth := &proxy.Auth{}
		if proxyRecord.Username != "" {
			auth.User = proxyRecord.Username
			auth.Password = proxyRecord.Password
		} else {
			auth = nil
		}

		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxyRecord.Host, proxyRecord.Port), auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return dialer.Dial, nil
	}

	return net.Dial, nil
}

import "encoding/json"
