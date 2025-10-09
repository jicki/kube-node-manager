# GitLab Runner Token 管理功能

## 概述

本文档描述了 GitLab Runner Token 管理功能，包括 Token 保存、查看和重置功能。

## 功能特性

### 1. 自动保存 Token

当通过平台创建 Runner 时，系统会自动保存 Runner 的认证 Token 到数据库中，便于后续查看和管理。

**实现细节：**
- Token 保存在 `gitlab_runners` 表中
- 记录包括：Runner ID、Token、描述、类型、创建者、创建时间
- 每个 Runner ID 是唯一的

### 2. 平台创建标识

在 Runner 列表中，系统会自动识别并标记哪些 Runner 是通过平台创建的。

**显示规则：**
- **平台创建**：绿色标签，表示该 Runner 是通过本平台创建的，可以查看和重置 Token
- **非平台创建**：灰色标签，表示该 Runner 是在其他地方创建的，无法管理 Token

### 3. 查看 Token

对于平台创建的 Runner，用户可以随时查看其认证 Token。

**使用方法：**
1. 在 Runner 列表中找到目标 Runner
2. 点击操作列的"查看Token"按钮
3. 在弹出的对话框中查看 Token 信息
4. 可以点击"复制"按钮将 Token 复制到剪贴板

**显示信息：**
- Runner ID
- 描述
- Runner 类型
- 创建者
- 创建时间
- Token（可复制）

### 4. 重置 Token

当 Token 泄露或需要重新注册 Runner 时，可以重置 Token。

**使用方法：**
1. 在 Runner 列表中找到目标 Runner
2. 点击操作列的"重置Token"按钮
3. 确认重置操作（会有警告提示）
4. 查看新生成的 Token

**注意事项：**
- ⚠️ 重置后，原 Token 立即失效
- ⚠️ 已使用旧 Token 注册的 Runner 需要重新注册
- ✅ 新 Token 会立即保存到数据库
- ✅ 重置后会显示新 Token，请妥善保存

## 数据库表结构

### gitlab_runners 表

```sql
CREATE TABLE gitlab_runners (
    id SERIAL PRIMARY KEY,
    runner_id INTEGER UNIQUE NOT NULL,  -- GitLab Runner ID
    token TEXT NOT NULL,                -- Runner 认证 Token
    description VARCHAR(255),           -- Runner 描述
    runner_type VARCHAR(50),            -- Runner 类型
    created_by VARCHAR(100),            -- 创建者用户名
    created_at TIMESTAMP,               -- 创建时间
    updated_at TIMESTAMP,               -- 更新时间
    deleted_at TIMESTAMP                -- 软删除时间
);

CREATE INDEX idx_gitlab_runners_runner_id ON gitlab_runners(runner_id);
CREATE INDEX idx_gitlab_runners_deleted_at ON gitlab_runners(deleted_at);
```

## API 端点

### 获取 Runner Token

```
GET /api/v1/gitlab/runners/:id/token
```

**响应示例：**
```json
{
  "runner_id": 12345,
  "token": "glrt-xxxxxxxxxxxxx",
  "description": "My Runner",
  "runner_type": "instance_type",
  "created_by": "admin",
  "created_at": "2025-10-09T10:00:00Z"
}
```

### 重置 Runner Token

```
POST /api/v1/gitlab/runners/:id/reset-token
```

**响应示例：**
```json
{
  "id": 12345,
  "token": "glrt-yyyyyyyyyyy",
  "description": "My Runner",
  "runner_type": "instance_type"
}
```

## 前端界面

### Runner 列表新增列

- **创建方式**列：显示 Runner 是否由平台创建
  - 平台创建：绿色标签
  - 非平台创建：灰色标签

### 操作按钮

对于平台创建的 Runner，操作列会显示以下额外按钮：

1. **查看Token**（绿色）：查看已保存的 Token
2. **重置Token**（橙色）：重置 Token 并获取新 Token

### Token 对话框模式

#### 创建模式
- 标题："Runner 创建成功" 或 "Token 重置成功"
- 警告提示：Token 只显示一次
- 显示信息：Runner ID、描述、类型、Token

#### 查看模式
- 标题："Runner Token"
- 信息提示：Token 可用于 Runner 注册
- 显示信息：Runner ID、描述、类型、创建者、创建时间、Token

## 安全性考虑

### Token 存储

- Token 以明文形式存储在数据库中
- 建议在生产环境中对 Token 进行加密存储
- 数据库访问权限应严格控制

### Token 泄露风险

如果 Token 泄露：
1. 立即使用"重置Token"功能
2. 重新注册受影响的 Runner
3. 检查审计日志确认是否有异常活动

### 访问控制

- 只有管理员可以访问 GitLab Runner 管理功能
- 所有 Token 操作都会记录在审计日志中
- 需要登录认证才能访问 API

## 数据迁移

首次部署此功能时，会自动创建 `gitlab_runners` 表。

已有的 Runner 不会自动标记为"平台创建"，只有通过平台新创建的 Runner 才会保存 Token。

## 故障排除

### Token 无法查看

**问题：**点击"查看Token"后提示"Runner token not found"

**可能原因：**
1. 该 Runner 不是通过平台创建的
2. Token 记录在数据库中丢失
3. 数据库连接失败

**解决方法：**
1. 确认 Runner 的创建方式标识
2. 检查数据库中 `gitlab_runners` 表的记录
3. 如果记录丢失，考虑重置 Token

### Token 重置失败

**问题：**重置 Token 时返回错误

**可能原因：**
1. GitLab API 连接失败
2. 权限不足
3. Runner 不存在

**解决方法：**
1. 检查 GitLab 连接设置
2. 确认 GitLab Token 具有足够权限
3. 检查 Runner 是否仍然存在于 GitLab 中

## 最佳实践

1. **定期审查 Token**
   - 定期检查哪些 Runner 具有保存的 Token
   - 删除不再使用的 Runner 及其 Token

2. **Token 轮换**
   - 定期重置 Token 以提高安全性
   - 在重要操作前后重置 Token

3. **备份 Token**
   - 创建 Runner 后立即保存 Token
   - 使用安全的密码管理器存储 Token

4. **监控 Token 使用**
   - 关注审计日志中的 Token 查看和重置操作
   - 对异常操作进行调查

## 更新日志

### v1.0.0 (2025-10-09)

- ✅ 添加 Token 自动保存功能
- ✅ 添加平台创建标识
- ✅ 添加查看 Token 功能
- ✅ 添加重置 Token 功能
- ✅ 创建 `gitlab_runners` 数据库表
- ✅ 前端界面优化

