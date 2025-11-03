package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// TagService 处理标签相关逻辑
type TagService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewTagService 创建 TagService 实例
func NewTagService(db *gorm.DB, logger *logger.Logger) *TagService {
	return &TagService{
		db:     db,
		logger: logger,
	}
}

// CreateTag 创建标签
func (s *TagService) CreateTag(req model.TagCreateRequest, userID uint) (*model.AnsibleTag, error) {
	// 检查标签名是否已存在
	var existing model.AnsibleTag
	if err := s.db.Where("name = ? AND user_id = ?", req.Name, userID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("tag with name '%s' already exists", req.Name)
	}

	// 设置默认颜色
	color := req.Color
	if color == "" {
		color = "#409EFF"
	}

	tag := &model.AnsibleTag{
		Name:        req.Name,
		Color:       color,
		Description: req.Description,
		UserID:      userID,
	}

	if err := s.db.Create(tag).Error; err != nil {
		s.logger.Errorf("Failed to create tag: %v", err)
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	s.logger.Infof("Created tag: %s (ID: %d) by user %d", tag.Name, tag.ID, userID)
	return tag, nil
}

// GetTag 获取标签详情
func (s *TagService) GetTag(id uint) (*model.AnsibleTag, error) {
	var tag model.AnsibleTag
	if err := s.db.Preload("User").First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return &tag, nil
}

// ListTags 获取标签列表
func (s *TagService) ListTags(req model.TagListRequest) ([]model.AnsibleTag, int64, error) {
	var tags []model.AnsibleTag
	var total int64

	query := s.db.Model(&model.AnsibleTag{}).Preload("User")

	// 关键字搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 计数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询
	if err := query.Order("created_at DESC").Find(&tags).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, total, nil
}

// UpdateTag 更新标签
func (s *TagService) UpdateTag(id uint, req model.TagUpdateRequest, userID uint) error {
	var tag model.AnsibleTag
	if err := s.db.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("tag not found")
		}
		return fmt.Errorf("failed to get tag: %w", err)
	}

	// 检查权限（只能编辑自己创建的标签）
	if tag.UserID != userID {
		return fmt.Errorf("permission denied: you can only edit your own tags")
	}

	// 如果更新了名称，检查是否与其他标签冲突
	if req.Name != "" && req.Name != tag.Name {
		var existing model.AnsibleTag
		if err := s.db.Where("name = ? AND user_id = ? AND id != ?", req.Name, userID, id).First(&existing).Error; err == nil {
			return fmt.Errorf("tag with name '%s' already exists", req.Name)
		}
		tag.Name = req.Name
	}

	if req.Color != "" {
		tag.Color = req.Color
	}

	if req.Description != "" {
		tag.Description = req.Description
	}

	if err := s.db.Save(&tag).Error; err != nil {
		s.logger.Errorf("Failed to update tag %d: %v", id, err)
		return fmt.Errorf("failed to update tag: %w", err)
	}

	s.logger.Infof("Updated tag %d by user %d", id, userID)
	return nil
}

// DeleteTag 删除标签
func (s *TagService) DeleteTag(id uint, userID uint) error {
	var tag model.AnsibleTag
	if err := s.db.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("tag not found")
		}
		return fmt.Errorf("failed to get tag: %w", err)
	}

	// 检查权限（只能删除自己创建的标签）
	if tag.UserID != userID {
		return fmt.Errorf("permission denied: you can only delete your own tags")
	}

	// 软删除标签
	if err := s.db.Delete(&tag).Error; err != nil {
		s.logger.Errorf("Failed to delete tag %d: %v", id, err)
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	s.logger.Infof("Deleted tag %d by user %d", id, userID)
	return nil
}

// AddTagsToTask 为任务添加标签
func (s *TagService) AddTagsToTask(taskID uint, tagIDs []uint) error {
	var task model.AnsibleTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 验证所有标签是否存在
	var tags []model.AnsibleTag
	if err := s.db.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	if len(tags) != len(tagIDs) {
		return fmt.Errorf("some tags not found")
	}

	// 使用事务添加标签
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, tagID := range tagIDs {
			taskTag := model.AnsibleTaskTag{
				TaskID: taskID,
				TagID:  tagID,
			}
			// 如果关联已存在，忽略错误（可能是唯一约束冲突）
			if err := tx.Create(&taskTag).Error; err != nil {
				s.logger.Infof("Tag %d may already exist for task %d, skipping: %v", tagID, taskID, err)
				// 继续处理下一个标签
			}
		}
		return nil
	})
}

