package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
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
	if err := query.Preload("Cluster").Preload("User").Order("created_at DESC").Find(&inventories).Error; err != nil {
		s.logger.Errorf("Failed to list inventories: %v", err)
		return nil, 0, fmt.Errorf("failed to list inventories: %w", err)
	}

	return inventories, total, nil
}

// GetInventory 获取主机清单详情
func (s *InventoryService) GetInventory(id uint) (*model.AnsibleInventory, error) {
	var inventory model.AnsibleInventory

	if err := s.db.Preload("Cluster").Preload("User").First(&inventory, id).Error; err != nil {
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

	inventory := &model.AnsibleInventory{
		Name:        req.Name,
		Description: req.Description,
		SourceType:  req.SourceType,
		ClusterID:   req.ClusterID,
		Content:     req.Content,
		HostsData:   req.HostsData,
		UserID:      userID,
	}

	if err := s.db.Create(inventory).Error; err != nil {
		s.logger.Errorf("Failed to create inventory: %v", err)
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	s.logger.Infof("Successfully created inventory: %s (ID: %d) by user %d", inventory.Name, inventory.ID, userID)
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

	if req.Content != "" {
		// 验证 inventory 内容格式
		if err := s.validateInventoryContent(req.Content); err != nil {
			return nil, fmt.Errorf("invalid inventory content: %w", err)
		}
		inventory.Content = req.Content
	}

	if req.HostsData != nil {
		inventory.HostsData = req.HostsData
	}

	if err := s.db.Save(&inventory).Error; err != nil {
		s.logger.Errorf("Failed to update inventory %d: %v", id, err)
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	s.logger.Infof("Successfully updated inventory: %s (ID: %d) by user %d", inventory.Name, inventory.ID, userID)
	return &inventory, nil
}

// DeleteInventory 删除主机清单
func (s *InventoryService) DeleteInventory(id uint, userID uint) error {
	var inventory model.AnsibleInventory

	if err := s.db.First(&inventory, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("inventory not found")
		}
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// 检查是否有关联的任务
	var taskCount int64
	if err := s.db.Model(&model.AnsibleTask{}).Where("inventory_id = ?", id).Count(&taskCount).Error; err != nil {
		return fmt.Errorf("failed to check related tasks: %w", err)
	}

	if taskCount > 0 {
		return fmt.Errorf("cannot delete inventory: %d tasks are using this inventory", taskCount)
	}

	if err := s.db.Delete(&inventory).Error; err != nil {
		s.logger.Errorf("Failed to delete inventory %d: %v", id, err)
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	s.logger.Infof("Successfully deleted inventory: %s (ID: %d) by user %d", inventory.Name, inventory.ID, userID)
	return nil
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

	// 获取节点列表
	nodes, err := s.k8sSvc.ListNodes(cluster.Name)
	if err != nil {
		s.logger.Errorf("Failed to list nodes from cluster %s: %v", cluster.Name, err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in cluster %s", cluster.Name)
	}

	// 过滤节点（根据标签）
	var filteredNodes []k8s.NodeInfo
	if len(req.NodeLabels) > 0 {
		for _, node := range nodes {
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
		}
	} else {
		filteredNodes = nodes
	}

	if len(filteredNodes) == 0 {
		return nil, fmt.Errorf("no nodes match the specified labels")
	}

	// 生成 inventory 内容（INI 格式）
	content := s.generateINIInventory(filteredNodes, cluster.Name)

	// 生成结构化主机数据
	hostsData := s.generateHostsData(filteredNodes)

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
		Content:     content,
		HostsData:   hostsData,
		UserID:      userID,
	}

	if err := s.db.Create(inventory).Error; err != nil {
		s.logger.Errorf("Failed to create inventory: %v", err)
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	s.logger.Infof("Successfully generated inventory from K8s cluster %s: %s (ID: %d, %d nodes)",
		cluster.Name, inventory.Name, inventory.ID, len(filteredNodes))
	return inventory, nil
}

// generateINIInventory 生成 INI 格式的 inventory 内容
func (s *InventoryService) generateINIInventory(nodes []k8s.NodeInfo, clusterName string) string {
	var builder strings.Builder

	// 写入组头
	builder.WriteString(fmt.Sprintf("[%s]\n", clusterName))

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

		// 格式: hostname ansible_host=ip ansible_user=root
		builder.WriteString(fmt.Sprintf("%s ansible_host=%s ansible_user=root\n", node.Name, ip))
	}

	// 写入变量组
	builder.WriteString(fmt.Sprintf("\n[%s:vars]\n", clusterName))
	builder.WriteString("ansible_python_interpreter=/usr/bin/python3\n")
	builder.WriteString("ansible_ssh_common_args='-o StrictHostKeyChecking=no'\n")

	return builder.String()
}

// generateHostsData 生成结构化主机数据
func (s *InventoryService) generateHostsData(nodes []k8s.NodeInfo) model.HostsData {
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
			"name":        node.Name,
			"ip":          ip,
			"internal_ip": node.InternalIP,
			"external_ip": node.ExternalIP,
			"roles":       node.Roles,
			"labels":      node.Labels,
			"version":     node.Version,
			"os":          node.OS,
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

	// 获取节点列表
	nodes, err := s.k8sSvc.ListNodes(inventory.Cluster.Name)
	if err != nil {
		s.logger.Errorf("Failed to list nodes from cluster %s: %v", inventory.Cluster.Name, err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in cluster %s", inventory.Cluster.Name)
	}

	// 重新生成 inventory 内容
	inventory.Content = s.generateINIInventory(nodes, inventory.Cluster.Name)
	inventory.HostsData = s.generateHostsData(nodes)

	if err := s.db.Save(&inventory).Error; err != nil {
		s.logger.Errorf("Failed to update inventory %d: %v", id, err)
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	s.logger.Infof("Successfully refreshed K8s inventory: %s (ID: %d, %d nodes)", inventory.Name, inventory.ID, len(nodes))
	return &inventory, nil
}

