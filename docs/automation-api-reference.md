# Kube Node Manager - 自动化 API 参考文档

## 📚 目录

- [概述](#概述)
- [认证](#认证)
- [通用响应格式](#通用响应格式)
- [Features API](#features-api)
- [Ansible API](#ansible-api)
- [SSH API](#ssh-api)
- [Scripts API](#scripts-api)
- [Workflows API](#workflows-api)
- [错误代码](#错误代码)
- [SDK 示例](#sdk-示例)

---

## 概述

Base URL: `http://<server>:<port>/api/v1`

所有 API 请求必须包含认证头（除了 Features API 的 GET 请求）。

## 认证

### JWT Token

所有需要认证的请求必须在 Header 中包含 JWT Token：

```http
Authorization: Bearer <jwt_token>
```

### 获取 Token

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

**响应**：
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

---

## 通用响应格式

### 成功响应

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    // 响应数据
  }
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "错误描述",
  "error": "详细错误信息"
}
```

---

## Features API

### 获取功能状态

获取系统所有功能开关状态（无需认证）。

```http
GET /api/v1/features
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "automation": {
      "enabled": true,
      "ansible_enabled": true,
      "ssh_enabled": true,
      "scripts_enabled": true,
      "workflows_enabled": true
    }
  }
}
```

### 更新自动化开关

更新自动化功能主开关（需要 admin 权限）。

```http
PUT /api/v1/features/automation/enabled
Authorization: Bearer <token>
Content-Type: application/json

{
  "enabled": true
}
```

**响应**：
```json
{
  "code": 200,
  "message": "更新成功"
}
```

### 更新 Ansible 配置

```http
PUT /api/v1/features/automation/ansible
Authorization: Bearer <token>
Content-Type: application/json

{
  "binary_path": "/usr/bin/ansible-playbook",
  "temp_dir": "/tmp/ansible-runs",
  "timeout": 3600,
  "max_concurrent": 10
}
```

### 更新 SSH 配置

```http
PUT /api/v1/features/automation/ssh
Authorization: Bearer <token>
Content-Type: application/json

{
  "timeout": 300,
  "max_concurrent": 50,
  "connection_pool_size": 20
}
```

---

## Ansible API

### 1. 列出 Playbooks

```http
GET /api/v1/automation/ansible/playbooks
Authorization: Bearer <token>

# 查询参数
?category=system       # 可选：按分类筛选
&is_builtin=true      # 可选：只显示内置 Playbook
&page=1               # 可选：页码
&page_size=20         # 可选：每页数量
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "total": 5,
    "playbooks": [
      {
        "id": 1,
        "name": "系统升级",
        "description": "执行系统安全更新",
        "content": "---\n- name: System Upgrade\n  hosts: all\n  ...",
        "category": "system",
        "is_builtin": true,
        "is_active": true,
        "version": 1,
        "created_at": "2025-10-29T10:00:00Z",
        "updated_at": "2025-10-29T10:00:00Z"
      }
    ]
  }
}
```

### 2. 获取 Playbook 详情

```http
GET /api/v1/automation/ansible/playbooks/:id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "id": 1,
    "name": "系统升级",
    "description": "执行系统安全更新",
    "content": "---\n- name: System Upgrade\n  ...",
    "category": "system",
    "is_builtin": true,
    "is_active": true,
    "version": 1,
    "tags": ["update", "security"],
    "created_at": "2025-10-29T10:00:00Z",
    "updated_at": "2025-10-29T10:00:00Z"
  }
}
```

### 3. 创建 Playbook

```http
POST /api/v1/automation/ansible/playbooks
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "配置 NTP 服务",
  "description": "同步时间服务器配置",
  "content": "---\n- name: Configure NTP\n  hosts: all\n  ...",
  "category": "configuration",
  "tags": ["ntp", "time"]
}
```

**响应**：
```json
{
  "code": 201,
  "message": "创建成功",
  "data": {
    "id": 10,
    "name": "配置 NTP 服务",
    ...
  }
}
```

### 4. 更新 Playbook

```http
PUT /api/v1/automation/ansible/playbooks/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "配置 NTP 服务 v2",
  "description": "更新的描述",
  "content": "---\n...",
  "category": "configuration"
}
```

**响应**：
```json
{
  "code": 200,
  "message": "更新成功",
  "data": {
    "id": 10,
    "version": 2,
    ...
  }
}
```

### 5. 删除 Playbook

```http
DELETE /api/v1/automation/ansible/playbooks/:id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "删除成功"
}
```

> **注意**：内置 Playbook（`is_builtin=true`）不能被删除。

### 6. 执行 Playbook

```http
POST /api/v1/automation/ansible/run
Authorization: Bearer <token>
Content-Type: application/json

