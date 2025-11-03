package ansible

import (
	"regexp"
	"strings"
)

// ExtractVariables 从 Playbook 内容中提取所有变量
func ExtractVariables(playbookContent string) []string {
	// Jinja2 变量格式: {{ variable_name }}
	// 支持的格式:
	// - {{ var }}
	// - {{ var.field }}
	// - {{ var | filter }}
	// - {{ var[0] }}
	
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_\.]*(?:\[[^\]]+\])?)(?:\s*\|[^}]+)?\s*\}\}`)
	matches := re.FindAllStringSubmatch(playbookContent, -1)
	
	// 使用 map 去重
	varMap := make(map[string]bool)
	
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			// 提取根变量名（去除 .field 或 [index] 部分）
			varName = extractRootVariable(varName)
			
			// 过滤掉 Ansible 内置变量
			if !isAnsibleBuiltinVar(varName) {
				varMap[varName] = true
			}
		}
	}
	
	// 转换为切片
	vars := make([]string, 0, len(varMap))
	for varName := range varMap {
		vars = append(vars, varName)
	}
	
	return vars
}

// extractRootVariable 提取根变量名
// 例如: "user.name" -> "user", "items[0]" -> "items"
func extractRootVariable(varName string) string {
	// 处理点号分隔的字段访问
	if idx := strings.Index(varName, "."); idx != -1 {
		return varName[:idx]
	}
	
	// 处理数组索引访问
	if idx := strings.Index(varName, "["); idx != -1 {
		return varName[:idx]
	}
	
	return varName
}

// isAnsibleBuiltinVar 判断是否为 Ansible 内置变量
func isAnsibleBuiltinVar(varName string) bool {
	builtinVars := map[string]bool{
		// Ansible 魔法变量
		"inventory_hostname":       true,
		"inventory_hostname_short": true,
		"groups":                   true,
		"group_names":              true,
		"hostvars":                 true,
		"ansible_facts":            true,
		"ansible_version":          true,
		"ansible_playbook_python":  true,
		
		// 主机变量
		"ansible_host":             true,
		"ansible_port":             true,
		"ansible_user":             true,
		"ansible_connection":       true,
		"ansible_ssh_private_key_file": true,
		
		// 任务执行变量
		"item":                     true,
		"ansible_loop":             true,
		"playbook_dir":             true,
		"role_path":                true,
		"inventory_dir":            true,
		
		// 其他常用内置变量
		"omit":                     true,
		"ansible_check_mode":       true,
		"ansible_diff_mode":        true,
		"ansible_verbosity":        true,
	}
	
	return builtinVars[varName]
}

// ValidateVariables 验证提供的变量是否包含所有必需变量
// 返回缺失的变量列表
func ValidateVariables(requiredVars []string, providedVars map[string]interface{}) []string {
	missing := make([]string, 0)
	
	for _, varName := range requiredVars {
		if _, exists := providedVars[varName]; !exists {
			missing = append(missing, varName)
		}
	}
	
	return missing
}

// GetVariableDescription 获取变量的描述（从注释中提取）
// 支持的注释格式:
// # @var variable_name: description
func GetVariableDescription(playbookContent string, varName string) string {
	// 简单实现：查找包含 @var 的注释行
	lines := strings.Split(playbookContent, "\n")
	pattern := regexp.MustCompile(`#\s*@var\s+` + regexp.QuoteMeta(varName) + `:\s*(.+)`)
	
	for _, line := range lines {
		if matches := pattern.FindStringSubmatch(line); matches != nil && len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

