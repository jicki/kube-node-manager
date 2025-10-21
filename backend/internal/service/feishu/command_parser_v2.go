package feishu

import (
	"fmt"
	"strings"
)

// CommandArgsV2 represents enhanced command arguments
type CommandArgsV2 struct {
	Positional []string          // 位置参数
	Named      map[string]string // --key=value 参数
	Flags      map[string]bool   // --flag 参数
}

// CommandV2 represents an enhanced parsed command
type CommandV2 struct {
	Name   string         // 命令名称
	Action string         // 子命令/动作
	Args   *CommandArgsV2 // 增强的参数
}

// ParseCommandV2 parses a command string with enhanced argument support
// Supports:
// - Positional arguments: /node list arg1 arg2
// - Named parameters: /node list --cluster=prod --namespace=default
// - Flags: /node list --all --force
// - Short flags: /node list -a -f
// - Combined short flags: /node list -af
func ParseCommandV2(cmdStr string) (*CommandV2, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil, fmt.Errorf("命令为空")
	}

	// Remove leading slash
	if strings.HasPrefix(cmdStr, "/") {
		cmdStr = cmdStr[1:]
	}

	// Split command into parts
	parts := smartSplit(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("无效的命令格式")
	}

	cmd := &CommandV2{
		Name: parts[0],
		Args: &CommandArgsV2{
			Positional: []string{},
			Named:      make(map[string]string),
			Flags:      make(map[string]bool),
		},
	}

	// Parse remaining parts
	i := 1
	if len(parts) > 1 && !strings.HasPrefix(parts[1], "-") {
		cmd.Action = parts[1]
		i = 2
	}

	// Parse arguments
	for ; i < len(parts); i++ {
		part := parts[i]

		if strings.HasPrefix(part, "--") {
			// Long option: --key=value or --flag
			opt := part[2:]
			if idx := strings.Index(opt, "="); idx > 0 {
				// --key=value
				key := opt[:idx]
				value := opt[idx+1:]
				cmd.Args.Named[key] = value
			} else {
				// --flag
				cmd.Args.Flags[opt] = true
			}
		} else if strings.HasPrefix(part, "-") && len(part) > 1 {
			// Short option: -f or -abc
			opts := part[1:]
			for _, ch := range opts {
				// Map short flags to long flags
				longFlag := mapShortFlag(string(ch))
				cmd.Args.Flags[longFlag] = true
			}
		} else {
			// Positional argument
			cmd.Args.Positional = append(cmd.Args.Positional, part)
		}
	}

	return cmd, nil
}

// smartSplit splits a command string intelligently, handling quotes
func smartSplit(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range s {
		switch {
		case (ch == '"' || ch == '\'') && !inQuote:
			inQuote = true
			quoteChar = ch
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// mapShortFlag maps short flags to long flags
func mapShortFlag(short string) string {
	flagMap := map[string]string{
		"a": "all",
		"f": "force",
		"h": "help",
		"v": "verbose",
		"q": "quiet",
		"y": "yes",
		"n": "no",
		"r": "recursive",
		"d": "debug",
	}

	if long, ok := flagMap[short]; ok {
		return long
	}
	return short
}

// HasFlag checks if a flag is set
func (a *CommandArgsV2) HasFlag(name string) bool {
	if a.Flags == nil {
		return false
	}
	return a.Flags[name]
}

// GetNamed gets a named parameter value
func (a *CommandArgsV2) GetNamed(key string) (string, bool) {
	if a.Named == nil {
		return "", false
	}
	val, ok := a.Named[key]
	return val, ok
}

// GetNamedOrDefault gets a named parameter value with default
func (a *CommandArgsV2) GetNamedOrDefault(key, defaultValue string) string {
	if val, ok := a.GetNamed(key); ok {
		return val
	}
	return defaultValue
}

// GetPositional gets a positional argument by index
func (a *CommandArgsV2) GetPositional(index int) (string, bool) {
	if a.Positional == nil || index < 0 || index >= len(a.Positional) {
		return "", false
	}
	return a.Positional[index], true
}

// GetPositionalOrDefault gets a positional argument with default
func (a *CommandArgsV2) GetPositionalOrDefault(index int, defaultValue string) string {
	if val, ok := a.GetPositional(index); ok {
		return val
	}
	return defaultValue
}

// PositionalCount returns the number of positional arguments
func (a *CommandArgsV2) PositionalCount() int {
	if a.Positional == nil {
		return 0
	}
	return len(a.Positional)
}

// ToCommand converts CommandV2 to original Command format for backward compatibility
func (c *CommandV2) ToCommand() *Command {
	args := c.Args.Positional

	// Add named parameters as key=value
	for k, v := range c.Args.Named {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	// Add flags
	for k, v := range c.Args.Flags {
		if v {
			args = append(args, fmt.Sprintf("--%s", k))
		}
	}

	return &Command{
		Name:   c.Name,
		Action: c.Action,
		Args:   args,
	}
}

// CommandAlias defines command aliases
var CommandAliases = map[string]string{
	"ls":  "list",
	"get": "info",
	"del": "delete",
	"rm":  "remove",
	"add": "create",
	"sw":  "switch",
	"st":  "status",
	"log": "logs",
	"h":   "help",
}

// ResolveAlias resolves command aliases
func ResolveAlias(action string) string {
	if resolved, ok := CommandAliases[action]; ok {
		return resolved
	}
	return action
}