{
  "playbook_id": 1,
  "cluster_name": "production",
  "target_nodes": ["node1", "node2", "node3"],
  "credential_id": 1,
  "extra_args": "--check",
  "extra_vars": {
    "ntp_server": "ntp.example.com",
    "timezone": "Asia/Shanghai"
  }
}
```

**响应**：
```json
{
  "code": 200,
  "message": "执行已启动",
  "data": {
    "task_id": "ansible-exec-1698550000-abc123",
    "status": "running",
    "playbook_id": 1,
    "playbook_name": "系统升级",
    "cluster_name": "production",
    "target_nodes": ["node1", "node2", "node3"],
    "start_time": "2025-10-29T14:30:00Z"
  }
}
```

### 7. 查询执行状态

```http
GET /api/v1/automation/ansible/status/:task_id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "task_id": "ansible-exec-1698550000-abc123",
    "status": "completed",
    "playbook_name": "系统升级",
    "cluster_name": "production",
    "target_nodes": ["node1", "node2", "node3"],
    "success_count": 3,
    "failed_count": 0,
    "start_time": "2025-10-29T14:30:00Z",
    "end_time": "2025-10-29T14:35:00Z",
    "duration": 300,
    "output": "PLAY [System Upgrade] ...\nTASK [Update apt cache] ...\nok: [node1]\nok: [node2]\n...",
    "results": [
      {
        "node": "node1",
        "status": "success",
        "message": "Upgrade completed successfully"
      },
      {
        "node": "node2",
        "status": "success",
        "message": "Upgrade completed successfully"
      },
      {
        "node": "node3",
        "status": "success",
        "message": "Upgrade completed successfully"
      }
    ]
  }
}
```

**状态值**：
- `pending`: 等待执行
- `running`: 正在执行
- `completed`: 执行完成
- `failed`: 执行失败
- `cancelled`: 已取消

### 8. 取消执行

```http
POST /api/v1/automation/ansible/cancel/:task_id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "取消成功"
}
```

### 9. 执行历史

```http
GET /api/v1/automation/ansible/history
Authorization: Bearer <token>

# 查询参数
?cluster_name=production    # 可选：按集群筛选
&status=completed           # 可选：按状态筛选
&start_date=2025-10-01     # 可选：开始日期
&end_date=2025-10-31       # 可选：结束日期
&page=1                     # 可选：页码
&page_size=20               # 可选：每页数量
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "total": 100,
    "executions": [
      {
        "id": 1,
        "task_id": "ansible-exec-1698550000-abc123",
        "playbook_name": "系统升级",
        "cluster_name": "production",
        "target_nodes": ["node1", "node2", "node3"],
        "status": "completed",
        "success_count": 3,
        "failed_count": 0,
        "start_time": "2025-10-29T14:30:00Z",
        "end_time": "2025-10-29T14:35:00Z",
        "duration": 300,
        "user_id": 1,
        "username": "admin"
      }
    ]
  }
}
```

---

## SSH API

### 1. 执行命令

```http
POST /api/v1/automation/ssh/execute
Authorization: Bearer <token>
Content-Type: application/json

