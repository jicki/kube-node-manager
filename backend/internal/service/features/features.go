package features

import (
	"encoding/json"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"sync"

	"gorm.io/gorm"
)

// Service 功能特性管理服务
type Service struct {
	logger *logger.Logger
	db     *gorm.DB
	cache  *FeatureCache
	mu     sync.RWMutex
}

// FeatureCache 功能特性缓存
type FeatureCache struct {
	Features map[string]interface{}
	mu       sync.RWMutex
}

// FeatureStatus 功能状态
type FeatureStatus struct {
	Automation AutomationFeatures `json:"automation"`
}

// AutomationFeatures 自动化功能开关
type AutomationFeatures struct {
	Enabled   bool              `json:"enabled"`
	Ansible   AnsibleFeatures   `json:"ansible"`
	SSH       SSHFeatures       `json:"ssh"`
	Scripts   ScriptsFeatures   `json:"scripts"`
	Workflows WorkflowsFeatures `json:"workflows"`
}

// AnsibleFeatures Ansible 功能配置
type AnsibleFeatures struct {
	Enabled    bool   `json:"enabled"`
	BinaryPath string `json:"binary_path"`
	TempDir    string `json:"temp_dir"`
	Timeout    int    `json:"timeout"` // 秒
}

// SSHFeatures SSH 功能配置
type SSHFeatures struct {
	Enabled            bool `json:"enabled"`
	Timeout            int  `json:"timeout"` // 秒
	MaxConcurrent      int  `json:"max_concurrent"`
	ConnectionPoolSize int  `json:"connection_pool_size"`
}

// ScriptsFeatures 脚本功能配置
type ScriptsFeatures struct {
	Enabled bool `json:"enabled"`
	Timeout int  `json:"timeout"` // 秒
}

// WorkflowsFeatures 工作流功能配置
type WorkflowsFeatures struct {
	Enabled     bool `json:"enabled"`
	MaxSteps    int  `json:"max_steps"`
	StepTimeout int  `json:"step_timeout"` // 秒
}

// NewService 创建功能特性服务
func NewService(logger *logger.Logger, db *gorm.DB) *Service {
	return &Service{
		logger: logger,
		db:     db,
		cache: &FeatureCache{
			Features: make(map[string]interface{}),
		},
	}
}

// GetFeatureStatus 获取所有功能状态
func (s *Service) GetFeatureStatus() (*FeatureStatus, error) {
	// 首先尝试从缓存获取
	s.cache.mu.RLock()
	if cachedStatus, ok := s.cache.Features["status"]; ok {
		s.cache.mu.RUnlock()
		if status, ok := cachedStatus.(*FeatureStatus); ok {
			return status, nil
		}
	}
	s.cache.mu.RUnlock()

	// 从数据库加载配置
	status, err := s.loadFeatureStatusFromDB()
	if err != nil {
		s.logger.Errorf("Failed to load feature status from database: %v", err)
		// 如果数据库加载失败，返回默认配置
		return s.getDefaultFeatureStatus(), nil
	}

	// 更新缓存
	s.cache.mu.Lock()
	s.cache.Features["status"] = status
	s.cache.mu.Unlock()

	return status, nil
}

// loadFeatureStatusFromDB 从数据库加载功能状态
func (s *Service) loadFeatureStatusFromDB() (*FeatureStatus, error) {
	var configs []model.AutomationConfig
	if err := s.db.Where("category = ?", "automation").Find(&configs).Error; err != nil {
		return nil, err
	}

	// 如果数据库中没有配置，初始化默认配置
	if len(configs) == 0 {
		return s.initializeDefaultConfig()
	}

	// 解析配置
	status := &FeatureStatus{}
	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	// 解析 automation 主配置
	if val, ok := configMap["automation.enabled"]; ok {
		var enabled bool
		json.Unmarshal([]byte(val), &enabled)
		status.Automation.Enabled = enabled
	}

	// 解析 ansible 配置
	if val, ok := configMap["automation.ansible"]; ok {
		json.Unmarshal([]byte(val), &status.Automation.Ansible)
	}

	// 解析 ssh 配置
	if val, ok := configMap["automation.ssh"]; ok {
		json.Unmarshal([]byte(val), &status.Automation.SSH)
	}

	// 解析 scripts 配置
	if val, ok := configMap["automation.scripts"]; ok {
		json.Unmarshal([]byte(val), &status.Automation.Scripts)
	}

	// 解析 workflows 配置
	if val, ok := configMap["automation.workflows"]; ok {
		json.Unmarshal([]byte(val), &status.Automation.Workflows)
	}

	return status, nil
}

// initializeDefaultConfig 初始化默认配置
func (s *Service) initializeDefaultConfig() (*FeatureStatus, error) {
	status := s.getDefaultFeatureStatus()

	// 保存到数据库
	configs := []model.AutomationConfig{
		{
			Key:      "automation.enabled",
			Value:    `false`,
			Category: "automation",
			IsSystem: true,
		},
		{
			Key:      "automation.ansible",
			Value:    `{"enabled":true,"binary_path":"/usr/bin/ansible-playbook","temp_dir":"/tmp/ansible-runs","timeout":3600}`,
			Category: "automation",
			IsSystem: true,
		},
		{
			Key:      "automation.ssh",
			Value:    `{"enabled":true,"timeout":30,"max_concurrent":50,"connection_pool_size":20}`,
			Category: "automation",
			IsSystem: true,
		},
		{
			Key:      "automation.scripts",
			Value:    `{"enabled":true,"timeout":600}`,
			Category: "automation",
			IsSystem: true,
		},
		{
			Key:      "automation.workflows",
			Value:    `{"enabled":true,"max_steps":50,"step_timeout":1800}`,
			Category: "automation",
			IsSystem: true,
		},
	}

	for _, config := range configs {
		if err := s.db.Create(&config).Error; err != nil {
			s.logger.Errorf("Failed to create default config %s: %v", config.Key, err)
			continue
		}
	}

	return status, nil
}

