package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// InventoryService 主机清单服务
type InventoryService struct {
	db      *gorm.DB
	logger  *logger.Logger
	k8sSvc  *k8s.Service
}

// NewInventoryService 创建主机清单服务实例
func NewInventoryService(db *gorm.DB, logger *logger.Logger, k8sSvc *k8s.Service) *InventoryService {
	return &InventoryService{
		db:     db,
		logger: logger,
		k8sSvc: k8sSvc,
	}
}

// ListInventories 列出主机清单
func (s *InventoryService) ListInventories(req model.InventoryListRequest, userID uint) ([]model.AnsibleInventory, int64, error) {
	var inventories []model.AnsibleInventory
	var total int64

	query := s.db.Model(&model.AnsibleInventory{})

	// 过滤条件
	if req.SourceType != "" {
		query = query.Where("source_type = ?", req.SourceType)
	}

	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorf("Failed to count inventories: %v", err)
		return nil, 0, fmt.Errorf("failed to count inventories: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询数据（包含关联）
	if err := query.Preload("Cluster").Preload("User").Preload("SSHKey").Order("created_at DESC").Find(&inventories).Error; err != nil {
		s.logger.Errorf("Failed to list inventories: %v", err)
		return nil, 0, fmt.Errorf("failed to list inventories: %w", err)
	}

	return inventories, total, nil
}

// GetInventory 获取主机清单详情
func (s *InventoryService) GetInventory(id uint) (*model.AnsibleInventory, error) {
	var inventory model.AnsibleInventory

	if err := s.db.Preload("Cluster").Preload("User").Preload("SSHKey").First(&inventory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory not found")
		}
		s.logger.Errorf("Failed to get inventory %d: %v", id, err)
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return &inventory, nil
}

// CreateInventory 创建主机清单
func (s *InventoryService) CreateInventory(req model.InventoryCreateRequest, userID uint) (*model.AnsibleInventory, error) {
	// 检查名称是否重复
	var count int64
	if err := s.db.Model(&model.AnsibleInventory{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		s.logger.Errorf("Failed to check inventory name: %v", err)
		return nil, fmt.Errorf("failed to check inventory name: %w", err)
	}

	if count > 0 {
		return nil, fmt.Errorf("inventory name already exists")
	}

	// 验证 inventory 内容格式
	if err := s.validateInventoryContent(req.Content); err != nil {
		return nil, fmt.Errorf("invalid inventory content: %w", err)
	}

	// 解析清单内容，生成主机数据
	hostsData := s.parseInventoryContent(req.Content)

	inventory := &model.AnsibleInventory{
		Name:        req.Name,
		Description: req.Description,
		SourceType:  req.SourceType,
		ClusterID:   req.ClusterID,
		SSHKeyID:    req.SSHKeyID, // 关联 SSH 密钥
		Content:     req.Content,
		HostsData:   hostsData,
		UserID:      userID,
	}

	if err := s.db.Create(inventory).Error; err != nil {
		s.logger.Errorf("Failed to create inventory: %v", err)
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	s.logger.Infof("Successfully created inventory: %s (ID: %d) by user %d with %d hosts", 
		inventory.Name, inventory.ID, userID, hostsData["total"])
	return inventory, nil
}

// UpdateInventory 更新主机清单
func (s *InventoryService) UpdateInventory(id uint, req model.InventoryUpdateRequest, userID uint) (*model.AnsibleInventory, error) {
	var inventory model.AnsibleInventory

	if err := s.db.First(&inventory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory not found")
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 检查名称是否与其他记录重复
	if req.Name != "" && req.Name != inventory.Name {
		var count int64
		if err := s.db.Model(&model.AnsibleInventory{}).
			Where("name = ? AND id != ?", req.Name, id).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to check inventory name: %w", err)
		}

		if count > 0 {
			return nil, fmt.Errorf("inventory name already exists")
		}

		inventory.Name = req.Name
	}

	// 更新字段
	if req.Description != "" {
		inventory.Description = req.Description
	}

	// 更新 SSH 密钥关联
	if req.SSHKeyID != nil {
		inventory.SSHKeyID = req.SSHKeyID
	}

	if req.Content != "" {
		// 验证 inventory 内容格式
		if err := s.validateInventoryContent(req.Content); err != nil {
			return nil, fmt.Errorf("invalid inventory content: %w", err)
		}
		inventory.Content = req.Content
		
		// 重新解析清单内容，更新主机数据
		inventory.HostsData = s.parseInventoryContent(req.Content)
	}

	if err := s.db.Save(&inventory).Error; err != nil {
		s.logger.Errorf("Failed to update inventory %d: %v", id, err)
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	s.logger.Infof("Successfully updated inventory: %s (ID: %d) by user %d", inventory.Name, inventory.ID, userID)
	return &inventory, nil
}

// DeleteInventory 删除主机清单
// 删除时会将关联任务的 inventory_id 设置为 NULL
func (s *InventoryService) DeleteInventory(id uint, userID uint) error {
	var inventory model.AnsibleInventory

	if err := s.db.First(&inventory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("inventory not found")
		}
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// 开启事务处理
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 将所有使用此清单的任务的 inventory_id 设置为 NULL
		if err := tx.Model(&model.AnsibleTask{}).
			Where("inventory_id = ?", id).
			Update("inventory_id", nil).Error; err != nil {
			s.logger.Errorf("Failed to update tasks for inventory %d: %v", id, err)
			return fmt.Errorf("failed to update related tasks: %w", err)
		}

		// 2. 执行软删除
		if err := tx.Delete(&inventory).Error; err != nil {
			s.logger.Errorf("Failed to delete inventory %d: %v", id, err)
			return fmt.Errorf("failed to delete inventory: %w", err)
		}

		s.logger.Infof("Successfully deleted inventory: %s (ID: %d) by user %d", inventory.Name, inventory.ID, userID)
		return nil
	})
}

// GenerateFromK8s 从 K8s 集群动态生成主机清单
func (s *InventoryService) GenerateFromK8s(req model.GenerateInventoryRequest, userID uint) (*model.AnsibleInventory, error) {
	// 获取集群信息
	var cluster model.Cluster
	if err := s.db.First(&cluster, req.ClusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found")
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// 获取 SSH 密钥的用户名（如果指定了 SSH 密钥）
	ansibleUser := "root" // 默认用户名
	if req.SSHKeyID != nil {
		var sshKey model.AnsibleSSHKey
		if err := s.db.First(&sshKey, *req.SSHKeyID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				s.logger.Warningf("SSH key %d not found, using default user 'root'", *req.SSHKeyID)
			} else {
				s.logger.Errorf("Failed to get SSH key %d: %v", *req.SSHKeyID, err)
			}
		} else {
			ansibleUser = sshKey.Username
			s.logger.Infof("Using SSH key username: %s", ansibleUser)
		}
	}

	// 获取节点列表
	nodes, err := s.k8sSvc.ListNodes(cluster.Name)
	if err != nil {
		s.logger.Errorf("Failed to list nodes from cluster %s: %v", cluster.Name, err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in cluster %s", cluster.Name)
	}

	// 过滤节点
	var filteredNodes []k8s.NodeInfo
	for _, node := range nodes {
		// 1. 首先过滤掉 master 和 control-plane 节点
		isMaster := false
		if _, exists := node.Labels["node-role.kubernetes.io/master"]; exists {
			isMaster = true
		}
		if _, exists := node.Labels["node-role.kubernetes.io/control-plane"]; exists {
			isMaster = true
		}
		
		// 如果是 master/control-plane 节点，跳过
		if isMaster {
			s.logger.Infof("Skipping master/control-plane node: %s", node.Name)
			continue
		}
		
		// 2. 然后根据用户指定的标签过滤（如果有）
		if len(req.NodeLabels) > 0 {
			match := true
			for key, value := range req.NodeLabels {
				if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
					match = false
					break
				}
			}
			if match {
				filteredNodes = append(filteredNodes, node)
			}
		} else {
			// 如果没有指定标签过滤条件，只要不是 master 节点就加入
			filteredNodes = append(filteredNodes, node)
		}
	}

	if len(filteredNodes) == 0 {
		return nil, fmt.Errorf("no nodes match the specified labels")
	}

	// 生成 inventory 内容（INI 格式），使用从 SSH 密钥获取的用户名
	content := s.generateINIInventory(filteredNodes, cluster.Name, ansibleUser)

	// 生成结构化主机数据，使用从 SSH 密钥获取的用户名
	hostsData := s.generateHostsData(filteredNodes, ansibleUser)

	// 检查名称是否重复
	var count int64
	if err := s.db.Model(&model.AnsibleInventory{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check inventory name: %w", err)
	}

	if count > 0 {
		return nil, fmt.Errorf("inventory name already exists")
	}

	// 创建 inventory
	inventory := &model.AnsibleInventory{
		Name:        req.Name,
		Description: req.Description,
		SourceType:  model.InventorySourceK8s,
		ClusterID:   &req.ClusterID,
		SSHKeyID:    req.SSHKeyID, // 关联 SSH 密钥
		Content:     content,
		HostsData:   hostsData,
		UserID:      userID,
	}

	if err := s.db.Create(inventory).Error; err != nil {
		s.logger.Errorf("Failed to create inventory: %v", err)
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	s.logger.Infof("Successfully generated inventory from K8s cluster %s: %s (ID: %d, %d nodes, user: %s)",
		cluster.Name, inventory.Name, inventory.ID, len(filteredNodes), ansibleUser)
	return inventory, nil
}

// generateINIInventory 生成 INI 格式的 inventory 内容
// ansibleUser: Ansible 连接使用的用户名，从 SSH 密钥获取或默认为 root
func (s *InventoryService) generateINIInventory(nodes []k8s.NodeInfo, clusterName string, ansibleUser string) string {
	var builder strings.Builder

	// 使用 [all] 组作为默认组
	builder.WriteString("[all]\n")

	// 写入主机
	for _, node := range nodes {
		// 优先使用 InternalIP
		ip := node.InternalIP
		if ip == "" {
			ip = node.ExternalIP
		}

		if ip == "" {
			s.logger.Warningf("Node %s has no IP address, skipping", node.Name)
			continue
		}

		// 格式: hostname ansible_host=ip ansible_user=<username>
		builder.WriteString(fmt.Sprintf("%s ansible_host=%s ansible_user=%s\n", node.Name, ip, ansibleUser))
	}

	// 写入变量组 [all:vars]
	builder.WriteString("\n[all:vars]\n")
	builder.WriteString("ansible_python_interpreter=/usr/bin/python3\n")
	builder.WriteString("ansible_ssh_common_args='-o StrictHostKeyChecking=no'\n")

	return builder.String()
}

// generateHostsData 生成结构化主机数据
// ansibleUser: Ansible 连接使用的用户名，从 SSH 密钥获取或默认为 root
func (s *InventoryService) generateHostsData(nodes []k8s.NodeInfo, ansibleUser string) model.HostsData {
	hostsData := make(model.HostsData)
	hosts := make([]map[string]interface{}, 0, len(nodes))

	for _, node := range nodes {
		ip := node.InternalIP
		if ip == "" {
			ip = node.ExternalIP
		}

		if ip == "" {
			continue
		}

		host := map[string]interface{}{
			"name":         node.Name,
			"ip":           ip,
			"internal_ip":  node.InternalIP,
			"external_ip":  node.ExternalIP,
			"roles":        node.Roles,
			"labels":       node.Labels,
			"version":      node.Version,
			"os":           node.OS,
			"ansible_user": ansibleUser, // 添加 ansible_user 信息
		}

		hosts = append(hosts, host)
	}

	hostsData["hosts"] = hosts
	hostsData["total"] = len(hosts)

	return hostsData
}

// validateInventoryContent 验证 inventory 内容格式
func (s *InventoryService) validateInventoryContent(content string) error {
	if content == "" {
		return fmt.Errorf("inventory content cannot be empty")
	}

	// 基本格式验证
	lines := strings.Split(content, "\n")
	hasGroup := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否有组定义
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			hasGroup = true
			continue
		}

		// 检查主机行格式（简单验证）
		if hasGroup && !strings.Contains(line, "=") {
			// 简单的主机名行
			continue
		}
	}

	if !hasGroup {
		return fmt.Errorf("inventory must contain at least one group")
	}

	return nil
}

// parseInventoryContent 解析 INI 格式的 inventory 内容，提取主机信息
func (s *InventoryService) parseInventoryContent(content string) model.HostsData {
	hostsData := make(model.HostsData)
	hosts := make([]map[string]interface{}, 0)
	hostSet := make(map[string]bool) // 用于去重

	lines := strings.Split(content, "\n")
	currentGroup := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是组定义
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentGroup = strings.Trim(line, "[]")
			// 跳过变量组
			if strings.Contains(currentGroup, ":") {
				currentGroup = ""
			}
			continue
		}

		// 如果在组内，解析主机行
		if currentGroup != "" {
			// 提取主机名或 IP（可能包含变量）
			// 格式可能是:
			// hostname
			// hostname ansible_host=ip ansible_user=root
			// ip ansible_user=root
			
			parts := strings.Fields(line)
			if len(parts) == 0 {
				continue
			}

			hostIdentifier := parts[0]
			
			// 如果已经解析过这个主机，跳过
			if hostSet[hostIdentifier] {
				continue
			}
			hostSet[hostIdentifier] = true

			// 解析主机变量
			ansibleHost := hostIdentifier
			ansibleUser := "root" // 默认用户
			ansiblePort := 22     // 默认端口

			// 解析变量
			for i := 1; i < len(parts); i++ {
				kv := strings.SplitN(parts[i], "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.TrimSpace(kv[1])

					switch key {
					case "ansible_host":
						ansibleHost = value
					case "ansible_user":
						ansibleUser = value
					case "ansible_port":
						if port, err := strconv.Atoi(value); err == nil {
							ansiblePort = port
						}
					}
				}
			}

			host := map[string]interface{}{
				"name":         hostIdentifier,
				"ip":           ansibleHost,
				"ansible_user": ansibleUser,
				"ansible_port": ansiblePort,
				"group":        currentGroup,
			}

			hosts = append(hosts, host)
		}
	}

	hostsData["hosts"] = hosts
	hostsData["total"] = len(hosts)

	return hostsData
}