{
  "cluster_name": "production",
  "target_nodes": ["node1", "node2"],
  "credential_id": 1,
  "command": "df -h",
  "timeout": 60
}
```

**响应**：
```json
{
  "code": 200,
  "message": "执行已启动",
  "data": {
    "task_id": "ssh-exec-1698550000-xyz789",
    "status": "running",
    "cluster_name": "production",
    "target_nodes": ["node1", "node2"],
    "command": "df -h",
    "start_time": "2025-10-29T15:00:00Z"
  }
}
```

### 2. 查询执行状态

```http
GET /api/v1/automation/ssh/status/:task_id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "task_id": "ssh-exec-1698550000-xyz789",
    "status": "completed",
    "cluster_name": "production",
    "target_nodes": ["node1", "node2"],
    "command": "df -h",
    "success_count": 2,
    "failed_count": 0,
    "start_time": "2025-10-29T15:00:00Z",
    "end_time": "2025-10-29T15:00:05Z",
    "duration": 5,
    "results": [
      {
        "node": "node1",
        "status": "success",
        "exit_code": 0,
        "stdout": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       50G   20G   28G  42% /",
        "stderr": ""
      },
      {
        "node": "node2",
        "status": "success",
        "exit_code": 0,
        "stdout": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       50G   18G   30G  38% /",
        "stderr": ""
      }
    ]
  }
}
```

### 3. 执行历史

```http
GET /api/v1/automation/ssh/history
Authorization: Bearer <token>

# 查询参数
?cluster_name=production
&status=completed
&page=1
&page_size=20
```

**响应**：类似 Ansible 执行历史格式。

---

## Scripts API

### 1. 列出脚本

```http
GET /api/v1/automation/scripts
Authorization: Bearer <token>

# 查询参数
?language=shell           # 可选：按语言筛选 (shell, python)
&category=diagnosis       # 可选：按分类筛选
&is_builtin=true         # 可选：只显示内置脚本
&page=1
&page_size=20
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "total": 10,
    "scripts": [
      {
        "id": 1,
        "name": "系统信息收集",
        "description": "收集 CPU、内存、磁盘、网络信息",
        "language": "shell",
        "category": "diagnosis",
        "is_builtin": true,
        "is_active": true,
        "version": 1,
        "created_at": "2025-10-29T10:00:00Z"
      }
    ]
  }
}
```

### 2. 获取脚本详情

```http
GET /api/v1/automation/scripts/:id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "id": 1,
    "name": "系统信息收集",
    "description": "收集系统信息",
    "content": "#!/bin/bash\n\necho \"=== CPU Info ===\"\nlscpu\n...",
    "language": "shell",
    "category": "diagnosis",
    "is_builtin": true,
    "is_active": true,
    "version": 1,
    "tags": ["system", "info"],
    "parameters": [
      {
        "name": "OUTPUT_DIR",
        "description": "输出目录",
        "default": "/tmp"
      }
    ]
  }
}
```

### 3. 创建脚本

```http
POST /api/v1/automation/scripts
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "检查端口占用",
  "description": "检查指定端口是否被占用",
  "content": "#!/bin/bash\n\nPORT=${PORT:-80}\n...",
  "language": "shell",
  "category": "diagnosis",
  "tags": ["port", "network"]
}
```

**响应**：
```json
{
  "code": 201,
  "message": "创建成功",
  "data": {
    "id": 20,
    "name": "检查端口占用",
    ...
  }
}
```

### 4. 更新脚本

```http
PUT /api/v1/automation/scripts/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "检查端口占用 v2",
  "description": "更新的描述",
  "content": "#!/bin/bash\n...",
  "category": "diagnosis"
}
```

### 5. 删除脚本

```http
DELETE /api/v1/automation/scripts/:id
Authorization: Bearer <token>
```

### 6. 执行脚本

```http
POST /api/v1/automation/scripts/execute
Authorization: Bearer <token>
Content-Type: application/json

{
  "script_id": 1,
  "cluster_name": "production",
  "target_nodes": ["node1", "node2"],
  "credential_id": 1,
  "parameters": {
    "PORT": "8080",
    "OUTPUT_DIR": "/tmp/logs"
  },
  "timeout": 300
}
```

**响应**：
```json
{
  "code": 200,
  "message": "执行已启动",
  "data": {
    "task_id": "script-exec-1698550000-def456",
    "status": "running",
    ...
  }
}
```

### 7. 查询执行状态

```http
GET /api/v1/automation/scripts/status/:task_id
Authorization: Bearer <token>
```

**响应**：类似 SSH 执行状态格式。

### 8. 执行历史

```http
GET /api/v1/automation/scripts/history
Authorization: Bearer <token>
```

**响应**：类似 Ansible 执行历史格式。

---

## Workflows API

### 1. 列出工作流

```http
GET /api/v1/automation/workflows
Authorization: Bearer <token>

