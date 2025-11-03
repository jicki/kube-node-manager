# Ansible 任务模板变量验证功能使用指南

## 功能概述

任务模板变量验证功能自动解析 Playbook 中的变量，并在创建任务时验证用户是否提供了所有必需的变量，避免因变量缺失导致的执行失败。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **自动变量提取**：从 Playbook 内容中自动识别所有变量
- ✅ **智能过滤**：自动过滤 Ansible 内置变量
- ✅ **必需变量验证**：创建任务时验证是否提供了所有必需变量
- ✅ **友好的输入界面**：自动为每个必需变量生成输入框
- ✅ **实时提示**：显示变量名称和输入提示

### 使用场景

#### 1. 避免执行失败
**问题**：经常因为忘记提供变量而导致任务执行失败  
**解决**：系统自动识别并要求填写所有必需变量

#### 2. 新用户引导
**问题**：新用户不知道模板需要哪些变量  
**解决**：界面自动显示所有必需变量的输入框

#### 3. 减少文档依赖
**问题**：需要查看文档才知道需要什么变量  
**解决**：界面自动提示所有必需变量

## 技术实现

### 后端实现

#### 1. 变量提取工具

**pkg/ansible/variables.go**:

```go
// ExtractVariables 从 Playbook 内容中提取所有变量
func ExtractVariables(playbookContent string) []string {
    // 使用正则表达式匹配 Jinja2 变量格式: {{ variable_name }}
    re := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_\.]*(?:\[[^\]]+\])?)(?:\s*\|[^}]+)?\s*\}\}`)
    
    // 提取根变量名（去除 .field 或 [index] 部分）
    // 过滤 Ansible 内置变量
    // 返回去重后的变量列表
}
```

**支持的变量格式**：
- `{{ var }}` - 简单变量
- `{{ var.field }}` - 对象字段访问
- `{{ var | filter }}` - 带过滤器的变量
- `{{ var[0] }}` - 数组索引访问

**过滤的内置变量**：
- `inventory_hostname`
- `hostvars`
- `ansible_facts`
- `ansible_host`
- `item`
- 等 30+ 个常用内置变量

#### 2. 数据模型

**backend/internal/model/ansible.go**:

```go
type AnsibleTemplate struct {
    // ... 其他字段
    RequiredVars []string `json:"required_vars"` // 必需变量列表
    // ...
}
```

#### 3. 自动提取逻辑

在模板创建/更新时自动提取：

```go
// CreateTemplate
requiredVars := ansibleUtil.ExtractVariables(req.PlaybookContent)
template.RequiredVars = requiredVars

// UpdateTemplate
if req.PlaybookContent != "" {
    requiredVars := ansibleUtil.ExtractVariables(req.PlaybookContent)
    template.RequiredVars = requiredVars
}
```

#### 4. 任务创建时验证

```go
// CreateTask
if len(template.RequiredVars) > 0 {
    missingVars := s.validateRequiredVariables(template.RequiredVars, req.ExtraVars)
    if len(missingVars) > 0 {
        return nil, fmt.Errorf("missing required variables: %v", missingVars)
    }
}
```

### 前端实现

#### 1. 自动显示变量输入

**TaskCenter.vue**:

```vue
<!-- 必需变量输入 -->
<template v-if="selectedTemplate && selectedTemplate.required_vars && selectedTemplate.required_vars.length > 0">
  <el-divider content-position="left">
    <el-icon><Setting /></el-icon>
    模板变量配置
  </el-divider>
  
  <el-alert
    title="请提供以下必需变量"
    type="info"
    :closable="false"
  >
    该模板需要以下 {{ selectedTemplate.required_vars.length }} 个变量
  </el-alert>
  
  <el-form-item 
    v-for="varName in selectedTemplate.required_vars" 
    :key="varName"
    :label="varName"
    :required="true"
  >
    <el-input 
      v-model="taskForm.extra_vars[varName]" 
      :placeholder="`请输入 ${varName} 的值`"
    >
      <template #prepend>
        <el-icon><Key /></el-icon>
      </template>
    </el-input>
  </el-form-item>
