package ansible

import (
	"fmt"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
)

// WorkflowValidator DAG 验证器
type WorkflowValidator struct {
	logger *logger.Logger
}

// NewWorkflowValidator 创建验证器实例
func NewWorkflowValidator(logger *logger.Logger) *WorkflowValidator {
	return &WorkflowValidator{
		logger: logger,
	}
}

// ValidateDAG 验证 DAG 的有效性
// 检查：节点完整性、边的有效性、循环依赖、可达性
func (v *WorkflowValidator) ValidateDAG(dag *model.WorkflowDAG) error {
	if dag == nil {
		return fmt.Errorf("DAG 不能为空")
	}

	if len(dag.Nodes) == 0 {
		return fmt.Errorf("DAG 至少需要一个节点")
	}

	// 1. 验证节点
	if err := v.validateNodes(dag.Nodes); err != nil {
		return err
	}

	// 2. 验证边
	if err := v.validateEdges(dag.Nodes, dag.Edges); err != nil {
		return err
	}

	// 3. 检测循环依赖
	if err := v.detectCycles(dag); err != nil {
		return err
	}

	// 4. 检查连通性（所有节点可达）
	if err := v.validateConnectivity(dag); err != nil {
		return err
	}

	v.logger.Infof("DAG validation passed: %d nodes, %d edges", len(dag.Nodes), len(dag.Edges))
	return nil
}