# 查询参数
?category=maintenance    # 可选：按分类筛选
&is_builtin=true        # 可选：只显示内置工作流
&page=1
&page_size=20
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "total": 5,
    "workflows": [
      {
        "id": 1,
        "name": "节点维护",
        "description": "安全地维护节点",
        "category": "maintenance",
        "is_builtin": true,
        "is_active": true,
        "version": 1,
        "step_count": 4,
        "created_at": "2025-10-29T10:00:00Z"
      }
    ]
  }
}
```

### 2. 获取工作流详情

```http
GET /api/v1/automation/workflows/:id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "id": 1,
    "name": "节点维护",
    "description": "安全地维护节点",
    "category": "maintenance",
    "is_builtin": true,
    "is_active": true,
    "version": 1,
    "definition": {
      "steps": [
        {
          "id": "step1",
          "name": "禁止调度",
          "type": "k8s",
          "action": "cordon",
          "timeout": 30
        },
        {
          "id": "step2",
          "name": "系统升级",
          "type": "ansible",
          "playbook_id": 1,
          "depends_on": ["step1"],
          "timeout": 1800
        },
        ...
      ]
    }
  }
}
```

### 3. 创建工作流

```http
POST /api/v1/automation/workflows
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "自定义维护流程",
  "description": "自定义的维护工作流",
  "category": "maintenance",
  "definition": {
    "steps": [
      {
        "id": "step1",
        "name": "检查状态",
        "type": "ssh",
        "command": "uptime",
        "timeout": 30
      },
      {
        "id": "step2",
        "name": "执行维护",
        "type": "ansible",
        "playbook_id": 1,
        "depends_on": ["step1"],
        "timeout": 1800
      }
    ]
  }
}
```

### 4. 更新工作流

```http
PUT /api/v1/automation/workflows/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "更新的工作流名称",
  "description": "更新的描述",
  "definition": { ... }
}
```

### 5. 删除工作流

```http
DELETE /api/v1/automation/workflows/:id
Authorization: Bearer <token>
```

### 6. 执行工作流

```http
POST /api/v1/automation/workflows/execute
Authorization: Bearer <token>
Content-Type: application/json

{
  "workflow_id": 1,
  "cluster_name": "production",
  "target_nodes": ["node1"],
  "credential_id": 1,
  "parameters": {
    "global_param1": "value1"
  }
}
```

**响应**：
```json
{
  "code": 200,
  "message": "执行已启动",
  "data": {
    "task_id": "workflow-exec-1698550000-ghi789",
    "status": "running",
    "workflow_name": "节点维护",
    "current_step": "step1",
    "total_steps": 4,
    "start_time": "2025-10-29T16:00:00Z"
  }
}
```

### 7. 查询执行状态

```http
GET /api/v1/automation/workflows/status/:task_id
Authorization: Bearer <token>
```

**响应**：
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "task_id": "workflow-exec-1698550000-ghi789",
    "status": "running",
    "workflow_name": "节点维护",
    "cluster_name": "production",
    "target_nodes": ["node1"],
    "current_step": "step2",
    "total_steps": 4,
    "start_time": "2025-10-29T16:00:00Z",
    "steps": [
      {
        "id": "step1",
        "name": "禁止调度",
        "status": "completed",
        "start_time": "2025-10-29T16:00:00Z",
        "end_time": "2025-10-29T16:00:05Z",
        "duration": 5,
        "result": "success"
      },
      {
        "id": "step2",
        "name": "系统升级",
        "status": "running",
        "start_time": "2025-10-29T16:00:05Z",
        "progress": 45
      },
      {
        "id": "step3",
        "name": "重启节点",
        "status": "pending"
      },
      {
        "id": "step4",
        "name": "恢复调度",
        "status": "pending"
      }
    ]
  }
}
```

