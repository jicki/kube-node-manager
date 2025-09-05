package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   string
	gitCommit string
	buildDate string
)

// SetVersionInfo 设置版本信息
func SetVersionInfo(v, commit, date string) {
	version = v
	gitCommit = commit
	buildDate = date
}

// rootCmd 代表基础命令
var rootCmd = &cobra.Command{
	Use:   "kubectl-node-mgr",
	Short: "Kubernetes 节点管理插件",
	Long: `kubectl-node-mgr 是一个 kubectl 插件，用于管理 Kubernetes 节点。

功能包括：
• 查看节点的 deeproute.cn/user-type 标签归属
• 对节点执行 cordon 操作并添加详细说明 annotations
• 管理节点的调度状态`,
	Example: `  # 查看所有节点的调度状态
  kubectl node-mgr get

  # 查看所有节点的用户类型标签
  kubectl node-mgr labels

  # 查看特定节点的标签
  kubectl node-mgr labels node1

  # 对节点执行 cordon 操作
  kubectl node-mgr cordon node1 --reason "维护升级"

  # 查看已 cordon 的节点
  kubectl node-mgr cordon list

  # 取消节点的 cordon 状态
  kubectl node-mgr uncordon node1`,
}

// Execute 添加所有子命令到根命令并设置标志
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 添加版本命令
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kubectl-node-mgr %s\n", version)
			fmt.Printf("Git Commit: %s\n", gitCommit)
			fmt.Printf("Build Date: %s\n", buildDate)
		},
	})

	// 设置全局标志
	rootCmd.PersistentFlags().String("kubeconfig", "", "kubeconfig 文件路径 (默认为 $HOME/.kube/config)")
	rootCmd.PersistentFlags().String("context", "", "要使用的 kubeconfig 上下文名称")
	rootCmd.PersistentFlags().StringP("namespace", "n", "", "如果存在，CLI 请求的命名空间范围")
}

// 检查错误的辅助函数
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
