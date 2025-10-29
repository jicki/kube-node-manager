package feishu

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// AutomationCommandHandler handles automation commands
type AutomationCommandHandler struct{}

// Description returns the command description
func (h *AutomationCommandHandler) Description() string {
	return "自动化运维命令"
}

// Handle processes the automation command
func (h *AutomationCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// 检查功能是否启用
	config, err := ctx.Service.db.Table("automation_configs").
		Where("config_key = ?", "automation.enabled").
		Select("config_value").
		First(&struct{ ConfigValue string }{}).Error
	
	if err == nil {
		var enabled bool
		if json.Unmarshal([]byte(config), &enabled); enabled == false {
			return &CommandResponse{
				Text: "❌ 自动化功能未启用\n\n请联系管理员在系统配置中启用自动化功能。",
			}, nil
		}
	}

	action := ctx.Command.Action
	args := ctx.Command.Args

	switch action {
	case "ansible":
		return h.handleAnsible(ctx, args)
	case "ssh":
		return h.handleSSH(ctx, args)
	case "script":
		return h.handleScript(ctx, args)
	case "workflow":
		return h.handleWorkflow(ctx, args)
	case "status":
		return h.handleStatus(ctx, args)
	case "help", "":
		return h.handleHelp(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 未知操作: %s\n\n输入 /automation help 查看帮助", action),
		}, nil
	}
}

// handleAnsible handles Ansible playbook commands
func (h *AutomationCommandHandler) handleAnsible(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 缺少参数\n\n用法:\n" +
				"/automation ansible list [category]\n" +
				"/automation ansible info <id>\n" +
				"/automation ansible run <id> <cluster> <nodes...>",
		}, nil
	}

	subAction := args[0]

	switch subAction {
	case "list", "ls":
		return h.listPlaybooks(ctx, args[1:])
	case "info":
		return h.playbookInfo(ctx, args[1:])
	case "run", "exec":
		return h.runPlaybook(ctx, args[1:])
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 未知操作: %s", subAction),
		}, nil
	}
}

// listPlaybooks lists available Ansible playbooks
func (h *AutomationCommandHandler) listPlaybooks(ctx *CommandContext, args []string) (*CommandResponse, error) {
	var playbooks []struct {
		ID          uint
		Name        string
		Description string
		Category    string
		IsBuiltin   bool
		IsActive    bool
	}

	query := ctx.Service.db.Table("ansible_playbooks").
		Where("is_active = ?", true)

	if len(args) > 0 {
		query = query.Where("category = ?", args[0])
	}

	if err := query.Find(&playbooks).Error; err != nil {
		return &CommandResponse{
			Text: "❌ 查询失败: " + err.Error(),
		}, nil
	}

	if len(playbooks) == 0 {
		return &CommandResponse{
			Text: "📋 未找到 Playbook",
		}, nil
	}

	// 构建响应
	var text strings.Builder
	text.WriteString(fmt.Sprintf("📋 **可用的 Ansible Playbooks** (共 %d 个)\n\n", len(playbooks)))

	for _, pb := range playbooks {
		tag := ""
		if pb.IsBuiltin {
			tag = "🔒 内置"
		}
		text.WriteString(fmt.Sprintf("**%d. %s** %s\n", pb.ID, pb.Name, tag))
		text.WriteString(fmt.Sprintf("   分类: %s\n", pb.Category))
		text.WriteString(fmt.Sprintf("   描述: %s\n\n", pb.Description))
	}

	text.WriteString("💡 **使用方法**:\n")
	text.WriteString("查看详情: `/automation ansible info <id>`\n")
	text.WriteString("执行: `/automation ansible run <id> <cluster> <node1> [node2...]`")

	return &CommandResponse{
		Text: text.String(),
	}, nil
}

