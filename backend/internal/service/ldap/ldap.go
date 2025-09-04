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
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Groups      []string `json:"groups"`
}

// AuthRequest LDAP认证请求
type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// NewService creates a new LDAP service instance
func NewService(logger *logger.Logger, cfg config.LDAPConfig) *Service {
	service := &Service{
		logger: logger,
		config: cfg,
	}

	// 记录LDAP服务初始化状态
	if cfg.Enabled {
		logger.Infof("LDAP authentication is ENABLED - Server: %s:%d, Base DN: %s", cfg.Host, cfg.Port, cfg.BaseDN)

		// 验证配置
		if err := service.validateConfig(); err != nil {
			logger.Errorf("LDAP configuration validation failed: %v", err)
			logger.Warning("LDAP authentication may not work properly due to configuration errors")
		} else {
			logger.Info("LDAP configuration validation passed")
		}

		// 测试连接（不阻塞启动）
		go func() {
			if err := service.TestConnection(); err != nil {
				logger.Warningf("LDAP connection test failed: %v", err)
			} else {
				logger.Info("LDAP connection test successful")
			}
		}()
	} else {
		logger.Info("LDAP authentication is DISABLED - using local authentication only")
	}

	return service
}

// IsEnabled 检查LDAP是否启用
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// validateConfig 校验LDAP配置的完整性
func (s *Service) validateConfig() error {
	if s.config.Host == "" {
		return fmt.Errorf("LDAP host is required")
	}

	if s.config.Port <= 0 {
		return fmt.Errorf("LDAP port must be greater than 0")
	}

	if s.config.BaseDN == "" {
		return fmt.Errorf("LDAP base DN is required")
	}

	if s.config.AdminDN == "" {
		return fmt.Errorf("LDAP admin DN is required")
	}

	if s.config.AdminPass == "" {
		return fmt.Errorf("LDAP admin password is required")
	}

	// 设置默认的用户过滤器
	if s.config.UserFilter == "" {
		s.logger.Info("Using default LDAP user filter: (uid=%s)")
	}

	return nil
}

