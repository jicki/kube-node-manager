# Kubernetes 资源管理策略

## 概述

本文档详细说明了 Kube Node Manager 如何管理不同类型的 Kubernetes 资源，以及为何采用不同的策略。

## 资源分类

### 1. 节点资源（Nodes）

#### 策略：✅ Informer + SmartCache（实时同步）

#### 特点
- **变化频率**：低（节点很少被添加或删除）
- **数量规模**：有限（通常 10-1000 个节点）
- **数据大小**：中等（每个节点 10-20KB）
- **查询频率**：高（用户频繁查看节点列表和详情）
- **实时性要求**：高（需要实时看到状态变化）

#### 实现方式

```go
// 1. Informer 实时监听
informer.StartInformer(clusterName, clientset)

// 2. SmartCache 存储数据
smartCache.SetNode(clusterName, node)

// 3. 查询从缓存获取
k8sNode, found := smartCache.GetNode(clusterName, nodeName)
if found {
    nodeInfo := s.nodeToNodeInfo(k8sNode)
    return nodeInfo // < 10ms 响应
}
```

#### 优势
- ✅ 查询响应时间：< 10ms
- ✅ 减少 K8s API 调用：99%
- ✅ 实时同步：1-2 秒延迟
- ✅ 降低 API Server 压力
- ✅ WebSocket 实时推送变化

#### 监听的属性
- **Schedulable（调度状态）** - 禁止/解除调度
- **Labels（标签）** - 节点标签变化
- **Taints（污点）** - 污点变化
- **Conditions（条件）** - 节点健康状态
- **Status（状态）** - Ready/NotReady
- **Resource（资源）** - CPU/内存容量（较少变化）

#### 内存占用
- 每个节点：约 10-20KB
- 100 个节点：约 1-2MB
- 1000 个节点：约 10-20MB
- **完全可接受** ✅

---

### 2. Pod 资源（Pods）

#### 策略：❌ 不使用 Informer，直接 API 查询

#### 特点
- **变化频率**：极高（Pod 频繁创建、删除、更新）
- **数量规模**：巨大（可能成千上万个）
- **数据大小**：大（每个 Pod 包含很多信息）
- **查询频率**：中（仅在特定操作时查询）
- **实时性要求**：中（不需要持续监控所有 Pod）

#### 为什么不使用 Informer？

1. **内存压力大**
   ```
   1000 个 Pod × 50KB = 50MB
   10000 个 Pod × 50KB = 500MB
   在大集群中不可接受
   ```

2. **变化太频繁**
   - Pod 每秒可能创建/删除数十个
   - Informer 会产生大量事件
   - 缓存更新压力大

3. **查询场景有限**
   - 主要用于统计节点上的 Pod 数量
   - 仅在驱逐操作时需要 Pod 列表
   - 不需要持续监控所有 Pod

#### 实现方式

```go
// 直接调用 K8s API，不使用缓存
func (s *Service) getNodesPodCounts(clusterName string) map[string]int {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    
    // 直接查询所有 Pod
    podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
    
    // 统计每个节点的 Pod 数量
    for _, pod := range podList.Items {
        if pod.Status.Phase != corev1.PodSucceeded && 
           pod.Status.Phase != corev1.PodFailed {
            podCounts[pod.Spec.NodeName]++
        }
    }
    
    return podCounts
}
```

#### 使用场景

**1. 统计节点 Pod 数量**
```go
// 批量获取多个节点的 Pod 数量（用于节点列表展示）
podCounts := s.getNodesPodCounts(clusterName, nodeNames)
```

**2. 驱逐节点前获取 Pod 列表**
```go
// 获取节点上需要驱逐的 Pod
func (s *Service) DrainNode(clusterName, nodeName string) error {
    // 直接查询节点上的 Pod
    podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
        FieldSelector: fields.SelectorFromSet(fields.Set{
            "spec.nodeName": nodeName,
        }).String(),
    })
    
    // 驱逐 Pod
    for _, pod := range podList.Items {
        s.evictPod(ctx, client, &pod)
    }
}
```

#### 优势
- ✅ 内存占用低（不缓存 Pod 数据）
- ✅ 数据始终最新（实时查询）
- ✅ 实现简单（无需复杂的缓存管理）

#### 性能优化
虽然直接查询 API，但通过以下方式优化：
- ✅ 使用 FieldSelector 过滤（只查询特定节点的 Pod）
- ✅ 批量查询（一次查询获取所有需要的数据）
- ✅ 超时控制（避免长时间等待）

---

### 3. Deployment 资源

#### 策略：📋 根据需求选择

#### 当前状态
- 本项目主要关注节点管理
- 不涉及 Deployment 的管理

#### 如果未来需要

**场景 A：实时监控 Deployment 状态**
- 使用 Informer + 缓存
- 类似节点的实现

**场景 B：偶尔查询 Deployment 列表**
- 直接 API 调用
- 类似 Pod 的实现

---

### 4. Service / ConfigMap / Secret 等资源

#### 策略：📋 根据需求选择

#### 推荐策略

| 资源类型 | 变化频率 | 数量 | 推荐策略 |
|---------|---------|------|---------|
| Services | 低 | 中 | Informer（如需实时监控） |
| ConfigMaps | 低 | 中 | 直接 API（按需查询） |
| Secrets | 低 | 中 | 直接 API（安全考虑） |
| PersistentVolumes | 低 | 小 | Informer |
| PersistentVolumeClaims | 中 | 中 | Informer |
| Namespaces | 极低 | 小 | Informer |
| Events | 极高 | 巨大 | 直接 API（仅查询近期） |