// playbookInfo shows playbook details
func (h *AutomationCommandHandler) playbookInfo(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 请指定 Playbook ID\n\n用法: /automation ansible info <id>",
		}, nil
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的 ID: " + args[0],
		}, nil
	}

	var playbook struct {
		ID          uint
		Name        string
		Description string
		Content     string
		Category    string
		IsBuiltin   bool
		Version     int
	}

	if err := ctx.Service.db.Table("ansible_playbooks").
		Where("id = ?", id).
		First(&playbook).Error; err != nil {
		return &CommandResponse{
			Text: fmt.Sprintf("❌ Playbook ID %d 不存在", id),
		}, nil
	}

	text := fmt.Sprintf("📋 **Playbook 详情**\n\n"+
		"**ID**: %d\n"+
		"**名称**: %s\n"+
		"**分类**: %s\n"+
		"**版本**: v%d\n"+
		"**类型**: %s\n\n"+
		"**描述**:\n%s\n\n"+
		"**内容预览**:\n```yaml\n%s\n```\n\n"+
		"💡 **执行命令**:\n"+
		"`/automation ansible run %d <cluster> <node1> [node2...]`",
		playbook.ID,
		playbook.Name,
		playbook.Category,
		playbook.Version,
		map[bool]string{true: "🔒 内置", false: "📝 自定义"}[playbook.IsBuiltin],
		playbook.Description,
		truncateString(playbook.Content, 500),
		playbook.ID,
	)

	return &CommandResponse{
		Text: text,
	}, nil
}

// runPlaybook executes an Ansible playbook
func (h *AutomationCommandHandler) runPlaybook(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) < 3 {
		return &CommandResponse{
			Text: "❌ 参数不足\n\n" +
				"用法: /automation ansible run <id> <cluster> <node1> [node2...]\n\n" +
				"示例: /automation ansible run 1 production node1 node2",
		}, nil
	}

	playbookID, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的 Playbook ID: " + args[0],
		}, nil
	}

	clusterName := args[1]
	targetNodes := args[2:]

	// 构建交互式确认卡片
	card := buildPlaybookExecutionCard(playbookID, clusterName, targetNodes)

	return &CommandResponse{
		Card: card,
	}, nil
}

// handleSSH handles SSH command execution
func (h *AutomationCommandHandler) handleSSH(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) < 3 {
		return &CommandResponse{
			Text: "❌ 参数不足\n\n" +
				"用法: /automation ssh <cluster> <command> <node1> [node2...]\n\n" +
				"示例: /automation ssh production \"uptime\" node1 node2",
		}, nil
	}

	clusterName := args[0]
	command := args[1]
	targetNodes := args[2:]

	// 构建交互式确认卡片
	card := buildSSHExecutionCard(clusterName, command, targetNodes)

	return &CommandResponse{
		Card: card,
	}, nil
}

// handleScript handles script management
func (h *AutomationCommandHandler) handleScript(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 缺少参数\n\n用法:\n" +
				"/automation script list [category]\n" +
				"/automation script info <id>\n" +
				"/automation script run <id> <cluster> <nodes...>",
		}, nil
	}

	subAction := args[0]

	switch subAction {
	case "list", "ls":
		return h.listScripts(ctx, args[1:])
	case "info":
		return h.scriptInfo(ctx, args[1:])
	case "run", "exec":
		return h.runScript(ctx, args[1:])
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 未知操作: %s", subAction),
		}, nil
	}
}

