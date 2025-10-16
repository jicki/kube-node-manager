package feishu

// HelpCommandHandler handles the help command
type HelpCommandHandler struct{}

// Handle executes the help command
func (h *HelpCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	return &CommandResponse{
		Card: BuildHelpCard(),
	}, nil
}

// Description returns the command description
func (h *HelpCommandHandler) Description() string {
	return "显示帮助信息"
}