### 8. 执行历史

```http
GET /api/v1/automation/workflows/history
Authorization: Bearer <token>
```

**响应**：类似 Ansible 执行历史格式。

---

## 错误代码

| 错误码 | 说明 |
|-------|------|
| 200 | 操作成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（Token 无效或过期） |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 422 | 验证失败 |
| 500 | 服务器内部错误 |

### 常见错误

#### 1. 认证失败

```json
{
  "code": 401,
  "message": "未授权",
  "error": "Token 已过期，请重新登录"
}
```

#### 2. 权限不足

```json
{
  "code": 403,
  "message": "权限不足",
  "error": "需要 admin 权限才能执行此操作"
}
```

#### 3. 资源不存在

```json
{
  "code": 404,
  "message": "资源不存在",
  "error": "Playbook ID 999 不存在"
}
```

#### 4. 危险命令拦截

```json
{
  "code": 422,
  "message": "命令验证失败",
  "error": "危险命令已拦截: rm -rf /"
}
```

#### 5. 功能未启用

```json
{
  "code": 403,
  "message": "功能未启用",
  "error": "自动化功能未启用，请联系管理员"
}
```

---

## SDK 示例

### JavaScript/TypeScript

```javascript
import axios from 'axios'

const API_BASE = 'http://localhost:8080/api/v1'
const token = 'your-jwt-token'

// 创建 axios 实例
const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
})

// 列出 Playbooks
async function listPlaybooks() {
  const response = await api.get('/automation/ansible/playbooks')
  return response.data.data.playbooks
}

// 执行 Playbook
async function runPlaybook(playbookId, clusterName, targetNodes, credentialId) {
  const response = await api.post('/automation/ansible/run', {
    playbook_id: playbookId,
    cluster_name: clusterName,
    target_nodes: targetNodes,
    credential_id: credentialId
  })
  return response.data.data.task_id
}

// 查询执行状态
async function getExecutionStatus(taskId) {
  const response = await api.get(`/automation/ansible/status/${taskId}`)
  return response.data.data
}

// 轮询执行状态
async function waitForCompletion(taskId, intervalMs = 2000) {
  while (true) {
    const status = await getExecutionStatus(taskId)
    
    if (status.status === 'completed' || status.status === 'failed') {
      return status
    }
    
    console.log(`Current status: ${status.status}`)
    await new Promise(resolve => setTimeout(resolve, intervalMs))
  }
}

// 使用示例
async function main() {
  try {
    // 列出所有 Playbooks
    const playbooks = await listPlaybooks()
    console.log('Available playbooks:', playbooks.length)
    
    // 执行第一个 Playbook
    const taskId = await runPlaybook(
      1,
      'production',
      ['node1', 'node2'],
      1
    )
    console.log('Task started:', taskId)
    
    // 等待完成
    const result = await waitForCompletion(taskId)
    console.log('Execution completed:', result)
    
    if (result.status === 'completed') {
      console.log('Success count:', result.success_count)
      console.log('Failed count:', result.failed_count)
    }
  } catch (error) {
    console.error('Error:', error.response?.data || error.message)
  }
}

main()
```

### Python