// listScripts lists available scripts
func (h *AutomationCommandHandler) listScripts(ctx *CommandContext, args []string) (*CommandResponse, error) {
	var scripts []struct {
		ID          uint
		Name        string
		Description string
		Language    string
		Category    string
		IsBuiltin   bool
	}

	query := ctx.Service.db.Table("scripts").
		Where("is_active = ?", true)

	if len(args) > 0 {
		query = query.Where("category = ?", args[0])
	}

	if err := query.Find(&scripts).Error; err != nil {
		return &CommandResponse{
			Text: "❌ 查询失败: " + err.Error(),
		}, nil
	}

	if len(scripts) == 0 {
		return &CommandResponse{
			Text: "📜 未找到脚本",
		}, nil
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📜 **可用的脚本** (共 %d 个)\n\n", len(scripts)))

	for _, s := range scripts {
		tag := ""
		if s.IsBuiltin {
			tag = "🔒 内置"
		}
		text.WriteString(fmt.Sprintf("**%d. %s** %s\n", s.ID, s.Name, tag))
		text.WriteString(fmt.Sprintf("   语言: %s | 分类: %s\n", s.Language, s.Category))
		text.WriteString(fmt.Sprintf("   描述: %s\n\n", s.Description))
	}

	text.WriteString("💡 **使用方法**:\n")
	text.WriteString("查看详情: `/automation script info <id>`\n")
	text.WriteString("执行: `/automation script run <id> <cluster> <node1> [node2...]`")

	return &CommandResponse{
		Text: text.String(),
	}, nil
}

// scriptInfo shows script details
func (h *AutomationCommandHandler) scriptInfo(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 请指定脚本 ID\n\n用法: /automation script info <id>",
		}, nil
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的 ID: " + args[0],
		}, nil
	}

	var script struct {
		ID          uint
		Name        string
		Description string
		Content     string
		Language    string
		Category    string
		IsBuiltin   bool
		Version     int
	}

	if err := ctx.Service.db.Table("scripts").
		Where("id = ?", id).
		First(&script).Error; err != nil {
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 脚本 ID %d 不存在", id),
		}, nil
	}

	text := fmt.Sprintf("📜 **脚本详情**\n\n"+
		"**ID**: %d\n"+
		"**名称**: %s\n"+
		"**语言**: %s\n"+
		"**分类**: %s\n"+
		"**版本**: v%d\n"+
		"**类型**: %s\n\n"+
		"**描述**:\n%s\n\n"+
		"**内容预览**:\n```%s\n%s\n```\n\n"+
		"💡 **执行命令**:\n"+
		"`/automation script run %d <cluster> <node1> [node2...]`",
		script.ID,
		script.Name,
		script.Language,
		script.Category,
		script.Version,
		map[bool]string{true: "🔒 内置", false: "📝 自定义"}[script.IsBuiltin],
		script.Description,
		script.Language,
		truncateString(script.Content, 500),
		script.ID,
	)

	return &CommandResponse{
		Text: text,
	}, nil
}

// runScript executes a script
func (h *AutomationCommandHandler) runScript(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) < 3 {
		return &CommandResponse{
			Text: "❌ 参数不足\n\n" +
				"用法: /automation script run <id> <cluster> <node1> [node2...]\n\n" +
				"示例: /automation script run 1 production node1 node2",
		}, nil
	}

	scriptID, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的脚本 ID: " + args[0],
		}, nil
	}

	clusterName := args[1]
	targetNodes := args[2:]

	// 构建交互式确认卡片
	card := buildScriptExecutionCard(scriptID, clusterName, targetNodes)

	return &CommandResponse{
		Card: card,
	}, nil
}

// handleWorkflow handles workflow management
func (h *AutomationCommandHandler) handleWorkflow(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 缺少参数\n\n用法:\n" +
				"/automation workflow list [category]\n" +
				"/automation workflow info <id>\n" +
				"/automation workflow run <id> <cluster> <nodes...>",
		}, nil
	}

	subAction := args[0]

	switch subAction {
	case "list", "ls":
		return h.listWorkflows(ctx, args[1:])
	case "info":
		return h.workflowInfo(ctx, args[1:])
	case "run", "exec":
		return h.runWorkflow(ctx, args[1:])
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 未知操作: %s", subAction),
		}, nil
	}
}

// listWorkflows lists available workflows
func (h *AutomationCommandHandler) listWorkflows(ctx *CommandContext, args []string) (*CommandResponse, error) {
	var workflows []struct {
		ID          uint
		Name        string
		Description string
		Category    string
		IsBuiltin   bool
		StepCount   int
	}

	query := ctx.Service.db.Table("workflows").
		Where("is_active = ?", true)

	if len(args) > 0 {
		query = query.Where("category = ?", args[0])
	}

	if err := query.Find(&workflows).Error; err != nil {
		return &CommandResponse{
			Text: "❌ 查询失败: " + err.Error(),
		}, nil
	}

	if len(workflows) == 0 {
		return &CommandResponse{
			Text: "🔄 未找到工作流",
		}, nil
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("🔄 **可用的工作流** (共 %d 个)\n\n", len(workflows)))

	for _, wf := range workflows {
		tag := ""
		if wf.IsBuiltin {
			tag = "🔒 内置"
		}
		text.WriteString(fmt.Sprintf("**%d. %s** %s\n", wf.ID, wf.Name, tag))
		text.WriteString(fmt.Sprintf("   分类: %s | 步骤数: %d\n", wf.Category, wf.StepCount))
		text.WriteString(fmt.Sprintf("   描述: %s\n\n", wf.Description))
	}

	text.WriteString("💡 **使用方法**:\n")
	text.WriteString("查看详情: `/automation workflow info <id>`\n")
	text.WriteString("执行: `/automation workflow run <id> <cluster> <node1> [node2...]`")

	return &CommandResponse{
		Text: text.String(),
	}, nil
}

