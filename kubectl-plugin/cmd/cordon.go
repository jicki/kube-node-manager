package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"kubectl-node-mgr/pkg/k8s"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	cordonReason string
	cordonOutput string
)

// cordonCmd 代表 cordon 命令
var cordonCmd = &cobra.Command{
	Use:   "cordon NODE_NAME[,NODE_NAME...]",
	Short: "对节点执行 cordon 操作并添加详细说明",
	Long: `对指定的节点执行 cordon 操作，阻止新的 Pod 调度到这些节点上。
同时会添加详细的说明 annotations，包括原因和时间信息。

支持批量操作多个节点，节点名称用逗号分隔。`,
	Example: `  # 对单个节点执行 cordon
  kubectl node-mgr cordon node1 --reason "系统维护"

  # 批量 cordon 多个节点
  kubectl node-mgr cordon node1,node2,node3 --reason "集群升级"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runCordonCommand(args)
	},
}

// cordonListCmd 代表 cordon list 子命令
var cordonListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看已 cordon 的节点及其说明",
	Long:  `显示所有已被 cordon 的节点及其详细说明信息。`,
	Example: `  # 查看所有已 cordon 的节点
  kubectl node-mgr cordon list

  # 以 JSON 格式输出
  kubectl node-mgr cordon list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runCordonListCommand()
	},
}

// CordonInfo 存储 cordon 节点的详细信息
type CordonInfo struct {
	Name       string    `json:"name" yaml:"name"`
	Status     string    `json:"status" yaml:"status"`
	Reason     string    `json:"reason" yaml:"reason"`
	CordonTime time.Time `json:"cordonTime" yaml:"cordonTime"`
}

func init() {
	rootCmd.AddCommand(cordonCmd)
	cordonCmd.AddCommand(cordonListCmd)

	// cordon 命令的标志
	cordonCmd.Flags().StringVar(&cordonReason, "reason", "", "Cordon 原因 (必需)")

	// 标记必需的标志
	cordonCmd.MarkFlagRequired("reason")

	// cordon list 命令的标志
	cordonListCmd.Flags().StringVarP(&cordonOutput, "output", "o", "table", "输出格式 (table|json|yaml)")
}

func runCordonCommand(args []string) {
	// 解析节点名称
	nodeNames := parseNodeNames(args[0])

	// 获取 kubeconfig 和 context 参数
	kubeconfig, _ := rootCmd.PersistentFlags().GetString("kubeconfig")
	contextName, _ := rootCmd.PersistentFlags().GetString("context")

	// 创建 Kubernetes 客户端
	clientset, err := k8s.CreateClientWithOptions(kubeconfig, contextName)
	checkError(err)

	// 获取当前时间
	currentTime := time.Now()

	// 对每个节点执行 cordon 操作
	for _, nodeName := range nodeNames {
		err := cordonNode(clientset, nodeName, currentTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to cordon node %s: %v\n", nodeName, err)
			continue
		}
		fmt.Printf("Node %s cordoned successfully with reason: %s\n", nodeName, cordonReason)
	}
}

func runCordonListCommand() {
	// 获取 kubeconfig 和 context 参数
	kubeconfig, _ := rootCmd.PersistentFlags().GetString("kubeconfig")
	contextName, _ := rootCmd.PersistentFlags().GetString("context")

	// 创建 Kubernetes 客户端
	clientset, err := k8s.CreateClientWithOptions(kubeconfig, contextName)
	checkError(err)

	// 获取所有节点
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	checkError(err)

	// 筛选已 cordon 的节点
	var cordonedNodes []CordonInfo
	for _, node := range nodeList.Items {
		if node.Spec.Unschedulable {
			cordonInfo := extractCordonInfo(&node)
			cordonedNodes = append(cordonedNodes, cordonInfo)
		}
	}

	// 根据输出格式显示结果
	switch cordonOutput {
	case "json":
		outputCordonJSON(cordonedNodes)
	case "yaml":
		outputCordonYAML(cordonedNodes)
	default:
		outputCordonTable(cordonedNodes)
	}
}

func parseNodeNames(input string) []string {
	return strings.Split(strings.TrimSpace(input), ",")
}

func cordonNode(clientset kubernetes.Interface, nodeName string, cordonTime time.Time) error {
	// 获取节点
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %v", nodeName, err)
	}

	// 设置 unschedulable
	node.Spec.Unschedulable = true

	// 添加 annotations
	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}

	node.Annotations["deeproute.cn/kube-node-mgr"] = cordonReason
	node.Annotations["deeproute.cn/kube-node-mgr-timestamp"] = cordonTime.Format(time.RFC3339)

	// 更新节点
	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	return err
}

func extractCordonInfo(node *corev1.Node) CordonInfo {
	info := CordonInfo{
		Name:   node.Name,
		Status: "SchedulingDisabled",
	}

	if node.Annotations != nil {
		info.Reason = node.Annotations["deeproute.cn/kube-node-mgr"]

		// 解析时间
		if timeStr := node.Annotations["deeproute.cn/kube-node-mgr-timestamp"]; timeStr != "" {
			if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
				info.CordonTime = parsedTime
			}
		}
	}

	// 如果没有我们的 annotations，可能是通过其他方式 cordon 的
	if info.Reason == "" {
		info.Reason = "Unknown"
	}

	return info
}

func outputCordonTable(cordonInfos []CordonInfo) {
	if len(cordonInfos) == 0 {
		fmt.Println("No cordoned nodes found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tREASON\tCORDON-TIME")

	for _, info := range cordonInfos {
		timeStr := "-"
		if !info.CordonTime.IsZero() {
			timeStr = info.CordonTime.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			info.Name,
			info.Status,
			info.Reason,
			timeStr,
		)
	}

	w.Flush()
}

func outputCordonJSON(cordonInfos []CordonInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "CordonInfoList",
		"items":      cordonInfos,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	checkError(err)
	fmt.Println(string(jsonData))
}

func outputCordonYAML(cordonInfos []CordonInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "CordonInfoList",
		"items":      cordonInfos,
	}

	yamlData, err := yaml.Marshal(output)
	checkError(err)
	fmt.Print(string(yamlData))
}
