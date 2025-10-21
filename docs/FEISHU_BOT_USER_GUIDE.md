# Kube 管理机器人使用指南

## 📋 目录

- [简介](#简介)
- [快速开始](#快速开始)
- [命令列表](#命令列表)
- [详细使用说明](#详细使用说明)
  - [节点管理](#节点管理)
  - [集群管理](#集群管理)
  - [标签管理](#标签管理)
  - [污点管理](#污点管理)
  - [批量操作](#批量操作)
  - [快捷命令](#快捷命令)
  - [审计日志](#审计日志)
- [交互式操作](#交互式操作)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 简介

Kube 管理机器人是一个功能强大的 Kubernetes 集群管理工具，通过飞书聊天界面提供便捷的集群和节点管理功能。

### 主要特性

- 🖥️ **节点管理** - 查看、搜索、禁止/恢复调度
- 🏢 **集群管理** - 查看集群列表、状态、快速切换
- 🏷️ **标签管理** - 添加、删除、查看节点标签
- ⚠️ **污点管理** - 管理节点污点，支持三种 Effect
- 📦 **批量操作** - 一次操作多个节点
- ⚡ **快捷命令** - 快速查看集群状态和问题节点
- 📊 **交互式卡片** - 点击按钮即可执行操作
- 📝 **审计日志** - 完整的操作记录和追溯

### 适用场景

- 日常运维巡检
- 节点维护管理
- 紧急故障处理
- 集群状态监控
- 批量节点操作

---

## 快速开始

### 1. 权限要求

- ✅ 需要是系统管理员（Admin）角色
- ✅ 飞书账号已与系统账号绑定
- ✅ 有权限访问目标 Kubernetes 集群

### 2. 第一次使用

```bash
# 1. 查看帮助
/help

# 2. 查看集群列表
/cluster list

# 3. 选择要操作的集群
/cluster set test-k8s-cluster

# 4. 查看节点列表
/node list
```

### 3. 获取帮助

```bash
/help              # 查看所有命令
/help node         # 节点命令帮助
/help label        # 标签管理帮助
/help taint        # 污点管理帮助
/help batch        # 批量操作帮助
/help quick        # 快捷命令帮助
```

---

## 命令列表

### 节点管理命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/node list` | 查看节点列表 | `/node list` |
| `/node list <关键词>` | 模糊搜索节点 | `/node list 10-3` |
| `/node info <节点名>` | 查看节点详情 | `/node info node-1` |
| `/node cordon <节点名> [原因]` | 禁止节点调度 | `/node cordon node-1 系统维护` |
| `/node uncordon <节点名>` | 恢复节点调度 | `/node uncordon node-1` |

### 集群管理命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/cluster list` | 查看集群列表 | `/cluster list` |
| `/cluster set <集群名>` | 切换当前集群 | `/cluster set prod` |
| `/cluster status [集群名]` | 查看集群状态 | `/cluster status` |

### 标签管理命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/label list <节点名>` | 查看节点标签 | `/label list node-1` |
| `/label add <节点名> <key>=<value>` | 添加标签 | `/label add node-1 env=prod` |
| `/label add <节点名> <k1>=<v1>,<k2>=<v2>` | 批量添加标签 | `/label add node-1 env=prod,tier=frontend` |
| `/label remove <节点名> <key>` | 删除标签 | `/label remove node-1 env` |

### 污点管理命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/taint list <节点名>` | 查看节点污点 | `/taint list node-1` |
| `/taint add <节点名> <key>=<value>:<effect>` | 添加污点 | `/taint add node-1 maintenance=true:NoSchedule` |
| `/taint remove <节点名> <key>` | 删除污点 | `/taint remove node-1 maintenance` |

**Effect 类型**:
- `NoSchedule` - 不调度新 Pod
- `PreferNoSchedule` - 尽量不调度新 Pod
- `NoExecute` - 不调度新 Pod 且驱逐现有 Pod（⚠️ 危险操作）

### 批量操作命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/node batch cordon <nodes> [原因]` | 批量禁止调度 | `/node batch cordon node-1,node-2,node-3 系统维护` |
| `/node batch uncordon <nodes>` | 批量恢复调度 | `/node batch uncordon node-1,node-2,node-3` |

**节点列表格式**: 用逗号分隔，如 `node-1,node-2,node-3`

### 快捷命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/quick status` | 当前集群概览 | `/quick status` |
| `/quick nodes` | 显示问题节点 | `/quick nodes` |
| `/quick health` | 所有集群健康检查 | `/quick health` |

### 审计日志命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/audit logs` | 查看操作日志 | `/audit logs` |

---

## 详细使用说明

### 节点管理

#### 查看节点列表

```bash
# 查看当前集群所有节点
/node list

# 模糊搜索节点（支持部分节点名）
/node list 10-3          # 搜索包含 "10-3" 的节点
/node list master        # 搜索包含 "master" 的节点
/node list worker-01     # 搜索包含 "worker-01" 的节点
```

**返回信息**:
- 节点名称
- 状态（🟢 Ready / 🔴 NotReady）
- 调度状态（✅ 可调度 / ⛔ 禁止调度）
- 节点类型（control-plane / worker / 自定义类型）
- 交互式按钮（详情、禁止调度、恢复调度）

#### 查看节点详情

```bash
/node info 10-1-2-15.desay.thor
```

**返回信息**:
- 基本信息：名称、状态、调度状态、IP 地址
- 系统信息：容器运行时、内核版本、操作系统
- 资源信息：
  - CPU：总量 / 可分配 / 使用量
  - 内存：总量 / 可分配 / 使用量
  - Pods：总量 / 可分配
  - GPU：总量 / 可分配 / 使用量

#### 禁止节点调度

```bash
# 基本用法
/node cordon node-1

# 添加原因说明
/node cordon node-1 系统维护
/node cordon node-1 硬件故障
/node cordon node-1 版本升级
```

**注意事项**:
- ✅ 禁止调度后，不会影响现有 Pod
- ✅ 新的 Pod 不会被调度到该节点
- ✅ 需要手动恢复调度才能重新使用

#### 恢复节点调度

```bash
/node uncordon node-1
```

---

### 集群管理

#### 查看集群列表

```bash
/cluster list
```

**返回信息**:
- 集群名称
- 状态（Healthy / Unavailable）
- 节点数量
- 交互式按钮（切换、查看状态）

#### 切换当前集群

```bash
# 命令方式
/cluster set test-k8s-cluster

# 或使用交互按钮
/cluster list
# 点击目标集群的【切换】按钮
```

**注意**: 切换集群后，所有后续操作都在新集群上执行

#### 查看集群状态

```bash
# 查看当前集群状态
/cluster status

# 查看指定集群状态
/cluster status prod-cluster
```

**返回信息**:
- 集群名称
- 健康状态
- 节点统计（总数、Ready、NotReady、禁止调度）

---

### 标签管理

#### 查看节点标签

```bash
/label list node-1
```

**标签分类显示**:
- 🏷️ **系统标签**: Kubernetes 自动添加的标签
  - `kubernetes.io/hostname`
  - `kubernetes.io/os`
  - `kubernetes.io/arch`
  - 等等

- 🔖 **用户标签**: 用户自定义的标签
  - `env=production`
  - `tier=frontend`
  - `team=platform`
  - 等等

#### 添加标签

```bash
# 添加单个标签
/label add node-1 env=production

# 批量添加标签（逗号分隔）
/label add node-1 env=production,tier=frontend,team=platform

# 添加或更新标签（如果标签已存在，会更新值）
/label add node-1 env=staging
```

**标签命名规范**:
- ✅ 推荐使用小写字母
- ✅ 可以包含数字、连字符、下划线
- ✅ 建议使用有意义的键名
- ❌ 避免使用 `kubernetes.io/` 前缀（系统保留）

#### 删除标签

```bash
# 删除单个标签
/label remove node-1 env

# 删除多个标签
/label remove node-1 env
/label remove node-1 tier
```

**注意**: 无法删除系统标签

---

### 污点管理

#### 查看节点污点

```bash
/taint list node-1
```

**返回信息**:
- 污点键（Key）
- 污点值（Value）
- 效果（Effect）

#### 添加污点

```bash
# NoSchedule - 不调度新 Pod
/taint add node-1 maintenance=true:NoSchedule

# PreferNoSchedule - 尽量不调度新 Pod
/taint add node-1 experimental=true:PreferNoSchedule

# NoExecute - 不调度新 Pod 且驱逐现有 Pod
/taint add node-1 critical=true:NoExecute
```

**Effect 说明**:

| Effect | 行为 | 使用场景 |
|--------|------|----------|
| `NoSchedule` | 不调度新 Pod | 节点维护、资源预留 |
| `PreferNoSchedule` | 尽量不调度新 Pod | 性能较差的节点 |
| `NoExecute` | 不调度新 Pod + 驱逐现有 Pod | ⚠️ 紧急故障处理 |

**NoExecute 警告**:
- ⚠️ 会立即驱逐节点上的所有 Pod
- ⚠️ 可能导致服务中断
- ⚠️ 建议在 Web 界面进行二次确认

#### 删除污点

```bash
/taint remove node-1 maintenance
```

---

### 批量操作

#### 批量禁止调度

```bash
# 基本用法
/node batch cordon node-1,node-2,node-3

# 添加原因
/node batch cordon node-1,node-2,node-3 系统升级
```

**使用场景**:
- 集群维护前批量禁止调度
- 多个节点同时故障
- 批量下线节点

**返回信息**:
- 总操作数
- 成功数量
- 失败数量
- 详细结果列表

#### 批量恢复调度

```bash
/node batch uncordon node-1,node-2,node-3
```

**注意事项**:
- ✅ 节点列表用逗号分隔，不要有空格
- ✅ 支持任意数量的节点
- ✅ 失败的节点会单独列出
- ✅ 建议先小批量测试，再大批量操作

---

### 快捷命令

#### 集群概览

```bash
/quick status
```

**返回信息**:
- 当前集群名称
- 节点总数
- Ready 节点数
- NotReady 节点数
- 禁止调度节点数

#### 问题节点

```bash
/quick nodes
```

**显示条件**:
- 状态为 NotReady 的节点
- 禁止调度的节点

**返回信息**:
- 问题节点数量
- 每个问题节点的名称、状态、调度状态
- 如果没有问题节点，显示 "✅ 太好了！当前没有问题节点"

#### 集群健康检查

```bash
/quick health
```

**返回信息**:
- 所有集群的健康状态
- 节点统计信息

---

### 审计日志

```bash
/audit logs
```

**记录内容**:
- 操作时间
- 操作用户
- 操作类型
- 目标对象
- 操作结果

**用途**:
- 操作追溯
- 问题排查
- 安全审计
- 合规检查

---

## 交互式操作

### 节点列表交互

执行 `/node list` 后，每个节点都有以下按钮：

| 按钮 | 功能 | 说明 |
|------|------|------|
| 📊 详情 | 查看节点详情 | 显示完整的节点信息 |
| ⛔ 禁止调度 | 禁止节点调度 | 原因：系统维护 by 用户名 |
| ✅ 恢复调度 | 恢复节点调度 | 立即生效 |

**操作流程**:
1. 输入 `/node list`
2. 点击目标节点的按钮
3. 等待操作完成
4. 查看操作结果

**优势**:
- ⚡ 无需输入节点名称
- 🖱️ 点击即可完成操作
- 📱 移动端友好
- ✅ 减少输入错误

### 集群列表交互

执行 `/cluster list` 后，每个集群都有以下按钮：

| 按钮 | 功能 | 说明 |
|------|------|------|
| 🔄 切换 | 切换到该集群 | 当前集群高亮显示 |
| 📊 状态 | 查看集群状态 | 显示详细统计 |

---

## 最佳实践

### 日常运维

#### 1. 早晨巡检

```bash
# 查看所有集群健康状态
/quick health

# 检查问题节点
/quick nodes

# 查看当前集群状态
/quick status
```

#### 2. 节点维护

```bash
# 1. 查看节点列表
/node list

# 2. 查看节点详情
/node info node-1

# 3. 禁止调度
/node cordon node-1 系统维护

# 4. 等待 Pod 迁移完成
# （通过其他方式确认）

# 5. 执行维护操作
# （物理操作或系统更新）

# 6. 恢复调度
/node uncordon node-1
```

#### 3. 批量节点下线

```bash
# 1. 搜索目标节点
/node list worker-old

# 2. 批量禁止调度
/node batch cordon worker-old-1,worker-old-2,worker-old-3 节点下线

# 3. 等待 Pod 迁移

# 4. 执行下线操作
```

### 标签管理

#### 环境标签

```bash
# 生产环境
/label add node-prod-1 env=production
/label add node-prod-1 tier=frontend

# 测试环境
/label add node-test-1 env=staging
/label add node-test-1 tier=backend
```

#### 团队标签

```bash
/label add node-1 team=platform,owner=ops,contact=ops@company.com
```

### 污点管理

#### 专用节点

```bash
# GPU 节点（只允许 GPU 任务）
/taint add gpu-node-1 nvidia.com/gpu=present:NoSchedule

# 存储节点（只允许存储任务）
/taint add storage-node-1 storage=dedicated:NoSchedule
```

#### 节点隔离

```bash
# 临时隔离
/taint add node-1 quarantine=true:NoSchedule

# 恢复
/taint remove node-1 quarantine
```

---

## 常见问题

### Q1: 为什么提示"没有权限操作"？

**原因**:
- 飞书账号未绑定系统账号
- 系统账号不是管理员角色
- 用户被禁用

**解决方案**:
1. 联系管理员绑定账号
2. 确认账号角色为 Admin
3. 检查账号状态

---

### Q2: 为什么提示"尚未选择集群"？

**原因**:
- 首次使用或切换会话
- 未执行 `/cluster set` 命令

**解决方案**:
```bash
# 查看集群列表
/cluster list

# 选择集群
/cluster set your-cluster-name
```

---

### Q3: 节点禁止调度后，现有 Pod 会怎样？

**答案**:
- ✅ 现有 Pod 不受影响，继续运行
- ❌ 新 Pod 不会被调度到该节点
- ✅ 需要手动恢复调度

---

### Q4: NoExecute 污点和 Cordon 有什么区别？

| 操作 | Cordon | NoExecute Taint |
|------|--------|----------------|
| 新 Pod | ❌ 不调度 | ❌ 不调度 |
| 现有 Pod | ✅ 继续运行 | ❌ 立即驱逐 |
| 使用场景 | 节点维护 | 紧急故障 |
| 风险等级 | 低 | ⚠️ 高 |

---

### Q5: 如何撤销误操作？

**禁止调度误操作**:
```bash
/node uncordon node-1
```

**标签误操作**:
```bash
/label remove node-1 wrong-label
```

**污点误操作**:
```bash
/taint remove node-1 wrong-taint
```

**注意**: 所有操作都有审计日志记录

---

### Q6: 批量操作失败了怎么办？

**步骤**:
1. 查看返回的错误信息
2. 检查失败的节点列表
3. 对失败的节点单独操作
4. 使用 `/audit logs` 查看详细记录

---

### Q7: 如何查看操作历史？

```bash
/audit logs
```

**包含信息**:
- 操作时间
- 操作用户
- 操作类型
- 操作结果

---

### Q8: 搜索功能支持哪些格式？

**支持的搜索**:
```bash
/node list 10-1      # IP 片段
/node list master    # 节点角色
/node list worker    # 节点角色
/node list prod      # 环境标识
/node list gpu       # 节点类型
```

**搜索特性**:
- ✅ 不区分大小写
- ✅ 支持部分匹配
- ✅ 实时过滤

---

### Q9: 交互按钮和命令有什么区别？

| 方式 | 优势 | 适用场景 |
|------|------|----------|
| 命令 | 精确控制、支持批量 | 自动化、脚本化 |
| 交互按钮 | 快捷方便、减少错误 | 日常操作、移动端 |

**建议**: 根据场景选择最合适的方式

---

### Q10: 如何获取更多帮助？

```bash
# 通用帮助
/help

# 特定命令帮助
/help node
/help label
/help taint
/help batch
/help quick
```

---

## 快速参考卡

### 常用命令速查

```bash
# 集群切换
/cluster list                           # 查看集群
/cluster set <集群名>                   # 切换集群

# 节点操作
/node list [关键词]                     # 查看/搜索节点
/node info <节点名>                     # 节点详情
/node cordon <节点名> [原因]            # 禁止调度
/node uncordon <节点名>                 # 恢复调度

# 标签管理
/label list <节点名>                    # 查看标签
/label add <节点名> <key>=<value>       # 添加标签
/label remove <节点名> <key>            # 删除标签

# 污点管理
/taint list <节点名>                    # 查看污点
/taint add <节点名> <k>=<v>:<effect>    # 添加污点
/taint remove <节点名> <key>            # 删除污点

# 批量操作
/node batch cordon <nodes> [原因]       # 批量禁止
/node batch uncordon <nodes>            # 批量恢复

# 快捷命令
/quick status                           # 集群概览
/quick nodes                            # 问题节点
/quick health                           # 健康检查
```

---

## 更新日志

### v2.0.0 (2024-10-21)

**新增功能**:
- ✅ Label 管理命令
- ✅ Taint 管理命令
- ✅ 批量操作命令
- ✅ 快捷命令
- ✅ 交互式按钮
- ✅ 模糊搜索功能
- ✅ 增强命令解析器
- ✅ 节点类型显示
- ✅ 性能优化（缓存）

**改进**:
- ✅ 统一权限错误提示
- ✅ 完善节点详情按钮
- ✅ 优化问题节点显示
- ✅ 改进帮助命令

---

## 技术支持

### 文档

- [完整技术文档](./FEISHU_BOT_IMPLEMENTATION_PROGRESS.md)
- [Label/Taint 实现](./feishu-bot-label-taint-implementation.md)
- [批量操作文档](./feishu-bot-batch-and-quick-commands.md)
- [交互式功能](./feishu-bot-interactive-and-parser.md)

### 联系方式

- 📧 技术支持：请联系平台团队
- 🐛 问题反馈：请提交工单或联系管理员
- 📖 更多文档：查看项目 docs 目录

---

**文档版本**: v2.0.0  
**更新日期**: 2024-10-21  
**维护团队**: Platform Team

