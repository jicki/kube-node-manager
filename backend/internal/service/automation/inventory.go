package automation

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// InventoryManager 动态 Inventory 管理器
type InventoryManager struct {
	hosts  map[string]*InventoryHost
	groups map[string]*InventoryGroup
}

// InventoryHost Ansible 主机配置
type InventoryHost struct {
	Name        string                 `json:"-"`
	Vars        map[string]interface{} `json:"vars,omitempty"`
	AnsibleHost string                 `json:"ansible_host,omitempty"`
	Groups      []string               `json:"-"`
}

// InventoryGroup Ansible 主机组
type InventoryGroup struct {
	Name     string                 `json:"-"`
	Hosts    []string               `json:"hosts,omitempty"`
	Children []string               `json:"children,omitempty"`
	Vars     map[string]interface{} `json:"vars,omitempty"`
}

// AnsibleInventory Ansible Inventory 格式
type AnsibleInventory struct {
	All    *InventoryGroup            `json:"_meta"`
	Groups map[string]*InventoryGroup `json:",inline"`
}

// NewInventoryManager 创建 Inventory 管理器
func NewInventoryManager() *InventoryManager {
	return &InventoryManager{
		hosts:  make(map[string]*InventoryHost),
		groups: make(map[string]*InventoryGroup),
	}
}

// AddHost 添加主机
func (im *InventoryManager) AddHost(name, ip string, vars map[string]interface{}) {
	if im.hosts[name] == nil {
		im.hosts[name] = &InventoryHost{
			Name:        name,
			Vars:        make(map[string]interface{}),
			AnsibleHost: ip,
			Groups:      []string{},
		}
	}

	// 设置 IP 地址
	if ip != "" {
		im.hosts[name].AnsibleHost = ip
	}

	// 合并变量
	for k, v := range vars {
		im.hosts[name].Vars[k] = v
	}
}

// AddGroup 添加主机组
func (im *InventoryManager) AddGroup(name string, vars map[string]interface{}) {
	if im.groups[name] == nil {
		im.groups[name] = &InventoryGroup{
			Name:     name,
			Hosts:    []string{},
			Children: []string{},
			Vars:     make(map[string]interface{}),
		}
	}

	// 合并变量
	for k, v := range vars {
		im.groups[name].Vars[k] = v
	}
}

// AddHostToGroup 将主机添加到组
func (im *InventoryManager) AddHostToGroup(hostName, groupName string) {
	// 确保组存在
	im.AddGroup(groupName, nil)

	// 添加主机到组
	if !contains(im.groups[groupName].Hosts, hostName) {
		im.groups[groupName].Hosts = append(im.groups[groupName].Hosts, hostName)
	}

	// 记录主机所属的组
	if host := im.hosts[hostName]; host != nil {
		if !contains(host.Groups, groupName) {
			host.Groups = append(host.Groups, groupName)
		}
	}
}

// AddGroupToGroup 添加子组
func (im *InventoryManager) AddGroupToGroup(childGroup, parentGroup string) {
	// 确保两个组都存在
	im.AddGroup(childGroup, nil)
	im.AddGroup(parentGroup, nil)

	// 添加子组
	if !contains(im.groups[parentGroup].Children, childGroup) {
		im.groups[parentGroup].Children = append(im.groups[parentGroup].Children, childGroup)
	}
}

// FromKubernetesNodes 从 Kubernetes 节点生成 Inventory
func (im *InventoryManager) FromKubernetesNodes(nodes []corev1.Node) {
	for _, node := range nodes {
		// 获取节点 IP
		nodeIP := getNodeInternalIP(node)

		// 基本主机信息
		hostVars := map[string]interface{}{
			"k8s_node_name": node.Name,
		}

		// 添加节点角色信息
		roles := getNodeRoles(node)
		if len(roles) > 0 {
			hostVars["k8s_roles"] = roles
		}

		// 添加主机
		im.AddHost(node.Name, nodeIP, hostVars)

		// 按角色分组
		for _, role := range roles {
			groupName := fmt.Sprintf("k8s_role_%s", role)
			im.AddHostToGroup(node.Name, groupName)
		}

		// 按标签分组
		for key, value := range node.Labels {
			// 跳过 Kubernetes 内部标签
			if strings.HasPrefix(key, "kubernetes.io/") ||
				strings.HasPrefix(key, "k8s.io/") ||
				strings.HasPrefix(key, "node.kubernetes.io/") {
				continue
			}

			// 创建标签组
			safeKey := strings.ReplaceAll(key, ".", "_")
			safeKey = strings.ReplaceAll(safeKey, "/", "_")
			groupName := fmt.Sprintf("label_%s_%s", safeKey, value)
			im.AddHostToGroup(node.Name, groupName)

			// 将标签添加到主机变量
			hostVars[fmt.Sprintf("label_%s", safeKey)] = value
		}

		// 按状态分组
		if isNodeReady(node) {
			im.AddHostToGroup(node.Name, "node_ready")
		} else {
			im.AddHostToGroup(node.Name, "node_not_ready")
		}

		// 按调度状态分组
		if node.Spec.Unschedulable {
			im.AddHostToGroup(node.Name, "node_unschedulable")
		} else {
			im.AddHostToGroup(node.Name, "node_schedulable")
		}
	}
}

