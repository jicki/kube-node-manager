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

// ClusterServiceInterface 集群服务接口
type ClusterServiceInterface interface {
	List(req interface{}, userID uint) (interface{}, error)
}

// NodeServiceInterface 节点服务接口
type NodeServiceInterface interface {
	List(req interface{}, userID uint) (interface{}, error)
	Get(req interface{}, userID uint) (interface{}, error)
	Cordon(req interface{}, userID uint) error
	Uncordon(req interface{}, userID uint) error
}

// AuditServiceInterface 审计服务接口
type AuditServiceInterface interface {
	List(req interface{}) (interface{}, error)
}

// LabelServiceInterface 标签服务接口
type LabelServiceInterface interface {
	UpdateNodeLabels(req interface{}, userID uint) error
	BatchUpdateLabels(req interface{}, userID uint) error
}

// TaintServiceInterface 污点服务接口
type TaintServiceInterface interface {
	UpdateNodeTaints(req interface{}, userID uint) error
	BatchUpdateTaints(req interface{}, userID uint) error
	RemoveTaint(clusterName, nodeName, taintKey string, userID uint) error
}

// Service handles Feishu (Lark) related operations
type Service struct {
	db             *gorm.DB
	logger         *logger.Logger
	commandRouter  *CommandRouter
	eventClient    *EventClient
	clusterService ClusterServiceInterface
	nodeService    NodeServiceInterface
	auditService   AuditServiceInterface
	labelService   LabelServiceInterface
	taintService   TaintServiceInterface
}

// NewService creates a new Feishu service
func NewService(db *gorm.DB, logger *logger.Logger) *Service {
	service := &Service{
		db:     db,
		logger: logger,
	}
	// Initialize command router
	service.commandRouter = NewCommandRouter()
	return service
}

// SetClusterService 设置集群服务
func (s *Service) SetClusterService(clusterSvc ClusterServiceInterface) {
	s.clusterService = clusterSvc
}

// SetNodeService 设置节点服务
func (s *Service) SetNodeService(nodeSvc NodeServiceInterface) {
	s.nodeService = nodeSvc
}

// SetAuditService 设置审计服务
func (s *Service) SetAuditService(auditSvc AuditServiceInterface) {
	s.auditService = auditSvc
}

// SetLabelService 设置标签服务
func (s *Service) SetLabelService(labelSvc LabelServiceInterface) {
	s.labelService = labelSvc
}

// SetTaintService 设置污点服务
func (s *Service) SetTaintService(taintSvc TaintServiceInterface) {
	s.taintService = taintSvc
}

// InitializeEventClient 初始化或重启事件客户端
func (s *Service) InitializeEventClient() error {
	// 停止现有客户端
	if s.eventClient != nil {
		s.eventClient.Stop()
		s.eventClient = nil
	}

	// 获取配置
	settings, err := s.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// 检查是否启用机器人
	if !settings.Enabled || !settings.BotEnabled || settings.AppID == "" || settings.AppSecret == "" {
		s.logger.Info("Feishu bot is not enabled or not configured")
		return nil
	}

	// 创建并启动新客户端
	s.eventClient = NewEventClient(s, settings.AppID, settings.AppSecret)
	if err := s.eventClient.Start(); err != nil {
		return fmt.Errorf("failed to start event client: %w", err)
	}

	s.logger.Info("Feishu event client initialized successfully")
	return nil
}

// IsEventClientConnected 检查事件客户端连接状态
func (s *Service) IsEventClientConnected() bool {
	if s.eventClient == nil {
		return false
	}
	return s.eventClient.IsConnected()
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

// GetSettingsWithStatus 获取设置并包含连接状态
func (s *Service) GetSettingsWithStatus() (*model.FeishuSettingsResponse, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	response := settings.ToResponse()
	response.BotConnected = s.IsEventClientConnected()
	return response, nil
}

// UpdateSettings updates or creates Feishu settings
func (s *Service) UpdateSettings(enabled bool, appID, appSecret string, botEnabled bool) (*model.FeishuSettings, error) {
	var settings model.FeishuSettings

	// Try to find existing settings
	result := s.db.First(&settings)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Update fields
	settings.Enabled = enabled
	settings.AppID = appID
	settings.BotEnabled = botEnabled

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

	// 如果机器人已启用且配置有效，初始化事件客户端
	if settings.Enabled && settings.BotEnabled {
		go func() {
			if err := s.InitializeEventClient(); err != nil {
				s.logger.Error("Failed to initialize event client: " + err.Error())
			}
		}()
	} else if s.eventClient != nil {
		// 如果机器人被禁用，停止事件客户端
		s.eventClient.Stop()
		s.eventClient = nil
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

// BindUser binds a Feishu user to a system user
func (s *Service) BindUser(feishuUserID string, systemUserID uint, username, feishuName string) (*model.FeishuUserMapping, error) {
	// Check if binding already exists
	var existing model.FeishuUserMapping
	result := s.db.Where("feishu_user_id = ?", feishuUserID).First(&existing)

	if result.Error == nil {
		// Update existing binding
		existing.SystemUserID = systemUserID
		existing.Username = username
		existing.FeishuName = feishuName
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Create new binding
	mapping := &model.FeishuUserMapping{
		FeishuUserID: feishuUserID,
		SystemUserID: systemUserID,
		Username:     username,
		FeishuName:   feishuName,
	}

	if err := s.db.Create(mapping).Error; err != nil {
		return nil, err
	}

	return mapping, nil
}

// UnbindUser unbinds a system user from Feishu
func (s *Service) UnbindUser(systemUserID uint) error {
	return s.db.Where("system_user_id = ?", systemUserID).Delete(&model.FeishuUserMapping{}).Error
}

// GetBindingByUserID retrieves the binding for a system user
func (s *Service) GetBindingByUserID(systemUserID uint) (*model.FeishuUserMapping, error) {
	var mapping model.FeishuUserMapping
	if err := s.db.Where("system_user_id = ?", systemUserID).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetOrCreateUserSession 获取或创建用户会话
func (s *Service) GetOrCreateUserSession(feishuUserID string) (*model.FeishuUserSession, error) {
	var session model.FeishuUserSession
	result := s.db.Where("feishu_user_id = ?", feishuUserID).First(&session)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 创建新会话
			session = model.FeishuUserSession{
				FeishuUserID:    feishuUserID,
				LastCommandTime: time.Now(),
			}
			if err := s.db.Create(&session).Error; err != nil {
				return nil, err
			}
			return &session, nil
		}
		return nil, result.Error
	}

	return &session, nil
}

// SetCurrentCluster 设置用户当前选择的集群
func (s *Service) SetCurrentCluster(feishuUserID, clusterName string) error {
	session, err := s.GetOrCreateUserSession(feishuUserID)
	if err != nil {
		return err
	}

	session.CurrentCluster = clusterName
	session.LastCommandTime = time.Now()

	return s.db.Save(session).Error
}

// GetCurrentCluster 获取用户当前选择的集群
func (s *Service) GetCurrentCluster(feishuUserID string) (string, error) {
	session, err := s.GetOrCreateUserSession(feishuUserID)
	if err != nil {
		return "", err
	}

	return session.CurrentCluster, nil
}
