# 飞书机器人集成真实数据 - 实施说明

## 🎯 目标

将飞书机器人从使用硬编码示例数据改为调用实际的集群和节点服务获取真实数据。

## ✅ 已完成的基础工作

1. ✅ 在 `feishu.Service` 中添加了 `ClusterServiceInterface` 和 `NodeServiceInterface`
2. ✅ 在 `services.go` 中设置了服务依赖关系
3. ✅ 创建了会话管理机制（`feishu_user_sessions` 表）

## 📝 需要完成的工作

由于代码量较大，建议分步完成。以下是详细的实施步骤：

### 步骤 1：修改 `handleListClusters` 获取真实集群列表

**当前代码**（`command_node.go`第40-59行）：
```go
func (h *NodeCommandHandler) handleListClusters(ctx *CommandContext) (*CommandResponse, error) {
    // 硬编码的示例数据
    clusters := []map[string]interface{}{
        {"name": "default", "status": "健康", "nodes": 2},
        {"name": "test-k8s-cluster", "status": "健康", "nodes": 2},
    }
    return &CommandResponse{
        Card: BuildClusterListCard(clusters),
    }, nil
}
```

**需要改为**：
```go
func (h *NodeCommandHandler) handleListClusters(ctx *CommandContext) (*CommandResponse, error) {
    // 调用实际的集群服务
    if ctx.Service.clusterService == nil {
        return &CommandResponse{
            Card: BuildErrorCard("集群服务未配置"),
        }, nil
    }
    
    // 调用集群服务获取列表（使用系统用户ID）
    result, err := ctx.Service.clusterService.List(struct {
        Page     int
        PageSize int
        Name     string
        Status   string
    }{
        Page:     1,
        PageSize: 100, // 获取所有集群
    }, ctx.UserMapping.SystemUserID)
    
    if err != nil {
        ctx.Service.logger.Error("获取集群列表失败: %v", err)
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("获取集群列表失败: %s", err.Error())),
        }, nil
    }
    
    // 类型断言
    listResp, ok := result.(*cluster.ListResponse)
    if !ok {
        return &CommandResponse{
            Card: BuildErrorCard("数据格式错误"),
        }, nil
    }
    
    // 转换为卡片需要的格式
    var clusters []map[string]interface{}
    for _, c := range listResp.Clusters {
        status := "健康"
        if c.Status != "active" {
            status = "不可用"
        }
        
        clusters = append(clusters, map[string]interface{}{
            "name":   c.Name,
            "status": status,
            "nodes":  c.NodeCount,
        })
    }
    
    if len(clusters) == 0 {
        return &CommandResponse{
            Card: BuildErrorCard("系统中没有配置集群\n\n请先在 Web 界面添加集群配置"),
        }, nil
    }
    
    return &CommandResponse{
        Card: BuildClusterListCard(clusters),
    }, nil
}
```

### 步骤 2：修改 `handleListNodes` 获取真实节点列表

**当前代码**（`command_node.go`第87-120行）：
```go
func (h *NodeCommandHandler) handleListNodes(ctx *CommandContext) (*CommandResponse, error) {
    // 硬编码的示例数据
    nodes := []map[string]interface{}{
        {"name": "node-1", "ready": true, "unschedulable": false},
        {"name": "node-2", "ready": true, "unschedulable": true},
    }
    return &CommandResponse{
        Card: BuildNodeListCard(nodes, clusterName),
    }, nil
}
```

**需要改为**：
```go
func (h *NodeCommandHandler) handleListNodes(ctx *CommandContext) (*CommandResponse, error) {
    // 获取用户当前选择的集群
    clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
    if err != nil {
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("获取当前集群失败: %s", err.Error())),
        }, nil
    }

    if clusterName == "" {
        return &CommandResponse{
            Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /node list 查看集群列表\n然后使用 /node set <集群名> 选择集群"),
        }, nil
    }

    // 调用节点服务获取真实数据
    if ctx.Service.nodeService == nil {
        return &CommandResponse{
            Card: BuildErrorCard("节点服务未配置"),
        }, nil
    }
    
    // 创建节点列表请求
    result, err := ctx.Service.nodeService.List(struct {
        ClusterName string
        Status      string
        Role        string
    }{
        ClusterName: clusterName,
    }, ctx.UserMapping.SystemUserID)
    
    if err != nil {
        ctx.Service.logger.Error("获取节点列表失败: %v", err)
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("获取节点列表失败: %s\n\n请检查集群连接是否正常", err.Error())),
        }, nil
    }
    
    // 类型断言 - node.List 返回 []k8s.NodeInfo
    nodeInfos, ok := result.([]k8s.NodeInfo)
    if !ok {
        return &CommandResponse{
            Card: BuildErrorCard("节点数据格式错误"),
        }, nil
    }
    
    // 转换为卡片需要的格式
    var nodes []map[string]interface{}
    for _, n := range nodeInfos {
        nodes = append(nodes, map[string]interface{}{
            "name":          n.Name,
            "ready":         n.Status == "Ready",
            "unschedulable": !n.Schedulable,
        })
    }
    
    if len(nodes) == 0 {
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("集群 %s 中没有节点", clusterName)),
        }, nil
    }

    return &CommandResponse{
        Card: BuildNodeListCard(nodes, clusterName),
    }, nil
}
```

## 🔧 类型定义问题

由于跨package调用，需要处理类型引用。有两种解决方案：

### 方案 1：使用 interface{} + 类型断言（推荐）

在 `feishu/feishu.go` 中已经使用了 interface{}，这样可以避免循环依赖。

### 方案 2：定义共享的数据结构

在 `command_node.go` 中需要import相关类型：
```go
import (
    "kube-node-manager/internal/service/cluster"
    "kube-node-manager/internal/service/k8s"
    "kube-node-manager/internal/service/node"
)
```

## ⚠️ 注意事项

1. **权限问题**：使用 `ctx.UserMapping.SystemUserID` 作为用户ID调用服务
2. **错误处理**：需要友好地处理各种错误情况
3. **空数据**：如果没有集群或节点，给出友好提示
4. **类型转换**：service返回的类型需要转换为卡片需要的格式

## 🚀 实施步骤

由于代码修改较多，建议：

1. 先完成 `handleListClusters` 的修改并测试
2. 然后完成 `handleListNodes` 的修改并测试
3. 最后修改其他命令（info, cordon, uncordon）

或者，我可以立即帮您完成所有修改。请确认是否继续？

## 📊 预期效果

修改完成后：
- `/node list` 将显示系统中配置的所有真实集群
- `/node set <集群名>` 后，`/node nodes` 将显示该集群的真实节点
- 节点名称将是真实的节点名（如 `10-9-9-28.vm.pd.sz.deeproute.ai`）
- 节点状态将反映实际的 Kubernetes 状态

---

**需要我继续完成所有代码修改吗？**