</template>
```

#### 2. 计算属性

```javascript
const selectedTemplate = computed(() => {
  if (!taskForm.template_id) return null
  return templates.value.find(t => t.id === taskForm.template_id)
})
```

### 数据库设计

#### 迁移文件

**012_add_template_required_vars.sql**:

```sql
ALTER TABLE ansible_templates ADD COLUMN required_vars JSONB;
COMMENT ON COLUMN ansible_templates.required_vars IS '必需变量列表';
```

## 使用示例

### 示例 1：创建带变量的模板

#### 1. Playbook 内容

```yaml
---
- name: Deploy Application
  hosts: all
  tasks:
    - name: Deploy {{ app_name }} version {{ app_version }}
      shell: |
        cd /opt/apps
        tar -xzf {{ app_name }}-{{ app_version }}.tar.gz
        
    - name: Configure environment
      template:
        src: config.j2
        dest: /etc/{{ app_name }}/config.yaml
      vars:
        db_host: "{{ database_host }}"
        db_port: "{{ database_port }}"
```

#### 2. 系统自动提取的变量

```json
{
  "required_vars": [
    "app_name",
    "app_version",
    "database_host",
    "database_port"
  ]
}
```

#### 3. 创建任务时的界面

```
模板变量配置
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ℹ️ 请提供以下必需变量
该模板需要以下 4 个变量

app_name *
🔑 [请输入 app_name 的值          ]
变量名: app_name

app_version *
🔑 [请输入 app_version 的值       ]
变量名: app_version

database_host *
🔑 [请输入 database_host 的值     ]
变量名: database_host

database_port *
🔑 [请输入 database_port 的值     ]
变量名: database_port

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 示例 2：验证变量完整性

#### 场景：缺少必需变量

```javascript
// 用户提供的变量
const extraVars = {
  app_name: "myapp",
  app_version: "1.0.0",
  database_host: "localhost"
  // 缺少 database_port
}

// 后端验证
POST /api/v1/ansible/tasks
{
  "name": "部署应用",
  "template_id": 10,
  "inventory_id": 5,
  "extra_vars": {
    "app_name": "myapp",
    "app_version": "1.0.0",
    "database_host": "localhost"
  }
}

// 响应：错误
{
  "error": "missing required variables: [database_port]"
}
```

## 变量提取规则

### 1. 支持的变量格式

| 格式 | 示例 | 提取结果 |
|------|------|----------|
| 简单变量 | `{{ app_name }}` | `app_name` |
| 对象字段 | `{{ user.email }}` | `user` |
| 数组索引 | `{{ items[0] }}` | `items` |
| 带过滤器 | `{{ name \| upper }}` | `name` |
| 默认值 | `{{ port \| default(8080) }}` | `port` |

### 2. 过滤的内置变量

系统自动过滤以下 Ansible 内置变量，不要求用户提供：

**魔法变量**：
- `inventory_hostname`
- `groups`
- `hostvars`
- `ansible_facts`

**主机变量**：
- `ansible_host`
- `ansible_port`
- `ansible_user`
- `ansible_connection`

**循环变量**：
- `item`
- `ansible_loop`

**路径变量**：
- `playbook_dir`
- `role_path`
- `inventory_dir`

### 3. 智能根变量提取

系统只提取根变量名，避免过度细化：

```yaml
# Playbook 中的变量
{{ user.name }}
{{ user.email }}
{{ user.role }}

# 提取结果（只要求提供 user 对象）
required_vars: ["user"]
```

## 最佳实践

### 1. 模板设计

#### 推荐：使用清晰的变量名

```yaml
- name: Deploy {{ app_name }} to {{ environment }}
  vars:
    deploy_path: "/opt/{{ app_name }}"
    config_file: "/etc/{{ app_name }}/config.yaml"
```

#### 不推荐：使用模糊的变量名

