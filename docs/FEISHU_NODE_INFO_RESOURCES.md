# 飞书机器人 - 节点信息资源显示

## ✅ 功能说明

为飞书机器人的 `/node info` 命令添加资源显示功能，展示节点的 CPU、内存、POD 和 GPU 资源信息。

## 🔧 技术实现

### 修改文件

1. **`backend/internal/service/feishu/card_builder.go`**
   - 修改 `BuildNodeInfoCard` 函数，添加资源显示卡片
   - 新增 `getStringValue` 辅助函数

2. **`backend/internal/service/feishu/command_node.go`**
   - 修改 `handleNodeInfo` 函数，传递资源信息到卡片构建器

### 核心代码

#### 1. 资源信息传递

```go
// command_node.go - handleNodeInfo
// 添加资源信息
capacity := map[string]interface{}{
    "cpu":    foundNode.Capacity.CPU,
    "memory": foundNode.Capacity.Memory,
    "pods":   foundNode.Capacity.Pods,
    "gpu":    foundNode.Capacity.GPU,
}
allocatable := map[string]interface{}{
    "cpu":    foundNode.Allocatable.CPU,
    "memory": foundNode.Allocatable.Memory,
    "pods":   foundNode.Allocatable.Pods,
    "gpu":    foundNode.Allocatable.GPU,
}
nodeInfo["capacity"] = capacity
nodeInfo["allocatable"] = allocatable

// 添加使用量信息（如果有）
if foundNode.Usage != nil {
    nodeInfo["cpu_usage"] = foundNode.Usage.CPU
    nodeInfo["memory_usage"] = foundNode.Usage.Memory
}
```

#### 2. 资源显示卡片

```go
// card_builder.go - BuildNodeInfoCard
// 添加资源显示
if capacity, ok := node["capacity"].(map[string]interface{}); ok {
    if allocatable, ok := node["allocatable"].(map[string]interface{}); ok {
        // CPU
        cpuCapacity := getStringValue(capacity, "cpu")
        cpuAllocatable := getStringValue(allocatable, "cpu")
        cpuUsage := getStringValue(node, "cpu_usage")
        if cpuUsage == "" {
            cpuUsage = "N/A"
        }

        // Memory
        memCapacity := getStringValue(capacity, "memory")
        memAllocatable := getStringValue(allocatable, "memory")
        memUsage := getStringValue(node, "memory_usage")
        if memUsage == "" {
            memUsage = "N/A"
        }

        // Pods
        podsCapacity := getStringValue(capacity, "pods")
        podsAllocatable := getStringValue(allocatable, "pods")

        // GPU
        gpuCapacity := "0"
        gpuAllocatable := "0"
        if gpuMap, ok := capacity["gpu"].(map[string]interface{}); ok && len(gpuMap) > 0 {
            for _, v := range gpuMap {
                if val, ok := v.(string); ok {
                    gpuCapacity = val
                    break
                }
            }
        }

        resourceContent := fmt.Sprintf(`🟢 **CPU**: %s / %s / %s
🔵 **内存**: %s / %s / %s
🟣 **POD**: %s / %s / N/A
🔴 **GPU**: %s / %s / N/A`,
            cpuCapacity, cpuAllocatable, cpuUsage,
            memCapacity, memAllocatable, memUsage,
            podsCapacity, podsAllocatable,
            gpuCapacity, gpuAllocatable,
        )
    }
}
```

## 📊 显示效果

### 命令使用

```bash
/node info 10-9-9-54.vm.pd.sz.deeproute.ai
```

### 卡片显示

```
┌────────────────────────────────────────┐
│        🖥️ 节点详情                     │
├────────────────────────────────────────┤
│ **节点名称**: 10-9-9-54.vm.pd.sz.deeproute.ai
│ **状态**: 🟢 Ready
│ **调度状态**: ✅ 可调度
│ **IP 地址**: 10.9.9.54
│ **容器运行时**: containerd://1.6.34
│ **内核版本**: 5.15.0-105-generic
│ **操作系统**: Ubuntu 20.04.6 LTS
│
│ ─────────────────────────────────────
│
│ **💾 资源显示**
│
│ 💡 总量 / 可分配 / 使用量
│
│ 🟢 **CPU**: 256 / 244 / 3.41 核
│ 🔵 **内存**: 1.5Ti / 1.4Ti / 3.5Gi
│ 🟣 **POD**: 250 / 250 / N/A
│ 🔴 **GPU**: 8 / 8 / N/A
└────────────────────────────────────────┘
```

## 🎨 资源类型说明

| 图标 | 资源类型 | 说明 |
|------|----------|------|
| 🟢 | CPU | 处理器核心数 |
| 🔵 | 内存 | 内存容量（带单位） |
| 🟣 | POD | Pod 最大数量 |
| 🔴 | GPU | GPU 设备数量 |

## 📋 资源信息格式

每个资源显示三个值：

1. **总量 (Capacity)**: 节点的总资源容量
2. **可分配 (Allocatable)**: 可供 Pod 使用的资源量
3. **使用量 (Usage)**: 当前已使用的资源量（如果可用）

**格式**: `总量 / 可分配 / 使用量`

### 示例

```
🟢 **CPU**: 256 / 244 / 3.41 核
```

- 总量: 256 核
- 可分配: 244 核（扣除系统预留）
- 使用量: 3.41 核（当前使用）

