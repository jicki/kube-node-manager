package logger

import (
	"regexp"
	"strings"
)

// SensitivePattern 敏感数据匹配规则
type SensitivePattern struct {
	Name    string         // 规则名称
	Pattern *regexp.Regexp // 正则表达式
	Replace string         // 替换字符串
}

// Sanitizer 日志脱敏器
type Sanitizer struct {
	patterns []SensitivePattern
	enabled  bool
}

// NewSanitizer 创建脱敏器实例
func NewSanitizer() *Sanitizer {
	s := &Sanitizer{
		enabled:  true,
		patterns: make([]SensitivePattern, 0),
	}
	
	// 注册默认脱敏规则
	s.RegisterDefaultPatterns()
	
	return s
}

// RegisterDefaultPatterns 注册默认的敏感数据模式
func (s *Sanitizer) RegisterDefaultPatterns() {
	// 1. 密码字段（password, passwd, pwd）
	s.AddPattern("password", regexp.MustCompile(`(?i)(password|passwd|pwd)[\s]*[:=][\s]*["']?([^"'\s\n]+)["']?`), "${1}=***REDACTED***")
	
	// 2. API Key 和 Token
	s.AddPattern("api_key", regexp.MustCompile(`(?i)(api[_-]?key|apikey|token|access[_-]?token)[\s]*[:=][\s]*["']?([a-zA-Z0-9_\-\.]{16,})["']?`), "${1}=***REDACTED***")
	
	// 3. SSH 私钥
	s.AddPattern("ssh_key", regexp.MustCompile(`(?s)-----BEGIN[A-Z ]+PRIVATE KEY-----.*?-----END[A-Z ]+PRIVATE KEY-----`), "***SSH_PRIVATE_KEY_REDACTED***")
	
	// 4. AWS Access Key
	s.AddPattern("aws_key", regexp.MustCompile(`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key)[\s]*[:=][\s]*["']?([A-Z0-9]{20,})["']?`), "${1}=***REDACTED***")
	
	// 5. 数据库连接字符串中的密码
	s.AddPattern("db_password", regexp.MustCompile(`(?i)(mysql|postgresql|postgres|mongodb|redis)://([^:]+):([^@]+)@`), "${1}://${2}:***REDACTED***@")
	
	// 6. 环境变量中的敏感信息
	s.AddPattern("env_secret", regexp.MustCompile(`(?i)(secret|credential|auth)[\s]*[:=][\s]*["']?([^"'\s\n]{8,})["']?`), "${1}=***REDACTED***")
	
	// 7. JWT Token
	s.AddPattern("jwt", regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`), "***JWT_TOKEN_REDACTED***")
	
	// 8. 信用卡号（简单匹配）
	s.AddPattern("credit_card", regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), "***CREDIT_CARD_REDACTED***")
	
	// 9. IP 地址后的密码（常见于 ansible 输出）
	s.AddPattern("ansible_pass", regexp.MustCompile(`(?i)ansible[_-]?password[\s]*[:=][\s]*["']?([^"'\s\n]+)["']?`), "ansible_password=***REDACTED***")
	
	// 10. become 密码
	s.AddPattern("become_pass", regexp.MustCompile(`(?i)become[_-]?pass(word)?[\s]*[:=][\s]*["']?([^"'\s\n]+)["']?`), "become_pass=***REDACTED***")
}

// AddPattern 添加自定义脱敏规则
func (s *Sanitizer) AddPattern(name string, pattern *regexp.Regexp, replace string) {
	s.patterns = append(s.patterns, SensitivePattern{
		Name:    name,
		Pattern: pattern,
		Replace: replace,
	})
}

// Sanitize 对文本进行脱敏处理
func (s *Sanitizer) Sanitize(text string) string {
	if !s.enabled || text == "" {
		return text
	}
	
	result := text
	
	// 应用所有脱敏规则
	for _, pattern := range s.patterns {
		result = pattern.Pattern.ReplaceAllString(result, pattern.Replace)
	}
	
	return result
}

// SanitizeMap 对 map 中的敏感数据进行脱敏
func (s *Sanitizer) SanitizeMap(data map[string]interface{}) map[string]interface{} {
	if !s.enabled {
		return data
	}
	
	result := make(map[string]interface{})
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"token", "api_key", "apikey", "access_token",
		"secret", "credential", "auth",
		"private_key", "ssh_key",
		"aws_access_key_id", "aws_secret_access_key",
	}
	
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		
		// 检查 key 是否包含敏感关键字
		isSensitive := false
		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitiveKey) {
				isSensitive = true
				break
			}
		}
		
		if isSensitive {
			result[key] = "***REDACTED***"
		} else {
			// 递归处理嵌套的 map
			if nestedMap, ok := value.(map[string]interface{}); ok {
				result[key] = s.SanitizeMap(nestedMap)
			} else if str, ok := value.(string); ok {
				result[key] = s.Sanitize(str)
			} else {
				result[key] = value
			}
		}
	}
	
	return result
}

// Enable 启用脱敏
func (s *Sanitizer) Enable() {
	s.enabled = true
}

// Disable 禁用脱敏
func (s *Sanitizer) Disable() {
	s.enabled = false
}

// IsEnabled 检查是否启用
func (s *Sanitizer) IsEnabled() bool {
	return s.enabled
}

// GetPatternNames 获取所有已注册的规则名称
func (s *Sanitizer) GetPatternNames() []string {
	names := make([]string, 0, len(s.patterns))
	for _, pattern := range s.patterns {
		names = append(names, pattern.Name)
	}
	return names
}