```yaml
- name: Deploy {{ a }} to {{ e }}
  vars:
    p: "/opt/{{ a }}"
    c: "/etc/{{ a }}/config.yaml"
```

### 2. 变量注释

虽然当前版本尚未完全支持，但建议在 Playbook 中使用注释：

```yaml
# @var app_name: 应用名称（例如：myapp）
# @var app_version: 应用版本号（例如：1.0.0）
# @var database_host: 数据库主机地址
# @var database_port: 数据库端口（默认：5432）
```

### 3. 变量默认值

为非关键变量提供默认值：

```yaml
- name: Configure application
  vars:
    db_port: "{{ database_port | default(5432) }}"
    log_level: "{{ app_log_level | default('INFO') }}"
```

### 4. 变量分组

对于复杂配置，使用对象分组：

```yaml
# 推荐：使用对象
{{ database.host }}
{{ database.port }}
{{ database.name }}

# 提供变量时
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "mydb"
  }
}
```

## 注意事项

### 1. 变量提取限制

**不支持的场景**：
- 动态生成的变量名：`{{ "var_" + index }}`
- 条件变量：`{{ var1 if condition else var2 }}`
- 复杂表达式：`{{ (a + b) * c }}`

**原因**：这些场景需要运行时解析，无法静态提取。

### 2. 内置变量判断

系统内置了常见的 Ansible 变量列表，但可能不完整。如果遇到误判（内置变量被要求输入），可以：
- 在创建任务时提供空值
- 或忽略该变量（Ansible 会自动提供）

### 3. 变量覆盖

如果模板中定义了默认变量（`vars:` 或 `defaults/`），用户提供的 `extra_vars` 会覆盖这些默认值。

### 4. 更新延迟

模板的 `required_vars` 字段只在模板创建或更新时提取。如果手动修改了数据库中的 Playbook 内容，需要重新保存模板以更新变量列表。

## 常见问题

### Q1：为什么某些变量没有被提取？

**A**：可能的原因：
1. 变量是 Ansible 内置变量（被过滤）
2. 变量格式不符合 Jinja2 标准
3. 变量在注释中（不会被提取）

### Q2：如何处理可选变量？

**A**：在 Playbook 中使用 `default` 过滤器：
```yaml
port: "{{ custom_port | default(8080) }}"
```
这样 `custom_port` 变量就是可选的。

### Q3：提供的变量值可以是复杂对象吗？

**A**：可以。`extra_vars` 字段存储为 JSONB，支持任意 JSON 结构：
```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "credentials": {
      "username": "admin",
      "password": "secret"
    }
  }
}
```

### Q4：如何批量提供变量？

**A**：在任务创建 API 中，`extra_vars` 是一个 JSON 对象，可以包含任意数量的变量：
```json
{
  "extra_vars": {
    "var1": "value1",
    "var2": "value2",
    "var3": {
      "nested": "value3"
    }
  }
}
```

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)
- [Ansible 分阶段执行](./ansible-batch-execution.md)

## 更新日志

### v2.25.0 (2025-11-03)

**后端实现**：
- ✅ 变量提取工具（`pkg/ansible/variables.go`）
- ✅ 自动提取必需变量（创建/更新模板时）
- ✅ 任务创建时验证变量完整性
- ✅ 数据库迁移（`required_vars` 字段）
- ✅ 智能过滤 30+ 个 Ansible 内置变量

**前端实现**：
- ✅ 自动显示必需变量输入框
- ✅ 计算属性获取选中模板
- ✅ 友好的变量输入界面
- ✅ 实时变量提示

**核心功能**：
1. **自动变量提取**：
   - 正则表达式匹配 Jinja2 变量
   - 提取根变量名
   - 过滤内置变量
   - 自动去重

2. **必需变量验证**：
   - 创建任务时验证
   - 返回缺失变量列表
   - 阻止执行直到变量完整

3. **友好的 UI**：
   - 自动为每个变量生成输入框
   - 显示变量名称
   - 键盘图标提示
   - 必填标记（*）

