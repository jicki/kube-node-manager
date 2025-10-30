# Ansible 任务中心模块实施总结

## 概述

本文档总结了 Ansible 任务中心模块的完整实现，该模块用于通过 Ansible 对 Kubernetes 集群节点执行自动化操作。

## 实施日期

2025-10-30

## 技术栈

### 后端
- Go 1.24
- Gin Web Framework
- GORM (ORM)
- gorilla/websocket
- gopkg.in/yaml.v3

### 前端
- Vue 3
- Element Plus
- Axios

## 功能特性

### 1. 任务管理
- ✅ 创建并执行 Ansible 任务
- ✅ 查看任务列表（支持分页和筛选）
- ✅ 查看任务详情和执行进度
- ✅ 实时查看任务日志
- ✅ 取消正在运行的任务
- ✅ 重试失败的任务
- ✅ 任务状态自动刷新

### 2. 模板管理
- ✅ 创建 Ansible Playbook 模板
- ✅ 编辑和删除模板
- ✅ Playbook YAML 语法验证
- ✅ 危险命令检测
- ✅ 模板变量定义和验证

### 3. 主机清单管理
- ✅ 从 Kubernetes 集群动态生成主机清单
- ✅ 手动创建主机清单
- ✅ 编辑和删除清单
- ✅ 刷新 K8s 来源的清单
- ✅ 支持按标签筛选节点

### 4. 任务执行
- ✅ 异步任务执行（最多 5 个并发）
- ✅ 实时日志收集和推送
- ✅ WebSocket 日志流
- ✅ 任务统计信息（成功/失败/跳过主机数）
- ✅ 执行时长计算
- ✅ 临时文件自动清理

### 5. 安全特性
- ✅ 用户身份验证
- ✅ Playbook 内容验证
- ✅ 危险命令检测
- ✅ 审计日志记录
- ✅ 删除保护（检查关联任务）

## 数据模型

### AnsibleTask (任务)
- ID、名称、状态
- 关联模板、集群、主机清单
- 执行信息（开始/结束时间、时长）
- 统计信息（主机总数、成功数、失败数、跳过数）
- Playbook 内容和额外变量

### AnsibleTemplate (模板)
- ID、名称、描述、标签
- Playbook 内容
- 变量定义
- 创建用户和时间戳

### AnsibleLog (日志)
- 任务 ID
- 日志类型（stdout/stderr/event）
- 日志内容和行号
- 时间戳

### AnsibleInventory (主机清单)
- ID、名称、描述
- 来源类型（k8s/manual）
- 清单内容（INI 格式）
- 结构化主机数据
- 关联集群（可选）

## API 端点

### 任务管理
- `GET /api/v1/ansible/tasks` - 列出任务
- `GET /api/v1/ansible/tasks/:id` - 获取任务详情
- `POST /api/v1/ansible/tasks` - 创建并执行任务
- `POST /api/v1/ansible/tasks/:id/cancel` - 取消任务
- `POST /api/v1/ansible/tasks/:id/retry` - 重试任务
- `GET /api/v1/ansible/tasks/:id/logs` - 获取任务日志
- `POST /api/v1/ansible/tasks/:id/refresh` - 刷新任务状态
- `GET /api/v1/ansible/statistics` - 获取统计信息

### 模板管理
- `GET /api/v1/ansible/templates` - 列出模板
- `GET /api/v1/ansible/templates/:id` - 获取模板详情
- `POST /api/v1/ansible/templates` - 创建模板
- `PUT /api/v1/ansible/templates/:id` - 更新模板
- `DELETE /api/v1/ansible/templates/:id` - 删除模板
- `POST /api/v1/ansible/templates/validate` - 验证模板

### 主机清单管理
- `GET /api/v1/ansible/inventories` - 列出主机清单
- `GET /api/v1/ansible/inventories/:id` - 获取清单详情
- `POST /api/v1/ansible/inventories` - 创建清单
- `PUT /api/v1/ansible/inventories/:id` - 更新清单
- `DELETE /api/v1/ansible/inventories/:id` - 删除清单
- `POST /api/v1/ansible/inventories/generate` - 从集群生成清单
- `POST /api/v1/ansible/inventories/:id/refresh` - 刷新清单

### WebSocket
- `WS /api/v1/ansible/tasks/:id/ws` - 任务日志实时流

## 文件结构

### 后端

```
backend/
├── internal/
│   ├── model/
│   │   └── ansible.go              # 数据模型定义
│   ├── service/
│   │   └── ansible/
│   │       ├── service.go          # 核心服务
│   │       ├── template.go         # 模板服务
│   │       ├── inventory.go        # 主机清单服务
│   │       └── executor.go         # 任务执行器
│   └── handler/
│       └── ansible/
│           ├── handler.go          # 任务 API 处理器
│           ├── template.go         # 模板 API 处理器
│           ├── inventory.go        # 清单 API 处理器
│           └── websocket.go        # WebSocket 处理器
```

### 前端

```
frontend/
├── src/
│   ├── api/
│   │   └── ansible.js              # API 封装
│   └── views/
│       └── ansible/
│           ├── TaskCenter.vue      # 任务中心页面
│           ├── TaskTemplates.vue   # 模板管理页面
│           └── InventoryManage.vue # 清单管理页面
```

## 使用说明

### 前置条件

1. **Ansible 安装**

   **选项 A：使用 Docker 镜像（推荐）**
   
   Docker 镜像已集成 Ansible，无需额外安装。镜像包含：
   - Ansible 核心工具
   - Python3 运行时
   - OpenSSH 客户端
   - 预配置的 Ansible 设置

   **选项 B：在主机上安装**
   ```bash
   # CentOS/RHEL
   yum install -y ansible

   # Ubuntu/Debian
   apt-get install -y ansible

   # Alpine (容器内)
   apk add ansible python3 openssh-client
   ```

