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
			ldapUser, ldapErr := s.ldap.Authenticate(req.Username, req.Password)
			if ldapErr != nil {
				s.audit.Log(audit.LogRequest{
					Action:       model.ActionLogin,
					ResourceType: model.ResourceUser,
					Details:      fmt.Sprintf("Failed login attempt for username: %s", req.Username),
					Status:       model.AuditStatusFailed,
					ErrorMsg:     "Invalid credentials",
					IPAddress:    ipAddress,
					UserAgent:    userAgent,
				})
				return nil, errors.New("invalid credentials")
			}
			
			user = model.User{
				Username: ldapUser.Username,
				Email:    ldapUser.Email,
				Role:     model.RoleUser,
				Status:   model.StatusActive,
			}
			user.HashPassword("") // LDAP users don't have local passwords
			if err := s.db.Create(&user).Error; err != nil {
				return nil, err
			}
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
	
	s.audit.Log(audit.LogRequest{
		UserID:       user.ID,
		Action:       model.ActionLogin,
		ResourceType: model.ResourceUser,
		Details:      fmt.Sprintf("Successful login for user: %s", user.Username),
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