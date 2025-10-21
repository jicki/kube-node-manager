package feishu

import "fmt"

// BotError represents a structured error for bot commands
type BotError struct {
	Code       string // 错误码，如 ERROR_NODE_NOT_FOUND
	Message    string // 用户友好的错误描述
	Suggestion string // 恢复建议
	Details    string // 技术细节（可选）
}

// Error implements the error interface
func (e *BotError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 错误码常量
const (
	// 集群相关错误
	ErrorClusterNotSelected = "ERROR_CLUSTER_NOT_SELECTED"
	ErrorClusterNotFound    = "ERROR_CLUSTER_NOT_FOUND"
	ErrorClusterConnection  = "ERROR_CLUSTER_CONNECTION"

	// 节点相关错误
	ErrorNodeNotFound   = "ERROR_NODE_NOT_FOUND"
	ErrorNodeOperation  = "ERROR_NODE_OPERATION"
	ErrorNodeDataFormat = "ERROR_NODE_DATA_FORMAT"

	// 标签相关错误
	ErrorLabelFormat     = "ERROR_LABEL_FORMAT"
	ErrorLabelOperation  = "ERROR_LABEL_OPERATION"
	ErrorLabelValidation = "ERROR_LABEL_VALIDATION"

	// 污点相关错误
	ErrorTaintFormat     = "ERROR_TAINT_FORMAT"
	ErrorTaintOperation  = "ERROR_TAINT_OPERATION"
	ErrorTaintValidation = "ERROR_TAINT_VALIDATION"

	// 参数相关错误
	ErrorInvalidArgument = "ERROR_INVALID_ARGUMENT"
	ErrorMissingArgument = "ERROR_MISSING_ARGUMENT"

	// 服务相关错误
	ErrorServiceNotConfigured = "ERROR_SERVICE_NOT_CONFIGURED"
	ErrorServiceUnavailable   = "ERROR_SERVICE_UNAVAILABLE"

	// 权限相关错误
	ErrorPermissionDenied = "ERROR_PERMISSION_DENIED"
	ErrorNotAuthenticated = "ERROR_NOT_AUTHENTICATED"
)

// 常见错误构造函数

// NewClusterNotSelectedError 创建未选择集群错误
func NewClusterNotSelectedError() *BotError {
	return &BotError{
		Code:    ErrorClusterNotSelected,
		Message: "尚未选择集群",
		Suggestion: "请先使用 /cluster list 查看集群列表\n" +
			"然后使用 /cluster set <集群名> 选择集群",
	}
}

// NewNodeNotFoundError 创建节点未找到错误
func NewNodeNotFoundError(nodeName, clusterName string) *BotError {
	return &BotError{
		Code:    ErrorNodeNotFound,
		Message: fmt.Sprintf("节点 `%s` 不存在", nodeName),
		Suggestion: fmt.Sprintf("• 使用 /node list 查看集群 %s 中的所有节点\n"+
			"• 检查节点名称是否正确\n"+
			"• 确认节点是否已被删除", clusterName),
		Details: fmt.Sprintf("cluster=%s, node=%s", clusterName, nodeName),
	}
}

// NewServiceNotConfiguredError 创建服务未配置错误
func NewServiceNotConfiguredError(serviceName string) *BotError {
	return &BotError{
		Code:       ErrorServiceNotConfigured,
		Message:    fmt.Sprintf("%s服务未配置", serviceName),
		Suggestion: "这是一个系统错误，请联系管理员检查服务配置",
		Details:    fmt.Sprintf("service=%s", serviceName),
	}
}

// NewInvalidArgumentError 创建无效参数错误
func NewInvalidArgumentError(argName, expectedFormat, actualValue string) *BotError {
	return &BotError{
		Code:    ErrorInvalidArgument,
		Message: fmt.Sprintf("参数 %s 格式错误", argName),
		Suggestion: fmt.Sprintf("• 期望格式: %s\n"+
			"• 实际值: %s\n"+
			"• 使用 /help 查看正确用法", expectedFormat, actualValue),
		Details: fmt.Sprintf("arg=%s, expected=%s, actual=%s", argName, expectedFormat, actualValue),
	}
}

// NewMissingArgumentError 创建缺少参数错误
func NewMissingArgumentError(command, usage string) *BotError {
	return &BotError{
		Code:       ErrorMissingArgument,
		Message:    "参数不足",
		Suggestion: fmt.Sprintf("• 正确用法: %s\n• 使用 /help %s 查看详细帮助", usage, command),
	}
}

// NewLabelFormatError 创建标签格式错误
func NewLabelFormatError(details string) *BotError {
	return &BotError{
		Code:    ErrorLabelFormat,
		Message: "标签格式错误",
		Suggestion: "• 正确格式: key=value\n" +
			"• 多个标签: key1=val1,key2=val2\n" +
			"• 使用 /help label 查看详细帮助",
		Details: details,
	}
}

// NewTaintFormatError 创建污点格式错误
func NewTaintFormatError(details string) *BotError {
	return &BotError{
		Code:    ErrorTaintFormat,
		Message: "污点格式错误",
		Suggestion: "• 正确格式: key=value:effect\n" +
			"• Effect 可选: NoSchedule, PreferNoSchedule, NoExecute\n" +
			"• 使用 /help taint 查看详细帮助",
		Details: details,
	}
}

// NewOperationFailedError 创建操作失败错误
func NewOperationFailedError(operation, resource, reason string) *BotError {
	code := ErrorNodeOperation
	if resource == "label" {
		code = ErrorLabelOperation
	} else if resource == "taint" {
		code = ErrorTaintOperation
	}

	return &BotError{
		Code:    code,
		Message: fmt.Sprintf("%s失败", operation),
		Suggestion: fmt.Sprintf("• 检查集群连接是否正常\n"+
			"• 检查%s名称是否正确\n"+
			"• 查看错误详情: %s\n"+
			"• 如问题持续，请联系管理员", resource, reason),
		Details: fmt.Sprintf("operation=%s, resource=%s, reason=%s", operation, resource, reason),
	}
}

// BuildErrorCardV2 builds an enhanced error message card with suggestions
func BuildErrorCardV2(err *BotError) string {
	return BuildEnhancedErrorCard(err.Code, err.Message, err.Suggestion, err.Details)
}