// validateNodes 验证节点有效性
func (v *WorkflowValidator) validateNodes(nodes []model.WorkflowNode) error {
	if len(nodes) == 0 {
		return fmt.Errorf("节点列表不能为空")
	}

	nodeIDs := make(map[string]bool)
	hasStart := false
	hasEnd := false

	for i, node := range nodes {
		// 检查节点 ID
		if node.ID == "" {
			return fmt.Errorf("节点 %d 的 ID 不能为空", i)
		}

		// 检查节点 ID 唯一性
		if nodeIDs[node.ID] {
			return fmt.Errorf("节点 ID 重复: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		// 检查节点类型
		if node.Type != "start" && node.Type != "end" && node.Type != "task" {
			return fmt.Errorf("节点 %s 的类型无效: %s (必须是 start/end/task)", node.ID, node.Type)
		}

		// 检查开始和结束节点
		if node.Type == "start" {
			if hasStart {
				return fmt.Errorf("只能有一个开始节点")
			}
			hasStart = true
		}
		if node.Type == "end" {
			if hasEnd {
				return fmt.Errorf("只能有一个结束节点")
			}
			hasEnd = true
		}

		// 检查任务节点配置
		if node.Type == "task" {
			if node.TaskConfig == nil {
				return fmt.Errorf("任务节点 %s 缺少任务配置", node.ID)
			}
			if err := v.validateTaskConfig(node.TaskConfig); err != nil {
				return fmt.Errorf("任务节点 %s 配置无效: %w", node.ID, err)
			}
		}

		// 检查标签
		if node.Label == "" {
			return fmt.Errorf("节点 %s 的标签不能为空", node.ID)
		}
	}

	// 必须有开始和结束节点
	if !hasStart {
		return fmt.Errorf("DAG 必须有一个开始节点")
	}
	if !hasEnd {
		return fmt.Errorf("DAG 必须有一个结束节点")
	}

	return nil
}

// validateTaskConfig 验证任务配置
func (v *WorkflowValidator) validateTaskConfig(config *model.TaskCreateRequest) error {
	if config.Name == "" {
		return fmt.Errorf("任务名称不能为空")
	}

	if config.PlaybookContent == "" {
		return fmt.Errorf("Playbook 内容不能为空")
	}

	if config.InventoryID == nil || *config.InventoryID == 0 {
		return fmt.Errorf("必须指定主机清单")
	}

	return nil
}

// validateEdges 验证边的有效性
func (v *WorkflowValidator) validateEdges(nodes []model.WorkflowNode, edges []model.WorkflowEdge) error {
	// 构建节点 ID 映射
	nodeMap := make(map[string]*model.WorkflowNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	edgeIDs := make(map[string]bool)

	for i, edge := range edges {
		// 检查边 ID
		if edge.ID == "" {
			return fmt.Errorf("边 %d 的 ID 不能为空", i)
		}

		// 检查边 ID 唯一性
		if edgeIDs[edge.ID] {
			return fmt.Errorf("边 ID 重复: %s", edge.ID)
		}
		edgeIDs[edge.ID] = true

		// 检查源节点
		if edge.Source == "" {
			return fmt.Errorf("边 %s 的源节点不能为空", edge.ID)
		}
		sourceNode, exists := nodeMap[edge.Source]
		if !exists {
			return fmt.Errorf("边 %s 的源节点不存在: %s", edge.ID, edge.Source)
		}

		// 检查目标节点
		if edge.Target == "" {
			return fmt.Errorf("边 %s 的目标节点不能为空", edge.ID)
		}
		targetNode, exists := nodeMap[edge.Target]
		if !exists {
			return fmt.Errorf("边 %s 的目标节点不存在: %s", edge.ID, edge.Target)
		}

		// 结束节点不能有出边
		if sourceNode.Type == "end" {
			return fmt.Errorf("结束节点不能有出边: %s -> %s", edge.Source, edge.Target)
		}

		// 开始节点不能有入边
		if targetNode.Type == "start" {
			return fmt.Errorf("开始节点不能有入边: %s -> %s", edge.Source, edge.Target)
		}

		// 不能自环
		if edge.Source == edge.Target {
			return fmt.Errorf("不允许自环: %s", edge.Source)
		}
	}

	return nil
}

// detectCycles 检测循环依赖（使用 DFS）
func (v *WorkflowValidator) detectCycles(dag *model.WorkflowDAG) error {
	// 构建邻接表
	graph := make(map[string][]string)
	for _, edge := range dag.Edges {
		graph[edge.Source] = append(graph[edge.Source], edge.Target)
	}

	// DFS 状态：0=未访问，1=正在访问，2=已完成
	visited := make(map[string]int)
	var path []string

	var dfs func(nodeID string) error
	dfs = func(nodeID string) error {
		visited[nodeID] = 1 // 标记为正在访问
		path = append(path, nodeID)

		for _, neighbor := range graph[nodeID] {
			if visited[neighbor] == 1 {
				// 发现循环
				cyclePath := append(path, neighbor)
				return fmt.Errorf("检测到循环依赖: %v", cyclePath)
			}
			if visited[neighbor] == 0 {
				if err := dfs(neighbor); err != nil {
					return err
				}
			}
		}

		visited[nodeID] = 2 // 标记为已完成
		path = path[:len(path)-1]
		return nil
	}

	// 对所有节点执行 DFS
	for _, node := range dag.Nodes {
		if visited[node.ID] == 0 {
			if err := dfs(node.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateConnectivity 验证连通性
// 确保从开始节点可以到达所有节点，所有节点可以到达结束节点
func (v *WorkflowValidator) validateConnectivity(dag *model.WorkflowDAG) error {
	// 找到开始和结束节点
	var startNode, endNode *model.WorkflowNode
	for i := range dag.Nodes {
		if dag.Nodes[i].Type == "start" {
			startNode = &dag.Nodes[i]
		}
		if dag.Nodes[i].Type == "end" {
			endNode = &dag.Nodes[i]
		}
	}

	if startNode == nil || endNode == nil {
		return fmt.Errorf("缺少开始或结束节点")
	}

	// 构建邻接表（正向和反向）
	forward := make(map[string][]string)
	backward := make(map[string][]string)
	for _, edge := range dag.Edges {
		forward[edge.Source] = append(forward[edge.Source], edge.Target)
		backward[edge.Target] = append(backward[edge.Target], edge.Source)
	}

	// BFS 检查从开始节点的可达性
	reachableFromStart := v.bfs(startNode.ID, forward)

	// BFS 检查到结束节点的可达性（反向图）
	reachableToEnd := v.bfs(endNode.ID, backward)

	// 检查是否所有节点都可达
	for _, node := range dag.Nodes {
		if !reachableFromStart[node.ID] {
			return fmt.Errorf("节点 %s 从开始节点不可达", node.ID)
		}
		if !reachableToEnd[node.ID] {
			return fmt.Errorf("节点 %s 无法到达结束节点", node.ID)
		}
	}

	return nil
}

// bfs 广度优先搜索
func (v *WorkflowValidator) bfs(startID string, graph map[string][]string) map[string]bool {
	visited := make(map[string]bool)
	queue := []string{startID}
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range graph[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return visited
}

// TopologicalSort 拓扑排序
// 返回按依赖顺序排列的节点 ID 列表
func (v *WorkflowValidator) TopologicalSort(dag *model.WorkflowDAG) ([]string, error) {
	// 构建邻接表和入度表
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// 初始化所有节点的入度为 0
	for _, node := range dag.Nodes {
		inDegree[node.ID] = 0
	}

	// 构建图和计算入度
	for _, edge := range dag.Edges {
		graph[edge.Source] = append(graph[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// 使用队列进行拓扑排序（Kahn 算法）
	var queue []string
	for _, node := range dag.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// 减少相邻节点的入度
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 如果排序结果的节点数不等于总节点数，说明有循环
	if len(sorted) != len(dag.Nodes) {
		return nil, fmt.Errorf("DAG 存在循环依赖")
	}

	return sorted, nil
}