// workflowInfo shows workflow details
func (h *AutomationCommandHandler) workflowInfo(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 请指定工作流 ID\n\n用法: /automation workflow info <id>",
		}, nil
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的 ID: " + args[0],
		}, nil
	}

	var workflow struct {
		ID          uint
		Name        string
		Description string
		Definition  string
		Category    string
		IsBuiltin   bool
		Version     int
	}

	if err := ctx.Service.db.Table("workflows").
		Where("id = ?", id).
		First(&workflow).Error; err != nil {
		return &CommandResponse{
			Text: fmt.Sprintf("❌ 工作流 ID %d 不存在", id),
		}, nil
	}

	// 解析工作流定义以获取步骤信息
	var def struct {
		Steps []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"steps"`
	}
	json.Unmarshal([]byte(workflow.Definition), &def)

	var text strings.Builder
	text.WriteString(fmt.Sprintf("🔄 **工作流详情**\n\n"+
		"**ID**: %d\n"+
		"**名称**: %s\n"+
		"**分类**: %s\n"+
		"**版本**: v%d\n"+
		"**类型**: %s\n\n"+
		"**描述**:\n%s\n\n"+
		"**步骤** (共 %d 步):\n",
		workflow.ID,
		workflow.Name,
		workflow.Category,
		workflow.Version,
		map[bool]string{true: "🔒 内置", false: "📝 自定义"}[workflow.IsBuiltin],
		workflow.Description,
		len(def.Steps),
	))

	for i, step := range def.Steps {
		text.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, step.Name, step.Type))
	}

	text.WriteString(fmt.Sprintf("\n💡 **执行命令**:\n"+
		"`/automation workflow run %d <cluster> <node1> [node2...]`", workflow.ID))

	return &CommandResponse{
		Text: text.String(),
	}, nil
}

// runWorkflow executes a workflow
func (h *AutomationCommandHandler) runWorkflow(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) < 3 {
		return &CommandResponse{
			Text: "❌ 参数不足\n\n" +
				"用法: /automation workflow run <id> <cluster> <node1> [node2...]\n\n" +
				"示例: /automation workflow run 1 production node1",
		}, nil
	}

	workflowID, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResponse{
			Text: "❌ 无效的工作流 ID: " + args[0],
		}, nil
	}

	clusterName := args[1]
	targetNodes := args[2:]

	// 构建交互式确认卡片
	card := buildWorkflowExecutionCard(workflowID, clusterName, targetNodes)

	return &CommandResponse{
		Card: card,
	}, nil
}

// handleStatus checks execution status
func (h *AutomationCommandHandler) handleStatus(ctx *CommandContext, args []string) (*CommandResponse, error) {
	if len(args) == 0 {
		return &CommandResponse{
			Text: "❌ 请指定任务 ID\n\n用法: /automation status <task_id>",
		}, nil
	}

	taskID := args[0]

	// 根据 taskID 前缀判断任务类型
	var status string
	var err error

	switch {
	case strings.HasPrefix(taskID, "ansible-exec-"):
		status, err = h.getAnsibleStatus(ctx, taskID)
	case strings.HasPrefix(taskID, "ssh-exec-"):
		status, err = h.getSSHStatus(ctx, taskID)
	case strings.HasPrefix(taskID, "script-exec-"):
		status, err = h.getScriptStatus(ctx, taskID)
	case strings.HasPrefix(taskID, "workflow-exec-"):
		status, err = h.getWorkflowStatus(ctx, taskID)
	default:
		return &CommandResponse{
			Text: "❌ 无效的任务 ID 格式",
		}, nil
	}

	if err != nil {
		return &CommandResponse{
			Text: "❌ 查询失败: " + err.Error(),
		}, nil
	}

	return &CommandResponse{
		Text: status,
	}, nil
}

// getAnsibleStatus gets Ansible execution status
func (h *AutomationCommandHandler) getAnsibleStatus(ctx *CommandContext, taskID string) (string, error) {
	var exec struct {
		TaskID       string
		PlaybookName string
		Status       string
		SuccessCount int
		FailedCount  int
		StartTime    string
		EndTime      *string
	}

	err := ctx.Service.db.Table("ansible_executions").
		Where("task_id = ?", taskID).
		First(&exec).Error

	if err != nil {
		return "", err
	}

	statusEmoji := map[string]string{
		"pending":   "⏳",
		"running":   "🔄",
		"completed": "✅",
		"failed":    "❌",
		"cancelled": "🚫",
	}

	text := fmt.Sprintf("📋 **Ansible 执行状态**\n\n"+
		"**任务 ID**: %s\n"+
		"**Playbook**: %s\n"+
		"**状态**: %s %s\n"+
		"**成功**: %d 个节点\n"+
		"**失败**: %d 个节点\n"+
		"**开始时间**: %s\n",
		exec.TaskID,
		exec.PlaybookName,
		statusEmoji[exec.Status],
		exec.Status,
		exec.SuccessCount,
		exec.FailedCount,
		exec.StartTime,
	)

	if exec.EndTime != nil {
		text += fmt.Sprintf("**结束时间**: %s\n", *exec.EndTime)
	}

	return text, nil
}

// getSSHStatus gets SSH execution status
func (h *AutomationCommandHandler) getSSHStatus(ctx *CommandContext, taskID string) (string, error) {
	// 类似 getAnsibleStatus 的实现
	return "🔄 SSH 执行状态查询功能待实现", nil
}

// getScriptStatus gets script execution status
func (h *AutomationCommandHandler) getScriptStatus(ctx *CommandContext, taskID string) (string, error) {
	// 类似 getAnsibleStatus 的实现
	return "🔄 脚本执行状态查询功能待实现", nil
}

// getWorkflowStatus gets workflow execution status
func (h *AutomationCommandHandler) getWorkflowStatus(ctx *CommandContext, taskID string) (string, error) {
	// 类似 getAnsibleStatus 的实现，但包含步骤详情
	return "🔄 工作流执行状态查询功能待实现", nil
}

// handleHelp shows help information
func (h *AutomationCommandHandler) handleHelp(ctx *CommandContext) (*CommandResponse, error) {
	text := `🤖 **自动化运维命令帮助**

**Ansible Playbook**:
• \`/automation ansible list [category]\` - 列出 Playbook
• \`/automation ansible info <id>\` - 查看 Playbook 详情
• \`/automation ansible run <id> <cluster> <nodes...>\` - 执行 Playbook

**SSH 命令**:
• \`/automation ssh <cluster> "<command>" <nodes...>\` - 执行 SSH 命令

**脚本管理**:
• \`/automation script list [category]\` - 列出脚本
• \`/automation script info <id>\` - 查看脚本详情
• \`/automation script run <id> <cluster> <nodes...>\` - 执行脚本

**工作流**:
• \`/automation workflow list [category]\` - 列出工作流
• \`/automation workflow info <id>\` - 查看工作流详情
• \`/automation workflow run <id> <cluster> <nodes...>\` - 执行工作流

**状态查询**:
• \`/automation status <task_id>\` - 查询任务执行状态

**示例**:
\`/automation ansible list system\`
\`/automation ansible run 1 production node1 node2\`
\`/automation ssh production "uptime" node1 node2\`
\`/automation status ansible-exec-1234567890-abc123\`

💡 提示: 执行命令前会显示确认卡片，请仔细检查后确认执行。`

	return &CommandResponse{
		Text: text,
	}, nil
}

