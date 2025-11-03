package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// FavoriteService 收藏服务
type FavoriteService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewFavoriteService 创建收藏服务实例
func NewFavoriteService(db *gorm.DB, logger *logger.Logger) *FavoriteService {
	return &FavoriteService{
		db:     db,
		logger: logger,
	}
}

// AddFavorite 添加收藏
func (s *FavoriteService) AddFavorite(userID uint, targetType string, targetID uint) error {
	// 检查是否已经收藏
	var count int64
	if err := s.db.Model(&model.AnsibleFavorite{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check favorite: %w", err)
	}
	
	if count > 0 {
		return fmt.Errorf("already in favorites")
	}
	
	// 创建收藏记录
	favorite := &model.AnsibleFavorite{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}
	
	if err := s.db.Create(favorite).Error; err != nil {
		s.logger.Errorf("Failed to add favorite: %v", err)
		return fmt.Errorf("failed to add favorite: %w", err)
	}
	
	s.logger.Infof("User %d added favorite: %s/%d", userID, targetType, targetID)
	return nil
}

// RemoveFavorite 移除收藏
func (s *FavoriteService) RemoveFavorite(userID uint, targetType string, targetID uint) error {
	result := s.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&model.AnsibleFavorite{})
	
	if result.Error != nil {
		s.logger.Errorf("Failed to remove favorite: %v", result.Error)
		return fmt.Errorf("failed to remove favorite: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("favorite not found")
	}
	
	s.logger.Infof("User %d removed favorite: %s/%d", userID, targetType, targetID)
	return nil
}

// ListFavorites 列出用户的收藏
func (s *FavoriteService) ListFavorites(userID uint, targetType string) ([]model.AnsibleFavorite, error) {
	var favorites []model.AnsibleFavorite
	
	query := s.db.Where("user_id = ?", userID)
	
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	
	// 根据类型预加载关联数据
	if targetType == "task" {
		query = query.Preload("Task")
	} else if targetType == "template" {
		query = query.Preload("Template")
	} else if targetType == "inventory" {
		query = query.Preload("Inventory")
	} else {
		// 全部加载
		query = query.Preload("Task").Preload("Template").Preload("Inventory")
	}
	
	if err := query.Order("created_at DESC").Find(&favorites).Error; err != nil {
		s.logger.Errorf("Failed to list favorites: %v", err)
		return nil, fmt.Errorf("failed to list favorites: %w", err)
	}
	
	return favorites, nil
}

// IsFavorite 检查是否已收藏
func (s *FavoriteService) IsFavorite(userID uint, targetType string, targetID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&model.AnsibleFavorite{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error; err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// AddOrUpdateTaskHistory 添加或更新任务历史
func (s *FavoriteService) AddOrUpdateTaskHistory(userID uint, task *model.AnsibleTask) error {
	// 查找是否存在相同配置的历史记录
	var history model.AnsibleTaskHistory
	err := s.db.Where("user_id = ? AND template_id = ? AND inventory_id = ? AND cluster_id = ?", 
		userID, task.TemplateID, task.InventoryID, task.ClusterID).
		First(&history).Error
	
	now := time.Now()
	
	if err == gorm.ErrRecordNotFound {
		// 创建新的历史记录
		history = model.AnsibleTaskHistory{
			UserID:          userID,
			TaskName:        task.Name,
			TemplateID:      task.TemplateID,
			InventoryID:     task.InventoryID,
			ClusterID:       task.ClusterID,
			PlaybookContent: task.PlaybookContent,
			ExtraVars:       task.ExtraVars,
			DryRun:          task.DryRun,
			BatchConfig:     task.BatchConfig,
			LastUsedAt:      now,
			UseCount:        1,
		}
		
		if err := s.db.Create(&history).Error; err != nil {
			s.logger.Errorf("Failed to create task history: %v", err)
			return fmt.Errorf("failed to create task history: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query task history: %w", err)
	} else {
		// 更新现有记录
		history.TaskName = task.Name
		history.PlaybookContent = task.PlaybookContent
		history.ExtraVars = task.ExtraVars
		history.DryRun = task.DryRun
		history.BatchConfig = task.BatchConfig
		history.LastUsedAt = now
		history.UseCount++
		
		if err := s.db.Save(&history).Error; err != nil {
			s.logger.Errorf("Failed to update task history: %v", err)
			return fmt.Errorf("failed to update task history: %w", err)
		}
	}
	
	s.logger.Debugf("Updated task history for user %d", userID)
	return nil
}

// GetRecentTaskHistory 获取最近使用的任务历史
func (s *FavoriteService) GetRecentTaskHistory(userID uint, limit int) ([]model.AnsibleTaskHistory, error) {
	var history []model.AnsibleTaskHistory
	
	if limit <= 0 {
		limit = 10
	}
	
	if err := s.db.Where("user_id = ?", userID).
		Preload("Template").
		Preload("Inventory").
		Preload("Cluster").
		Order("last_used_at DESC").
		Limit(limit).
		Find(&history).Error; err != nil {
		s.logger.Errorf("Failed to get recent task history: %v", err)
		return nil, fmt.Errorf("failed to get recent task history: %w", err)
	}
	
	return history, nil
}

// GetTaskHistory 获取指定的任务历史
func (s *FavoriteService) GetTaskHistory(id uint, userID uint) (*model.AnsibleTaskHistory, error) {
	var history model.AnsibleTaskHistory
	
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).
		Preload("Template").
		Preload("Inventory").
		Preload("Cluster").
		First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task history not found")
		}
		return nil, fmt.Errorf("failed to get task history: %w", err)
	}
	
	return &history, nil
}

// DeleteTaskHistory 删除任务历史
func (s *FavoriteService) DeleteTaskHistory(id uint, userID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.AnsibleTaskHistory{})
	
	if result.Error != nil {
		s.logger.Errorf("Failed to delete task history: %v", result.Error)
		return fmt.Errorf("failed to delete task history: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("task history not found")
	}
	
	s.logger.Infof("User %d deleted task history %d", userID, id)
	return nil
}

// CleanupOldHistory 清理旧的历史记录（保留最近N条）
func (s *FavoriteService) CleanupOldHistory(userID uint, keepCount int) error {
	if keepCount <= 0 {
		keepCount = 50 // 默认保留50条
	}
	
	// 查找要保留的记录ID
	var keepIDs []uint
	if err := s.db.Model(&model.AnsibleTaskHistory{}).
		Where("user_id = ?", userID).
		Order("last_used_at DESC").
		Limit(keepCount).
		Pluck("id", &keepIDs).Error; err != nil {
		return fmt.Errorf("failed to get keep ids: %w", err)
	}
	
	if len(keepIDs) == 0 {
		return nil
	}
	
	// 删除不在保留列表中的记录
	result := s.db.Where("user_id = ? AND id NOT IN ?", userID, keepIDs).
		Delete(&model.AnsibleTaskHistory{})
	
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old history: %w", result.Error)
	}
	
	if result.RowsAffected > 0 {
		s.logger.Infof("Cleaned up %d old task history records for user %d", result.RowsAffected, userID)
	}
	
	return nil
}

