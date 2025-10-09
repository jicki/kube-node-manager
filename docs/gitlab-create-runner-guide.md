# GitLab Runner 创建功能使用指南

## 功能概述

新增了在系统中直接创建 GitLab Instance Runner 的功能，无需手动到 GitLab 后台操作。

## 主要特性

- ✅ 创建 **Instance Runner**（实例级）：
  - 可用于 GitLab 实例中的所有项目和组
  - 简单易用，配置快速

- ✅ 配置选项：
  - **Runner 描述**（必填）- 简短描述此 Runner 的用途
  - **标签列表**（可选）- 指定 Runner 可执行的作业类型
  - **运行未打标签作业**（可选，默认：否）- 是否允许执行没有标签的作业
  - **Runner 额外描述**（可选）- 添加更多关于 Runner 的详细信息
  - **受保护**（可选，默认：否）- 是否仅用于受保护的分支
  - **锁定到当前项目**（可选，默认：否）- 是否锁定 Runner
  - **最大作业超时**（可选）- Runner 可运行作业的最大时间（最少 600 秒）
  - **已暂停**（可选，默认：否）- 创建后是否暂停 Runner

- ✅ 安全的 Token 管理：
  - 创建成功后显示 Token（仅一次）
  - 一键复制 Token
  - 自动生成注册命令
  - 一键复制注册命令

## 使用步骤

### 1. 打开创建对话框

在 GitLab Runners 页面，点击右上角的 **"新建 Runner"** 按钮。

### 2. 填写 Runner 信息

系统会自动创建 Instance Runner（实例级），可用于所有项目和组。

#### 必填字段

- **描述**：给 Runner 一个有意义的描述（1-100个字符）

#### 可选字段

**基础配置**:
- **标签**：用于匹配 CI/CD 作业的标签，可添加多个（如：docker、linux、production）
- **运行未打标签作业**：是否运行没有标签的作业（默认：否）
- **Runner 额外描述**：添加关于此 Runner 的详细描述信息（最多 200 字符）

**高级配置**:
- **受保护**：是否仅用于受保护的分支（默认：否）
- **锁定到当前项目**：是否锁定 Runner（默认：否）
- **最大作业超时**：Runner 在结束作业前可以运行的最大时间，单位为秒（最少 600 秒，留空使用默认值）
- **已暂停**：创建后 Runner 是否处于暂停状态（默认：否）

### 3. 创建并获取 Token

点击 **"创建"** 按钮后：

1. 系统会调用 GitLab API 创建 Instance Runner
2. 弹出 Token 对话框，显示：
   - Runner ID
   - Runner 描述
   - Runner 类型（Instance Runner）
   - **Runner Token**（仅显示一次！）
   - 自动生成的注册命令

### 4. 保存 Token

⚠️ **重要提示**：Token 只会显示一次，请务必保存！

有两种方式保存：

1. **复制 Token**：点击 Token 输入框后的"复制"按钮
2. **复制注册命令**：点击注册命令后的"复制"按钮

### 5. 在目标机器上注册 Runner

在安装了 GitLab Runner 的机器上执行注册命令：

```bash
gitlab-runner register \
  --url https://your-gitlab.com \
  --token glrt-xxxxxxxxxxxxxxxxxxxx \
  --executor docker \
  --description "Your Runner Description"
```

你可以根据需要修改 executor 和其他参数。

### 6. 验证 Runner

注册成功后：
1. 点击"我已保存 Token"关闭对话框
2. Runner 列表会自动刷新
3. 在列表中找到新创建的 Runner
4. 检查其状态是否变为"在线"

## API 接口

### 创建 Runner

**端点**: `POST /api/v1/gitlab/runners`

**请求体**:
```json
{
  "runner_type": "instance_type",
  "description": "Production Docker Runner - Build and Deploy",
  "tag_list": ["docker", "production", "linux"],
  "run_untagged": false,
  "locked": false,
  "access_level": "not_protected",
  "maximum_timeout": 3600,
  "paused": false
}
```

**响应**:
```json
{
  "id": 123,
  "token": "glrt-xxxxxxxxxxxxxxxxxxxx",
  "description": "My Runner",
  "active": true,
  "paused": false,
  "is_shared": true,
  "runner_type": "instance_type",
  "tag_list": ["docker", "production"]
}
```

## 常见问题

### 1. Token 丢失了怎么办？

Token 只显示一次，无法再次查看。如果丢失：
- 删除该 Runner
- 重新创建一个新的 Runner

### 2. Runner 创建成功但显示离线？

这是正常的，因为：
1. Runner 配置已在 GitLab 创建
2. 但还没有实际的 Runner 进程连接
3. 需要在目标机器上执行注册命令
4. 注册并启动后，状态会变为"在线"

### 3. 标签有什么作用？

标签用于匹配 CI/CD 作业：
- `.gitlab-ci.yml` 中可以指定作业的标签
- 只有匹配标签的 Runner 才会执行该作业
- 如果没有标签，需要勾选"运行未打标签作业"

## 技术实现

### 后端修改

1. **Service 层** (`gitlab.go`):
   - 新增 `CreateRunnerRequest` 结构
   - 新增 `CreateRunnerResponse` 结构
   - 新增 `CreateRunner()` 方法

2. **Handler 层** (`gitlab.go`):
   - 新增 `CreateRunner()` 处理函数

3. **路由** (`main.go`):
   - 注册 `POST /api/v1/gitlab/runners` 端点

### 前端修改

1. **API 层** (`gitlab.js`):
   - 新增 `createGitlabRunner()` 接口

2. **UI 层** (`GitlabRunners.vue`):
   - 新增"新建 Runner"按钮
   - 新增创建 Runner 对话框
   - 新增 Token 显示对话框
   - 新增复制 Token 和注册命令功能

## 安全注意事项

1. **Token 管理**:
   - Token 只在创建时显示一次
   - 对话框关闭后，Token 会被清除
   - 建议立即保存到安全的地方

2. **权限控制**:
   - 只有管理员可以创建 Runner
   - 需要有效的 GitLab API Token
   - 遵循 GitLab 的权限设置

3. **权限要求**:
   - Instance Runner：需要 GitLab 管理员权限
   - 确保配置了有效的 GitLab API Token

## 最佳实践

1. **命名规范**:
   - 使用有意义的描述
   - 包含环境信息（如：prod-docker-runner）
   - 便于识别和管理

2. **标签管理**:
   - 使用清晰的标签分类
   - 如：docker、kubernetes、production、staging

3. **Runner 管理**:
   - 定期审查和清理不用的 Runner
   - 监控 Runner 的使用情况
   - 保持 Runner 版本更新

## 相关文档

- [GitLab Runner 官方文档](https://docs.gitlab.com/runner/)
- [GitLab API 文档](https://docs.gitlab.com/ee/api/runners.html)
- [GitLab Runner 配置指南](./gitlab-runner-configuration.md)

---

**提示**: 如有问题，请查看后端日志或联系系统管理员。