// Helper functions

// truncateString truncates a string to maxLength
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// buildPlaybookExecutionCard builds a confirmation card for playbook execution
func buildPlaybookExecutionCard(playbookID int, clusterName string, targetNodes []string) string {
	// 这里应该构建飞书卡片 JSON
	// 简化版本，实际应该使用完整的卡片构建器
	card := fmt.Sprintf(`{
  "config": {"wide_screen_mode": true},
  "header": {
    "title": {"tag": "plain_text", "content": "📋 执行 Ansible Playbook"},
    "template": "blue"
  },
  "elements": [
    {"tag": "div", "text": {"tag": "lark_md", "content": "**Playbook ID**: %d\n**集群**: %s\n**目标节点**: %s\n\n**请确认后点击执行按钮**"}},
    {"tag": "action", "actions": [
      {"tag": "button", "text": {"tag": "plain_text", "content": "✅ 确认执行"}, "type": "primary", "value": "confirm"},
      {"tag": "button", "text": {"tag": "plain_text", "content": "❌ 取消"}, "type": "default", "value": "cancel"}
    ]}
  ]
}`, playbookID, clusterName, strings.Join(targetNodes, ", "))

	return card
}

// buildSSHExecutionCard builds a confirmation card for SSH execution
func buildSSHExecutionCard(clusterName, command string, targetNodes []string) string {
	card := fmt.Sprintf(`{
  "config": {"wide_screen_mode": true},
  "header": {
    "title": {"tag": "plain_text", "content": "🔧 执行 SSH 命令"},
    "template": "orange"
  },
  "elements": [
    {"tag": "div", "text": {"tag": "lark_md", "content": "**命令**: \`%s\`\n**集群**: %s\n**目标节点**: %s\n\n**请确认后点击执行按钮**"}},
    {"tag": "action", "actions": [
      {"tag": "button", "text": {"tag": "plain_text", "content": "✅ 确认执行"}, "type": "primary", "value": "confirm"},
      {"tag": "button", "text": {"tag": "plain_text", "content": "❌ 取消"}, "type": "default", "value": "cancel"}
    ]}
  ]
}`, command, clusterName, strings.Join(targetNodes, ", "))

	return card
}

