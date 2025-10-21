# 飞书机器人批量操作和快捷命令实现文档

## 📋 概述

本文档记录了飞书机器人新增的**批量操作**和**快捷命令**功能的实现细节。

---

## 1. 批量操作功能

### 1.1 功能描述

批量操作允许用户一次性对多个节点执行相同的操作，提高运维效率。

### 1.2 支持的命令

```bash
/node batch cordon <node1,node2,node3> [reason]  # 批量禁止调度
/node batch uncordon <node1,node2,node3>         # 批量恢复调度
```

### 1.3 命令参数

- `operation`: 操作类型（cordon/uncordon）
- `nodes`: 节点列表，使用逗号分隔（如：node-1,node-2,node-3）
- `reason`: 可选，禁止调度的原因（仅用于 cordon 操作）

### 1.4 使用示例

**批量禁止调度**:
```
/node batch cordon node-1,node-2,node-3 维护升级
```

**批量恢复调度**:
```
/node batch uncordon node-1,node-2,node-3
```

### 1.5 功能特性

- ✅ 支持批量 cordon/uncordon 操作
- ✅ 显示详细的执行结果（成功/失败统计）
- ✅ 失败节点会显示具体错误信息
- ✅ 成功节点列表展示（限制 10 个以内完整显示）
- ✅ 操作审计记录（每个节点单独记录）

### 1.6 结果展示

批量操作完成后，会显示以下信息：
- 操作类型和集群名称
- 总节点数、成功数、失败数
- 失败节点列表及详细错误
- 成功节点列表（少于 10 个时显示）

卡片颜色：
- 🟢 全部成功：绿色
- 🔴 全部失败：红色
- 🟠 部分失败：橙色

---

## 2. 快捷命令功能

### 2.1 功能描述

快捷命令提供快速查看集群和节点状态的能力，聚合常用信息。

### 2.2 支持的命令

```bash
/quick status  # 当前集群概览
/quick nodes   # 显示问题节点（NotReady/禁止调度）
/quick health  # 所有集群健康检查
```

### 2.3 命令详解

#### 2.3.1 /quick status

显示当前选择集群的概览信息：
- 集群名称
- 节点统计（总数、Ready、NotReady、禁止调度）
- 如果有问题节点，会显示警告提示

#### 2.3.2 /quick nodes

显示当前集群的问题节点：
- NotReady 状态的节点
- SchedulingDisabled（禁止调度）的节点
- 如果没有问题节点，显示"太好了"提示

#### 2.3.3 /quick health

执行所有集群的健康检查（简化版）：
- 显示健康检查完成信息
- 提示使用详细命令查看

### 2.4 使用场景

- **早上上班**: 使用 `/quick status` 快速查看集群状态
- **发现告警**: 使用 `/quick nodes` 快速定位问题节点
- **整体巡检**: 使用 `/quick health` 检查所有集群

---

## 3. 实现细节

### 3.1 文件结构

```
backend/internal/service/feishu/
├── command_node.go          # 扩展了批量操作处理
├── command_quick.go         # 新增：快捷命令处理器
├── card_builder.go          # 新增批量和快捷命令卡片构建器
├── command.go               # 注册新命令处理器
└── command_help.go          # 更新帮助信息
```

### 3.2 关键函数

#### 批量操作

**command_node.go**:
- `handleBatchOperation`: 批量操作入口
- `handleBatchCordon`: 批量禁止调度
- `handleBatchUncordon`: 批量恢复调度
- `parseNodeList`: 解析节点列表

**card_builder.go**:
- `BuildBatchHelpCard`: 批量操作帮助卡片
- `BuildBatchOperationResultCard`: 批量操作结果卡片

#### 快捷命令

**command_quick.go**:
- `handleQuickStatus`: 集群状态概览
- `handleQuickNodes`: 问题节点列表
- `handleQuickHealth`: 健康检查

**card_builder.go**:
- `BuildQuickHelpCard`: 快捷命令帮助卡片
- `BuildQuickStatusCard`: 状态概览卡片
- `BuildQuickNodesCard`: 问题节点卡片
- `BuildQuickHealthCard`: 健康检查卡片

### 3.3 错误处理

所有批量操作和快捷命令都包含：
- ✅ 用户绑定验证
- ✅ 集群选择验证
- ✅ 服务可用性检查
- ✅ 数据格式验证
- ✅ 详细错误提示

---

## 4. 使用指南

### 4.1 获取帮助

```bash
/help                 # 查看所有命令
/help batch          # 查看批量操作帮助
/help quick          # 查看快捷命令帮助
```

### 4.2 批量操作流程

1. 选择集群：`/cluster set <集群名>`
2. 执行批量操作：`/node batch cordon node-1,node-2 维护`
3. 查看结果卡片，确认成功和失败节点
4. 如有失败，根据错误提示处理

### 4.3 快捷命令流程

1. 快速查看当前集群：`/quick status`
2. 发现问题节点：查看是否有警告
3. 查看详情：`/quick nodes`
4. 处理问题节点

---

## 5. 注意事项

### 5.1 批量操作

- ⚠️ 节点名称之间用逗号分隔，不要有空格
- ⚠️ 批量操作会逐个执行，部分失败不影响其他节点
- ⚠️ 每个节点操作都会生成单独的审计日志
- ⚠️ 建议一次批量操作不超过 20 个节点

### 5.2 快捷命令

- ⚠️ `/quick status` 和 `/quick nodes` 需要先选择集群
- ⚠️ 快捷命令提供概览信息，详细信息请使用具体命令
- ℹ️ `/quick health` 功能当前为简化版本

---

## 6. 后续优化建议

### 6.1 批量操作

- [ ] 支持标签选择器批量操作（如：role=worker）
- [ ] 批量操作进度实时反馈
- [ ] 支持批量标签和污点操作
- [ ] 批量操作事务性（全部成功或全部回滚）

### 6.2 快捷命令

- [ ] `/quick health` 显示所有集群的详细健康信息
- [ ] `/quick drain` 快速驱逐（cordon + drain）
- [ ] 快捷命令支持过滤参数
- [ ] 添加资源使用率快速查看

---

## 7. 版本历史

### v1.0.0 (2024-10-21)

- ✅ 实现批量 cordon/uncordon 操作
- ✅ 实现快捷命令 status/nodes/health
- ✅ 添加批量操作结果卡片
- ✅ 添加快捷命令帮助卡片
- ✅ 更新主帮助信息

---

## 8. 相关文档

- [飞书机器人 Label 和 Taint 管理实现](./feishu-bot-label-taint-implementation.md)
- [飞书机器人功能优化与新增分析](./-----------.plan.md)

---

**编写者**: AI Assistant  
**更新日期**: 2024-10-21  
**版本**: 1.0.0

