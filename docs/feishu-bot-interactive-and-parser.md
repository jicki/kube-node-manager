# 飞书机器人交互式按钮和命令解析增强实现文档

## 📋 概述

本文档记录了飞书机器人新增的**交互式按钮**和**命令解析增强**功能的实现细节。

---

## 1. 交互式按钮功能

### 1.1 功能描述

交互式按钮允许用户通过点击卡片上的按钮执行操作，无需输入命令，提高用户体验和操作效率。

### 1.2 支持的交互式卡片

#### 1.2.1 节点列表卡片（带操作按钮）

每个节点显示以下按钮：
- **📊 详情** - 查看节点详细信息（主操作按钮）
- **⛔ 禁止调度** - 禁止节点调度（默认按钮）
- **✅ 恢复调度** - 恢复节点调度（默认按钮）

按钮根据节点当前状态动态显示：
- 如果节点已禁止调度，显示"恢复调度"按钮
- 如果节点可调度，显示"禁止调度"按钮

#### 1.2.2 节点详情卡片（带操作按钮）

节点详情卡片显示以下按钮：
- **🔄 刷新** - 刷新节点信息（默认按钮）
- **✅ 恢复调度** - 恢复节点调度（主操作按钮）
- **⛔ 禁止调度** - 禁止节点调度（危险按钮）

#### 1.2.3 集群列表卡片（带切换按钮）

每个非当前集群显示以下按钮：
- **🔄 切换** - 切换到该集群（主操作按钮）
- **📊 状态** - 查看集群状态（默认按钮）

当前集群会标记为 `👉` 并且不显示按钮。

#### 1.2.4 确认操作卡片

用于危险操作的二次确认：
- **✅ 确认执行** - 确认并执行操作（危险按钮）
- **❌ 取消** - 取消操作（默认按钮）

### 1.3 按钮操作类型

| 操作类型 | 说明 | 参数 |
|---------|------|------|
| `node_info` | 查看节点详情 | node, cluster |
| `node_cordon` | 禁止节点调度 | node, cluster |
| `node_uncordon` | 恢复节点调度 | node, cluster |
| `node_refresh` | 刷新节点信息 | node, cluster |
| `cluster_switch` | 切换集群 | cluster |
| `cluster_status` | 查看集群状态 | cluster |
| `confirm_action` | 确认危险操作 | command |
| `cancel_action` | 取消操作 | 无 |

### 1.4 使用场景

#### 场景 1：快速操作节点

```bash
# 传统方式
/node list                    # 查看节点列表
/node cordon node-1 维护      # 手动输入命令

# 交互式方式
/node list                    # 查看节点列表（带按钮）
点击节点旁的【禁止调度】按钮   # 一键执行
```

#### 场景 2：快速切换集群

```bash
# 传统方式
/cluster list                 # 查看集群列表
/cluster set production       # 手动输入命令

# 交互式方式
/cluster list                 # 查看集群列表（带按钮）
点击集群旁的【切换】按钮       # 一键切换
```

---

## 2. 命令解析增强功能

### 2.1 功能描述

增强的命令解析器支持多种参数格式，提供更灵活的命令输入方式。

### 2.2 支持的参数格式

#### 2.2.1 位置参数（原有）

```bash
/node list arg1 arg2 arg3
```

#### 2.2.2 命名参数（新增）

```bash
/node list --cluster=production --namespace=default
/node cordon node-1 --reason="System maintenance"
```

#### 2.2.3 标志参数（新增）

```bash
/node list --all --force
/node drain node-1 --ignore-daemonsets
```

#### 2.2.4 短标志（新增）

```bash
/node list -a -f
/help -h
```

#### 2.2.5 组合短标志（新增）

```bash
/node list -af    # 等同于 -a -f
```

#### 2.2.6 引号支持（新增）

```bash
/node cordon node-1 --reason="System upgrade in progress"
/label add node-1 "comment=This is a test node"
```

### 2.3 短标志映射

| 短标志 | 长标志 | 说明 |
|-------|--------|------|
| `-a` | `--all` | 全部 |
| `-f` | `--force` | 强制 |
| `-h` | `--help` | 帮助 |
| `-v` | `--verbose` | 详细 |
| `-q` | `--quiet` | 安静模式 |
| `-y` | `--yes` | 自动确认 |
| `-n` | `--no` | 自动拒绝 |
| `-r` | `--recursive` | 递归 |
| `-d` | `--debug` | 调试模式 |

### 2.4 命令别名

| 别名 | 实际命令 | 说明 |
|------|---------|------|
| `ls` | `list` | 列表 |
| `get` | `info` | 获取信息 |
| `del` | `delete` | 删除 |
| `rm` | `remove` | 移除 |
| `add` | `create` | 添加/创建 |
| `sw` | `switch` | 切换 |
| `st` | `status` | 状态 |
| `log` | `logs` | 日志 |
| `h` | `help` | 帮助 |

### 2.5 使用示例

#### 示例 1：过滤节点列表

```bash
# 原方式（不支持）
/node list

# 新方式
/node list --status=Ready --role=worker
/node list --label=env=production
/node ls -a    # 使用别名和短标志
```

#### 示例 2：带原因的禁止调度

```bash
# 原方式
/node cordon node-1 系统维护

# 新方式
/node cordon node-1 --reason="系统维护升级"
```

#### 示例 3：强制操作

```bash
# 使用标志
/node drain node-1 --force --ignore-daemonsets
/node drain node-1 -f    # 使用短标志
```

---

## 3. 实现细节

### 3.1 文件结构

