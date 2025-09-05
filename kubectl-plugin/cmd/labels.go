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
	labelsOutput   string
	labelsSelector string
	showAllLabels  bool
)

// labelsCmd 代表 labels 命令
var labelsCmd = &cobra.Command{
	Use:   "labels [NODE_NAME]",
	Short: "查看节点的 deeproute.cn/user-type 标签归属",
	Long: `查看节点的 deeproute.cn/user-type 标签归属信息。

如果不指定节点名称，将显示所有节点的标签信息。
可以使用标签选择器来过滤节点。`,
	Example: `  # 查看所有节点的用户类型标签
  kubectl node-mgr labels

  # 查看特定节点的标签
  kubectl node-mgr labels node1

  # 使用标签选择器过滤节点
  kubectl node-mgr labels -l "kubernetes.io/arch=amd64"

  # 显示所有标签，不仅仅是用户类型
  kubectl node-mgr labels --show-all

  # 以 JSON 格式输出
  kubectl node-mgr labels -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runLabelsCommand(args)
	},
}

// NodeLabelInfo 节点标签信息结构
type NodeLabelInfo struct {
	Name      string            `json:"name" yaml:"name"`
	Status    string            `json:"status" yaml:"status"`
	Roles     []string          `json:"roles" yaml:"roles"`
	Age       string            `json:"age" yaml:"age"`
	UserType  string            `json:"userType" yaml:"userType"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt time.Time         `json:"createdAt" yaml:"createdAt"`
}

func init() {
	rootCmd.AddCommand(labelsCmd)

	// 添加标志
	labelsCmd.Flags().StringVarP(&labelsOutput, "output", "o", "table", "输出格式 (table|json|yaml)")
	labelsCmd.Flags().StringVarP(&labelsSelector, "selector", "l", "", "标签选择器")
	labelsCmd.Flags().BoolVar(&showAllLabels, "show-all", false, "显示所有标签，不仅仅是 deeproute.cn/user-type")
}

func runLabelsCommand(args []string) {
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
		if labelsSelector != "" {
			listOptions.LabelSelector = labelsSelector
		}

		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), listOptions)
		checkError(err)
		nodes = nodeList.Items
	}

	// 处理节点信息
	nodeInfos := processNodeLabels(nodes)

	// 根据输出格式显示结果
	switch labelsOutput {
	case "json":
		outputJSON(nodeInfos)
	case "yaml":
		outputYAML(nodeInfos)
	default:
		outputTable(nodeInfos)
	}
}

func processNodeLabels(nodes []corev1.Node) []NodeLabelInfo {
	var nodeInfos []NodeLabelInfo

	for _, node := range nodes {
		info := NodeLabelInfo{
			Name:      node.Name,
			Status:    getNodeStatus(&node),
			Roles:     getNodeRoles(&node),
			Age:       formatAge(node.CreationTimestamp.Time),
			UserType:  getUserType(&node),
			CreatedAt: node.CreationTimestamp.Time,
		}

		// 如果需要显示所有标签
		if showAllLabels {
			info.Labels = node.Labels
		}

		nodeInfos = append(nodeInfos, info)
	}

	return nodeInfos
}

func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				if node.Spec.Unschedulable {
					return "Ready,SchedulingDisabled"
				}
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeRoles(node *corev1.Node) []string {
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

func getUserType(node *corev1.Node) string {
	if userType, exists := node.Labels["deeproute.cn/user-type"]; exists {
		return userType
	}
	return "-"
}

func formatAge(creationTime time.Time) string {
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

func outputTable(nodeInfos []NodeLabelInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	if showAllLabels {
		fmt.Fprintln(w, "NAME\tSTATUS\tROLES\tAGE\tUSER-TYPE\tLABELS")
		for _, info := range nodeInfos {
			labels := formatLabelsForTable(info.Labels)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				info.Name,
				info.Status,
				strings.Join(info.Roles, ","),
				info.Age,
				info.UserType,
				labels,
			)
		}
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tROLES\tAGE\tUSER-TYPE")
		for _, info := range nodeInfos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				info.Name,
				info.Status,
				strings.Join(info.Roles, ","),
				info.Age,
				info.UserType,
			)
		}
	}

	w.Flush()
}

func formatLabelsForTable(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	var labelPairs []string
	count := 0
	maxDisplay := 3 // 最多显示3个标签

	for key, value := range labels {
		if count >= maxDisplay {
			labelPairs = append(labelPairs, "...")
			break
		}
		labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", key, value))
		count++
	}

	return strings.Join(labelPairs, ",")
}

func outputJSON(nodeInfos []NodeLabelInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NodeLabelList",
		"items":      nodeInfos,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	checkError(err)
	fmt.Println(string(jsonData))
}

func outputYAML(nodeInfos []NodeLabelInfo) {
	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NodeLabelList",
		"items":      nodeInfos,
	}

	yamlData, err := yaml.Marshal(output)
	checkError(err)
	fmt.Print(string(yamlData))
}