// ToJSON 导出为 JSON 格式
func (im *InventoryManager) ToJSON() (string, error) {
	// 构建 Ansible 动态 Inventory 格式
	inventory := make(map[string]interface{})

	// 添加 _meta 部分（主机变量）
	metaHostvars := make(map[string]map[string]interface{})
	for name, host := range im.hosts {
		hostVars := make(map[string]interface{})

		// 添加 ansible_host
		if host.AnsibleHost != "" {
			hostVars["ansible_host"] = host.AnsibleHost
		}

		// 添加自定义变量
		for k, v := range host.Vars {
			hostVars[k] = v
		}

		metaHostvars[name] = hostVars
	}

	inventory["_meta"] = map[string]interface{}{
		"hostvars": metaHostvars,
	}

	// 添加组
	for name, group := range im.groups {
		groupData := make(map[string]interface{})

		if len(group.Hosts) > 0 {
			groupData["hosts"] = group.Hosts
		}

		if len(group.Children) > 0 {
			groupData["children"] = group.Children
		}

		if len(group.Vars) > 0 {
			groupData["vars"] = group.Vars
		}

		inventory[name] = groupData
	}

	// 添加 all 组（包含所有主机）
	allHosts := make([]string, 0, len(im.hosts))
	for name := range im.hosts {
		allHosts = append(allHosts, name)
	}
	inventory["all"] = map[string]interface{}{
		"hosts": allHosts,
	}

	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal inventory: %w", err)
	}

	return string(data), nil
}

// ToINI 导出为 INI 格式
func (im *InventoryManager) ToINI() string {
	var builder strings.Builder

	// 写入未分组的主机（如果有）
	ungroupedHosts := make([]string, 0)
	for name, host := range im.hosts {
		if len(host.Groups) == 0 {
			line := name
			if host.AnsibleHost != "" {
				line += fmt.Sprintf(" ansible_host=%s", host.AnsibleHost)
			}
			for k, v := range host.Vars {
				line += fmt.Sprintf(" %s=%v", k, v)
			}
			ungroupedHosts = append(ungroupedHosts, line)
		}
	}

	if len(ungroupedHosts) > 0 {
		builder.WriteString("[ungrouped]\n")
		for _, line := range ungroupedHosts {
			builder.WriteString(line + "\n")
		}
		builder.WriteString("\n")
	}

	// 写入组
	for groupName, group := range im.groups {
		builder.WriteString(fmt.Sprintf("[%s]\n", groupName))

		// 写入主机
		for _, hostName := range group.Hosts {
			host := im.hosts[hostName]
			line := hostName
			if host != nil && host.AnsibleHost != "" {
				line += fmt.Sprintf(" ansible_host=%s", host.AnsibleHost)
			}
			builder.WriteString(line + "\n")
		}

		// 写入子组
		if len(group.Children) > 0 {
			builder.WriteString(fmt.Sprintf("\n[%s:children]\n", groupName))
			for _, child := range group.Children {
				builder.WriteString(child + "\n")
			}
		}

		// 写入组变量
		if len(group.Vars) > 0 {
			builder.WriteString(fmt.Sprintf("\n[%s:vars]\n", groupName))
			for k, v := range group.Vars {
				builder.WriteString(fmt.Sprintf("%s=%v\n", k, v))
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// getNodeInternalIP 获取节点内部 IP
func getNodeInternalIP(node corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

// getNodeRoles 获取节点角色
func getNodeRoles(node corev1.Node) []string {
	roles := []string{}
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}
	return roles
}

// isNodeReady 检查节点是否就绪
func isNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// contains 检查字符串切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