// Authenticate 验证LDAP用户身份
func (s *Service) Authenticate(req AuthRequest) (*UserInfo, error) {
	// 记录LDAP认证尝试
	s.logger.Infof("LDAP authentication attempt for user: %s", req.Username)

	if !s.config.Enabled {
		s.logger.Warningf("LDAP authentication attempted but LDAP is disabled for user: %s", req.Username)
		return nil, fmt.Errorf("LDAP authentication is not enabled")
	}

	// 校验LDAP配置完整性
	if err := s.validateConfig(); err != nil {
		s.logger.Errorf("LDAP configuration validation failed: %v", err)
		return nil, fmt.Errorf("LDAP configuration error: %w", err)
	}

	// 连接到LDAP服务器
	s.logger.Infof("Connecting to LDAP server: %s:%d", s.config.Host, s.config.Port)
	conn, err := s.connect()
	if err != nil {
		s.logger.Errorf("Failed to connect to LDAP server %s:%d for user %s: %v", s.config.Host, s.config.Port, req.Username, err)
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// 使用管理员账号绑定
	s.logger.Infof("Binding with admin DN: %s", s.config.AdminDN)
	if err := s.bindAdmin(conn); err != nil {
		s.logger.Errorf("Failed to bind admin user for authentication of %s: %v", req.Username, err)
		return nil, fmt.Errorf("failed to bind admin user: %w", err)
	}

	// 搜索用户
	s.logger.Infof("Searching for user %s in base DN: %s", req.Username, s.config.BaseDN)
	userDN, userInfo, err := s.searchUser(conn, req.Username)
	if err != nil {
		s.logger.Errorf("Failed to search user %s: %v", req.Username, err)
		return nil, fmt.Errorf("failed to search user: %w", err)
	}

	if userDN == "" {
		s.logger.Warningf("User not found in LDAP directory: %s", req.Username)
		return nil, fmt.Errorf("user not found: %s", req.Username)
	}

	s.logger.Infof("Found user %s with DN: %s", req.Username, userDN)

	// 验证用户密码
	if err := s.authenticateUser(conn, userDN, req.Password); err != nil {
		s.logger.Errorf("Failed to authenticate user %s with DN %s: %v", req.Username, userDN, err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	s.logger.Infof("LDAP authentication successful for user: %s (DN: %s, Email: %s)", req.Username, userDN, userInfo.Email)
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
				s.logger.Warning("Failed to start TLS, continuing with plain connection: %v", err)
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

	s.logger.Infof("LDAP search filter: %s", filter)
	s.logger.Infof("LDAP search base DN: %s", s.config.BaseDN)

	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		10, // 增加返回结果数量用于调试
		0,
		false,
		filter,
		[]string{"dn", "uid", "cn", "mail", "displayName", "memberOf", "sAMAccountName", "userPrincipalName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		s.logger.Errorf("LDAP search error: %v", err)
		return "", nil, fmt.Errorf("search failed: %w", err)
	}

	s.logger.Infof("LDAP search returned %d entries", len(sr.Entries))

	// 如果没有找到用户，尝试不同的过滤器进行诊断
	if len(sr.Entries) == 0 {
		s.logger.Infof("No entries found with filter %s, trying alternative filters for diagnosis...", filter)
		s.tryAlternativeFilters(conn, username)
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

// tryAlternativeFilters 尝试不同的过滤器进行诊断
func (s *Service) tryAlternativeFilters(conn *ldap.Conn, username string) {
	// 常见的用户过滤器
	alternativeFilters := []string{
		"(sAMAccountName=%s)",    // Active Directory
		"(userPrincipalName=%s)", // Active Directory UPN
		"(cn=%s)",                // Common Name
		"(mail=%s)",              // Email address
		"(displayName=%s)",       // Display Name
		"(name=%s)",              // Name attribute
	}

	for _, filterTemplate := range alternativeFilters {
		filter := fmt.Sprintf(filterTemplate, username)
		s.logger.Infof("Trying alternative filter: %s", filter)

		searchRequest := ldap.NewSearchRequest(
			s.config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			5,
			0,
			false,
			filter,
			[]string{"dn", "uid", "cn", "sAMAccountName", "userPrincipalName", "mail", "displayName"},
			nil,
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			s.logger.Warningf("Alternative filter %s failed: %v", filter, err)
			continue
		}

		if len(sr.Entries) > 0 {
			s.logger.Infof("SUCCESS: Found %d entries with filter %s", len(sr.Entries), filter)
			for i, entry := range sr.Entries {
				s.logger.Infof("Entry %d: DN=%s", i+1, entry.DN)
				s.logger.Infof("  uid: %s", entry.GetAttributeValue("uid"))
				s.logger.Infof("  cn: %s", entry.GetAttributeValue("cn"))
				s.logger.Infof("  sAMAccountName: %s", entry.GetAttributeValue("sAMAccountName"))
				s.logger.Infof("  userPrincipalName: %s", entry.GetAttributeValue("userPrincipalName"))
				s.logger.Infof("  mail: %s", entry.GetAttributeValue("mail"))
				s.logger.Infof("  displayName: %s", entry.GetAttributeValue("displayName"))
			}
			s.logger.Infof("SUGGESTION: Consider updating your user_filter to: %s", filterTemplate)
			return
		}
	}

	s.logger.Warningf("No entries found with any alternative filters for user: %s", username)
	s.logger.Info("This could indicate:")
	s.logger.Info("1. The user does not exist in the LDAP directory")
	s.logger.Info("2. The Base DN is incorrect or too restrictive")
	s.logger.Info("3. The user is in a different OU not covered by the Base DN")
	s.logger.Info("4. Insufficient permissions for the admin account to search")
}

// DiagnoseDirectory 诊断 LDAP 目录结构，帮助确定正确的配置
func (s *Service) DiagnoseDirectory() error {
	if !s.config.Enabled {
		return fmt.Errorf("LDAP is not enabled")
	}

	conn, err := s.connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	if err := s.bindAdmin(conn); err != nil {
		return fmt.Errorf("failed to bind admin: %w", err)
	}

	s.logger.Info("=== LDAP Directory Diagnosis ===")
	s.logger.Infof("Base DN: %s", s.config.BaseDN)

	// 搜索组织单位
	s.logger.Info("--- Organizational Units ---")
	ouSearchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		20,
		0,
		false,
		"(objectClass=organizationalUnit)",
		[]string{"dn", "ou", "description"},
		nil,
	)

	ouResult, err := conn.Search(ouSearchRequest)
	if err != nil {
		s.logger.Warningf("Failed to search OUs: %v", err)
	} else {
		for i, entry := range ouResult.Entries {
			s.logger.Infof("OU %d: %s", i+1, entry.DN)
			if desc := entry.GetAttributeValue("description"); desc != "" {
				s.logger.Infof("  Description: %s", desc)
			}
		}
	}

	// 搜索前几个用户样本
	s.logger.Info("--- Sample Users (first 5) ---")
	userSearchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		5,
		0,
		false,
		"(objectClass=person)",
		[]string{"dn", "uid", "cn", "sAMAccountName", "userPrincipalName", "mail", "displayName"},
		nil,
	)

	userResult, err := conn.Search(userSearchRequest)
	if err != nil {
		s.logger.Warningf("Failed to search users: %v", err)
	} else {
		for i, entry := range userResult.Entries {
			s.logger.Infof("User %d: %s", i+1, entry.DN)
			s.logger.Infof("  uid: %s", entry.GetAttributeValue("uid"))
			s.logger.Infof("  cn: %s", entry.GetAttributeValue("cn"))
			s.logger.Infof("  sAMAccountName: %s", entry.GetAttributeValue("sAMAccountName"))
			s.logger.Infof("  userPrincipalName: %s", entry.GetAttributeValue("userPrincipalName"))
			s.logger.Infof("  mail: %s", entry.GetAttributeValue("mail"))
			s.logger.Infof("  displayName: %s", entry.GetAttributeValue("displayName"))
		}
	}

	s.logger.Info("=== End of Diagnosis ===")
	return nil
}