```
🔵 **内存**: 1.5Ti / 1.4Ti / 3.5Gi
```

- 总量: 1.5Ti
- 可分配: 1.4Ti（扣除系统预留）
- 使用量: 3.5Gi（当前使用）

## 🔍 数据来源

### Kubernetes API

资源数据来自 Kubernetes Node 对象：

```go
// NodeInfo 结构
type NodeInfo struct {
    Name        string       `json:"name"`
    Capacity    ResourceInfo `json:"capacity"`    // 总资源
    Allocatable ResourceInfo `json:"allocatable"` // 可分配资源
    Usage       *ResourceUsageInfo `json:"usage,omitempty"` // 使用情况
}

// ResourceInfo 结构
type ResourceInfo struct {
    CPU    string            `json:"cpu"`
    Memory string            `json:"memory"`
    Pods   string            `json:"pods"`
    GPU    map[string]string `json:"gpu,omitempty"`
}
```

### 资源计算

- **Capacity**: 来自 `node.Status.Capacity`
- **Allocatable**: 来自 `node.Status.Allocatable`
- **Usage**: 来自 Metrics API（如果可用）

## 💡 特殊处理

### GPU 资源

GPU 资源通过 `map[string]string` 存储，支持多种 GPU 类型：

```go
// 示例
GPU: map[string]string{
    "nvidia.com/gpu": "8",
    "amd.com/gpu": "4",
}
```

在卡片中，显示第一个找到的 GPU 类型的数量。

### 使用量 N/A

如果 Metrics API 不可用或未启用，使用量会显示为 `N/A`：

```
🟢 **CPU**: 256 / 244 / N/A
🔵 **内存**: 1.5Ti / 1.4Ti / N/A
```

## ✅ 优势

1. **直观显示** - 一目了然的资源信息
2. **完整信息** - 包含总量、可分配和使用量
3. **颜色区分** - 不同资源类型使用不同颜色图标
4. **格式统一** - 与 Web 界面保持一致
5. **灵活扩展** - 支持多种 GPU 类型

## 🔄 与 Web 界面对比

### Web 界面

```
资源显示
总量 / 可分配 / 使用量

🟢 CPU
256.00 核 / 244.00 核 / 3.41 核

🔵 内存
1.5 Ti / 1.4 Ti / 3.5 Gi

🟣 POD
250 / 250 / N/A

🔴 GPU
8 / 8 / N/A
```

### 飞书机器人

```
💾 资源显示
💡 总量 / 可分配 / 使用量

🟢 **CPU**: 256 / 244 / 3.41 核
🔵 **内存**: 1.5Ti / 1.4Ti / 3.5Gi
🟣 **POD**: 250 / 250 / N/A
🔴 **GPU**: 8 / 8 / N/A
```

**一致性**: 两者显示的信息完全一致，只是格式略有不同以适应飞书卡片。

## 🔧 辅助函数

```go
// getStringValue 从 map 中安全获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
    if val, ok := m[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}
```

**作用**: 安全地从 map 中提取字符串值，避免类型断言失败。

## 📝 使用示例

### 查看节点信息

```bash
# 1. 选择集群
/cluster set test-k8s-cluster

# 2. 查看节点信息
/node info 10-9-9-54.vm.pd.sz.deeproute.ai
```

### 返回结果

```
🖥️ 节点详情

**节点名称**: 10-9-9-54.vm.pd.sz.deeproute.ai
**状态**: 🟢 Ready
**调度状态**: ✅ 可调度
**IP 地址**: 10.9.9.54
**容器运行时**: containerd://1.6.34
**内核版本**: 5.15.0-105-generic
**操作系统**: Ubuntu 20.04.6 LTS

─────────────────────────────────────

💾 资源显示

💡 总量 / 可分配 / 使用量

🟢 **CPU**: 256 / 244 / 3.41 核
🔵 **内存**: 1.5Ti / 1.4Ti / 3.5Gi
🟣 **POD**: 250 / 250 / N/A
🔴 **GPU**: 8 / 8 / N/A
```

## 🎯 应用场景

1. **快速查看节点资源** - 无需登录 Web 界面
2. **资源规划** - 了解节点可用资源
3. **故障排查** - 检查资源使用情况
4. **容量规划** - 评估集群容量

## ⚠️ 注意事项

1. **使用量显示**
   - 需要 Metrics Server 支持
   - 如果未启用，显示 N/A

2. **GPU 资源**
   - 显示第一个找到的 GPU 类型
   - 如果有多种 GPU，只显示一种

3. **单位处理**
   - 资源单位来自 Kubernetes，可能包含：
     - CPU: 核数或 millicores（如 "2000m" = 2核）
     - Memory: Ki, Mi, Gi, Ti 等

## 🚀 后续优化建议

1. **资源使用百分比**
   - 显示资源使用率（已使用/可分配）
   - 例如: `CPU: 3.41 / 244 (1.4%)`

2. **多 GPU 类型支持**
   - 显示所有 GPU 类型
   - 例如: `nvidia.com/gpu: 8, amd.com/gpu: 4`

3. **颜色提示**
   - 根据使用率改变颜色
   - 高使用率显示为红色警告

4. **资源趋势**
   - 显示资源使用趋势
   - 例如: `↑ 5%` 表示使用率上升

---

**更新时间**: 2025/10/17  
**版本**: v2.5  
**影响范围**: 飞书机器人节点信息显示  
**兼容性**: 完全兼容

