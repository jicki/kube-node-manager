package auth

import (
	"errors"
	"fmt"
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/ldap"
	"kube-node-manager/pkg/logger"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *logger.Logger
	jwtCfg config.JWTConfig
	ldap   *ldap.Service
	audit  *audit.Service
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token"`
	User         model.User `json:"user"`
	ExpiresAt    time.Time  `json:"expires_at"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type Claims struct {
	UserID   uint           `json:"user_id"`
	Username string         `json:"username"`
	Role     model.UserRole `json:"role"`
	Type     string         `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type UpdateProfileRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type ProfileStatsResponse struct {
	LoginCount     int `json:"loginCount"`
	OperationCount int `json:"operationCount"`
}

func NewService(db *gorm.DB, logger *logger.Logger, jwtCfg config.JWTConfig, ldap *ldap.Service, audit *audit.Service) *Service {
	return &Service{
		db:     db,
		logger: logger,
		jwtCfg: jwtCfg,
		ldap:   ldap,
		audit:  audit,
	}
}

func (s *Service) Login(req LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	var user model.User
	err := s.db.Where("username = ?", req.Username).First(&user).Error

	var isLDAPAuth bool
	if err != nil {
		if err == gorm.ErrRecordNotFound && s.ldap.IsEnabled() {
			s.logger.Infof("User %s not found locally, attempting LDAP authentication", req.Username)
			ldapUser, ldapErr := s.ldap.Authenticate(ldap.AuthRequest{
				Username: req.Username,
				Password: req.Password,
			})
			if ldapErr != nil {
				s.logger.Warningf("LDAP authentication failed for user %s: %v", req.Username, ldapErr)
				s.audit.Log(audit.LogRequest{
					Action:       model.ActionLogin,
					ResourceType: model.ResourceUser,
					Details:      fmt.Sprintf("Failed LDAP login attempt for username: %s - %s", req.Username, ldapErr.Error()),
					Status:       model.AuditStatusFailed,
					ErrorMsg:     "LDAP authentication failed",
					IPAddress:    ipAddress,
					UserAgent:    userAgent,
				})
				return nil, errors.New("invalid credentials")
			}

			s.logger.Infof("LDAP authentication successful for user %s, creating local user record", req.Username)
			user = model.User{
				Username: ldapUser.Username,
				Email:    ldapUser.Email,
				Role:     model.RoleUser,
				Status:   model.StatusActive,
			}
			user.HashPassword("") // LDAP users don't have local passwords
			if err := s.db.Create(&user).Error; err != nil {
				s.logger.Errorf("Failed to create local user record for LDAP user %s: %v", req.Username, err)
				return nil, err
			}
			s.logger.Infof("Local user record created for LDAP user %s (ID: %d)", req.Username, user.ID)
			isLDAPAuth = true
		} else {
			return nil, err
		}
	}

	if user.Status != model.StatusActive {
		s.audit.Log(audit.LogRequest{
			UserID:       user.ID,
			Action:       model.ActionLogin,
			ResourceType: model.ResourceUser,
			Details:      fmt.Sprintf("Login attempt for inactive user: %s", user.Username),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "Account is inactive",
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
		})
		return nil, errors.New("account is inactive")
	}

	if !isLDAPAuth {
		if !user.CheckPassword(req.Password) {
			s.audit.Log(audit.LogRequest{
				UserID:       user.ID,
				Action:       model.ActionLogin,
				ResourceType: model.ResourceUser,
				Details:      fmt.Sprintf("Failed login attempt for user: %s", user.Username),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     "Invalid password",
				IPAddress:    ipAddress,
				UserAgent:    userAgent,
			})
			return nil, errors.New("invalid credentials")
		}
	}

	expiresAt := time.Now().Add(time.Duration(s.jwtCfg.ExpireTime) * time.Second)

	token, err := s.generateToken(user.ID, user.Username, user.Role, "access", expiresAt)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(user.ID, user.Username, user.Role, "refresh", time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.db.Model(&user).Update("last_login", now)

	// 记录成功登录的审计日志
	loginDetails := fmt.Sprintf("Successful login for user: %s", user.Username)
	if isLDAPAuth {
		loginDetails = fmt.Sprintf("Successful LDAP login for user: %s", user.Username)
		s.logger.Infof("User %s successfully logged in via LDAP", user.Username)
	} else {
		s.logger.Infof("User %s successfully logged in via local authentication", user.Username)
	}

	s.audit.Log(audit.LogRequest{
		UserID:       user.ID,
		Action:       model.ActionLogin,
		ResourceType: model.ResourceUser,
		Details:      loginDetails,
		Status:       model.AuditStatusSuccess,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) RefreshToken(req RefreshTokenRequest) (*LoginResponse, error) {
	claims, err := s.validateToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	var user model.User
	if err := s.db.First(&user, claims.UserID).Error; err != nil {
		return nil, err
	}

	if user.Status != model.StatusActive {
		return nil, errors.New("account is inactive")
	}

	expiresAt := time.Now().Add(time.Duration(s.jwtCfg.ExpireTime) * time.Second)

	token, err := s.generateToken(user.ID, user.Username, user.Role, "access", expiresAt)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(user.ID, user.Username, user.Role, "refresh", time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString)
}

func (s *Service) generateToken(userID uint, username string, role model.UserRole, tokenType string, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.Secret))
}

func (s *Service) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtCfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetUserByID 根据ID获取用户信息
func (s *Service) GetUserByID(userID uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateProfile 更新用户个人信息
func (s *Service) UpdateProfile(userID uint, req UpdateProfileRequest) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	if req.Email != "" && req.Email != user.Email {
		// 检查邮箱是否已存在
		var existingUser model.User
		if err := s.db.Where("email = ? AND id != ?", req.Email, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Updated profile for user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return &user, nil
}

// ChangePassword 修改用户密码
func (s *Service) ChangePassword(userID uint, req ChangePasswordRequest) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if !user.CheckPassword(req.OldPassword) {
		s.audit.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceUser,
			Details:      fmt.Sprintf("Failed password change attempt for user: %s", user.Username),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "Incorrect current password",
		})
		return errors.New("current password is incorrect")
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		return err
	}

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	s.audit.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Password changed successfully for user: %s", user.Username),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// TestLDAPConnection 测试LDAP连接配置
func (s *Service) TestLDAPConnection() (*TestLDAPResponse, error) {
	response := &TestLDAPResponse{
		Enabled: s.ldap.IsEnabled(),
	}

	if !s.ldap.IsEnabled() {
		response.Status = "LDAP is disabled"
		response.Success = false
		s.logger.Info("LDAP connection test skipped - LDAP is disabled")
		return response, nil
	}

	if err := s.ldap.TestConnection(); err != nil {
		response.Status = fmt.Sprintf("Connection failed: %v", err)
		response.Success = false
		s.logger.Warningf("LDAP connection test failed: %v", err)
		return response, nil
	}

	response.Status = "Connection successful"
	response.Success = true
	s.logger.Info("LDAP connection test successful")

	return response, nil
}

// DiagnoseLDAP 诊断 LDAP 目录结构
func (s *Service) DiagnoseLDAP() error {
	if !s.ldap.IsEnabled() {
		return fmt.Errorf("LDAP is not enabled")
	}

	s.logger.Info("Starting LDAP directory diagnosis...")
	return s.ldap.DiagnoseDirectory()
}

// TestLDAPResponse LDAP连接测试响应
type TestLDAPResponse struct {
	Enabled bool   `json:"enabled"`
	Success bool   `json:"success"`
	Status  string `json:"status"`
}

// GetProfileStats 获取用户统计信息
func (s *Service) GetProfileStats(userID uint) (*ProfileStatsResponse, error) {
	stats := &ProfileStatsResponse{
		LoginCount:     0,
		OperationCount: 0,
	}

	// 统计登录次数 - 从审计日志中统计成功登录记录
	var loginCount int64
	if err := s.db.Model(&model.AuditLog{}).
		Where("user_id = ? AND action = ? AND resource_type = ? AND status = ?",
			userID, model.ActionLogin, model.ResourceUser, model.AuditStatusSuccess).
		Count(&loginCount).Error; err != nil {
		s.logger.Error("Failed to count login records: %v", err)
	}
	stats.LoginCount = int(loginCount)

	// 统计操作记录 - 从审计日志中统计所有操作记录
	var operationCount int64
	if err := s.db.Model(&model.AuditLog{}).
		Where("user_id = ? AND status = ?", userID, model.AuditStatusSuccess).
		Count(&operationCount).Error; err != nil {
		s.logger.Error("Failed to count operation records: %v", err)
	}
	stats.OperationCount = int(operationCount)

	return stats, nil
}