// buildScriptExecutionCard builds a confirmation card for script execution
func buildScriptExecutionCard(scriptID int, clusterName string, targetNodes []string) string {
	card := fmt.Sprintf(`{
  "config": {"wide_screen_mode": true},
  "header": {
    "title": {"tag": "plain_text", "content": "📜 执行脚本"},
    "template": "green"
  },
  "elements": [
    {"tag": "div", "text": {"tag": "lark_md", "content": "**脚本 ID**: %d\n**集群**: %s\n**目标节点**: %s\n\n**请确认后点击执行按钮**"}},
    {"tag": "action", "actions": [
      {"tag": "button", "text": {"tag": "plain_text", "content": "✅ 确认执行"}, "type": "primary", "value": "confirm"},
      {"tag": "button", "text": {"tag": "plain_text", "content": "❌ 取消"}, "type": "default", "value": "cancel"}
    ]}
  ]
}`, scriptID, clusterName, strings.Join(targetNodes, ", "))

	return card
}

// buildWorkflowExecutionCard builds a confirmation card for workflow execution
func buildWorkflowExecutionCard(workflowID int, clusterName string, targetNodes []string) string {
	card := fmt.Sprintf(`{
  "config": {"wide_screen_mode": true},
  "header": {
    "title": {"tag": "plain_text", "content": "🔄 执行工作流"},
    "template": "purple"
  },
  "elements": [
    {"tag": "div", "text": {"tag": "lark_md", "content": "**工作流 ID**: %d\n**集群**: %s\n**目标节点**: %s\n\n**请确认后点击执行按钮**"}},
    {"tag": "action", "actions": [
      {"tag": "button", "text": {"tag": "plain_text", "content": "✅ 确认执行"}, "type": "primary", "value": "confirm"},
      {"tag": "button", "text": {"tag": "plain_text", "content": "❌ 取消"}, "type": "default", "value": "cancel"}
    ]}
  ]
}`, workflowID, clusterName, strings.Join(targetNodes, ", "))

	return card
}

