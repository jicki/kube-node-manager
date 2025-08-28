package ldap

import (
	"crypto/tls"
	"fmt"
	"kube-node-manager/internal/config"
	"kube-node-manager/pkg/logger"

	"github.com/go-ldap/ldap/v3"
)

// Service LDAP认证服务
type Service struct {
	logger *logger.Logger
	config config.LDAPConfig
}

// UserInfo LDAP用户信息
type UserInfo struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Groups      []string `json:"groups"`
}

// AuthRequest LDAP认证请求
type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// NewService creates a new LDAP service instance
func NewService(logger *logger.Logger, cfg config.LDAPConfig) *Service {
	return &Service{
		logger: logger,
		config: cfg,
	}
}

// IsEnabled 检查LDAP是否启用
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// Authenticate 验证LDAP用户身份
func (s *Service) Authenticate(req AuthRequest) (*UserInfo, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("LDAP authentication is not enabled")
	}

	// 连接到LDAP服务器
	conn, err := s.connect()
	if err != nil {
		s.logger.Error("Failed to connect to LDAP server", err)
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// 使用管理员账号绑定
	if err := s.bindAdmin(conn); err != nil {
		s.logger.Error("Failed to bind admin user", err)
		return nil, fmt.Errorf("failed to bind admin user: %w", err)
	}

	// 搜索用户
	userDN, userInfo, err := s.searchUser(conn, req.Username)
	if err != nil {
		s.logger.Error("Failed to search user", err)
		return nil, fmt.Errorf("failed to search user: %w", err)
	}

	if userDN == "" {
		return nil, fmt.Errorf("user not found: %s", req.Username)
	}

	// 验证用户密码
	if err := s.authenticateUser(conn, userDN, req.Password); err != nil {
		s.logger.Error("Failed to authenticate user", err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	s.logger.Info("LDAP authentication successful for user: %s", req.Username)
	return userInfo, nil
}

// connect 连接到LDAP服务器
func (s *Service) connect() (*ldap.Conn, error) {
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	var conn *ldap.Conn
	var err error
	
	if s.config.Port == 636 {
		// LDAPS连接
		conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			ServerName: s.config.Host,
		})
	} else {
		// LDAP连接
		conn, err = ldap.Dial("tcp", address)
		if err == nil && s.config.Port != 389 {
			// 尝试启动TLS
			if err := conn.StartTLS(&tls.Config{
				ServerName: s.config.Host,
			}); err != nil {
				s.logger.Warn("Failed to start TLS, continuing with plain connection: %v", err)
			}
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}
	
	return conn, nil
}

// bindAdmin 使用管理员账号绑定
func (s *Service) bindAdmin(conn *ldap.Conn) error {
	if s.config.AdminDN == "" || s.config.AdminPass == "" {
		return fmt.Errorf("admin DN and password are required")
	}
	
	if err := conn.Bind(s.config.AdminDN, s.config.AdminPass); err != nil {
		return fmt.Errorf("failed to bind admin: %w", err)
	}
	
	return nil
}

// searchUser 搜索用户
func (s *Service) searchUser(conn *ldap.Conn, username string) (string, *UserInfo, error) {
	// 构建搜索过滤器
	filter := fmt.Sprintf(s.config.UserFilter, username)
	if s.config.UserFilter == "" {
		filter = fmt.Sprintf("(uid=%s)", username)
	}
	
	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // 只需要一个结果
		0,
		false,
		filter,
		[]string{"dn", "uid", "cn", "mail", "displayName", "memberOf"},
		nil,
	)
	
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return "", nil, fmt.Errorf("search failed: %w", err)
	}
	
	if len(sr.Entries) == 0 {
		return "", nil, nil
	}
	
	entry := sr.Entries[0]
	userInfo := &UserInfo{
		Username:    entry.GetAttributeValue("uid"),
		Email:       entry.GetAttributeValue("mail"),
		DisplayName: entry.GetAttributeValue("displayName"),
		Groups:      entry.GetAttributeValues("memberOf"),
	}
	
	// 如果没有displayName，使用cn
	if userInfo.DisplayName == "" {
		userInfo.DisplayName = entry.GetAttributeValue("cn")
	}
	
	// 如果没有uid，使用用户名
	if userInfo.Username == "" {
		userInfo.Username = username
	}
	
	return entry.DN, userInfo, nil
}

// authenticateUser 验证用户密码
func (s *Service) authenticateUser(conn *ldap.Conn, userDN, password string) error {
	// 创建新的连接来验证用户
	userConn, err := s.connect()
	if err != nil {
		return fmt.Errorf("failed to create user connection: %w", err)
	}
	defer userConn.Close()
	
	// 尝试使用用户凭据绑定
	if err := userConn.Bind(userDN, password); err != nil {
		return fmt.Errorf("bind failed: %w", err)
	}
	
	return nil
}

// TestConnection 测试LDAP连接
func (s *Service) TestConnection() error {
	if !s.config.Enabled {
		return fmt.Errorf("LDAP is not enabled")
	}
	
	conn, err := s.connect()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()
	
	if err := s.bindAdmin(conn); err != nil {
		return fmt.Errorf("admin bind failed: %w", err)
	}
	
	return nil
}

// GetUserGroups 获取用户组信息
func (s *Service) GetUserGroups(username string) ([]string, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}
	
	conn, err := s.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	
	if err := s.bindAdmin(conn); err != nil {
		return nil, fmt.Errorf("failed to bind admin: %w", err)
	}
	
	_, userInfo, err := s.searchUser(conn, username)
	if err != nil {
		return nil, fmt.Errorf("failed to search user: %w", err)
	}
	
	if userInfo == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	return userInfo.Groups, nil
}

// SearchUsers 搜索多个用户
func (s *Service) SearchUsers(filter string, limit int) ([]*UserInfo, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}
	
	conn, err := s.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	
	if err := s.bindAdmin(conn); err != nil {
		return nil, fmt.Errorf("failed to bind admin: %w", err)
	}
	
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	
	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		limit,
		0,
		false,
		filter,
		[]string{"dn", "uid", "cn", "mail", "displayName", "memberOf"},
		nil,
	)
	
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	var users []*UserInfo
	for _, entry := range sr.Entries {
		userInfo := &UserInfo{
			Username:    entry.GetAttributeValue("uid"),
			Email:       entry.GetAttributeValue("mail"),
			DisplayName: entry.GetAttributeValue("displayName"),
			Groups:      entry.GetAttributeValues("memberOf"),
		}
		
		if userInfo.DisplayName == "" {
			userInfo.DisplayName = entry.GetAttributeValue("cn")
		}
		
		users = append(users, userInfo)
	}
	
	return users, nil
}