# 飞书机器人 Cordon 功能修复与增强

## 🐛 问题描述

用户反馈三个问题：

1. `/node cordon` 操作成功后实际节点并未被禁止调度
2. `/node cordon` 缺少禁止调度原因选项
3. `/node nodes` 没有显示节点类型（如 master）

## 🔍 根本原因

### 问题 1：Cordon 不生效
- `handleCordon` 和 `handleUncordon` 只返回成功消息，但没有调用实际的节点服务
- 代码中有 `TODO` 注释，表示功能未实现

### 问题 2：无原因选项
- 虽然支持原因参数，但没有给用户提供常用原因的参考

### 问题 3：缺少节点类型
- 节点列表卡片未显示节点的角色信息（master/worker）

## ✅ 解决方案

### 1. 实现实际的 Cordon/Uncordon 操作

#### 更新接口定义

**文件**: `backend/internal/service/feishu/feishu.go`

```go
type NodeServiceInterface interface {
    List(req interface{}, userID uint) (interface{}, error)
    Get(req interface{}, userID uint) (interface{}, error)
    Cordon(req interface{}, userID uint) error      // 新增
    Uncordon(req interface{}, userID uint) error    // 新增
}
```

#### 更新适配器

**文件**: `backend/internal/service/services.go`

```go
func (a *nodeServiceAdapter) Cordon(req interface{}, userID uint) error {
    cordonReq, ok := req.(node.CordonRequest)
    if !ok {
        return fmt.Errorf("invalid request type")
    }
    return a.svc.Cordon(cordonReq, userID)
}

func (a *nodeServiceAdapter) Uncordon(req interface{}, userID uint) error {
    uncordonReq, ok := req.(node.CordonRequest)
    if !ok {
        return fmt.Errorf("invalid request type")
    }
    return a.svc.Uncordon(uncordonReq, userID)
}
```

#### 实现 Cordon 调用

**文件**: `backend/internal/service/feishu/command_node.go`

```go
func (h *NodeCommandHandler) handleCordon(ctx *CommandContext) (*CommandResponse, error) {
    // ... 获取集群和节点名称 ...
    
    // 参数不足时显示帮助
    if len(ctx.Command.Args) < 1 {
        return &CommandResponse{
            Card: BuildCordonHelpCard(),
        }, nil
    }

    nodeName := ctx.Command.Args[0]
    reason := ""
    if len(ctx.Command.Args) > 1 {
        reason = joinArgs(ctx.Command.Args[1:]) // 合并多个参数为原因
    }

    // 调用实际的节点服务
    err = ctx.Service.nodeService.Cordon(node.CordonRequest{
        ClusterName: clusterName,
        NodeName:    nodeName,
        Reason:      reason,
    }, ctx.UserMapping.SystemUserID)

    if err != nil {
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("禁止调度节点失败: %s", err.Error())),
        }, nil
    }

    // 成功消息包含原因
    reasonText := ""
    if reason != "" {
        reasonText = fmt.Sprintf("\n原因: %s", reason)
    }

    return &CommandResponse{
        Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功设置为禁止调度\n\n节点: %s\n集群: %s%s", nodeName, clusterName, reasonText)),
    }, nil
}
```

#### 实现 Uncordon 调用

```go
func (h *NodeCommandHandler) handleUncordon(ctx *CommandContext) (*CommandResponse, error) {
    // ... 获取集群和节点名称 ...

    err = ctx.Service.nodeService.Uncordon(node.CordonRequest{
        ClusterName: clusterName,
        NodeName:    nodeName,
    }, ctx.UserMapping.SystemUserID)

    if err != nil {
        return &CommandResponse{
            Card: BuildErrorCard(fmt.Sprintf("恢复调度节点失败: %s", err.Error())),
        }, nil
    }

    return &CommandResponse{
        Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功恢复调度\n\n节点: %s\n集群: %s", nodeName, clusterName)),
    }, nil
}
```

### 2. 添加禁止调度原因帮助

#### 创建帮助卡片

