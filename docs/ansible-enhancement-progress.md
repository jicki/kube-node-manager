# Ansible 模块功能增强实施进度

## 实施日期

2025-10-31 (进行中)

## 已完成功能

### 一、任务调度和自动化（后端）✅

#### 1.1 定时任务调度 ✅

**数据模型**：
- ✅ 添加 `AnsibleSchedule` 模型到 `backend/internal/model/ansible.go`
- ✅ 支持字段：name, description, template_id, inventory_id, cluster_id, cron_expr, extra_vars, enabled, last_run_at, next_run_at, run_count
- ✅ 创建数据库迁移文件 `007_add_ansible_schedules.sql`

**调度服务**：
- ✅ 实现 `ScheduleService` (`backend/internal/service/ansible/schedule.go`)
- ✅ 集成 `github.com/robfig/cron/v3` 库
- ✅ 支持秒级精度的 Cron 表达式
- ✅ 自动加载已启用的定时任务
- ✅ 最大活跃任务数限制（100个）
- ✅ 优雅启动和停止

**核心功能**：
- ✅ `Start()` - 启动调度器并加载所有已启用任务
- ✅ `Stop()` - 优雅停止调度器
- ✅ `AddSchedule()` - 添加定时任务到调度器
- ✅ `RemoveSchedule()` - 从调度器移除任务
- ✅ `executeSchedule()` - 执行定时任务（自动创建 Ansible Task）
- ✅ `CreateSchedule()` - 创建定时任务
- ✅ `GetSchedule()` - 获取定时任务详情
- ✅ `ListSchedules()` - 列出定时任务（支持筛选和分页）
- ✅ `UpdateSchedule()` - 更新定时任务
- ✅ `DeleteSchedule()` - 删除定时任务（软删除）
- ✅ `ToggleSchedule()` - 启用/禁用定时任务
- ✅ `RunNow()` - 立即执行定时任务

**API 端点**：
- ✅ `GET /api/v1/ansible/schedules` - 列出定时任务
- ✅ `GET /api/v1/ansible/schedules/:id` - 获取详情
- ✅ `POST /api/v1/ansible/schedules` - 创建定时任务
- ✅ `PUT /api/v1/ansible/schedules/:id` - 更新定时任务
- ✅ `DELETE /api/v1/ansible/schedules/:id` - 删除定时任务
- ✅ `POST /api/v1/ansible/schedules/:id/toggle` - 启用/禁用
- ✅ `POST /api/v1/ansible/schedules/:id/run-now` - 立即执行

**集成**：
- ✅ 集成到 `Service` (`backend/internal/service/ansible/service.go`)
- ✅ 创建 `ScheduleHandler` (`backend/internal/handler/ansible/schedule.go`)
- ✅ 注册到 `Handlers` 结构体
- ✅ 添加路由到 `main.go`
- ✅ 主程序启动时自动启动调度服务
- ✅ 主程序关闭时优雅停止调度服务

#### 1.2 失败重试策略 ✅

**数据模型**：
- ✅ 添加 `RetryPolicy` 结构体
- ✅ 在 `AnsibleTask` 中添加字段：
  - `retry_policy` (JSONB) - 重试策略配置
  - `retry_count` (int) - 当前重试次数
  - `max_retries` (int) - 最大重试次数
- ✅ 创建数据库迁移文件 `008_add_retry_and_environment_fields.sql`

**重试机制**：
- ✅ 实现 `checkAndRetryTask()` 方法检查是否需要重试
- ✅ 实现 `retryTask()` 方法执行重试
- ✅ 支持配置重试次数和重试间隔
- ✅ 自动延迟后重试失败的任务
- ✅ 记录重试次数和重试历史
- ✅ 达到最大重试次数后停止重试

**重试策略配置**：
```json
{
  "max_retries": 3,          // 最大重试次数
  "retry_interval": 60,      // 重试间隔（秒）
  "retry_on_error": true     // 是否在错误时重试
}
```

### 二、环境保护和权限管理（后端）✅

#### 2.1 环境标签和风险等级 ✅

**数据模型扩展**：
- ✅ `AnsibleInventory` 添加 `environment` 字段 (dev/staging/production)
- ✅ `AnsibleTemplate` 添加 `risk_level` 字段 (low/medium/high)
- ✅ 数据库迁移已包含在 `008_add_retry_and_environment_fields.sql`
- ✅ 为现有记录设置默认值

**权限控制**：
- ✅ 保持现有的 admin 权限检查机制
- ✅ 所有 Ansible API 仅限 admin 角色访问
- ✅ 不实施细粒度权限控制

### 三、前端开发 ✅

