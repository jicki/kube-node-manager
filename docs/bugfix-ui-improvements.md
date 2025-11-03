# Bug 修复：UI 改进与收藏功能修复

## 修复时间
2025/01/13

## 修复内容

### 1. 执行模式选择器样式修复

**问题描述**:
执行模式选择器中的文字显示不在框框中，布局混乱。

**解决方案**:
- 使用 `flex` 布局确保内容正确对齐
- 调整 padding 和间距
- 使用 `flex: 1` 确保两个选项等宽
- 设置 `height: auto` 允许内容撑开高度
- 使用 `align-items: flex-start` 确保图标和文字顶部对齐

**效果**:
```
┌────────────────────────┐  ┌────────────────────────┐
│ 🔧 正常模式            │  │ 👁 检查模式 (Dry Run)  │
│ 实际执行并应用变更     │  │ 仅模拟执行，不实际变更 │
└────────────────────────┘  └────────────────────────┘
```

### 2. 任务列表模式标识

**新增功能**:
为所有任务添加执行模式标识，不仅限于 Dry Run 模式。

**实现效果**:
- **正常模式**: 蓝色设置图标 + 蓝色"正常"标签（plain 效果）
- **检查模式**: 绿色眼睛图标 + 深绿色"检查"标签（dark 效果）

**任务列表显示**:
```
🔧 任务名称  [正常]
👁 任务名称  [检查]
```

### 3. 最近使用卡片模式标识

**新增功能**:
在"最近使用"卡片中为每个历史任务添加执行模式标签。

**实现效果**:
- 默认显示"正常模式"标签（蓝色 plain 效果）
- Dry Run 任务显示"检查模式"标签（绿色 dark 效果）
- 标签包含相应的图标
- 与分批执行等其他标签并列显示

**卡片显示**:
```
┌─────────────────────────┐
│ 任务名称: Dry           │
│ 📄 Example              │
│ 📋 test-node            │
│ 📅 9 分钟前             │
│ 📊 使用 1 次            │
│                         │
│ [👁 检查模式]           │
│                         │
│ [快速执行]              │
└─────────────────────────┘
```

### 4. 收藏功能关系预加载修复

**问题描述**:
```
failed to list favorites: Template: unsupported relations for schema AnsibleFavorite
failed to list favorites: Inventory: unsupported relations for schema AnsibleFavorite
```

**根本原因**:
`AnsibleFavorite` 模型中已移除外键关联字段，但 `ListFavorites` 方法仍尝试使用 `Preload` 预加载关联数据，导致 GORM 报错。

**解决方案**:
1. 修改 `ListFavorites` 方法，移除所有 `Preload` 调用
2. 返回纯收藏记录（只包含 target_type 和 target_id）
3. 前端可根据需要通过 target_type 和 target_id 单独获取详细信息

**代码修改** (`backend/internal/service/ansible/favorite.go`):
```go
// 修改前
query = query.Preload("Task").Preload("Template").Preload("Inventory")

// 修改后
// 不使用 Preload，因为 TargetID 是动态引用
// 前端可以根据需要通过 target_type 和 target_id 单独获取详细信息
```

### 5. 数据库外键约束清理

**提供修复脚本**:
创建了 `scripts/fix_favorites_constraints.sql` 脚本用于清理错误的外键约束。

**执行方法**:

**PostgreSQL**:
```bash
psql -U username -d database_name -f scripts/fix_favorites_constraints.sql
```

**SQLite**:
```bash
sqlite3 database.db < scripts/fix_favorites_constraints.sql
```

**或者在数据库客户端中直接执行**:
```sql
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task_id;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template_id;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory_id;

CREATE INDEX IF NOT EXISTS idx_ansible_favorites_user_type_target ON ansible_favorites(user_id, target_type, target_id);
```

## 修改文件清单

### 前端
- `frontend/src/views/ansible/TaskCenter.vue`
  - 修复执行模式选择器样式
  - 任务列表添加正常模式标识
  - 最近使用卡片添加模式标识

### 后端
- `backend/internal/service/ansible/favorite.go`
  - 移除 Preload 调用
  - 简化 ListFavorites 实现

### 脚本
- `scripts/fix_favorites_constraints.sql` - 数据库修复脚本

## 部署说明

### 1. 重新构建和部署
```bash
make docker-build
# 重新部署应用
```

### 2. 执行数据库修复脚本
部署后需要手动执行修复脚本来删除错误的外键约束：

```bash
# PostgreSQL
kubectl exec -it <pod-name> -- psql -U $DB_USER -d $DB_NAME -c "
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory;
CREATE INDEX IF NOT EXISTS idx_ansible_favorites_user_type_target ON ansible_favorites(user_id, target_type, target_id);
"
```

### 3. 验证修复
- 尝试添加收藏，确认不再报错
- 查看任务列表，确认模式标识正确显示
- 查看最近使用卡片，确认标识显示
- 检查执行模式选择器布局是否正确

## 测试建议

### UI 测试
- [ ] 检查执行模式选择器布局是否正确，文字是否在框内
- [ ] 创建正常模式任务，确认列表中显示蓝色"正常"标签
- [ ] 创建检查模式任务，确认列表中显示绿色"检查"标签
- [ ] 查看最近使用卡片，确认所有任务都有模式标签
- [ ] 确认图标和颜色使用正确

### 功能测试
- [ ] 测试添加任务收藏，确认不报错
- [ ] 测试添加模板收藏，确认不报错
- [ ] 测试添加清单收藏，确认不报错
- [ ] 测试列出收藏，确认返回正确数据
- [ ] 测试删除收藏，确认正常工作

## 视觉效果对比

### 执行模式选择器
**修改前**: 文字溢出，布局混乱
**修改后**: 整齐对齐，内容在框内，视觉清晰

### 任务列表
**修改前**: 只有 Dry Run 任务有标识
**修改后**: 所有任务都有明确的模式标识（正常/检查）

### 最近使用卡片
**修改前**: 只标记 Dry Run 模式
**修改后**: 所有任务都标记模式（正常模式/检查模式）

## 版本信息
- 修复版本: v2.22.12
- 相关功能: 任务收藏、执行模式 UI、任务列表展示