```python
import requests
import time
from typing import Dict, List

API_BASE = 'http://localhost:8080/api/v1'
TOKEN = 'your-jwt-token'

class AutomationClient:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        }
    
    def list_playbooks(self) -> List[Dict]:
        """列出所有 Playbooks"""
        response = requests.get(
            f'{self.base_url}/automation/ansible/playbooks',
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()['data']['playbooks']
    
    def run_playbook(
        self,
        playbook_id: int,
        cluster_name: str,
        target_nodes: List[str],
        credential_id: int
    ) -> str:
        """执行 Playbook"""
        payload = {
            'playbook_id': playbook_id,
            'cluster_name': cluster_name,
            'target_nodes': target_nodes,
            'credential_id': credential_id
        }
        response = requests.post(
            f'{self.base_url}/automation/ansible/run',
            headers=self.headers,
            json=payload
        )
        response.raise_for_status()
        return response.json()['data']['task_id']
    
    def get_execution_status(self, task_id: str) -> Dict:
        """查询执行状态"""
        response = requests.get(
            f'{self.base_url}/automation/ansible/status/{task_id}',
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()['data']
    
    def wait_for_completion(self, task_id: str, interval: int = 2) -> Dict:
        """轮询执行状态直到完成"""
        while True:
            status = self.get_execution_status(task_id)
            
            if status['status'] in ['completed', 'failed', 'cancelled']:
                return status
            
            print(f"Current status: {status['status']}")
            time.sleep(interval)

# 使用示例
def main():
    client = AutomationClient(API_BASE, TOKEN)
    
    try:
        # 列出所有 Playbooks
        playbooks = client.list_playbooks()
        print(f'Available playbooks: {len(playbooks)}')
        
        # 执行第一个 Playbook
        task_id = client.run_playbook(
            playbook_id=1,
            cluster_name='production',
            target_nodes=['node1', 'node2'],
            credential_id=1
        )
        print(f'Task started: {task_id}')
        
        # 等待完成
        result = client.wait_for_completion(task_id)
        print(f'Execution completed: {result["status"]}')
        
        if result['status'] == 'completed':
            print(f'Success count: {result["success_count"]}')
            print(f'Failed count: {result["failed_count"]}')
            
    except requests.exceptions.RequestException as e:
        print(f'Error: {e}')

if __name__ == '__main__':
    main()
```

### Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	APIBase = "http://localhost:8080/api/v1"
	Token   = "your-jwt-token"
)

type AutomationClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewAutomationClient(baseURL, token string) *AutomationClient {
	return &AutomationClient{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *AutomationClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

func (c *AutomationClient) RunPlaybook(
	playbookID int,
	clusterName string,
	targetNodes []string,
	credentialID int,
) (string, error) {
	payload := map[string]interface{}{
		"playbook_id":   playbookID,
		"cluster_name":  clusterName,
		"target_nodes":  targetNodes,
		"credential_id": credentialID,
	}

	respBody, err := c.doRequest("POST", "/automation/ansible/run", payload)
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	return result.Data.TaskID, nil
}

func (c *AutomationClient) GetExecutionStatus(taskID string) (map[string]interface{}, error) {
	respBody, err := c.doRequest("GET", "/automation/ansible/status/"+taskID, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func main() {
	client := NewAutomationClient(APIBase, Token)

	// 执行 Playbook
	taskID, err := client.RunPlaybook(
		1,
		"production",
		[]string{"node1", "node2"},
		1,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Task started: %s\n", taskID)

	// 轮询状态
	for {
		status, err := client.GetExecutionStatus(taskID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Status: %v\n", status["status"])

		if status["status"] == "completed" || status["status"] == "failed" {
			fmt.Printf("Execution finished: %v\n", status)
			break
		}

		time.Sleep(2 * time.Second)
	}
}
```

---

## WebSocket 实时进度

对于长时间运行的任务，可以通过 WebSocket 接收实时进度更新。

### 连接 WebSocket

```javascript
const ws = new WebSocket(`ws://localhost:8080/api/v1/progress/ws?token=${token}`)

ws.onopen = () => {
  console.log('WebSocket connected')
  
  // 订阅任务进度
  ws.send(JSON.stringify({
    action: 'subscribe',
    task_id: 'ansible-exec-1698550000-abc123'
  }))
}

ws.onmessage = (event) => {
  const data = JSON.parse(event.data)
  console.log('Progress update:', data)
  
  // 数据格式
  // {
  //   task_id: "ansible-exec-1698550000-abc123",
  //   type: "progress",
  //   progress: 50,
  //   message: "Executing task 2 of 4...",
  //   data: { ... }
  // }
}

ws.onerror = (error) => {
  console.error('WebSocket error:', error)
}

ws.onclose = () => {
  console.log('WebSocket disconnected')
}
```

---

**版本**：v2.17.0  
**更新日期**：2025-10-29  
**作者**：Kube Node Manager Team

