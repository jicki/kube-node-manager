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
	"sigs.k8s.io/yaml"
)

var (
	getOutput   string
	getSelector string
)

// getCmd 代表 get 命令
var getCmd = &cobra.Command{
	Use:   "get [NODE_NAME]",
	Short: "查看节点调度状态和禁止调度信息",
	Long: `查看节点的调度状态，包括是否被 cordon、禁止调度的原因和时间。

如果不指定节点名称，将显示所有节点的调度状态信息。
可以使用标签选择器来过滤节点。`,
	Example: `  # 查看所有节点的调度状态
  kubectl node_mgr get

  # 查看特定节点的调度状态
  kubectl node_mgr get node1

  # 使用标签选择器过滤节点
  kubectl node_mgr get -l "kubernetes.io/arch=amd64"

  # 以 JSON 格式输出
  kubectl node_mgr get -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runGetCommand(args)
	},
}

// NodeScheduleInfo 节点调度状态信息结构
type NodeScheduleInfo struct {
	Name          string    `json:"name" yaml:"name"`
	Status        string    `json:"status" yaml:"status"`
	Roles         []string  `json:"roles" yaml:"roles"`
	Age           string    `json:"age" yaml:"age"`
	Schedulable   string    `json:"schedulable" yaml:"schedulable"`
	CordonReason  string    `json:"cordonReason,omitempty" yaml:"cordonReason,omitempty"`
	CordonTime    time.Time `json:"cordonTime,omitempty" yaml:"cordonTime,omitempty"`
	CordonTimeStr string    `json:"cordonTimeStr,omitempty" yaml:"cordonTimeStr,omitempty"`
	CreatedAt     time.Time `json:"createdAt" yaml:"createdAt"`
}

func init() {
	rootCmd.AddCommand(getCmd)

	// 添加标志
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "table", "输出格式 (table|json|yaml)")
	getCmd.Flags().StringVarP(&getSelector, "selector", "l", "", "标签选择器")
}

func runGetCommand(args []string) {
	// 获取 kubeconfig 和 context 参数
	kubeconfig, _ := rootCmd.PersistentFlags().GetString("kubeconfig")
	contextName, _ := rootCmd.PersistentFlags().GetString("context")

	// 创建 Kubernetes 客户端
	clientset, err := k8s.CreateClientWithOptions(kubeconfig, contextName)
	checkError(err)

	var nodes []corev1.Node
	var nodeName string

	if len(args) > 0 {
		nodeName = args[0]
		// 获取特定节点
		node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		checkError(err)
		nodes = []corev1.Node{*node}
	} else {
		// 获取所有节点
		listOptions := metav1.ListOptions{}
		if getSelector != "" {
			listOptions.LabelSelector = getSelector
		}

		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), listOptions)
		checkError(err)
		nodes = nodeList.Items
	}

	// 处理节点信息
	nodeInfos := processNodeScheduleInfo(nodes)

	// 根据输出格式显示结果
	switch getOutput {
	case "json":
		outputGetJSON(nodeInfos)
	case "yaml":
		outputGetYAML(nodeInfos)
	default:
		outputGetTable(nodeInfos)
	}
}

func processNodeScheduleInfo(nodes []corev1.Node) []NodeScheduleInfo {
	var nodeInfos []NodeScheduleInfo

	for _, node := range nodes {
		info := NodeScheduleInfo{
			Name:      node.Name,
			Status:    getNodeStatusForGet(&node),
			Roles:     getNodeRolesForGet(&node),
			Age:       formatAgeForGet(node.CreationTimestamp.Time),
			CreatedAt: node.CreationTimestamp.Time,
		}

		// 检查调度状态
		if node.Spec.Unschedulable {
			info.Schedulable = "SchedulingDisabled"

			// 尝试获取 cordon 信息
			if node.Annotations != nil {
				if reason := node.Annotations["deeproute.cn/kube-node-mgr"]; reason != "" {
					info.CordonReason = reason
				} else {
					info.CordonReason = "Unknown"
				}

				// 解析时间
				if timeStr := node.Annotations["deeproute.cn/kube-node-mgr-timestamp"]; timeStr != "" {
					if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
						info.CordonTime = parsedTime
						info.CordonTimeStr = parsedTime.Format("2006-01-02 15:04:05")
					}
				}
			} else {
				info.CordonReason = "Unknown"
			}
		} else {
			info.Schedulable = "Schedulable"
		}

		nodeInfos = append(nodeInfos, info)
	}

	return nodeInfos
}

func outputGetTable(nodeInfos []NodeScheduleInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tROLES\tAGE\tSCHEDULABLE\tCORDON-REASON\tCORDON-TIME")

	for _, info := range nodeInfos {
		reason := info.CordonReason
		if reason == "" {
			reason = "-"
		}

		cordonTime := info.CordonTimeStr
		if cordonTime == "" {
			cordonTime = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			info.Name,
			info.Status,
			strings.Join(info.Roles, ","),
			info.Age,
			info.Schedulable,
			reason,
			cordonTime,
		)
	}

	w.Flush()
}

func outputGetJSON(nodeInfos []NodeScheduleInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NodeScheduleInfoList",
		"items":      nodeInfos,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	checkError(err)
	fmt.Println(string(jsonData))
}

func outputGetYAML(nodeInfos []NodeScheduleInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NodeScheduleInfoList",
		"items":      nodeInfos,
	}

	yamlData, err := yaml.Marshal(output)
	checkError(err)
	fmt.Print(string(yamlData))
}

// 辅助函数
func getNodeStatusForGet(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeRolesForGet(node *corev1.Node) []string {
	roles := []string{}

	// 检查常见的角色标签
	roleLabels := []string{
		"node-role.kubernetes.io/master",
		"node-role.kubernetes.io/control-plane",
		"node-role.kubernetes.io/worker",
		"node-role.kubernetes.io/etcd",
	}

	for _, roleLabel := range roleLabels {
		if _, exists := node.Labels[roleLabel]; exists {
			// 提取角色名称
			parts := strings.Split(roleLabel, "/")
			if len(parts) > 1 {
				role := parts[1]
				// 将 control-plane 转换为 master 以保持一致性
				if role == "control-plane" {
					role = "master"
				}
				roles = append(roles, role)
			}
		}
	}

	// 如果没有找到角色，检查是否有其他角色标签
	if len(roles) == 0 {
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				parts := strings.Split(label, "/")
				if len(parts) > 1 && parts[1] != "" {
					roles = append(roles, parts[1])
				}
			}
		}
	}

	// 如果仍然没有角色，默认为 worker
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	return roles
}

func formatAgeForGet(creationTime time.Time) string {
	duration := time.Since(creationTime)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(duration.Minutes())
	return fmt.Sprintf("%dm", minutes)
}
