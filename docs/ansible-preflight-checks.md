# Ansible 任务执行前置检查功能使用指南

## 功能概述

任务执行前置检查功能在任务实际执行前进行一系列自动化检查，提前发现潜在问题，避免执行失败，提升运维效率和成功率。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **主机清单检查**：验证清单内容和主机数量
- ✅ **SSH 连接检查**：验证 SSH 密钥配置和认证类型
- ✅ **Playbook 语法检查**：基本的 Playbook 格式验证
- ✅ **检查结果存储**：将检查结果保存到任务记录
- ✅ **分类展示**：按类别（connectivity/resources/config）组织检查项
- ✅ **状态汇总**：pass/warning/fail 三级状态

### 使用场景

#### 1. 避免执行失败
**问题**：任务执行后才发现配置错误，浪费时间  
**解决**：执行前检查，提前发现问题

#### 2. 新手引导
**问题**：新用户不知道如何配置任务  
**解决**：检查结果提供明确的问题说明和建议

#### 3. 批量任务保障
**问题**：大规模任务执行失败影响严重  
**解决**：执行前检查所有前置条件

## 检查项说明

### 1. 主机清单检查 (config)

**检查内容**：
- 清单内容是否为空
- 主机数量是否为 0
- 清单配置是否正常

**状态判断**：
- ✅ **pass**: 清单正常，包含有效主机
- ⚠️ **warning**: 清单中没有主机
- ❌ **fail**: 清单内容为空

**示例输出**：
```json
{
  "name": "主机清单检查",
  "category": "config",
  "status": "pass",
  "message": "主机清单正常，包含 20 个主机",
  "details": "清单名称: 生产环境主机, 来源: k8s",
  "duration": 5
}
```

### 2. SSH 连接检查 (connectivity)

**检查内容**：
- SSH 密钥是否配置
- SSH 认证类型是否有效
- 私钥内容是否存在（key 认证）

**状态判断**：
- ✅ **pass**: SSH 配置正常
- ⚠️ **warning**: 未配置 SSH 密钥（使用默认配置）
- ❌ **fail**: SSH 密钥不存在或私钥为空

**示例输出**：
```json
{
  "name": "SSH 连接检查",
  "category": "connectivity",
  "status": "pass",
  "message": "SSH 密钥认证已配置",
  "details": "SSH 用户: root, 密钥长度: 1679 bytes",
  "duration": 12
}
```

### 3. Playbook 语法检查 (config)

**检查内容**：
- Playbook 内容是否为空
- 基本格式是否正确（包含 hosts: 或 - name:）

**状态判断**：
- ✅ **pass**: Playbook 格式正常
- ⚠️ **warning**: Playbook 格式可能不正确
- ❌ **fail**: Playbook 内容为空

**注意**：如果系统安装了 `ansible-playbook` 命令，会尝试进行更严格的语法检查。

**示例输出**：
```json
{
  "name": "Playbook 语法检查",
  "category": "config",
  "status": "pass",
  "message": "Playbook 格式正常",
  "details": "Playbook 大小: 1024 bytes",
  "duration": 8
}
```

## 技术实现

### 后端实现

#### 1. 数据模型

**backend/internal/model/ansible.go**:

```go
// PreflightCheckResult 前置检查结果
type PreflightCheckResult struct {
    Status      string              `json:"status"`       // overall/pass/warning/fail
    CheckedAt   time.Time           `json:"checked_at"`   // 检查时间
    Duration    int                 `json:"duration"`     // 检查耗时（毫秒）
    Checks      []PreflightCheck    `json:"checks"`       // 检查项列表
    Summary     PreflightSummary    `json:"summary"`      // 检查摘要
}

// PreflightCheck 单个检查项
type PreflightCheck struct {
    Name        string    `json:"name"`         // 检查项名称
    Category    string    `json:"category"`     // 类别
    Status      string    `json:"status"`       // pass/warning/fail
    Message     string    `json:"message"`      // 检查结果消息
    Details     string    `json:"details"`      // 详细信息
    CheckedAt   time.Time `json:"checked_at"`   // 检查时间
    Duration    int       `json:"duration"`     // 耗时（毫秒）
}

// PreflightSummary 检查摘要
type PreflightSummary struct {
    Total       int `json:"total"`        // 总检查项数
    Passed      int `json:"passed"`       // 通过数
    Warnings    int `json:"warnings"`     // 警告数
    Failed      int `json:"failed"`       // 失败数
}
```

