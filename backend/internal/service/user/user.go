package user

import (
	"errors"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/pkg/logger"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *logger.Logger
	audit  *audit.Service
}

type ListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type ListResponse struct {
	Users    []model.User `json:"users"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type CreateRequest struct {
	Username string           `json:"username" binding:"required"`
	Email    string           `json:"email" binding:"required,email"`
	Password string           `json:"password" binding:"required,min=6"`
	Role     model.UserRole   `json:"role"`
	Status   model.UserStatus `json:"status"`
}

type UpdateRequest struct {
	Username string           `json:"username"`
	Email    string           `json:"email" binding:"omitempty,email"`
	Role     model.UserRole   `json:"role"`
	Status   model.UserStatus `json:"status"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordRequest 管理员重置用户密码请求
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

func NewService(db *gorm.DB, logger *logger.Logger, audit *audit.Service) *Service {
	return &Service{
		db:     db,
		logger: logger,
		audit:  audit,
	}
}

func (s *Service) List(req ListRequest) (*ListResponse, error) {
	query := s.db.Model(&model.User{})

	if req.Username != "" {
		query = query.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Email != "" {
		query = query.Where("email LIKE ?", "%"+req.Email+"%")
	}
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	var users []model.User
	if err := query.Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	return &ListResponse{
		Users:    users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *Service) GetByID(id uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) GetByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) Create(req CreateRequest, operatorID uint) (*model.User, error) {
	var existingUser model.User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("username or email already exists")
	}

	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		Status:   req.Status,
	}

	if user.Role == "" {
		user.Role = model.RoleUser
	}
	if user.Status == "" {
		user.Status = model.StatusActive
	}

	if err := user.HashPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       operatorID,
		Action:       model.ActionCreate,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Created user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return &user, nil
}

func (s *Service) Update(id uint, req UpdateRequest, operatorID uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}

	// 检查是否为 LDAP 用户
	if user.IsLDAPUser {
		// LDAP 用户只允许修改角色和状态
		if req.Username != "" && req.Username != user.Username {
			return nil, errors.New("cannot modify username for LDAP users. Username is managed by LDAP directory")
		}
		if req.Email != "" && req.Email != user.Email {
			return nil, errors.New("cannot modify email for LDAP users. Email is managed by LDAP directory")
		}

		// 只允许修改角色和状态
		if req.Role != "" {
			user.Role = req.Role
		}
		if req.Status != "" {
			user.Status = req.Status
		}
	} else {
		// 非 LDAP 用户允许修改所有字段
		if req.Username != "" && req.Username != user.Username {
			var existingUser model.User
			if err := s.db.Where("username = ? AND id != ?", req.Username, id).First(&existingUser).Error; err == nil {
				return nil, errors.New("username already exists")
			}
			user.Username = req.Username
		}

		if req.Email != "" && req.Email != user.Email {
			var existingUser model.User
			if err := s.db.Where("email = ? AND id != ?", req.Email, id).First(&existingUser).Error; err == nil {
				return nil, errors.New("email already exists")
			}
			user.Email = req.Email
		}

		if req.Role != "" {
			user.Role = req.Role
		}
		if req.Status != "" {
			user.Status = req.Status
		}
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	updateDetails := fmt.Sprintf("Updated user: %s", user.Username)
	if user.IsLDAPUser {
		updateDetails = fmt.Sprintf("Updated LDAP user (role/status only): %s", user.Username)
	}

	s.audit.Log(audit.LogRequest{
		UserID:       operatorID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceUser,
		Details:      updateDetails,
		Status:       model.AuditStatusSuccess,
	})

	return &user, nil
}

func (s *Service) Delete(id uint, operatorID uint) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return err
	}

	if user.ID == operatorID {
		return errors.New("cannot delete yourself")
	}

	// 检查是否为 LDAP 用户
	if user.IsLDAPUser {
		return errors.New("cannot delete LDAP users. LDAP users are managed through the LDAP directory")
	}

	if err := s.db.Delete(&user).Error; err != nil {
		return err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       operatorID,
		Action:       model.ActionDelete,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Deleted user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

func (s *Service) UpdatePassword(id uint, req UpdatePasswordRequest, operatorID uint) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return err
	}

	// 检查是否为 LDAP 用户
	if user.IsLDAPUser {
		return errors.New("LDAP users cannot change password locally. Please contact your LDAP administrator")
	}

	if !user.CheckPassword(req.CurrentPassword) {
		return errors.New("current password is incorrect")
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		return err
	}

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       operatorID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Updated password for user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// ResetPassword 管理员重置用户密码（不需要当前密码验证）
func (s *Service) ResetPassword(id uint, req ResetPasswordRequest, operatorID uint) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return err
	}

	// 检查是否为 LDAP 用户
	if user.IsLDAPUser {
		return errors.New("Cannot reset password for LDAP users. LDAP users are authenticated through LDAP directory")
	}

	// 直接设置新密码，不需要验证当前密码
	if err := user.HashPassword(req.Password); err != nil {
		return err
	}

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       operatorID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Reset password for user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

func (s *Service) UpdateLastLogin(id uint) error {
	now := time.Now()
	return s.db.Model(&model.User{}).Where("id = ?", id).Update("last_login", now).Error
}

// GetLDAPUserCount 获取 LDAP 用户数量统计
func (s *Service) GetLDAPUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&model.User{}).Where("is_ldap_user = ?", true).Count(&count).Error
	return count, err
}

// GetLocalUserCount 获取本地用户数量统计
func (s *Service) GetLocalUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&model.User{}).Where("is_ldap_user = ?", false).Count(&count).Error
	return count, err
}