#### 3.1 定时任务管理页面 ✅
- ✅ 创建 `frontend/src/views/ansible/ScheduleManage.vue`
- ✅ 定时任务列表（表格视图）
- ✅ 创建/编辑定时任务对话框
- ✅ Cron 表达式输入和验证
- ✅ Cron 表达式帮助文档
- ✅ 下次执行时间预览
- ✅ 启用/禁用开关
- ✅ 立即执行按钮
- ✅ 删除定时任务
- ✅ 添加路由 `/ansible-schedules`

#### 3.2 环境标签和二次确认 ✅
- ✅ 创建 `ConfirmDialog.vue` 通用二次确认组件
- ✅ 在 `TaskCenter.vue` 中添加环境和风险检查
- ✅ 在 `TaskTemplates.vue` 中显示风险等级（Tag 标签）
- ✅ 在 `TaskTemplates.vue` 中添加风险等级选择器
- ✅ 在 `InventoryManage.vue` 中显示环境标签（Tag 标签）
- ✅ 在 `InventoryManage.vue` 中添加环境选择器
- ✅ 生产环境任务执行二次确认（带风险提示）
- ✅ 高风险模板执行二次确认（带风险提示）
- ✅ 组合确认（生产环境 + 高风险）

#### 3.3 前端 API 封装 ✅
- ✅ 在 `frontend/src/api/ansible.js` 中添加定时任务相关 API：
  - ✅ `listSchedules(params)` - 列出定时任务
  - ✅ `getSchedule(id)` - 获取定时任务详情
  - ✅ `createSchedule(data)` - 创建定时任务
  - ✅ `updateSchedule(id, data)` - 更新定时任务
  - ✅ `deleteSchedule(id)` - 删除定时任务
  - ✅ `toggleSchedule(id, enabled)` - 切换启用状态
  - ✅ `runScheduleNow(id)` - 立即执行定时任务

## 待实施功能

### 四、可选增强功能 ✅

#### 4.1 日志查看器增强 ✅
- ✅ 创建 `LogViewer.vue` 组件
- ✅ 日志级别过滤（info/warning/error/debug）
- ✅ 关键字搜索和高亮
- ✅ 日志复制功能
- ✅ 日志下载功能
- ✅ 自动滚动到底部
- ✅ 显示行号和时间戳
- ✅ 限制最大行数防止性能问题
- ✅ 在 TaskCenter 中集成

#### 4.2 任务执行可视化 ✅
- ✅ 创建 `TaskTimeline.vue` 组件
- ✅ 时间线视图展示任务执行步骤
- ✅ 显示各阶段耗时
- ✅ 状态图标和颜色区分
- ✅ 显示任务详情
- ✅ 错误信息展示
- ✅ 重试次数显示

#### 4.3 UI 优化 ✅
- ✅ 模板快速筛选标签（按风险等级）
- ✅ 模板克隆功能
- ✅ 批量删除任务（已有）
- ✅ 任务选择控制（只能选择已完成的任务）
- ✅ 响应式布局优化

#### 4.4 Monaco Editor 集成 ✅
- ✅ 添加 npm 依赖到 `package.json`
  - `monaco-editor: ^0.44.0`
  - `@guolao/vue-monaco-editor: ^1.3.1`
  - `vite-plugin-monaco-editor: ^1.1.0` (devDependencies)
- ✅ 配置 `vite.config.js` 插件
- ✅ 创建 `MonacoEditor.vue` 封装组件
- ✅ YAML 语法高亮支持
- ✅ Ansible 模块自动补全（40+ 模块）
- ✅ Ansible 关键字补全（20+ 关键字）
- ✅ 工具栏功能（格式化、撤销、重做、全屏）
- ✅ 主题切换支持（vs, vs-dark, hc-black）
- ✅ 代码折叠和查找替换
- ✅ 行号和列号显示
- ✅ 只读模式支持
- ✅ 完整的集成文档
- ✅ 已集成到 `TaskTemplates.vue`（Playbook 内容编辑）

**使用说明**：
1. 运行 `npm install` 安装依赖
2. 在需要的页面导入 `MonacoEditor` 组件
3. 参考 `docs/monaco-editor-integration.md` 文档

**已集成页面**：
- ✅ 任务模板管理（TaskTemplates.vue）- Playbook 内容编辑器

### 五、测试和文档 ✅

#### 5.1 功能测试 ✅
- ✅ 创建测试指南文档 (`docs/ansible-testing-guide.md`)
- ✅ 定时任务测试用例（7个）
- ✅ 失败重试测试用例（2个）
- ✅ 环境保护测试用例（3个）
- ✅ UI 增强测试用例（4个）
- ✅ 权限测试用例（2个）
- ✅ 性能测试用例（3个）
- ✅ 异常情况测试用例（3个）

**注**：测试指南提供详细的测试步骤和预期结果，可用于手动测试或自动化测试参考。

