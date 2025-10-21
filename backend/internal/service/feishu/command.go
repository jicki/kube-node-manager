package feishu

import (
	"kube-node-manager/internal/model"
	"regexp"
	"strings"
)

// Command represents a parsed bot command
type Command struct {
	Name      string   // 命令名称，如 "help", "node", "cluster"
	Action    string   // 操作，如 "list", "info", "cordon"
	Args      []string // 参数列表
	RawString string   // 原始命令字符串
}

// CommandContext contains the context for command execution
type CommandContext struct {
	Command     *Command
	UserMapping *model.FeishuUserMapping
	ChatID      string
	MessageID   string
	Service     *Service
}

// CommandResponse represents the response from a command execution
type CommandResponse struct {
	Text string // Plain text response
	Card string // Card JSON response
}

// CommandHandler defines the interface for command handlers
type CommandHandler interface {
	Handle(ctx *CommandContext) (*CommandResponse, error)
	Description() string
}

// CommandRouter routes commands to appropriate handlers
type CommandRouter struct {
	handlers map[string]CommandHandler
}

// NewCommandRouter creates a new command router
func NewCommandRouter() *CommandRouter {
	router := &CommandRouter{
		handlers: make(map[string]CommandHandler),
	}

	// Register command handlers
	router.Register("help", &HelpCommandHandler{})
	router.Register("node", &NodeCommandHandler{})
	router.Register("cluster", &ClusterCommandHandler{})
	router.Register("audit", &AuditCommandHandler{})
	router.Register("label", &LabelCommandHandler{})
	router.Register("taint", &TaintCommandHandler{})

	return router
}

// Register registers a command handler
func (r *CommandRouter) Register(name string, handler CommandHandler) {
	r.handlers[name] = handler
}

// Route routes a command to the appropriate handler
func (r *CommandRouter) Route(ctx *CommandContext) (*CommandResponse, error) {
	handler, exists := r.handlers[ctx.Command.Name]
	if !exists {
		return &CommandResponse{
			Text: "未知命令。输入 /help 查看可用命令列表。",
		}, nil
	}

	return handler.Handle(ctx)
}

// ParseCommand parses a command string
func ParseCommand(cmdStr string) *Command {
	cmdStr = strings.TrimSpace(cmdStr)
	if !strings.HasPrefix(cmdStr, "/") {
		return nil
	}

	// Remove leading /
	cmdStr = strings.TrimPrefix(cmdStr, "/")

	// Split by whitespace
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil
	}

	cmd := &Command{
		Name:      parts[0],
		RawString: cmdStr,
	}

	// Parse action and args
	if len(parts) > 1 {
		cmd.Action = parts[1]
	}

	if len(parts) > 2 {
		cmd.Args = parts[2:]
		// 清理参数中的 Markdown 超链接格式
		cmd.Args = cleanMarkdownLinks(cmd.Args)
	}

	return cmd
}

// cleanMarkdownLinks 清理参数中的 Markdown 超链接格式
// 将 [text](url) 格式转换为 text
func cleanMarkdownLinks(args []string) []string {
	// Markdown 链接格式的正则表达式: [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)

	cleaned := make([]string, len(args))
	for i, arg := range args {
		// 替换所有 Markdown 链接为纯文本
		cleaned[i] = linkRegex.ReplaceAllString(arg, "$1")
	}

	return cleaned
}
