package cmd

import (
	"context"
	"fmt"
	"os"

	"kubectl-node-mgr/pkg/k8s"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// uncordonCmd 代表 uncordon 命令
var uncordonCmd = &cobra.Command{
	Use:   "uncordon NODE_NAME[,NODE_NAME...]",
	Short: "取消节点的 cordon 状态",
	Long: `取消指定节点的 cordon 状态，允许新的 Pod 调度到这些节点上。
同时会清理相关的说明 annotations。

支持批量操作多个节点，节点名称用逗号分隔。`,
	Example: `  # 取消单个节点的 cordon 状态
  kubectl node-mgr uncordon node1

  # 批量取消多个节点的 cordon 状态
  kubectl node-mgr uncordon node1,node2,node3`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runUncordonCommand(args)
	},
}

func init() {
	rootCmd.AddCommand(uncordonCmd)
}

func runUncordonCommand(args []string) {
	// 解析节点名称
	nodeNames := parseNodeNames(args[0])

	// 获取 kubeconfig 和 context 参数
	kubeconfig, _ := rootCmd.PersistentFlags().GetString("kubeconfig")
	contextName, _ := rootCmd.PersistentFlags().GetString("context")

	// 创建 Kubernetes 客户端
	clientset, err := k8s.CreateClientWithOptions(kubeconfig, contextName)
	checkError(err)

	// 对每个节点执行 uncordon 操作
	for _, nodeName := range nodeNames {
		err := uncordonNode(clientset, nodeName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to uncordon node %s: %v\n", nodeName, err)
			continue
		}
		fmt.Printf("Node %s uncordoned successfully\n", nodeName)
	}
}

func uncordonNode(clientset kubernetes.Interface, nodeName string) error {
	// 获取节点
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %v", nodeName, err)
	}

	// 检查节点是否已经是 uncordon 状态
	if !node.Spec.Unschedulable {
		return fmt.Errorf("node %s is already uncordoned", nodeName)
	}

	// 设置 schedulable
	node.Spec.Unschedulable = false

	// 清理相关的 annotations
	if node.Annotations != nil {
		annotationsToRemove := []string{
			"deeproute.cn/kube-node-mgr",
			"deeproute.cn/kube-node-mgr-timestamp",
		}

		for _, annotation := range annotationsToRemove {
			delete(node.Annotations, annotation)
		}
	}

	// 更新节点
	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	return err
}
