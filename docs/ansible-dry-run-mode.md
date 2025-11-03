# Ansible Dry Run 模式使用指南

## 功能概述

Dry Run 模式（也称为检查模式）允许您在实际执行 Ansible 任务之前模拟运行，查看任务将要进行的变更，而不会对目标主机进行实际修改。这是一个重要的安全功能，特别适用于生产环境的变更验证。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **模拟执行**：任务以检查模式运行，显示将要进行的变更
- ✅ **无副作用**：不会对目标主机进行任何实际修改
- ✅ **变更预览**：查看哪些配置将被修改、哪些文件将被创建/删除
- ✅ **风险评估**：在实际执行前评估变更影响范围
- ✅ **状态标识**：任务列表中清晰标识 Dry Run 任务

### 使用场景

1. **生产环境验证**
   - 在生产环境执行任务前验证 Playbook 的正确性
   - 确认变更范围符合预期
   - 避免意外的配置修改

2. **Playbook 测试**
   - 测试新编写的 Playbook 逻辑
   - 验证变量和条件判断是否正确
   - 检查任务依赖关系

3. **变更审核**
   - 向团队展示将要进行的变更
   - 获取变更批准前的预览
   - 生成变更报告

4. **学习和培训**
   - 学习 Ansible 时安全地测试命令
   - 了解 Playbook 的执行流程
   - 不会影响实际系统

## 使用方法

### 前端界面操作

1. **创建任务时启用 Dry Run**
   
   a. 点击 "启动任务" 按钮打开任务创建对话框
   
   b. 填写任务基本信息：
      - 任务名称
      - 选择模板（可选）
      - 选择主机清单
      - 选择集群（可选）
   
   c. 找到 "Dry Run 模式" 开关
   
   d. 将开关切换至 "启用" 状态
      - 开关会显示：`启用（检查模式，不实际执行变更）`
      - 下方会显示提示信息
   
   e. 点击 "检查任务" 按钮（按钮文本会根据 Dry Run 状态自动变化）

2. **查看 Dry Run 任务**
   
   - 在任务列表中，Dry Run 任务会显示 "Dry Run" 标签
   - 任务状态和进度显示与普通任务相同
   - 可以查看执行日志，了解将要进行的变更

3. **日志解读**
   
   在 Dry Run 模式下，日志会包含以下信息：
   - `changed` 状态表示该任务将进行变更
   - `ok` 状态表示该任务不需要变更（配置已是目标状态）
   - `skipped` 状态表示任务被跳过

### API 调用

```bash
# 创建 Dry Run 任务
curl -X POST http://your-server/api/v1/ansible/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "检查系统更新",
    "template_id": 1,
    "inventory_id": 2,
    "dry_run": true
  }'
```

## 技术实现

### 后端实现

#### 1. 数据模型

在 `AnsibleTask` 模型中添加 `dry_run` 字段：

```go
type AnsibleTask struct {
    // ... 其他字段
    DryRun bool `json:"dry_run" gorm:"default:false;comment:是否为检查模式(Dry Run)"`
}
```

#### 2. 任务创建请求

在 `TaskCreateRequest` 中添加 `dry_run` 参数：

```go
type TaskCreateRequest struct {
    Name            string
    TemplateID      *uint
    InventoryID     *uint
    DryRun          bool // 是否为检查模式
}
```

#### 3. 命令构建

在执行器的 `buildAnsibleCommand` 函数中添加 `--check` 参数：

```go
if task.DryRun {
    args = append(args, "--check")
    e.logger.Infof("Task %d: Running in Dry Run mode (--check), no changes will be made", task.ID)
}
```

### 前端实现

#### 1. 表单控件

使用 `el-switch` 组件控制 Dry Run 模式：

```vue
<el-form-item label="Dry Run 模式">
  <el-switch 
    v-model="taskForm.dry_run" 
    active-text="启用（检查模式，不实际执行变更）"
    inactive-text="禁用"
  />
</el-form-item>
```

#### 2. 任务标识