**文件**: `backend/internal/service/feishu/card_builder.go`

```go
func BuildCordonHelpCard() string {
    elements := []interface{}{
        // 用法说明
        map[string]interface{}{
            "tag": "markdown",
            "content": "**📋 用法**\n```\n/node cordon <节点名> [原因]\n```",
        },
        // 常用原因
        map[string]interface{}{
            "tag": "markdown",
            "content": "**🔖 常用原因**（可直接复制使用）",
        },
        // 6个常用原因选项
        // - 🔧 维护
        // - ⬆️ 升级
        // - 🔍 故障排查
        // - ⚠️ 资源不足
        // - 🔄 重启
        // - 🧪 测试
    }
    // ...
}
```

#### 常用原因列表

| 图标 | 原因 | 用法示例 |
|------|------|---------|
| 🔧 | 维护 | `/node cordon <节点名> 维护` |
| ⬆️ | 升级 | `/node cordon <节点名> 升级` |
| 🔍 | 故障排查 | `/node cordon <节点名> 故障排查` |
| ⚠️ | 资源不足 | `/node cordon <节点名> 资源不足` |
| 🔄 | 重启 | `/node cordon <节点名> 重启` |
| 🧪 | 测试 | `/node cordon <节点名> 测试` |

### 3. 添加节点类型显示

#### 更新节点数据

**文件**: `backend/internal/service/feishu/command_node.go`

```go
// 转换为卡片需要的格式
var nodes []map[string]interface{}
for _, n := range nodeInfos {
    nodes = append(nodes, map[string]interface{}{
        "name":          n.Name,
        "ready":         n.Status == "Ready",
        "unschedulable": !n.Schedulable,
        "roles":         n.Roles, // 添加节点类型
    })
}
```

#### 更新卡片显示

**文件**: `backend/internal/service/feishu/card_builder.go`

```go
// 处理节点类型
roleText := ""
if roles, ok := node["roles"].([]string); ok && len(roles) > 0 {
    roleIcons := map[string]string{
        "master":        "👑",
        "control-plane": "👑",
        "worker":        "⚙️",
    }
    for _, role := range roles {
        icon := roleIcons[role]
        if icon == "" {
            icon = "📌"
        }
        if roleText != "" {
            roleText += " "
        }
        roleText += fmt.Sprintf("%s %s", icon, role)
    }
} else {
    roleText = "⚙️ worker"
}

nodeInfo := fmt.Sprintf("**%s**\n类型: %s\n状态: %s | 调度: %s", node["name"], roleText, status, schedulable)
```

## 📊 效果对比

### 问题 1：Cordon 不生效

#### 之前 ❌
```
/node cordon 10-9-9-33.vm.pd.sz.deeproute.ai

显示：✅ 节点已成功设置为禁止调度
实际：节点仍然可调度（Unschedulable: false）
```

#### 现在 ✅
```
/node cordon 10-9-9-33.vm.pd.sz.deeproute.ai 维护

显示：✅ 节点已成功设置为禁止调度
      节点: 10-9-9-33.vm.pd.sz.deeproute.ai
      集群: test-k8s-cluster
      原因: 维护
实际：节点被禁止调度（SchedulingDisabled）
```

### 问题 2：无原因选项

#### 之前 ❌
```
/node cordon

显示：参数不足。用法: /node cordon <节点名> [原因]
```

#### 现在 ✅
```
/node cordon

显示：💡 节点禁止调度指南
      
      📋 用法
      /node cordon <节点名> [原因]
      
      🔖 常用原因（可直接复制使用）
      🔧 维护      ⬆️ 升级
      🔍 故障排查  ⚠️ 资源不足
      🔄 重启      🧪 测试
      
      📝 示例
      /node cordon 10-9-9-28.vm.pd.sz.deeproute.ai 维护升级
```

### 问题 3：缺少节点类型

