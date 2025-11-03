# Ansible 任务快速操作功能使用指南

## 功能概述

任务快速操作功能提供了收藏夹和历史记录功能，让用户可以快速访问和重新执行常用的任务配置，大幅提升日常运维效率。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **收藏功能**：收藏常用的任务、模板和清单
- ✅ **历史记录**：自动记录任务执行配置
- ✅ **快速重新执行**：基于历史记录一键重新执行任务
- ✅ **使用频率统计**：记录每个配置的使用次数
- ✅ **最近使用排序**：按最后使用时间排序
- ✅ **参数记忆**：记住上次执行的所有配置（包括 Dry Run、分批执行等）

### 使用场景

#### 1. 日常运维任务
- **场景**：每天需要执行相同的系统检查任务
- **方案**：收藏检查任务模板，一键执行
- **效率提升**：从填写表单 2 分钟减少到点击按钮 5 秒

#### 2. 紧急故障处理
- **场景**：故障发生时需要快速执行已验证的修复 Playbook
- **方案**：收藏常用的故障修复模板和清单
- **效率提升**：紧急情况下快速响应，减少人为错误

#### 3. 测试和开发
- **场景**：开发 Playbook 时需要反复测试
- **方案**：系统自动记录每次执行配置，快速重新执行
- **效率提升**：避免重复填写相同参数

#### 4. 定期维护任务
- **场景**：每月执行相同的维护任务
- **方案**：收藏维护任务配置，每月快速执行
- **效率提升**：保证配置一致性，避免遗漏参数

## 功能说明

### 1. 收藏功能

#### 可收藏的对象类型

- **任务（Task）**：收藏已完成的任务，方便查看和参考
- **模板（Template）**：收藏常用的 Playbook 模板
- **清单（Inventory）**：收藏常用的主机清单

#### API 接口

```bash
# 添加收藏
POST /api/v1/ansible/favorites
{
  "target_type": "template",  # task/template/inventory
  "target_id": 1
}

# 移除收藏
DELETE /api/v1/ansible/favorites?target_type=template&target_id=1

# 列出收藏
GET /api/v1/ansible/favorites?target_type=template
```

### 2. 任务执行历史

#### 自动记录机制

系统会在每次创建任务时自动记录以下信息：
- 任务名称
- 模板 ID
- 清单 ID
- 集群 ID
- Playbook 内容
- 额外变量
- Dry Run 配置
- 分批执行配置
- 最后使用时间
- 使用次数

#### 历史记录特点

- **智能去重**：相同配置的任务会合并为一条记录，更新使用次数
- **自动更新**：每次执行会更新最后使用时间和使用次数
- **参数完整**：保存完整的任务配置，包括所有高级选项
- **排序优化**：按最后使用时间倒序排列

#### API 接口

```bash
# 获取最近使用的任务（默认10条）
GET /api/v1/ansible/recent-tasks?limit=10

# 获取任务历史详情
GET /api/v1/ansible/task-history/{id}

# 删除任务历史
DELETE /api/v1/ansible/task-history/{id}
```

### 3. 快速重新执行

#### 使用流程

1. 在"最近使用"列表中选择一个历史记录
2. 点击"重新执行"按钮
3. 系统自动填充所有参数：
   - 任务名称（可修改）
   - 模板
   - 清单
   - 集群
   - 额外变量
   - Dry Run 设置
   - 分批执行配置
4. 确认后执行

#### 优点

- **参数一致性**：避免手动填写导致的参数错误
- **快速响应**：紧急情况下快速执行已验证的配置
- **配置追溯**：可以查看历史执行使用的配置

## 技术实现

### 后端实现

#### 1. 数据模型

```go
// 收藏记录
type AnsibleFavorite struct {
    ID         uint
    UserID     uint
    TargetType string  // task/template/inventory
    TargetID   uint
    CreatedAt  time.Time
}

// 任务执行历史
type AnsibleTaskHistory struct {
    ID              uint
    UserID          uint
    TaskName        string
    TemplateID      *uint
    InventoryID     *uint
    ClusterID       *uint
    PlaybookContent string
    ExtraVars       ExtraVars
    DryRun          bool
    BatchConfig     *BatchExecutionConfig
    LastUsedAt      time.Time
    UseCount        int
}
```

#### 2. 服务层（FavoriteService）

核心方法：
- `AddFavorite()` - 添加收藏
- `RemoveFavorite()` - 移除收藏
- `ListFavorites()` - 列出收藏
- `IsFavorite()` - 检查是否已收藏
- `AddOrUpdateTaskHistory()` - 添加或更新任务历史
- `GetRecentTaskHistory()` - 获取最近使用的任务
- `GetTaskHistory()` - 获取任务历史详情
- `DeleteTaskHistory()` - 删除任务历史
- `CleanupOldHistory()` - 清理旧历史记录