在任务列表中显示 Dry Run 标签：

```vue
<el-tag v-if="row.dry_run" type="info" size="small">
  Dry Run
</el-tag>
```

### 数据库迁移

执行数据库迁移添加 `dry_run` 字段：

```sql
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS dry_run BOOLEAN DEFAULT FALSE;
COMMENT ON COLUMN ansible_tasks.dry_run IS '是否为检查模式(Dry Run)';
```

## Ansible 原生支持

Dry Run 模式使用 Ansible 的原生 `--check` 参数：

```bash
ansible-playbook playbook.yml --check
```

### 注意事项

1. **模块支持**
   - 大部分 Ansible 模块支持检查模式
   - 少数模块可能不支持（如 `shell`、`command`）
   - 不支持的模块会被跳过或返回警告

2. **限制条件**
   - 某些复杂的条件判断可能无法在检查模式下正确评估
   - 依赖于前置任务结果的任务可能无法准确预测
   - 文件内容的复杂修改可能无法完全预览

3. **最佳实践**
   - 在生产环境执行前始终使用 Dry Run
   - 仔细review Dry Run 的输出日志
   - 将 Dry Run 结果作为变更审批的依据

## 使用示例

### 示例 1：检查系统软件包更新

```yaml
---
- name: 检查系统更新
  hosts: all
  tasks:
    - name: 更新 yum 缓存
      yum:
        name: '*'
        state: latest
```

**Dry Run 输出**：
```
TASK [更新 yum 缓存] *****************************************
changed: [host1]  # 表示 host1 有可用更新
ok: [host2]       # 表示 host2 已是最新
```

### 示例 2：检查配置文件修改

```yaml
---
- name: 检查 Nginx 配置
  hosts: webservers
  tasks:
    - name: 更新 Nginx 配置文件
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
```

**Dry Run 输出**：
```
TASK [更新 Nginx 配置文件] ***********************************
changed: [web1]  # 表示配置文件将被修改
```

### 示例 3：检查服务状态变更

```yaml
---
- name: 检查服务重启
  hosts: all
  tasks:
    - name: 重启 Nginx 服务
      service:
        name: nginx
        state: restarted
```

**Dry Run 输出**：
```
TASK [重启 Nginx 服务] ***************************************
changed: [host1]  # 表示服务将被重启
```

## 常见问题

### Q1: Dry Run 模式是否会连接目标主机？

**A**: 是的，Dry Run 模式会连接目标主机并收集必要的信息（如当前配置状态），但不会执行实际的变更操作。

### Q2: 所有 Ansible 模块都支持 Dry Run 吗？

**A**: 大部分模块支持，但某些模块（如 `shell`、`command`、`raw`）不支持检查模式。这些模块会在 Dry Run 时被跳过或使用 `check_mode: false` 声明不参与检查。

### Q3: Dry Run 的日志和正常执行有什么区别？

**A**: Dry Run 的日志会标识哪些任务将进行变更（`changed`），但实际上没有执行这些变更。正常执行的日志则反映实际的变更结果。

### Q4: 如何判断 Dry Run 是否准确？

**A**: Dry Run 基于当前系统状态进行预测，在大部分情况下是准确的。但对于依赖运行时状态或外部因素的任务，预测可能不完全准确。最好的做法是先在测试环境验证。

### Q5: Dry Run 任务是否计入统计？

**A**: 是的，Dry Run 任务会正常记录在任务历史中，并标记为 Dry Run 模式。这有助于审计和追溯。

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible 增强进度](./ansible-enhancement-progress.md)
- [Ansible 官方文档 - Check Mode](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_checkmode.html)

## 更新日志

### v2.19.0 (2025-11-03)

- ✅ 实现 Dry Run 基础功能
- ✅ 前端添加 Dry Run 开关
- ✅ 任务列表显示 Dry Run 标签
- ✅ 数据库迁移脚本
- ✅ API 支持 dry_run 参数
- ✅ 执行器集成 --check 参数
- ✅ 完善文档和使用指南