---

## 决策树

```
需要管理某个 K8s 资源？
    ↓
考虑以下因素：
    ↓
┌─────────────────────────────────────┐
│ 1. 变化频率是否低？                   │
│    (节点、Service、Namespace 等)     │
│    ↓ 是                              │
│ 2. 数量规模是否可控？                 │
│    (< 10000 个)                     │
│    ↓ 是                              │
│ 3. 需要实时监控状态变化？             │
│    ↓ 是                              │
│ 4. 查询频率是否高？                   │
│    ↓ 是                              │
│                                     │
│ ✅ 使用 Informer + 缓存              │
└─────────────────────────────────────┘
             │ 否（任何一项）
             ↓
┌─────────────────────────────────────┐
│ ❌ 使用直接 API 查询                 │
│                                     │
│ 适用于：                             │
│ - 高变化频率资源（Pod、Event）        │
│ - 大规模资源（成千上万个）            │
│ - 低查询频率场景                     │
│ - 安全敏感资源（Secret）             │
└─────────────────────────────────────┘
```

---

## 性能对比

### 节点查询（使用 Informer）

| 指标 | 数值 |
|------|------|
| 响应时间 | < 10ms |
| API 调用 | 0 次/查询 |
| 内存占用 | 1-20MB |
| 实时性 | 1-2 秒 |
| CPU 占用 | 极低 |

### Pod 查询（直接 API）

| 指标 | 数值 |
|------|------|
| 响应时间 | 100-500ms |
| API 调用 | 1 次/查询 |
| 内存占用 | 0（不缓存） |
| 实时性 | 即时 |
| CPU 占用 | 低 |

---

## 最佳实践

### 1. 使用 Informer 的最佳实践

**初始化数据**
```go
// 在启动 Informer 前，立即获取初始数据
nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
for i := range nodeList.Items {
    smartCache.SetNode(clusterName, &nodeList.Items[i])
}

// 然后启动 Informer
informerSvc.StartInformer(clusterName, clientset)
```

**事件过滤**
```go
// 只处理有意义的变化
func hasSignificantChanges(oldNode, newNode *corev1.Node) bool {
    if oldNode.Spec.Unschedulable != newNode.Spec.Unschedulable {
        return true // 调度状态变化
    }
    if !reflect.DeepEqual(oldNode.Labels, newNode.Labels) {
        return true // 标签变化
    }
    // ... 其他关键属性
    return false
}
```

**资源清理**
```go
// 集群删除时，停止 Informer 并清理缓存
func (m *Manager) UnregisterCluster(clusterName string) {
    m.informerSvc.StopInformer(clusterName)
    m.smartCache.InvalidateCluster(clusterName)
}
```

### 2. 直接 API 查询的最佳实践

**超时控制**
```go
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
```

**字段过滤**
```go
// 只查询特定节点的 Pod
podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
    FieldSelector: fields.SelectorFromSet(fields.Set{
        "spec.nodeName": nodeName,
    }).String(),
})
```

**错误处理**
```go
if err != nil {
    if apierrors.IsNotFound(err) {
        // 资源不存在
    } else if apierrors.IsForbidden(err) {
        // 权限不足
    } else {
        // 其他错误
    }
}
```

---

## 监控和日志

### Informer 监控

**关键日志**
```
INFO: Informer for cluster my-cluster started and synced
INFO: Node updated: cluster=my-cluster, node=node-1, changes=[Schedulable]
INFO: SmartCache: Updated node node-1 in cluster my-cluster
```

**性能指标**
- Informer 同步延迟
- SmartCache 命中率
- WebSocket 连接数
- 事件处理速率

### API 查询监控

**关键日志**
```
INFO: Successfully retrieved 100 pods from cluster my-cluster
WARNING: Failed to list pods for cluster my-cluster: timeout
```

**性能指标**
- API 调用次数
- API 响应时间
- API 错误率
- 超时次数

---

## 总结

### 当前实现

| 资源 | 策略 | 原因 | 状态 |
|------|------|------|------|
| Nodes | Informer + Cache | 低频变化、高频查询、需实时 | ✅ 已实现 |
| Pods | 直接 API | 高频变化、大数量、低频查询 | ✅ 已实现 |
| Deployments | - | 不在项目范围 | - |
| Services | - | 不在项目范围 | - |

### 架构优势

1. ✅ **高性能** - 节点查询 < 10ms
2. ✅ **低延迟** - 实时同步 1-2 秒
3. ✅ **低开销** - 内存占用 1-20MB
4. ✅ **可扩展** - 支持 1000+ 节点
5. ✅ **高可用** - 容错性强
6. ✅ **实时性** - WebSocket 推送

### 设计原则

1. **按需选择** - 根据资源特点选择策略
2. **性能优先** - 高频操作使用缓存
3. **内存可控** - 不缓存大规模动态资源
4. **实时性平衡** - Informer 提供 1-2 秒延迟的实时性
5. **容错性** - 即使 Informer 失败也有降级方案

这个策略确保了系统在性能、实时性和资源占用之间达到最佳平衡！🎯