#### 3. 自动记录机制

在 `CreateTask()` 方法中添加：

```go
// 添加到任务历史（用于快速重新执行）
go func() {
    if err := s.favoriteSvc.AddOrUpdateTaskHistory(userID, task); err != nil {
        s.logger.Errorf("Failed to add task history: %v", err)
    }
}()
```

#### 4. 智能去重逻辑

```go
func (s *FavoriteService) AddOrUpdateTaskHistory(userID uint, task *model.AnsibleTask) error {
    // 查找是否存在相同配置的历史记录
    var history model.AnsibleTaskHistory
    err := s.db.Where("user_id = ? AND template_id = ? AND inventory_id = ? AND cluster_id = ?", 
        userID, task.TemplateID, task.InventoryID, task.ClusterID).
        First(&history).Error
    
    if err == gorm.ErrRecordNotFound {
        // 创建新记录
        history = model.AnsibleTaskHistory{...}
        s.db.Create(&history)
    } else {
        // 更新现有记录
        history.LastUsedAt = time.Now()
        history.UseCount++
        s.db.Save(&history)
    }
}
```

### 前端实现

#### 1. API 封装

```javascript
// 收藏管理
export function addFavorite(data)
export function removeFavorite(targetType, targetId)
export function listFavorites(targetType)

// 历史记录
export function getRecentTasks(limit = 10)
export function getTaskHistory(id)
export function deleteTaskHistory(id)
```

#### 2. UI 组件（待实现）

**组件建议**：
- `FavoriteButton.vue` - 收藏按钮组件
- `RecentTasks.vue` - 最近使用任务列表
- `QuickActions.vue` - 快速操作面板

**集成位置**：
- 任务列表：添加收藏按钮
- 模板列表：添加收藏按钮
- 清单列表：添加收藏按钮
- 任务创建对话框：添加"最近使用"快捷选项

### 数据库设计

#### ansible_favorites 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| user_id | INTEGER | 用户 ID |
| target_type | VARCHAR(50) | 目标类型 |
| target_id | INTEGER | 目标 ID |
| created_at | TIMESTAMP | 创建时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

**索引**：
- `idx_ansible_favorites_user_id`
- `idx_ansible_favorites_unique` - 唯一索引（user_id, target_type, target_id）

#### ansible_task_history 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| user_id | INTEGER | 用户 ID |
| task_name | VARCHAR(255) | 任务名称 |
| template_id | INTEGER | 模板 ID |
| inventory_id | INTEGER | 清单 ID |
| cluster_id | INTEGER | 集群 ID |
| playbook_content | TEXT | Playbook 内容 |
| extra_vars | JSONB | 额外变量 |
| dry_run | BOOLEAN | 是否 Dry Run |
| batch_config | JSONB | 分批配置 |
| last_used_at | TIMESTAMP | 最后使用时间 |
| use_count | INTEGER | 使用次数 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引**：
- `idx_ansible_task_history_user_id`
- `idx_ansible_task_history_last_used_at`

## 实施状态

### ✅ 已完成

#### 后端
- ✅ 数据模型定义
- ✅ FavoriteService 实现
- ✅ FavoriteHandler 实现
- ✅ API 路由注册
- ✅ 自动记录任务历史
- ✅ 数据库迁移文件

#### 前端
- ✅ API 接口封装

### ✅ 已实现（前端 UI）

#### 前端 UI
- ✅ 最近使用面板（TaskCenter.vue）
- ✅ 一键重新执行功能
- ✅ 任务历史记录删除
- ✅ 模板收藏按钮（TaskTemplates.vue）
- ✅ 清单收藏按钮（InventoryManage.vue）
- ✅ 收藏状态自动加载和同步

## 使用示例

### 示例 1：收藏常用模板

