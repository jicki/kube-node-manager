package automation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandSecurityChecker_DangerousCommands(t *testing.T) {
	checker := NewCommandSecurityChecker()

	tests := []struct {
		name        string
		command     string
		shouldFail  bool
		description string
	}{
		{
			name:        "safe_command_uptime",
			command:     "uptime",
			shouldFail:  false,
			description: "Safe command should pass",
		},
		{
			name:        "safe_command_df",
			command:     "df -h",
			shouldFail:  false,
			description: "Safe command with flags should pass",
		},
		{
			name:        "dangerous_rm_root",
			command:     "rm -rf /",
			shouldFail:  true,
			description: "Dangerous rm -rf / should be blocked",
		},
		{
			name:        "dangerous_rm_wildcard",
			command:     "rm -rf /*",
			shouldFail:  true,
			description: "Dangerous rm -rf /* should be blocked",
		},
		{
			name:        "dangerous_fork_bomb",
			command:     ":(){ :|:& };:",
			shouldFail:  true,
			description: "Fork bomb should be blocked",
		},
		{
			name:        "dangerous_dd_zero",
			command:     "dd if=/dev/zero of=/dev/sda",
			shouldFail:  true,
			description: "DD to disk should be blocked",
		},
		{
			name:        "dangerous_mkfs",
			command:     "mkfs.ext4 /dev/sda1",
			shouldFail:  true,
			description: "mkfs command should be blocked",
		},
		{
			name:        "safe_grep",
			command:     "grep error /var/log/syslog",
			shouldFail:  false,
			description: "Grep command should pass",
		},
		{
			name:        "safe_systemctl",
			command:     "systemctl status docker",
			shouldFail:  false,
			description: "Systemctl status should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckCommand(tt.command)
			if tt.shouldFail {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestCommandSecurityChecker_CaseSensitivity(t *testing.T) {
	checker := NewCommandSecurityChecker()

	// 测试大小写变体
	dangerousCommands := []string{
		"rm -rf /",
		"RM -RF /",
		"Rm -Rf /",
		"RM -rf /",
	}

	for _, cmd := range dangerousCommands {
		err := checker.CheckCommand(cmd)
		assert.Error(t, err, "Command should be blocked regardless of case: %s", cmd)
	}
}