// RemoveTagsFromTask 从任务移除标签
func (s *TagService) RemoveTagsFromTask(taskID uint, tagIDs []uint) error {
	var task model.AnsibleTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 删除关联
	if err := s.db.Where("task_id = ? AND tag_id IN ?", taskID, tagIDs).
		Delete(&model.AnsibleTaskTag{}).Error; err != nil {
		s.logger.Errorf("Failed to remove tags from task %d: %v", taskID, err)
		return fmt.Errorf("failed to remove tags: %w", err)
	}

	s.logger.Infof("Removed tags %v from task %d", tagIDs, taskID)
	return nil
}

// BatchAddTagsToTasks 批量为任务添加标签
func (s *TagService) BatchAddTagsToTasks(taskIDs []uint, tagIDs []uint) error {
	// 验证任务是否存在
	var taskCount int64
	if err := s.db.Model(&model.AnsibleTask{}).Where("id IN ?", taskIDs).Count(&taskCount).Error; err != nil {
		return fmt.Errorf("failed to count tasks: %w", err)
	}
	if int(taskCount) != len(taskIDs) {
		return fmt.Errorf("some tasks not found")
	}

	// 验证标签是否存在
	var tagCount int64
	if err := s.db.Model(&model.AnsibleTag{}).Where("id IN ?", tagIDs).Count(&tagCount).Error; err != nil {
		return fmt.Errorf("failed to count tags: %w", err)
	}
	if int(tagCount) != len(tagIDs) {
		return fmt.Errorf("some tags not found")
	}

	// 批量添加关联
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, taskID := range taskIDs {
			for _, tagID := range tagIDs {
				taskTag := model.AnsibleTaskTag{
					TaskID: taskID,
					TagID:  tagID,
				}
				// 如果关联已存在，忽略错误
				if err := tx.Create(&taskTag).Error; err != nil {
					// 继续处理下一个
					s.logger.Infof("Tag %d may already exist for task %d, skipping: %v", tagID, taskID, err)
				}
			}
		}
		s.logger.Infof("Batch added tags %v to tasks %v", tagIDs, taskIDs)
		return nil
	})
}

// BatchRemoveTagsFromTasks 批量从任务移除标签
func (s *TagService) BatchRemoveTagsFromTasks(taskIDs []uint, tagIDs []uint) error {
	if err := s.db.Where("task_id IN ? AND tag_id IN ?", taskIDs, tagIDs).
		Delete(&model.AnsibleTaskTag{}).Error; err != nil {
		s.logger.Errorf("Failed to batch remove tags: %v", err)
		return fmt.Errorf("failed to remove tags: %w", err)
	}

	s.logger.Infof("Batch removed tags %v from tasks %v", tagIDs, taskIDs)
	return nil
}

// GetTaskTags 获取任务的所有标签
func (s *TagService) GetTaskTags(taskID uint) ([]model.AnsibleTag, error) {
	var tags []model.AnsibleTag
	if err := s.db.Joins("JOIN ansible_task_tags ON ansible_task_tags.tag_id = ansible_tags.id").
		Where("ansible_task_tags.task_id = ?", taskID).
		Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to get task tags: %w", err)
	}
	return tags, nil
}

// GetTagStats 获取标签统计信息
func (s *TagService) GetTagStats(tagID uint) (map[string]interface{}, error) {
	var tag model.AnsibleTag
	if err := s.db.First(&tag, tagID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	stats := make(map[string]interface{})
	stats["tag_id"] = tagID
	stats["tag_name"] = tag.Name

	// 统计使用该标签的任务总数
	var totalTasks int64
	if err := s.db.Model(&model.AnsibleTaskTag{}).Where("tag_id = ?", tagID).Count(&totalTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}
	stats["total_tasks"] = totalTasks

	// 按状态统计任务数
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := s.db.Model(&model.AnsibleTask{}).
		Select("status, COUNT(*) as count").
		Joins("JOIN ansible_task_tags ON ansible_task_tags.task_id = ansible_tasks.id").
		Where("ansible_task_tags.tag_id = ?", tagID).
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by status: %w", err)
	}

	statusMap := make(map[string]int64)
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
	}
	stats["by_status"] = statusMap

	return stats, nil
}

