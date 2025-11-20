# 批量操作优化文档

## 概述

本次更新对节点批量操作功能进行了全面优化，提供了更好的进度追踪和错误反馈机制。

## 主要改进

### 1. 降低批量操作阈值

- **之前**: 需要选择 5 个以上节点才会显示进度弹窗
- **现在**: 选择 **2 个或更多**节点即显示进度弹窗
- **影响范围**: 所有批量操作（标签、污点、禁止/解除调度、驱逐）

### 2. 进度弹窗完全重新设计

#### UI 布局
- **宽度**: 从 500px 增加到 800px
- **布局**: 全新三栏卡片式设计

#### 三栏状态展示

```
┌─────────────────────────────────────────────────────┐
│                  批量操作进度                         │
│  [████████████████████░░░░░░░░] 80%                 │
│  8 / 10 个节点已完成                                  │
├─────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ 处理中(1) │  │ 已成功(8) │  │ 已失败(1) │          │
│  ├──────────┤  ├──────────┤  ├──────────┤          │
│  │⟳ node-9  │  │✓ node-1  │  │✗ node-3  │          │
│  │          │  │✓ node-2  │  │(悬停查看)│          │
│  │          │  │...       │  │          │          │
│  └──────────┘  └──────────┘  └──────────┘          │
├─────────────────────────────────────────────────────┤
│ 总计 10 个节点：成功 8 个，失败 1 个                   │
│                                      [关闭]          │
└─────────────────────────────────────────────────────┘
```

#### 关键特性

1. **强制手动关闭**
   - 禁用遮罩层点击关闭 (`:close-on-click-modal="false"`)
   - 禁用 ESC 键关闭 (`:close-on-press-escape="false"`)
   - 隐藏右上角 X 按钮 (`:show-close="false"`)
   - 必须点击底部"关闭"按钮

2. **实时节点状态**
   - **处理中**: 蓝色背景，旋转的 loading 图标
   - **已成功**: 绿色背景，✓ 图标
   - **已失败**: 红色背景，✗ 图标，可悬停查看错误详情

3. **流畅动画**
   - 节点状态变化时的过渡动画
   - 列表项淡入淡出效果
   - Loading 图标旋转动画

4. **错误详情**
   - 点击或悬停失败节点可查看完整错误信息
   - 使用 Popover 组件展示错误详情

### 3. 后端数据结构增强

#### 新增模型

```go
// NodeError 节点错误信息
type NodeError struct {
    NodeName string `json:"node_name"`
    Error    string `json:"error"`
}
```

#### 增强的字段

**ProgressTask**:
- `SuccessNodes` (TEXT): 成功节点列表（JSON）
- `FailedNodes` (TEXT): 失败节点及错误信息（JSON）

**ProgressMessage**:
- `CurrentNode` (VARCHAR): 当前处理的节点
- `SuccessNodes` (TEXT): 成功节点列表（JSON）
- `FailedNodes` (TEXT): 失败节点及错误信息（JSON）

### 4. 批量处理逻辑优化

#### 错误处理策略

- **继续执行**: 单个节点失败不会中断整个批量操作
- **详细记录**: 每个失败节点的错误信息都会被记录
- **实时更新**: 成功和失败列表实时更新到前端

#### 进度追踪

```go
// 处理过程中实时追踪
for _, nodeName := range nodeNames {
    err := processNode(nodeName)
    if err != nil {
        failedNodes = append(failedNodes, NodeError{
            NodeName: nodeName,
            Error: err.Error(),
        })
    } else {
        successNodes = append(successNodes, nodeName)
    }
    // 更新进度
    updateProgress(successNodes, failedNodes)
}
```

### 5. WebSocket 消息增强

#### 新的消息格式

