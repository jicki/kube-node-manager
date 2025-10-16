package feishu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// Service handles Feishu (Lark) related operations
type Service struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewService creates a new Feishu service
func NewService(db *gorm.DB, logger *logger.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// TenantAccessTokenResponse represents the response from Feishu token API
type TenantAccessTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

// ChatInfo represents Feishu chat information
type ChatInfo struct {
	ChatID      string `json:"chat_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
	OwnerID     string `json:"owner_id"`
	OwnerIDType string `json:"owner_id_type"`
	External    bool   `json:"external"`
	TenantKey   string `json:"tenant_key"`
}

// ChatListResponse represents the response from Feishu chat list API
type ChatListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items     []ChatInfo `json:"items"`
		PageToken string     `json:"page_token"`
		HasMore   bool       `json:"has_more"`
	} `json:"data"`
}

// ChatInfoResponse represents the response from Feishu chat info API
type ChatInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ChatID      string `json:"chat_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
		OwnerID     string `json:"owner_id"`
		OwnerIDType string `json:"owner_id_type"`
		External    bool   `json:"external"`
		TenantKey   string `json:"tenant_key"`
	} `json:"data"`
}

// GetSettings retrieves Feishu settings
func (s *Service) GetSettings() (*model.FeishuSettings, error) {
	var settings model.FeishuSettings

	// Get the first (and should be only) settings record
	if err := s.db.First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default settings if none exist
			return &model.FeishuSettings{
				Enabled:   false,
				AppID:     "",
				AppSecret: "",
			}, nil
		}
		return nil, err
	}

	return &settings, nil
}

// UpdateSettings updates or creates Feishu settings
func (s *Service) UpdateSettings(enabled bool, appID, appSecret string) (*model.FeishuSettings, error) {
	var settings model.FeishuSettings

	// Try to find existing settings
	result := s.db.First(&settings)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Update fields
	settings.Enabled = enabled
	settings.AppID = appID

	// Only update app_secret if provided (non-empty)
	if appSecret != "" {
		// In production, this should be encrypted
		settings.AppSecret = appSecret
	}

	// Create or update
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if err := s.db.Create(&settings).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Save(&settings).Error; err != nil {
			return nil, err
		}
	}

	return &settings, nil
}

// getTenantAccessToken retrieves tenant access token from Feishu
func (s *Service) getTenantAccessToken(appID, appSecret string) (string, error) {
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal/"

	requestBody := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp TenantAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("feishu API error: code=%d, msg=%s", tokenResp.Code, tokenResp.Msg)
	}

	return tokenResp.TenantAccessToken, nil
}

// TestConnection tests the connection to Feishu API
func (s *Service) TestConnection(appID, appSecret string) error {
	// Use provided credentials or fetch from database
	if appID == "" || appSecret == "" {
		settings, err := s.GetSettings()
		if err != nil {
			return fmt.Errorf("failed to get settings: %w", err)
		}
		if appID == "" {
			appID = settings.AppID
		}
		if appSecret == "" {
			appSecret = settings.AppSecret
		}
	}

	if appID == "" || appSecret == "" {
		return errors.New("app_id and app_secret are required")
	}

	// Try to get tenant access token
	_, err := s.getTenantAccessToken(appID, appSecret)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// GetChatInfo retrieves information about a specific chat by chat_id
func (s *Service) GetChatInfo(chatID string) (*ChatInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if !settings.Enabled {
		return nil, errors.New("feishu is not enabled")
	}

	if settings.AppID == "" || settings.AppSecret == "" {
		return nil, errors.New("feishu is not configured")
	}

	// Get access token
	token, err := s.getTenantAccessToken(settings.AppID, settings.AppSecret)
	if err != nil {
		return nil, err
	}

	// Get chat info
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats/%s", chatID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatInfoResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Code != 0 {
		return nil, fmt.Errorf("feishu API error: code=%d, msg=%s", chatResp.Code, chatResp.Msg)
	}

	chatInfo := &ChatInfo{
		ChatID:      chatID, // 使用传入的 chatID，因为飞书 API 返回的数据中可能不包含此字段
		Name:        chatResp.Data.Name,
		Description: chatResp.Data.Description,
		Avatar:      chatResp.Data.Avatar,
		OwnerID:     chatResp.Data.OwnerID,
		OwnerIDType: chatResp.Data.OwnerIDType,
		External:    chatResp.Data.External,
		TenantKey:   chatResp.Data.TenantKey,
	}

	return chatInfo, nil
}

// ListChats retrieves all chats that the bot is a member of
func (s *Service) ListChats() ([]ChatInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if !settings.Enabled {
		return nil, errors.New("feishu is not enabled")
	}

	if settings.AppID == "" || settings.AppSecret == "" {
		return nil, errors.New("feishu is not configured")
	}

	// Get access token
	token, err := s.getTenantAccessToken(settings.AppID, settings.AppSecret)
	if err != nil {
		return nil, err
	}

	var allChats []ChatInfo
	pageToken := ""

	// Paginate through all chats
	for {
		url := "https://open.feishu.cn/open-apis/im/v1/chats?page_size=100"
		if pageToken != "" {
			url += "&page_token=" + pageToken
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var listResp ChatListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if listResp.Code != 0 {
			return nil, fmt.Errorf("feishu API error: code=%d, msg=%s", listResp.Code, listResp.Msg)
		}

		allChats = append(allChats, listResp.Data.Items...)

		if !listResp.Data.HasMore {
			break
		}

		pageToken = listResp.Data.PageToken
	}

	return allChats, nil
}
