package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// TemplateService 任务模板服务
type TemplateService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewTemplateService 创建模板服务实例
func NewTemplateService(db *gorm.DB, logger *logger.Logger) *TemplateService {
	return &TemplateService{
		db:     db,
		logger: logger,
	}
}

// ListTemplates 列出任务模板
func (s *TemplateService) ListTemplates(req model.TemplateListRequest) ([]model.AnsibleTemplate, int64, error) {
	var templates []model.AnsibleTemplate
	var total int64

	query := s.db.Model(&model.AnsibleTemplate{})

	// 过滤条件
	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR tags LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorf("Failed to count templates: %v", err)
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询数据（包含关联）
	if err := query.Preload("User").Order("created_at DESC").Find(&templates).Error; err != nil {
		s.logger.Errorf("Failed to list templates: %v", err)
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// GetTemplate 获取模板详情
func (s *TemplateService) GetTemplate(id uint) (*model.AnsibleTemplate, error) {
	var template model.AnsibleTemplate

	if err := s.db.Preload("User").First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		s.logger.Errorf("Failed to get template %d: %v", id, err)
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(req model.TemplateCreateRequest, userID uint) (*model.AnsibleTemplate, error) {
	// 检查名称是否重复
	var count int64
	if err := s.db.Model(&model.AnsibleTemplate{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		s.logger.Errorf("Failed to check template name: %v", err)
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}

	if count > 0 {
		return nil, fmt.Errorf("template name already exists")
	}

	// 验证 playbook 内容
	if err := s.ValidatePlaybook(req.PlaybookContent); err != nil {
		return nil, fmt.Errorf("invalid playbook: %w", err)
	}

	template := &model.AnsibleTemplate{
		Name:            req.Name,
		Description:     req.Description,
		PlaybookContent: req.PlaybookContent,
		Variables:       req.Variables,
		Tags:            req.Tags,
		UserID:          userID,
	}

	if err := s.db.Create(template).Error; err != nil {
		s.logger.Errorf("Failed to create template: %v", err)
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.Infof("Successfully created template: %s (ID: %d) by user %d", template.Name, template.ID, userID)
	return template, nil
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(id uint, req model.TemplateUpdateRequest, userID uint) (*model.AnsibleTemplate, error) {
	var template model.AnsibleTemplate

	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 检查名称是否与其他记录重复
	if req.Name != "" && req.Name != template.Name {
		var count int64
		if err := s.db.Model(&model.AnsibleTemplate{}).
			Where("name = ? AND id != ?", req.Name, id).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to check template name: %w", err)
		}

		if count > 0 {
			return nil, fmt.Errorf("template name already exists")
		}

		template.Name = req.Name
	}

	// 更新字段
	if req.Description != "" {
		template.Description = req.Description
	}

	if req.PlaybookContent != "" {
		// 验证 playbook 内容
		if err := s.ValidatePlaybook(req.PlaybookContent); err != nil {
			return nil, fmt.Errorf("invalid playbook: %w", err)
		}
		template.PlaybookContent = req.PlaybookContent
	}

	if req.Variables != nil {
		template.Variables = req.Variables
	}

	if req.Tags != "" {
		template.Tags = req.Tags
	}

	if err := s.db.Save(&template).Error; err != nil {
		s.logger.Errorf("Failed to update template %d: %v", id, err)
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	s.logger.Infof("Successfully updated template: %s (ID: %d) by user %d", template.Name, template.ID, userID)
	return &template, nil
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(id uint, userID uint) error {
	var template model.AnsibleTemplate

	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 检查是否有关联的任务（排除已软删除的任务）
	var taskCount int64
	if err := s.db.Model(&model.AnsibleTask{}).Where("template_id = ? AND deleted_at IS NULL", id).Count(&taskCount).Error; err != nil {
		return fmt.Errorf("failed to check related tasks: %w", err)
	}

	if taskCount > 0 {
		return fmt.Errorf("cannot delete template: %d tasks are using this template", taskCount)
	}

	if err := s.db.Delete(&template).Error; err != nil {
		s.logger.Errorf("Failed to delete template %d: %v", id, err)
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Infof("Successfully deleted template: %s (ID: %d) by user %d", template.Name, template.ID, userID)
	return nil
}

// ValidatePlaybook 验证 playbook 语法
func (s *TemplateService) ValidatePlaybook(content string) error {
	if content == "" {
		return fmt.Errorf("playbook content cannot be empty")
	}

	// 尝试解析为 YAML
	var playbook interface{}
	if err := yaml.Unmarshal([]byte(content), &playbook); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// 检查是否为数组格式（Ansible playbook 应该是一个数组）
	playbookSlice, ok := playbook.([]interface{})
	if !ok {
		return fmt.Errorf("playbook must be an array of plays")
	}

	if len(playbookSlice) == 0 {
		return fmt.Errorf("playbook must contain at least one play")
	}

	// 验证每个 play
	for i, play := range playbookSlice {
		playMap, ok := play.(map[string]interface{})
		if !ok {
			return fmt.Errorf("play %d must be a dictionary", i+1)
		}

		// 检查必需字段
		if _, hasName := playMap["name"]; !hasName {
			s.logger.Warningf("Play %d has no name field (optional but recommended)", i+1)
		}

		// 检查是否有 hosts 或 import_playbook
		hasHosts := false
		hasImport := false
		for key := range playMap {
			if key == "hosts" {
				hasHosts = true
			}
			if key == "import_playbook" || key == "include" {
				hasImport = true
			}
		}

		if !hasHosts && !hasImport {
			return fmt.Errorf("play %d must have 'hosts' or 'import_playbook' field", i+1)
		}

		// 检查是否有任务或角色
		hasTasks := false
		for key := range playMap {
			if key == "tasks" || key == "roles" || key == "pre_tasks" || key == "post_tasks" || key == "import_playbook" {
				hasTasks = true
				break
			}
		}

		if hasHosts && !hasTasks {
			s.logger.Warningf("Play %d has no tasks or roles (optional but recommended)", i+1)
		}
	}

	// 检查是否包含危险命令
	if err := s.checkDangerousCommands(content); err != nil {
		return err
	}

	return nil
}

// checkDangerousCommands 检查危险命令
func (s *TemplateService) checkDangerousCommands(content string) error {
	// 定义危险命令列表
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=/dev/zero",
		":(){ :|:& };:",  // fork bomb
		"> /dev/sda",
		"format c:",
	}

	contentLower := strings.ToLower(content)

	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return fmt.Errorf("playbook contains potentially dangerous command: %s", pattern)
		}
	}

	return nil
}

// GetTemplateByName 根据名称获取模板
func (s *TemplateService) GetTemplateByName(name string) (*model.AnsibleTemplate, error) {
	var template model.AnsibleTemplate

	if err := s.db.Where("name = ?", name).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// ValidateTemplateVariables 验证模板变量
func (s *TemplateService) ValidateTemplateVariables(templateID uint, providedVars map[string]interface{}) error {
	template, err := s.GetTemplate(templateID)
	if err != nil {
		return err
	}

	// 如果模板没有定义变量，则不需要验证
	if len(template.Variables) == 0 {
		return nil
	}

	// 检查必需的变量是否都提供了
	for varName, varDef := range template.Variables {
		varDefMap, ok := varDef.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查是否为必需变量
		if required, ok := varDefMap["required"].(bool); ok && required {
			if _, provided := providedVars[varName]; !provided {
				return fmt.Errorf("required variable '%s' is not provided", varName)
			}
		}

		// 检查变量类型（可选）
		if expectedType, ok := varDefMap["type"].(string); ok {
			if providedValue, provided := providedVars[varName]; provided {
				if !s.checkVariableType(providedValue, expectedType) {
					return fmt.Errorf("variable '%s' has incorrect type (expected: %s)", varName, expectedType)
				}
			}
		}
	}

	return nil
}

// checkVariableType 检查变量类型
func (s *TemplateService) checkVariableType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "int", "integer":
		_, ok := value.(int)
		if !ok {
			_, ok = value.(float64) // JSON 数字通常解析为 float64
		}
		return ok
	case "bool", "boolean":
		_, ok := value.(bool)
		return ok
	case "array", "list":
		_, ok := value.([]interface{})
		return ok
	case "object", "dict", "map":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		// 未知类型，不验证
		return true
	}
}

