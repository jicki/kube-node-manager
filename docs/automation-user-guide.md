# Kube Node Manager - 自动化运维用户指南

## 📚 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [功能模块](#功能模块)
  - [Ansible Playbook 管理](#ansible-playbook-管理)
  - [SSH 命令执行](#ssh-命令执行)
  - [脚本管理](#脚本管理)
  - [工作流编排](#工作流编排)
- [安全特性](#安全特性)
- [常见问题](#常见问题)
- [最佳实践](#最佳实践)

---

## 概述

Kube Node Manager 自动化运维系统为 Kubernetes 节点提供了强大的批量管理和自动化操作能力。通过本系统，您可以：

- ✅ 执行 Ansible Playbook 进行配置管理
- ✅ 批量执行 SSH 命令进行快速操作
- ✅ 管理和执行自定义脚本
- ✅ 编排复杂的运维工作流

### 系统架构

```
┌─────────────────────────────────────────────────┐
│           Web UI / 飞书机器人                    │
├─────────────────────────────────────────────────┤
│                  API Gateway                    │
├──────────┬──────────┬──────────┬────────────────┤
│  Ansible │   SSH    │  Script  │   Workflow     │
│  Service │  Service │  Service │   Service      │
├──────────┴──────────┴──────────┴────────────────┤
│          Progress & Audit Service               │
├─────────────────────────────────────────────────┤
│              Kubernetes Cluster                 │
└─────────────────────────────────────────────────┘
```

### 权限说明

| 角色 | 权限描述 |
|------|---------|
| **Admin** | 完全控制：管理所有自动化功能、创建/修改/删除资源、执行所有操作 |
| **User** | 执行操作：执行预定义的安全 Playbook、脚本和工作流 |
| **Viewer** | 只读查看：查看执行历史和结果 |

---

## 快速开始

### 1. 启用自动化功能

**前提条件**：
- 您需要 **Admin** 权限
- 系统配置文件中 `automation.enabled = true`
- 已设置加密密钥环境变量 `AUTOMATION_ENCRYPTION_KEY`

**操作步骤**：

1. 登录系统
2. 导航到：**系统配置** → **自动化配置**
3. 开启 **"启用自动化功能"** 开关
4. 配置各项参数（Ansible、SSH、脚本、工作流）
5. 点击 **保存配置**

### 2. 配置 SSH 凭据

在执行任何自动化操作前，需要先配置 SSH 凭据：

1. 导航到：**自动化** → **SSH 命令执行** → **凭据管理**
2. 点击 **添加凭据**
3. 填写信息：
   - 名称：`prod-nodes-key`
   - 描述：生产环境节点凭据
   - 用户名：`ubuntu`
   - 认证方式：选择 **私钥** 或 **密码**
   - 私钥/密码：输入对应内容
   - 端口：`22`（默认）
4. 点击 **保存**

> **安全提示**：所有凭据使用 AES-256-GCM 加密存储，绝不明文保存。

### 3. 第一次执行

#### 方式一：执行预置 Playbook

1. 导航到：**自动化** → **Ansible Playbooks**
2. 在列表中找到 **"Docker 服务重启"**（内置 Playbook）
3. 点击 **执行** 按钮
4. 在弹出对话框中：
   - 选择目标集群
   - 选择目标节点（可多选）
   - 选择 SSH 凭据
   - 点击 **开始执行**
5. 实时查看执行进度和输出

#### 方式二：执行 SSH 命令

1. 导航到：**自动化** → **SSH 命令执行**
2. 点击 **执行命令**
3. 填写信息：
   - 命令：`uptime`
   - 选择集群和节点
   - 选择凭据
4. 点击 **执行**
5. 实时查看输出

---

## 功能模块

### Ansible Playbook 管理

#### 内置 Playbook

系统提供 4 个常用 Playbook：

| Playbook | 描述 | 用途 |
|----------|------|------|
| 系统升级 | 执行 `apt-get update && apt-get upgrade` | 系统安全更新 |
| Docker 重启 | 重启 Docker 服务 | Docker 故障恢复 |
| 内核升级 | 升级 Linux 内核并重启 | 内核漏洞修复 |
| 安全补丁 | 只安装安全相关更新 | 快速安全修复 |

#### 创建自定义 Playbook

**步骤**：

1. 导航到：**自动化** → **Ansible Playbooks**
2. 点击 **新建 Playbook**
3. 填写基本信息：
   ```
   名称：配置 NTP 服务
   描述：同步时间服务器配置
   分类：configuration
   ```
4. 在编辑器中输入 Playbook 内容：
   ```yaml
   ---
   - name: Configure NTP Service
     hosts: all
     become: yes
     tasks:
       - name: Install NTP
         apt:
           name: ntp
           state: present
           
       - name: Configure NTP servers
         template:
           src: ntp.conf.j2
           dest: /etc/ntp.conf
         notify: restart ntp
         
       - name: Start NTP service
         service:
           name: ntp
           state: started
           enabled: yes
     
     handlers:
       - name: restart ntp
         service:
           name: ntp
           state: restarted
   ```
5. 点击 **保存**

#### 执行 Playbook

**完整流程**：

1. 在 Playbook 列表中找到目标 Playbook
2. 点击 **执行** 按钮
3. 在执行对话框中配置：

   **基本配置**：
   - 集群：选择目标 Kubernetes 集群
   - 节点：选择目标节点（支持多选、全选）
   
   **凭据配置**：
   - SSH 凭据：选择预先配置的凭据
   
   **高级选项**（可选）：
   - 额外参数：`--check`（检查模式，不实际执行）
   - 额外变量：
     ```json
     {
       "ntp_server": "ntp.example.com",
       "timezone": "Asia/Shanghai"
     }
     ```
   - 超时时间：3600 秒（默认）

4. 点击 **开始执行**

5. 实时监控：
   - 查看任务进度条
   - 查看每个节点的执行状态
   - 查看实时输出日志
   - 遇到错误时可点击 **取消执行**

#### Playbook 版本管理

- **查看历史版本**：点击 Playbook 详情 → **版本历史**
- **回滚到历史版本**：选择版本 → **恢复此版本**
- **对比版本差异**：选择两个版本 → **对比**

---

### SSH 命令执行

SSH 命令执行允许您快速在多个节点上执行 Shell 命令。

#### 执行命令

**步骤**：

1. 导航到：**自动化** → **SSH 命令执行**
2. 点击 **执行命令**
3. 填写配置：
   ```
   命令：df -h | grep -v tmpfs
   集群：production-cluster
   节点：[node1, node2, node3]
   凭据：prod-nodes-key
   超时：60 秒
   ```
4. 点击 **执行**

#### 实时输出

执行时会看到：
```
[node1] 正在连接...
[node1] 已连接，执行命令...
[node1] ========== 输出 ==========
[node1] Filesystem      Size  Used Avail Use% Mounted on
[node1] /dev/sda1       50G   20G   28G  42% /
[node1] /dev/sdb1      200G   80G  110G  42% /data

[node2] 正在连接...
[node2] 已连接，执行命令...
...
```

#### 安全限制

系统内置危险命令检测，以下命令会被拦截：

- ❌ `rm -rf /`
- ❌ `rm -rf /*`
- ❌ `dd if=/dev/zero of=/dev/sda`
- ❌ `mkfs.*`
- ❌ `:(){ :|:& };:`（Fork bomb）
- ❌ `> /dev/sda`
- ❌ `chmod -R 777 /`

如需执行危险操作，请使用 Ansible Playbook 并经过审批。

#### 命令模板

您可以保存常用命令为模板：

**创建模板**：
1. 执行命令后，点击 **保存为模板**
2. 输入模板名称：`检查磁盘空间`
3. 点击保存

**使用模板**：
1. 点击 **从模板加载**
2. 选择模板
3. 自动填充命令

---

### 脚本管理

脚本管理允许您上传、编辑和执行 Shell 或 Python 脚本。

#### 内置脚本

系统提供 4 个常用脚本：

| 脚本名称 | 语言 | 功能描述 |
|---------|------|---------|
| 系统信息收集 | Shell | 收集 CPU、内存、磁盘、网络信息 |
| 磁盘清理 | Shell | 清理临时文件、日志、缓存 |
| 日志收集 | Shell | 归档系统日志和应用日志 |
| 性能诊断 | Shell | CPU、内存、IO 性能分析 |

#### 创建脚本

**步骤**：

1. 导航到：**自动化** → **脚本管理**
2. 点击 **新建脚本**
3. 填写信息：

   **基本信息**：
   ```
   名称：检查端口占用
   描述：检查指定端口是否被占用
   语言：Shell
   分类：diagnosis
   ```
   
   **脚本内容**：
   ```bash
   #!/bin/bash
   
   # 参数：PORT
   PORT=${PORT:-80}
   
   echo "检查端口 $PORT 占用情况..."
   
   if netstat -tuln | grep ":$PORT " > /dev/null; then
       echo "端口 $PORT 已被占用："
       netstat -tulnp | grep ":$PORT "
       
       PID=$(lsof -ti:$PORT)
       if [ -n "$PID" ]; then
           echo "进程信息："
           ps aux | grep $PID | grep -v grep
       fi
   else
       echo "端口 $PORT 未被占用"
   fi
   ```

4. 点击 **语法检查**（自动验证语法）
5. 点击 **保存**

#### 执行脚本

**步骤**：

1. 在脚本列表中找到脚本
2. 点击 **执行** 按钮
3. 配置执行参数：
   ```
   集群：production-cluster
   节点：[node1, node2]
   凭据：prod-nodes-key
   参数：
     PORT: 8080
   超时：300 秒
   ```
4. 点击 **开始执行**

#### 脚本参数

脚本支持参数化：

**定义参数**：
```bash
#!/bin/bash
# 参数定义（注释格式）：
# @param PORT 端口号 (default: 80)
# @param LOG_DIR 日志目录 (default: /var/log)

PORT=${PORT:-80}
LOG_DIR=${LOG_DIR:-/var/log}
```

**执行时传参**：
```json
{
  "PORT": "8080",
  "LOG_DIR": "/app/logs"
}
```

---

### 工作流编排

工作流编排允许您将多个操作组合成一个自动化流程。

#### 内置工作流

| 工作流名称 | 步骤数 | 描述 |
|-----------|--------|------|
| 节点维护 | 4 步 | Cordon → 系统升级 → 重启 → Uncordon |
| 故障诊断 | 3 步 | 信息收集 → 日志分析 → 报告生成 |
| 批量部署 | 3 步 | 环境检查 → 软件安装 → 配置更新 |

#### 工作流结构

```json
{
  "name": "节点维护工作流",
  "description": "安全地维护节点",
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
      "timeout": 1800,
      "on_failure": "stop"
    },
    {
      "id": "step3",
      "name": "重启节点",
      "type": "ssh",
      "command": "sudo reboot",
      "depends_on": ["step2"],
      "timeout": 300,
      "continue_on_error": false
    },
    {
      "id": "step4",
      "name": "恢复调度",
      "type": "k8s",
      "action": "uncordon",
      "depends_on": ["step3"],
      "condition": "always",
      "timeout": 30
    }
  ]
}
```

#### 步骤类型

**K8s 操作**：
```json
{
  "type": "k8s",
  "action": "cordon | uncordon | drain | label | taint"
}
```

**Ansible Playbook**：
```json
{
  "type": "ansible",
  "playbook_id": 1,
  "extra_vars": {}
}
```

**SSH 命令**：
```json
{
  "type": "ssh",
  "command": "systemctl restart docker"
}
```

**脚本执行**：
```json
{
  "type": "script",
  "script_id": 1,
  "parameters": {}
}
```

#### 条件执行

**on_success**：前置步骤成功时执行（默认）
```json
{
  "depends_on": ["step1"],
  "condition": "on_success"
}
```

**on_failure**：前置步骤失败时执行
```json
{
  "depends_on": ["step2"],
  "condition": "on_failure"
}
```

**always**：无论前置步骤成功或失败都执行
```json
{
  "depends_on": ["step3"],
  "condition": "always"
}
```

#### 重试机制

```json
{
  "id": "step2",
  "name": "下载软件包",
  "type": "ssh",
  "command": "wget https://example.com/package.tar.gz",
  "retry": {
    "max_attempts": 3,
    "delay": 10
  }
}
```

#### 执行工作流

**步骤**：

1. 导航到：**自动化** → **工作流管理**
2. 选择工作流，点击 **执行**
3. 选择目标集群和节点
4. 点击 **开始执行**
5. 查看工作流执行图：
   ```
   [Step 1: 禁止调度]  ✅ 已完成 (3 秒)
          ↓
   [Step 2: 系统升级]  🔄 执行中... (120 秒)
          ↓
   [Step 3: 重启节点]  ⏳ 等待中
          ↓
   [Step 4: 恢复调度]  ⏳ 等待中
   ```

---

## 安全特性

### 凭据加密

- 所有 SSH 凭据使用 **AES-256-GCM** 加密存储
- 加密密钥通过环境变量 `AUTOMATION_ENCRYPTION_KEY` 设置
- 密钥长度必须为 32 字节
- 支持密钥轮换

### 命令安全检查

**黑名单机制**：
- 系统维护危险命令黑名单
- 执行前自动检查命令内容
- 匹配黑名单命令会被拒绝

**白名单模式**（可选）：
```yaml
automation:
  security:
    command_whitelist_mode: true
    command_whitelist:
      - "uptime"
      - "df -h"
      - "systemctl status *"
```

### 审计日志

所有自动化操作自动记录审计日志：

| 字段 | 说明 |
|------|------|
| 用户 | 执行操作的用户 |
| 时间 | 操作时间戳 |
| 操作类型 | Ansible/SSH/Script/Workflow |
| 目标节点 | 受影响的节点列表 |
| 执行内容 | Playbook/命令/脚本内容 |
| 执行结果 | 成功/失败/部分成功 |
| 错误信息 | 失败时的详细错误 |

**查看审计日志**：
- 导航到：**系统配置** → **审计日志**
- 筛选：操作类型 = "自动化"

### 权限控制

| 操作 | Admin | User | Viewer |
|------|-------|------|--------|
| 查看 Playbook/脚本/工作流 | ✅ | ✅ | ✅ |
| 执行内置资源 | ✅ | ✅ | ❌ |
| 创建自定义资源 | ✅ | ❌ | ❌ |
| 修改/删除资源 | ✅ | ❌ | ❌ |
| 管理凭据 | ✅ | ❌ | ❌ |
| 配置系统 | ✅ | ❌ | ❌ |

---

## 常见问题

### Q1: 启用自动化功能后，侧边栏没有显示"自动化"菜单？

**A**: 请检查：
1. 系统配置中 `automation.enabled = true`
2. 刷新浏览器页面（Ctrl+F5 强制刷新）
3. 检查用户权限（至少需要 User 角色）

### Q2: 执行 Playbook 时报错："Failed to connect to the host"

**A**: 可能原因：
1. **SSH 凭据不正确**：检查凭据配置
2. **网络不通**：从服务器手动 SSH 测试
3. **端口不对**：检查节点的 SSH 端口（默认 22）
4. **防火墙阻止**：检查防火墙规则

**排查步骤**：
```bash
# 1. 测试网络连通性
ping <node-ip>

# 2. 测试 SSH 连接
ssh -i <private-key> <user>@<node-ip>

# 3. 查看系统日志
tail -f /var/log/kube-node-manager/automation.log
```

### Q3: SSH 命令执行被拦截？

**A**: 如果命令包含危险操作，会被安全检查拦截。

**解决方案**：
1. 使用 Ansible Playbook 代替直接命令
2. 联系管理员临时禁用白名单模式
3. 将命令添加到白名单（需要管理员权限）

### Q4: 工作流执行失败，如何恢复？

**A**: 工作流支持从失败点恢复：

1. 导航到：**工作流管理** → **执行历史**
2. 找到失败的执行记录
3. 点击 **查看详情**
4. 查看失败步骤的错误信息
5. 修复问题后，点击 **从失败点继续**

### Q5: 如何批量执行但跳过某些节点？

**A**: 

**方法一**：在节点选择器中取消勾选要跳过的节点

**方法二**：使用节点标签筛选
```json
{
  "target_selector": {
    "labels": {
      "env": "production",
      "maintenance": "true"
    }
  }
}
```

### Q6: 执行进度卡住不动？

**A**: 可能原因：
1. **WebSocket 连接断开**：刷新页面重新连接
2. **超时设置太短**：增加超时时间
3. **节点无响应**：检查节点状态

**检查方法**：
```bash
# 查看执行状态
GET /api/v1/automation/ansible/status/<task_id>

# 取消执行
POST /api/v1/automation/ansible/cancel/<task_id>
```

---

## 最佳实践

### 1. 测试先行

**始终在测试环境先验证**：
```
开发环境 → 测试环境 → 预发布环境 → 生产环境
```

**使用检查模式**：
```bash
# Ansible 检查模式（不实际执行）
--check --diff
```

### 2. 小批量滚动

**分批执行，降低风险**：
```
第一批: 10% 节点 → 观察 30 分钟
第二批: 30% 节点 → 观察 30 分钟
第三批: 60% 节点 → 观察 30 分钟
```

### 3. 备份重要数据

**执行前备份**：
- 数据库备份
- 配置文件备份
- 日志归档

### 4. 设置合理超时

| 操作类型 | 推荐超时 |
|---------|---------|
| 简单命令 | 60 秒 |
| 系统升级 | 1800 秒 (30 分钟) |
| 内核升级 | 3600 秒 (1 小时) |
| 重启节点 | 300 秒 (5 分钟) |

### 5. 使用工作流编排

**复杂操作使用工作流**，好处：
- ✅ 步骤清晰，易于理解
- ✅ 自动处理依赖关系
- ✅ 失败自动回滚
- ✅ 可重复执行

### 6. 监控和告警

**执行后检查**：
- 节点状态是否正常
- 服务是否正常运行
- 监控指标是否异常
- 查看审计日志

### 7. 文档化操作

**记录每次操作**：
```markdown
## 2025-10-29 系统升级操作

**目标**：升级生产环境 Docker 版本

**执行人**：张三

**影响范围**：10 个节点

**Playbook**：Docker 升级

**执行时间**：14:00 - 14:30

**结果**：成功

**备注**：无异常
```

### 8. 定期清理

**清理历史数据**：
- 保留 90 天执行记录
- 归档重要执行日志
- 清理临时文件

### 9. 权限最小化

**按需分配权限**：
- 开发人员：User 权限
- 运维人员：Admin 权限
- 监控人员：Viewer 权限

### 10. 定期审查

**每月审查**：
- 审计日志分析
- 异常操作排查
- 凭据有效性检查
- 工作流效率评估

---

## 获取帮助

### 文档资源

- **技术文档**：`docs/automation-implementation-summary.md`
- **API 文档**：`http://<server>/swagger-ui`
- **更新日志**：`CHANGELOG.md`

### 支持渠道

- **飞书群组**：kube-node-manager 用户群
- **GitHub Issues**：https://github.com/your-org/kube-node-manager/issues
- **邮件支持**：support@example.com

### 反馈建议

欢迎通过以下方式提供反馈：
- 🐛 提交 Bug 报告
- 💡 功能建议
- 📖 文档改进
- ⭐ GitHub Star

---

**版本**：v2.17.0  
**更新日期**：2025-10-29  
**作者**：Kube Node Manager Team