2. **SSH 密钥配置（用于连接目标主机）**
   
   ```bash
   # 生成 SSH 密钥
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/ansible_id_rsa
   
   # 复制公钥到目标主机
   ssh-copy-id -i ~/.ssh/ansible_id_rsa.pub root@target-host
   
   # Docker 部署时挂载密钥
   docker run -v ~/.ssh:/root/.ssh:ro your-registry/kube-node-manager:latest
   ```

3. **用户权限**
   
   必须使用具有 `admin` 角色的用户账号才能访问 Ansible 模块。

### 使用流程

1. **创建主机清单**
   - 从 K8s 集群自动生成
   - 或手动创建 INI 格式清单

2. **创建任务模板（可选）**
   - 编写 Ansible Playbook YAML
   - 定义模板变量
   - 验证语法

3. **执行任务**
   - 选择模板或直接提供 Playbook
   - 选择主机清单
   - 提供额外变量
   - 启动任务

4. **监控任务**
   - 查看实时日志
   - 监控执行进度
   - 查看统计信息

## 配置说明

### 数据库

模块使用现有数据库配置，新增以下表：
- `ansible_tasks`
- `ansible_templates`
- `ansible_logs`
- `ansible_inventories`

### 并发控制

默认最多同时执行 5 个任务，可在 `executor.go` 中修改：
```go
maxConcurrent: 5
```

### 临时文件清理

自动清理 24 小时前的临时文件：
- Playbook 文件
- Inventory 文件

## 安全注意事项

1. **权限管理**
   - ✅ **仅限管理员访问**：所有 Ansible 模块的 API 接口都要求 `admin` 角色
   - 在每个 Handler 方法中使用 `checkAdminPermission()` 进行权限验证
   - 非管理员用户访问时返回 `403 Forbidden` 错误

2. **Playbook 验证**
   - 自动检测危险命令（如 `rm -rf /`）
   - YAML 语法验证
   - 必需字段检查

3. **SSH 密钥管理**
   - 使用 SSH 密钥而非密码
   - 限制 ansible 用户权限
   - 定期轮换密钥
   - 在 Docker/K8s 部署时通过 Volume 挂载密钥

4. **网络隔离**
   - 确保 Ansible 执行节点可以访问目标主机
   - 使用防火墙规则限制访问

5. **审计日志**
   - 所有操作记录用户信息
   - 记录任务执行历史
   - 保留日志便于追溯

## 性能优化

1. **日志批量保存**
   - 每 50 条日志或每秒批量写入数据库
   - 减少数据库 I/O

2. **异步执行**
   - 使用 goroutine 异步执行任务
   - 不阻塞 API 响应

3. **WebSocket 推送**
   - 实时日志通过 WebSocket 推送
   - 避免频繁轮询

4. **自动刷新**
   - 前端每 5 秒自动刷新运行中的任务
   - 只在有运行中任务时刷新

## 故障排查

### 任务执行失败

1. 检查 Ansible 是否已安装
```bash
ansible --version
```

2. 检查 SSH 连接
```bash
ssh -i /path/to/key user@target-host
```

3. 查看任务日志
   - 在前端任务列表点击"查看日志"
   - 或查询数据库 `ansible_logs` 表

### WebSocket 连接失败

1. 检查防火墙规则
2. 检查 Nginx 配置（如果使用代理）
3. 确认 WebSocket 升级成功

### 清单生成失败

1. 检查 K8s 集群连接
2. 确认节点有 IP 地址
3. 查看后端日志

## 未来改进

1. **任务流程可视化**
   - 使用流程图展示任务执行链
   - 显示依赖关系

2. **更强大的编辑器**
   - 集成 Monaco Editor 或 CodeMirror
   - YAML 语法高亮和自动补全

3. **任务调度**
   - 支持定时任务
   - 支持 Cron 表达式

4. **权限管理**
   - 基于角色的访问控制
   - 模板和清单权限管理

5. **任务审批流程**
   - 危险操作需要审批
   - 多级审批

6. **Ansible Vault 支持**
   - 加密敏感变量
   - 密钥管理

## 维护建议

1. **定期清理旧数据**
   - 使用 `CleanupOldTasks()` 方法
   - 建议保留最近 30 天的数据

2. **监控执行器状态**
   - 监控并发任务数
   - 监控临时文件大小

3. **备份数据库**
   - 定期备份任务和模板数据
   - 保留重要的 Playbook

4. **更新 Ansible**
   - 保持 Ansible 版本更新
   - 测试新版本兼容性

## 相关文档

- [Ansible 安全和部署更新](./ansible-security-and-deployment-update.md) - 权限管理和 Docker 集成详情
- [部署指南](../deploy/README.md) - Kubernetes 和 Docker 部署说明

## 支持与反馈

如有问题或建议，请联系开发团队。

## 变更日志

### v1.1.0 (2025-10-30)
- ✅ **安全增强**：添加管理员权限验证，限制 Ansible 模块访问
- ✅ **Docker 集成**：在 Docker 镜像中集成 Ansible 命令行工具
- ✅ **文档完善**：添加权限管理和部署指南

### v1.0.0 (2025-10-30)
- ✅ 初始版本发布
- ✅ 实现完整的任务管理功能
- ✅ 实现模板和清单管理
- ✅ 实现 WebSocket 实时日志
- ✅ 实现从 K8s 动态生成清单

## 许可证

遵循项目整体许可证。