// RefreshK8sInventory 刷新来自 K8s 的主机清单
func (s *InventoryService) RefreshK8sInventory(id uint, userID uint) (*model.AnsibleInventory, error) {
	var inventory model.AnsibleInventory

	if err := s.db.Preload("Cluster").First(&inventory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory not found")
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 检查是否为 K8s 来源
	if inventory.SourceType != model.InventorySourceK8s {
		return nil, fmt.Errorf("only k8s-sourced inventories can be refreshed")
	}

	if inventory.ClusterID == nil {
		return nil, fmt.Errorf("inventory has no associated cluster")
	}

	if inventory.Cluster == nil {
		return nil, fmt.Errorf("cluster not found")
	}

	// 获取 SSH 密钥的用户名（如果清单关联了 SSH 密钥）
	ansibleUser := "root" // 默认用户名
	if inventory.SSHKeyID != nil {
		var sshKey model.AnsibleSSHKey
		if err := s.db.First(&sshKey, *inventory.SSHKeyID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				s.logger.Warningf("SSH key %d not found, using default user 'root'", *inventory.SSHKeyID)
			} else {
				s.logger.Errorf("Failed to get SSH key %d: %v", *inventory.SSHKeyID, err)
			}
		} else {
			ansibleUser = sshKey.Username
			s.logger.Infof("Refreshing inventory with SSH key username: %s", ansibleUser)
		}
	}

	// 获取节点列表
	nodes, err := s.k8sSvc.ListNodes(inventory.Cluster.Name)
	if err != nil {
		s.logger.Errorf("Failed to list nodes from cluster %s: %v", inventory.Cluster.Name, err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in cluster %s", inventory.Cluster.Name)
	}

	// 重新生成 inventory 内容，使用从 SSH 密钥获取的用户名
	inventory.Content = s.generateINIInventory(nodes, inventory.Cluster.Name, ansibleUser)
	inventory.HostsData = s.generateHostsData(nodes, ansibleUser)

	if err := s.db.Save(&inventory).Error; err != nil {
		s.logger.Errorf("Failed to update inventory %d: %v", id, err)
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	s.logger.Infof("Successfully refreshed K8s inventory: %s (ID: %d, %d nodes, user: %s)", 
		inventory.Name, inventory.ID, len(nodes), ansibleUser)
	return &inventory, nil
}