// getDefaultFeatureStatus 获取默认功能状态
func (s *Service) getDefaultFeatureStatus() *FeatureStatus {
	return &FeatureStatus{
		Automation: AutomationFeatures{
			Enabled: false, // 默认关闭
			Ansible: AnsibleFeatures{
				Enabled:    true,
				BinaryPath: "/usr/bin/ansible-playbook",
				TempDir:    "/tmp/ansible-runs",
				Timeout:    3600,
			},
			SSH: SSHFeatures{
				Enabled:            true,
				Timeout:            30,
				MaxConcurrent:      50,
				ConnectionPoolSize: 20,
			},
			Scripts: ScriptsFeatures{
				Enabled: true,
				Timeout: 600,
			},
			Workflows: WorkflowsFeatures{
				Enabled:     true,
				MaxSteps:    50,
				StepTimeout: 1800,
			},
		},
	}
}

// UpdateAutomationEnabled 更新自动化主开关
func (s *Service) UpdateAutomationEnabled(enabled bool, userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, _ := json.Marshal(enabled)

	var config model.AutomationConfig
	result := s.db.Where("key = ?", "automation.enabled").First(&config)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新配置
		config = model.AutomationConfig{
			Key:       "automation.enabled",
			Value:     string(value),
			Category:  "automation",
			IsSystem:  true,
			UpdatedBy: userID,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		// 更新现有配置
		config.Value = string(value)
		config.UpdatedBy = userID
		if err := s.db.Save(&config).Error; err != nil {
			return err
		}
	}

	// 清除缓存
	s.cache.mu.Lock()
	delete(s.cache.Features, "status")
	s.cache.mu.Unlock()

	s.logger.Infof("Automation feature %s by user %d", map[bool]string{true: "enabled", false: "disabled"}[enabled], userID)
	return nil
}

// UpdateAnsibleConfig 更新 Ansible 配置
func (s *Service) UpdateAnsibleConfig(config AnsibleFeatures, userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, _ := json.Marshal(config)

	var dbConfig model.AutomationConfig
	result := s.db.Where("key = ?", "automation.ansible").First(&dbConfig)

	if result.Error == gorm.ErrRecordNotFound {
		dbConfig = model.AutomationConfig{
			Key:       "automation.ansible",
			Value:     string(value),
			Category:  "automation",
			IsSystem:  true,
			UpdatedBy: userID,
		}
		if err := s.db.Create(&dbConfig).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		dbConfig.Value = string(value)
		dbConfig.UpdatedBy = userID
		if err := s.db.Save(&dbConfig).Error; err != nil {
			return err
		}
	}

	// 清除缓存
	s.cache.mu.Lock()
	delete(s.cache.Features, "status")
	s.cache.mu.Unlock()

	return nil
}

// UpdateSSHConfig 更新 SSH 配置
func (s *Service) UpdateSSHConfig(config SSHFeatures, userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, _ := json.Marshal(config)

	var dbConfig model.AutomationConfig
	result := s.db.Where("key = ?", "automation.ssh").First(&dbConfig)

	if result.Error == gorm.ErrRecordNotFound {
		dbConfig = model.AutomationConfig{
			Key:       "automation.ssh",
			Value:     string(value),
			Category:  "automation",
			IsSystem:  true,
			UpdatedBy: userID,
		}
		if err := s.db.Create(&dbConfig).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		dbConfig.Value = string(value)
		dbConfig.UpdatedBy = userID
		if err := s.db.Save(&dbConfig).Error; err != nil {
			return err
		}
	}

	// 清除缓存
	s.cache.mu.Lock()
	delete(s.cache.Features, "status")
	s.cache.mu.Unlock()

	return nil
}

// IsAutomationEnabled 检查自动化功能是否启用
func (s *Service) IsAutomationEnabled() bool {
	status, err := s.GetFeatureStatus()
	if err != nil {
		return false
	}
	return status.Automation.Enabled
}

// IsAnsibleEnabled 检查 Ansible 功能是否启用
func (s *Service) IsAnsibleEnabled() bool {
	status, err := s.GetFeatureStatus()
	if err != nil {
		return false
	}
	return status.Automation.Enabled && status.Automation.Ansible.Enabled
}

// IsSSHEnabled 检查 SSH 功能是否启用
func (s *Service) IsSSHEnabled() bool {
	status, err := s.GetFeatureStatus()
	if err != nil {
		return false
	}
	return status.Automation.Enabled && status.Automation.SSH.Enabled
}

// IsScriptsEnabled 检查脚本功能是否启用
func (s *Service) IsScriptsEnabled() bool {
	status, err := s.GetFeatureStatus()
	if err != nil {
		return false
	}
	return status.Automation.Enabled && status.Automation.Scripts.Enabled
}

// IsWorkflowsEnabled 检查工作流功能是否启用
func (s *Service) IsWorkflowsEnabled() bool {
	status, err := s.GetFeatureStatus()
	if err != nil {
		return false
	}
	return status.Automation.Enabled && status.Automation.Workflows.Enabled
}

// ReloadCache 重新加载缓存
func (s *Service) ReloadCache() error {
	s.cache.mu.Lock()
	delete(s.cache.Features, "status")
	s.cache.mu.Unlock()

	_, err := s.GetFeatureStatus()
	return err
}