```json
{
  "task_id": "label_batch_123_1234567890",
  "type": "progress",
  "action": "batch_label",
  "current": 8,
  "total": 10,
  "progress": 80,
  "current_node": "node-9",
  "success_nodes": ["node-1", "node-2", "node-4", ...],
  "failed_nodes": [
    {
      "node_name": "node-3",
      "error": "connection timeout"
    }
  ],
  "message": "正在处理节点 node-9 (8/10)",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## 数据库迁移

### 迁移文件

- `023_add_node_tracking_to_progress.sql` (PostgreSQL)
- `023_add_node_tracking_to_progress_sqlite.sql` (SQLite)

### 迁移内容

1. 添加新字段到 `progress_tasks` 表
2. 添加新字段到 `progress_messages` 表
3. 创建索引以提高查询性能

### 运行迁移

```bash
# 开发环境
cd backend
go run tools/migrate.go up

# Kubernetes 环境
kubectl apply -f deploy/k8s/migration-job.yaml
```

## 影响的批量操作

所有以下操作现在都支持增强的进度追踪（阈值 >= 2 个节点）：

### 节点操作
- ✅ 批量禁止调度 (Batch Cordon)
- ✅ 批量解除调度 (Batch Uncordon)
- ✅ 批量驱逐节点 (Batch Drain)

### 标签操作
- ✅ 批量添加标签
- ✅ 批量删除标签

### 污点操作
- ✅ 批量添加污点
- ✅ 批量删除污点

## 修改的文件

### 后端

```
backend/internal/model/task.go                        # 数据模型
backend/internal/service/progress/progress.go         # 进度服务
backend/internal/service/progress/database.go         # 数据库进度服务
backend/migrations/023_add_node_tracking_to_progress.sql       # PostgreSQL 迁移
backend/migrations/023_add_node_tracking_to_progress_sqlite.sql # SQLite 迁移
```

### 前端

```
frontend/src/components/common/ProgressDialog.vue    # 进度弹窗组件
frontend/src/views/nodes/NodeList.vue                # 节点列表页面
```

## 测试建议

### 1. 成功场景
- 选择 2 个节点执行批量操作
- 确认进度弹窗正常显示
- 确认所有节点显示在"已成功"栏

### 2. 失败场景
- 选择包含无效节点的批量操作
- 确认失败节点显示在"已失败"栏
- 悬停查看错误详情是否正确显示

### 3. 部分失败场景
- 选择 5+ 个节点，其中部分会失败
- 确认成功和失败节点分别显示
- 确认操作继续处理所有节点
- 确认最终统计信息正确

### 4. 用户体验
- 尝试点击遮罩层关闭（应该无效）
- 尝试按 ESC 键关闭（应该无效）
- 确认只能通过"关闭"按钮关闭

## 向后兼容性

- ✅ 旧的批量操作 API 仍然可用（单节点操作）
- ✅ 数据库字段可为空，不影响现有数据
- ✅ 前端优雅降级，旧消息格式仍可处理

## 性能优化

- 节点列表最大高度限制（240px），超出滚动
- 使用虚拟滚动（transition-group）优化大量节点
- 消息合并，减少 WebSocket 流量
- 数据库索引优化查询性能

## 已知限制

1. **节点数量**: 建议单次批量操作不超过 100 个节点
2. **WebSocket 重连**: 最多尝试 5 次重连
3. **消息保留**: 完成的任务消息保留 60 秒后清理

## 未来改进方向

1. **取消功能**: 支持中途取消批量操作
2. **暂停/恢复**: 支持暂停和恢复批量操作
3. **优先级**: 支持设置节点处理优先级
4. **导出报告**: 支持导出批量操作结果报告
5. **批量重试**: 一键重试所有失败节点

## 更新日志

### v1.0.0 (2025-01-15)
- 初始实现批量操作进度优化
- 降低批量操作阈值到 2 个节点
- 重新设计进度弹窗 UI
- 添加成功/失败节点追踪
- 优化错误处理机制

---

**文档版本**: 1.0.0  
**最后更新**: 2025-01-15  
**作者**: Kube Node Manager Team