#### 之前 ❌
```
/node nodes

显示：
**10-9-9-28.vm.pd.sz.deeproute.ai**
状态: 🟢 Ready | 调度: ⛔ 禁止调度

**10-9-9-33.vm.pd.sz.deeproute.ai**
状态: 🟢 Ready | 调度: 👑 master
```

#### 现在 ✅
```
/node nodes

显示：
**10-9-9-28.vm.pd.sz.deeproute.ai**
类型: 👑 control-plane 👑 master
状态: 🟢 Ready | 调度: ⛔ 禁止调度

**10-9-9-33.vm.pd.sz.deeproute.ai**
类型: 👑 control-plane 👑 master
状态: 🟢 Ready | 调度: ✅ 可调度

**10-9-9-30.vm.pd.sz.deeproute.ai**
类型: ⚙️ worker
状态: 🟢 Ready | 调度: ✅ 可调度
```

## 🔧 技术实现细节

### 1. 参数合并
```go
func joinArgs(args []string) string {
    result := ""
    for i, arg := range args {
        if i > 0 {
            result += " "
        }
        result += arg
    }
    return result
}
```

这样用户可以输入：
```
/node cordon 节点名 维护 升级 内核  
```
会被合并为原因：`维护 升级 内核`

### 2. 节点类型图标映射
```go
roleIcons := map[string]string{
    "master":        "👑",
    "control-plane": "👑",
    "worker":        "⚙️",
}
```

### 3. 错误处理
- 服务未配置：提示"节点服务未配置"
- 操作失败：显示具体错误信息
- 节点不存在：由 node service 返回错误

## 🎯 修改文件清单

| 文件 | 修改内容 |
|------|---------|
| `backend/internal/service/feishu/feishu.go` | 添加 Cordon/Uncordon 接口方法 |
| `backend/internal/service/services.go` | 添加适配器方法 |
| `backend/internal/service/feishu/command_node.go` | 实现 Cordon/Uncordon 调用，添加节点类型 |
| `backend/internal/service/feishu/card_builder.go` | 添加帮助卡片，更新节点列表显示 |

## 🚀 测试步骤

### 1. 测试 Cordon 功能

```bash
# 在飞书中发送
/node set test-k8s-cluster
/node cordon 10-9-9-33.vm.pd.sz.deeproute.ai 维护
```

**预期**：
- 飞书显示成功消息，包含原因
- Kubernetes 节点状态变为 SchedulingDisabled
- Web 管理平台显示禁止调度状态和原因

```bash
# 验证节点状态
kubectl get nodes
```

### 2. 测试 Cordon 帮助

```bash
# 在飞书中发送
/node cordon
```

**预期**：
- 显示用法指南
- 显示 6 个常用原因选项
- 显示示例

### 3. 测试节点类型显示

```bash
# 在飞书中发送
/node nodes
```

**预期**：
- Master 节点显示：`类型: 👑 control-plane 👑 master`
- Worker 节点显示：`类型: ⚙️ worker`

### 4. 测试 Uncordon 功能

```bash
# 在飞书中发送
/node uncordon 10-9-9-33.vm.pd.sz.deeproute.ai
```

**预期**：
- 飞书显示成功消息
- Kubernetes 节点状态变为可调度
- Web 管理平台显示可调度状态

## ⚠️ 注意事项

1. **权限检查**：操作使用 `ctx.UserMapping.SystemUserID` 进行权限验证
2. **审计日志**：所有操作都会记录到审计日志（由 node service 处理）
3. **原因可选**：原因参数是可选的，但建议填写
4. **多参数支持**：原因可以是多个单词，会自动合并

## 🎉 总结

通过本次修复：

1. ✅ **Cordon 功能正常工作**：实际调用 Kubernetes API 禁止/恢复调度
2. ✅ **提供原因参考**：6 个常用原因选项，方便用户选择
3. ✅ **显示节点类型**：清晰区分 master 和 worker 节点
4. ✅ **改善用户体验**：更友好的提示和错误信息

---

**修复时间**：2025/10/17  
**影响模块**：飞书机器人节点管理
**兼容性**：向后兼容，不影响现有功能

