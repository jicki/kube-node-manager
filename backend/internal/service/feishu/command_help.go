package feishu

// HelpCommandHandler handles the help command
type HelpCommandHandler struct{}

// Handle executes the help command
func (h *HelpCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// 如果指定了具体命令，显示该命令的帮助
	// /help label 会被解析为 Action="label"
	if ctx.Command.Action != "" {
		cmdName := ctx.Command.Action
		switch cmdName {
		case "label":
			return &CommandResponse{
				Card: BuildLabelHelpCard(),
			}, nil
		case "taint":
			return &CommandResponse{
				Card: BuildTaintHelpCard(),
			}, nil
		case "batch":
			return &CommandResponse{
				Card: BuildBatchHelpCard(),
			}, nil
		case "quick":
			return &CommandResponse{
				Card: BuildQuickHelpCard(),
			}, nil
		default:
			return &CommandResponse{
				Card: BuildHelpCard(),
			}, nil
		}
	}

	return &CommandResponse{
		Card: BuildHelpCard(),
	}, nil
}

// Description returns the command description
func (h *HelpCommandHandler) Description() string {
	return "显示帮助信息"
}