```
backend/internal/service/feishu/
├── card_interactive.go           # 交互式卡片构建器
├── card_action_handler.go        # 按钮操作处理器
├── command_parser_v2.go          # 增强命令解析器
└── (现有文件...)
```

### 3.2 交互式卡片构建器

**card_interactive.go**:
- `BuildNodeListCardWithActions`: 带操作按钮的节点列表卡片
- `BuildNodeInfoCardWithActions`: 带操作按钮的节点详情卡片
- `BuildClusterListCardWithActions`: 带切换按钮的集群列表卡片
- `BuildConfirmActionCard`: 危险操作确认卡片

### 3.3 按钮操作处理器

**card_action_handler.go**:
- `CardActionHandler`: 按钮操作处理器结构
- `HandleCardAction`: 处理按钮点击事件
- `handleNodeInfo`: 处理节点详情按钮
- `handleNodeCordon`: 处理禁止调度按钮
- `handleNodeUncordon`: 处理恢复调度按钮
- `handleNodeRefresh`: 处理刷新按钮
- `handleClusterSwitch`: 处理集群切换按钮
- `handleClusterStatus`: 处理集群状态按钮
- `handleConfirmAction`: 处理确认按钮
- `handleCancelAction`: 处理取消按钮

### 3.4 增强命令解析器

**command_parser_v2.go**:
- `CommandV2`: 增强命令结构
- `CommandArgsV2`: 增强参数结构
- `ParseCommandV2`: 增强命令解析函数
- `smartSplit`: 智能分割（支持引号）
- `mapShortFlag`: 短标志映射
- `ResolveAlias`: 别名解析
- Helper 方法：`HasFlag`, `GetNamed`, `GetPositional` 等

### 3.5 按钮数据格式

按钮的 `value` 字段使用 JSON 格式存储上下文数据：

```json
{
  "action": "node_cordon",
  "node": "node-1",
  "cluster": "production"
}
```

---

## 4. 集成方式

### 4.1 在命令处理器中使用交互式卡片

```go
// 在 command_node.go 中
func (h *NodeCommandHandler) handleListNodes(ctx *CommandContext) (*CommandResponse, error) {
    // 获取节点列表
    nodes := getNodes()
    
    // 返回交互式卡片
    return &CommandResponse{
        Card: BuildNodeListCardWithActions(nodes, clusterName),
    }, nil
}
```

### 4.2 处理按钮回调

```go
// 在 bot.go 中处理卡片操作回调
func (s *Service) HandleCardCallback(callbackData string, userID string) {
    handler := NewCardActionHandler(s)
    userMapping := s.GetBindingByFeishuUserID(userID)
    response, err := handler.HandleCardAction(callbackData, userMapping)
    // 发送响应
}
```

### 4.3 使用增强解析器

```go
// 使用 V2 解析器
cmd, err := ParseCommandV2("/node list --status=Ready -a")
if err != nil {
    // 处理错误
}

// 检查标志
if cmd.Args.HasFlag("all") {
    // 显示所有节点
}

// 获取命名参数
if status, ok := cmd.Args.GetNamed("status"); ok {
    // 按状态过滤
}

// 兼容旧版
oldCmd := cmd.ToCommand()  // 转换为旧格式
```

---

## 5. 安全性考虑

### 5.1 按钮操作验证

- ✅ 所有按钮操作都验证用户绑定状态
- ✅ 验证操作目标（节点名、集群名）的合法性
- ✅ 危险操作需要二次确认
- ✅ 操作审计日志记录

### 5.2 参数验证

- ✅ 参数格式验证
- ✅ 参数值范围检查
- ✅ 引号配对验证
- ✅ 特殊字符转义

---

## 6. 后续优化建议

### 6.1 交互式按钮

- [ ] 支持更多操作类型（标签管理、污点管理）
- [ ] 添加批量选择功能（多选框）
- [ ] 操作进度实时反馈
- [ ] 按钮状态管理（禁用/启用）
- [ ] 操作结果通知优化

### 6.2 命令解析

- [ ] 参数类型自动推断和转换
- [ ] 参数验证规则定义
- [ ] 子命令嵌套支持
- [ ] Tab 自动补全建议
- [ ] 命令历史和快捷输入

---

## 7. 测试用例

### 7.1 交互式按钮测试

```bash
# 测试节点列表交互
/node list
# 点击任意节点的"禁止调度"按钮
# 验证操作成功

# 测试集群切换
/cluster list
# 点击非当前集群的"切换"按钮
# 验证切换成功
```

### 7.2 命令解析测试

```bash
# 测试命名参数
/node list --cluster=prod --status=Ready

# 测试短标志
/node list -a -f

# 测试组合短标志
/node list -af

# 测试引号
/node cordon node-1 --reason="System maintenance"

# 测试别名
/node ls    # 等同于 /node list
```

---

## 8. 版本历史

### v1.2.0 (2024-10-21)

- ✅ 实现交互式按钮（节点/集群列表）
- ✅ 实现按钮操作处理器
- ✅ 实现增强命令解析器
- ✅ 添加命令别名支持
- ✅ 添加短标志支持
- ✅ 添加引号字符串支持

---

## 9. 相关文档

- [飞书机器人 Label 和 Taint 管理实现](./feishu-bot-label-taint-implementation.md)
- [飞书机器人批量操作和快捷命令](./feishu-bot-batch-and-quick-commands.md)
- [飞书机器人实现进度](./FEISHU_BOT_IMPLEMENTATION_PROGRESS.md)

---

**编写者**: AI Assistant  
**更新日期**: 2024-10-21  
**版本**: 1.2.0