#### 5.2 文档编写 ✅
- ✅ `docs/ansible-scheduling-guide.md` - 定时任务使用指南（完整）
- ✅ `docs/ansible-v2.19-features.md` - 版本功能总结
- ✅ `docs/ansible-enhancement-progress.md` - 实施进度文档
- ✅ `docs/ansible-testing-guide.md` - 测试指南
- ✅ API 文档（内嵌在代码注释中）
- ✅ 故障排查指南（已有）

## 技术细节

### 数据库表结构

#### ansible_schedules

| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| name | VARCHAR(255) | 调度任务名称 |
| description | TEXT | 描述 |
| template_id | INTEGER | 关联模板ID |
| inventory_id | INTEGER | 关联清单ID |
| cluster_id | INTEGER | 关联集群ID（可选） |
| cron_expr | VARCHAR(100) | Cron表达式 |
| extra_vars | JSONB | 额外变量 |
| enabled | BOOLEAN | 是否启用 |
| last_run_at | TIMESTAMP | 上次执行时间 |
| next_run_at | TIMESTAMP | 下次执行时间 |
| run_count | INTEGER | 执行次数 |
| user_id | INTEGER | 创建用户ID |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 删除时间（软删除） |

#### ansible_tasks (新增字段)

| 字段 | 类型 | 说明 |
|------|------|------|
| retry_policy | JSONB | 重试策略 |
| retry_count | INTEGER | 当前重试次数 |
| max_retries | INTEGER | 最大重试次数 |

#### ansible_inventories (新增字段)

| 字段 | 类型 | 说明 |
|------|------|------|
| environment | VARCHAR(20) | 环境标签 (dev/staging/production) |

#### ansible_templates (新增字段)

| 字段 | 类型 | 说明 |
|------|------|------|
| risk_level | VARCHAR(20) | 风险等级 (low/medium/high) |

### Cron 表达式支持

支持标准 Cron 表达式和秒级精度：

```
# 标准 5 字段格式
* * * * *        # 每分钟执行
0 * * * *        # 每小时整点执行
0 0 * * *        # 每天午夜执行
0 0 * * 0        # 每周日午夜执行
0 0 1 * *        # 每月1号午夜执行

# 扩展表达式
@hourly          # 每小时
@daily           # 每天
@weekly          # 每周
@monthly         # 每月
@yearly          # 每年
```

### 重试策略示例

```json
{
  "retry_policy": {
    "max_retries": 3,
    "retry_interval": 60,
    "retry_on_error": true
  }
}
```

## 性能优化

1. **定时任务数量限制**：最多 100 个活跃定时任务
2. **并发控制**：任务执行器最多同时执行 5 个任务
3. **索引优化**：
   - `idx_ansible_schedules_enabled` - 查询已启用任务
   - `idx_ansible_schedules_next_run_at` - 查询即将执行的任务
   - `idx_ansible_schedules_name_active` - 唯一名称索引（支持软删除）

## 安全考虑

1. **权限控制**：所有定时任务操作仅限 admin 用户
2. **Playbook 验证**：创建定时任务时验证 Playbook 语法和危险命令
3. **资源限制**：限制最大活跃定时任务数量
4. **审计日志**：记录所有定时任务的创建、修改、删除操作
5. **环境保护**：前端对生产环境和高风险操作进行二次确认

## 向后兼容性

所有新功能都是增量添加，不会破坏现有功能：
- ✅ 新增数据表和字段都是可选的
- ✅ 现有 API 保持不变
- ✅ 前端新增页面不影响现有页面
- ✅ 重试策略为可选配置
- ✅ 环境标签有默认值

## 下一步工作

1. 实施前端定时任务管理页面
2. 集成 Monaco Editor
3. 实现环境标签二次确认
4. 增强日志查看器
5. 添加任务执行可视化
6. 编写用户文档和测试用例

## 变更日志

### v2.19.0 (2025-10-31) - 进行中

**后端新增**：
- ✅ 定时任务调度服务（支持 Cron 表达式）
- ✅ 失败重试机制
- ✅ 环境标签和风险等级字段
- ✅ 定时任务 CRUD API
- ✅ 数据库迁移文件（007, 008）

**前端新增**：
- ✅ 定时任务管理页面 (`ScheduleManage.vue`)
- ✅ 环境保护二次确认组件 (`ConfirmDialog.vue`)
- ✅ 风险等级和环境标签显示
- ✅ 前端 API 封装
- ✅ 路由配置

**待开发（可选）**：
- ⏳ Monaco Editor 集成
- ⏳ 增强日志查看器
- ⏳ 任务执行时间线
- ⏳ UI/UX 进一步优化

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [实施计划](../ansible.plan.md)