#### 2. 前置检查服务

**backend/internal/service/ansible/preflight.go**:

核心方法：
- `RunPreflightChecks(taskID)` - 执行所有检查
- `checkInventory()` - 检查主机清单
- `checkSSHConnectivity()` - 检查 SSH 连接
- `checkPlaybookSyntax()` - 检查 Playbook 语法
- `calculateSummary()` - 计算检查摘要
- `GetPreflightChecks(taskID)` - 获取检查结果

#### 3. API 端点

```go
POST /api/v1/ansible/tasks/:id/preflight-checks  // 执行前置检查
GET  /api/v1/ansible/tasks/:id/preflight-checks  // 获取检查结果
```

#### 4. 检查流程

```
1. 获取任务信息（含清单、模板等）
   ↓
2. 执行主机清单检查
   ↓
3. 执行 SSH 连接检查（如果配置了密钥）
   ↓
4. 执行 Playbook 语法检查
   ↓
5. 计算摘要和总体状态
   ↓
6. 保存结果到任务记录
   ↓
7. 返回检查结果
```

### 前端实现

#### 1. API 封装

**frontend/src/api/ansible.js**:

```javascript
// 执行前置检查
export function runPreflightChecks(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/preflight-checks`,
    method: 'post'
  })
}

// 获取前置检查结果
export function getPreflightChecks(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/preflight-checks`,
    method: 'get'
  })
}
```

#### 2. UI 组件（待实现）

**建议的 UI 设计**：

1. **触发按钮**：
   - 在任务列表中添加"执行检查"按钮
   - 在任务创建对话框中添加"检查"按钮

2. **结果展示对话框**：
   - 总体状态（pass/warning/fail）使用颜色标识
   - 摘要信息（通过/警告/失败数量）
   - 检查项列表（按类别分组）
   - 每个检查项显示：名称、状态图标、消息、详情

3. **状态图标**：
   - ✅ pass - 绿色对勾
   - ⚠️ warning - 黄色警告
   - ❌ fail - 红色叉号

### 数据库设计

#### 迁移文件

**013_add_preflight_checks.sql**:

```sql
ALTER TABLE ansible_tasks ADD COLUMN preflight_checks JSONB;
COMMENT ON COLUMN ansible_tasks.preflight_checks IS '前置检查结果';
```

## 使用示例

### 示例 1：执行前置检查

```bash
# 创建任务后，执行前置检查
POST /api/v1/ansible/tasks/123/preflight-checks

# 响应
{
  "code": 200,
  "data": {
    "status": "pass",
    "checked_at": "2025-11-03T10:30:00Z",
    "duration": 25,
    "checks": [
      {
        "name": "主机清单检查",
        "category": "config",
        "status": "pass",
        "message": "主机清单正常，包含 20 个主机",
        "details": "清单名称: 生产环境主机, 来源: k8s",
        "checked_at": "2025-11-03T10:30:00Z",
        "duration": 5
      },
      {
        "name": "SSH 连接检查",
        "category": "connectivity",
        "status": "pass",
        "message": "SSH 密钥认证已配置",
        "details": "SSH 用户: root, 密钥长度: 1679 bytes",
        "checked_at": "2025-11-03T10:30:00Z",
        "duration": 12
      },
      {
        "name": "Playbook 语法检查",
        "category": "config",
        "status": "pass",
        "message": "Playbook 格式正常",
        "details": "Playbook 大小: 1024 bytes",
        "checked_at": "2025-11-03T10:30:00Z",
        "duration": 8
      }
    ],
    "summary": {
      "total": 3,
      "passed": 3,
      "warnings": 0,
      "failed": 0
    }
  }
}
```

### 示例 2：检查失败场景

```json
{
  "status": "fail",
  "checked_at": "2025-11-03T10:35:00Z",
  "duration": 18,
  "checks": [
    {
      "name": "主机清单检查",
      "category": "config",
      "status": "warning",
      "message": "主机清单中没有主机",
      "details": "清单配置可能存在问题，建议检查"
    },
    {
      "name": "SSH 连接检查",
      "category": "connectivity",
      "status": "fail",
      "message": "SSH 私钥为空",
      "details": "请配置有效的 SSH 私钥"
    },
    {
      "name": "Playbook 语法检查",
      "category": "config",
      "status": "pass",
      "message": "Playbook 格式正常",
      "details": "Playbook 大小: 512 bytes"
    }
  ],
  "summary": {
    "total": 3,
    "passed": 1,
    "warnings": 1,
    "failed": 1
  }
}
```

## 最佳实践

### 1. 什么时候执行检查

**建议场景**：
- ✅ 首次创建任务时
- ✅ 修改了清单或 SSH 密钥后
- ✅ 执行批量任务前
- ✅ 生产环境任务执行前

**可以跳过的场景**：
- 已经检查过且配置未变更
- 测试环境的简单任务
- 紧急修复任务

### 2. 如何处理检查结果

**pass（全部通过）**：
- 可以直接执行任务
- 无需额外操作

**warning（有警告）**：
- 查看警告详情
- 评估风险
- 决定是否继续执行或先修复

**fail（有失败）**：
- 必须修复失败项
- 修复后重新检查
- 确认通过后再执行

### 3. 常见问题修复

| 检查项 | 失败原因 | 修复方法 |
|--------|----------|----------|
| 主机清单 | 内容为空 | 重新生成或配置清单 |
| 主机清单 | 没有主机 | 检查清单配置，确保包含主机 |
| SSH 连接 | 密钥不存在 | 创建并配置 SSH 密钥 |
| SSH 连接 | 私钥为空 | 上传有效的私钥文件 |
| Playbook | 内容为空 | 选择模板或编写 Playbook |
| Playbook | 格式错误 | 修正 Playbook 语法 |

## 扩展性

当前实现是一个基础框架，可以方便地扩展更多检查项：

### 1. 可添加的检查项

**资源检查**：
- 磁盘空间检查
- 内存可用性检查
- CPU 负载检查

**环境检查**：
- 必需命令是否存在
- 端口是否被占用
- 依赖服务是否运行

**网络检查**：
- 主机是否可达（ping）
- 端口是否开放（telnet）
- DNS 解析是否正常

### 2. 扩展方法

在 `preflight.go` 中添加新的检查方法：

```go
// checkDiskSpace 检查磁盘空间
func (s *PreflightService) checkDiskSpace(task *model.AnsibleTask) model.PreflightCheck {
    // 实现检查逻辑
}

// 在 RunPreflightChecks 中调用
check4 := s.checkDiskSpace(&task)
checks = append(checks, check4)
```

## 注意事项

### 1. 性能考虑

- 前置检查会增加任务启动时间
- 当前实现的检查都比较轻量（< 100ms）
- 如果添加网络检查，需要设置合理的超时时间

### 2. 权限要求

- 前置检查需要管理员权限
- SSH 连接检查不会实际连接主机（仅检查配置）
- 不会对目标主机产生任何影响

### 3. 检查的局限性

**当前检查无法发现的问题**：
- 主机是否真的可达（未实际连接）
- SSH 密码是否正确（未实际认证）
- 目标主机的实际资源状况
- 网络连通性问题

**原因**：
- 避免对生产环境产生影响
- 减少检查时间
- 简化实现

如果需要更全面的检查，建议使用 Dry Run 模式进行实际测试。

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)
- [Ansible 任务模板变量验证](./ansible-variable-validation.md)

## 更新日志

### v2.26.0 (2025-11-03)

**后端实现**（已完成）：
- ✅ 前置检查数据模型（PreflightCheckResult, PreflightCheck, PreflightSummary）
- ✅ PreflightService 完整实现
- ✅ 三个核心检查项（主机清单、SSH连接、Playbook语法）
- ✅ 检查结果存储到任务
- ✅ API 端点（执行检查、获取结果）
- ✅ 数据库迁移

**前端实现**（已完成）：
- ✅ API 接口封装（runPreflightChecks, getPreflightChecks）

**前端 UI**（待实现）：
- ⏳ 执行检查按钮
- ⏳ 检查结果展示对话框
- ⏳ 状态图标和颜色标识
- ⏳ 分类展示检查项

**核心功能**：
1. **主机清单检查**：验证清单内容和主机数量
2. **SSH 连接检查**：验证 SSH 配置和认证类型
3. **Playbook 语法检查**：基本的格式验证
4. **结果存储**：将检查结果保存到任务记录
5. **三级状态**：pass/warning/fail
6. **分类组织**：connectivity/resources/config