```bash
# 添加模板到收藏
curl -X POST http://your-server/api/v1/ansible/favorites \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "target_type": "template",
    "target_id": 5
  }'

# 查看收藏的模板
curl -X GET "http://your-server/api/v1/ansible/favorites?target_type=template" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 示例 2：获取最近使用的任务

```bash
# 获取最近10个任务配置
curl -X GET "http://your-server/api/v1/ansible/recent-tasks?limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应示例
{
  "data": [
    {
      "id": 123,
      "task_name": "系统更新",
      "template_id": 5,
      "inventory_id": 3,
      "cluster_id": 1,
      "dry_run": false,
      "batch_config": {
        "enabled": true,
        "batch_percent": 20
      },
      "last_used_at": "2025-11-03T10:30:00Z",
      "use_count": 15
    }
  ],
  "total": 10
}
```

### 示例 3：基于历史记录重新执行

```bash
# 1. 获取历史记录详情
curl -X GET http://your-server/api/v1/ansible/task-history/123 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 2. 使用历史记录中的配置创建新任务
curl -X POST http://your-server/api/v1/ansible/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "系统更新 (重新执行)",
    "template_id": 5,
    "inventory_id": 3,
    "cluster_id": 1,
    "dry_run": false,
    "batch_config": {
      "enabled": true,
      "batch_percent": 20
    }
  }'
```

## 最佳实践

### 1. 收藏管理

- **分类收藏**：按用途收藏不同类型的对象
- **定期清理**：删除不再使用的收藏
- **命名规范**：使用清晰的名称便于识别

### 2. 历史记录使用

- **参数核对**：重新执行前检查参数是否仍然适用
- **谨慎生产**：生产环境重新执行时启用 Dry Run 验证
- **定期清理**：系统会自动保留最近50条记录

### 3. 快速操作流程

```
常用操作流程：
1. 首次执行：完整填写所有参数
2. 系统自动记录到历史
3. 下次执行：从"最近使用"中选择
4. 修改必要参数（如任务名称）
5. 快速执行
```

## 性能优化

### 1. 历史记录自动清理

系统会自动清理旧的历史记录，默认保留每个用户最近50条：

```go
func (s *FavoriteService) CleanupOldHistory(userID uint, keepCount int) error {
    // 保留最近的 keepCount 条记录
    // 删除其余记录
}
```

### 2. 数据库索引优化

- 用户 ID 索引：加速按用户查询
- 最后使用时间索引：加速排序查询
- 唯一索引：防止重复收藏

### 3. 预加载关联数据

```go
query.Preload("Template").Preload("Inventory").Preload("Cluster")
```

## 未来增强

### 1. 智能推荐
- 根据使用频率推荐常用配置
- 根据时间模式推荐（如每周/每月任务）

### 2. 配置对比
- 对比两次执行的配置差异
- 查看配置变更历史

### 3. 团队共享
- 共享常用配置给团队成员
- 组织级别的配置库

### 4. 移动端优化
- 移动端快速执行
- 语音命令执行常用任务

## 常见问题

### Q1：历史记录会一直增长吗？

**A**：不会。系统会自动保留每个用户最近50条记录，旧记录会被自动清理。

### Q2：收藏的对象被删除后会怎样？

**A**：收藏记录会通过数据库的级联删除自动清理。

### Q3：历史记录是否占用大量存储空间？

**A**：
- 相同配置会合并为一条记录（去重）
- 自动清理机制限制总数
- JSONB 字段压缩存储
- 通常每个用户占用 < 1MB

### Q4：如何批量清理历史记录？

**A**：可以通过API删除单条，或者系统会自动保持在合理数量。未来可以添加批量删除功能。

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)
- [Ansible 分阶段执行](./ansible-batch-execution.md)

## 更新日志

### v2.23.0 (2025-11-03)

**后端实现**（已完成）：
- ✅ AnsibleFavorite 数据模型
- ✅ AnsibleTaskHistory 数据模型
- ✅ FavoriteService 完整实现
- ✅ FavoriteHandler API 实现
- ✅ 自动记录任务历史机制
- ✅ 智能去重逻辑
- ✅ 数据库迁移脚本（011_add_favorites_and_history.sql）
- ✅ API 路由注册

**前端实现**（已完成）：
- ✅ API 接口封装（frontend/src/api/ansible.js）
- ✅ 最近使用任务面板（TaskCenter.vue）
- ✅ 一键重新执行功能（基于历史记录）
- ✅ 任务历史记录管理（查看、删除）
- ✅ 模板收藏功能（TaskTemplates.vue）
- ✅ 清单收藏功能（InventoryManage.vue）
- ✅ 收藏状态实时同步

**核心功能**：
1. **最近使用面板**：
   - 显示最近6个任务配置
   - 相对时间显示（刚刚、N分钟前、N小时前、N天前）
   - 使用次数统计
   - 快速执行和删除操作

2. **一键重新执行**：
   - 自动填充所有历史参数
   - 支持 Dry Run 配置恢复
   - 支持分批执行配置恢复
   - 支持额外变量恢复

3. **收藏功能**：
   - 模板收藏/取消收藏
   - 清单收藏/取消收藏
   - 收藏状态实时显示（⭐ 已收藏 / 收藏）
   - 收藏状态自动同步

