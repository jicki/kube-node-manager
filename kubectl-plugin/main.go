package main

import (
	"fmt"
	"os"

	"kubectl-node-mgr/cmd"
)

// 插件版本信息
var (
	Version   = "v1.0.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	// 设置版本信息
	cmd.SetVersionInfo(Version, GitCommit, BuildDate)

	// 执行根命令
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
