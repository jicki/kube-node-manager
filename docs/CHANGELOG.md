# 变更日志 (CHANGELOG)

本文档记录了 Kube Node Manager 的所有版本变更历史。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [v2.13.2] - 2025-10-27 (下午-第二次修复)

### 🐛 Bug 修复

#### 节点健康度表格布局和数据显示修复

**1. 表格宽度自适应优化**
- ✅ 修复表格右侧大量空白问题
- ✅ 将固定宽度列改为最小宽度（`width` → `min-width`）
- ✅ 表格现在能够自适应填满整个卡片容器
- ✅ 长文本列添加 `show-overflow-tooltip`，自动截断并显示提示
- ✅ 数值列添加 `align="center"`，居中对齐更美观
- ✅ 优化后的列宽配置：
  - 节点名称：min-width 200px（可扩展）
  - 集群：min-width 150px（可扩展）
  - 健康度评分：min-width 220px（可扩展）
  - 平均恢复时间：min-width 160px（可扩展）
  - 其他列保持固定宽度

**2. 平均恢复时间字段名修复**
- ✅ 修复前后端字段名不匹配问题
- ✅ 后端返回 `avg_mttr`，前端错误使用 `avg_recovery_time`
- ✅ 统一修改为 `avg_mttr`，共3处：
  - 节点健康度排行表格
  - 节点健康详情对话框
  - MTTR图表数据映射
- ✅ 同时修复 `last_anomaly_time` → `last_anomaly`
- ✅ 现在平均恢复时间能够正常显示

**3. MTTR图表数据字段修复**
- ✅ 修复 MTTR 图表数据字段：`avg_recovery_time` → `mttr`
- ✅ 与后端 `MTTRStatistics` 模型字段对应
- ✅ 图表现在能够正常显示平均恢复时间数据

### 📄 文档更新
- ✅ 新增 `docs/fix-table-layout-and-mttr.md` - 表格布局和MTTR字段修复详细报告

---

## [v2.13.1] - 2025-10-27 (下午-首次修复)

### 🐛 Bug 修复

#### 统计分析页面优化修复

**1. 节点健康度排行榜表格自动适配**
- ✅ 为健康度排行榜表格添加固定高度（600px）
- ✅ 超过10条数据时自动显示滚动条
- ✅ 表头固定，滚动时保持可见
- ✅ 改善了大数据量时的用户体验

**2. 平均恢复时间显示优化**
- ✅ MTTR图表添加友好的空状态提示："暂无已恢复的异常"
- ✅ 添加子标题说明："只有异常恢复后才能统计平均恢复时间"
- ✅ 表格中无数据时显示"-"，并提供工具提示："该节点暂无已恢复的异常记录"
- ✅ 避免用户误以为是系统错误

**3. 查看详情功能实现**
- ✅ 实现节点健康详情对话框
- ✅ 显示完整的节点健康信息：
  - 节点名称、集群名称
  - 健康度评分（带进度条可视化）
  - 健康等级标签（极好/良好/一般/较差/极差）
  - 总异常数、活跃异常数
  - 平均恢复时间、最近异常时间
- ✅ 添加统计卡片：健康指数、异常率百分比
- ✅ 根据异常状态显示不同提示（警告/成功）
- ✅ 提供"查看节点详情"按钮，可跳转到节点列表页面
- ✅ 添加精美的对话框样式

### 📄 文档更新
- ✅ 新增 `docs/fix-analytics-issues.md` - 详细的问题修复报告

---

## [v2.13.0] - 2025-10-27 (上午)

### ✨ 新功能

#### 1. 统计分析功能全面升级

**高级统计API** (10+ 新增接口):

- ✅ **按角色聚合统计** - 统计不同节点角色的异常分布
  - `GET /api/v1/anomalies/role-statistics`
- ✅ **按集群聚合统计** - 统计各集群的异常情况
  - `GET /api/v1/anomalies/cluster-aggregate`
- ✅ **单节点历史趋势** - 查看单个节点的异常变化趋势
  - `GET /api/v1/anomalies/node-trend`
- ✅ **MTTR 统计** - 计算平均恢复时间（Mean Time To Recovery）
  - `GET /api/v1/anomalies/mttr`
- ✅ **SLA 可用性** - 计算节点/集群的SLA可用性百分比
  - `GET /api/v1/anomalies/sla`
- ✅ **恢复率和复发率** - 统计异常恢复率和复发率
  - `GET /api/v1/anomalies/recovery-metrics`
- ✅ **节点健康度评分** - 综合评估节点健康状况（0-100分）
  - `GET /api/v1/anomalies/node-health`
- ✅ **热力图数据** - 时间 × 节点矩阵的异常分布
  - `GET /api/v1/anomalies/heatmap`
- ✅ **日历图数据** - 按日期聚合的异常数量
  - `GET /api/v1/anomalies/calendar`
- ✅ **Top 不健康节点** - 健康度最低的节点列表
  - `GET /api/v1/anomalies/top-unhealthy-nodes`

**关键指标计算**:

- **MTTR**: `AVG(duration) WHERE status='Resolved'`
- **SLA**: `(总时间 - 异常累计时长) / 总时间 × 100%`
- **恢复率**: `已恢复数 / 总异常数 × 100%`
- **复发率**: `重复异常次数 / 总恢复次数 × 100%`
- **健康度评分**: 综合异常次数、类型、持续时长、恢复速度等指标

#### 2. 定时报告配置管理（数据库驱动）

**核心特性**:

- ✅ **UI 界面管理** - 通过"系统配置 → 分析报告"页面管理报告配置
- ✅ **数据库存储** - 配置存储在数据库中，无需修改配置文件
- ✅ **Cron 调度** - 使用 Cron 表达式灵活配置执行时间
- ✅ **飞书自动推送** - 定时生成报告并推送到飞书群聊
- ✅ **多报告支持** - 可创建多个报告配置（日报、周报、月报等）
- ✅ **测试发送** - 配置前可测试推送渠道
- ✅ **手动执行** - 支持手动触发报告生成

**报告内容**:

- 时间范围内的统计摘要（总异常、活跃、已恢复、受影响节点）
- 异常趋势数据
- 异常类型分布
- Top 10 异常节点
- MTTR 和 SLA 关键指标
- 健康度最低的节点列表

**API 接口** (7个新增接口):

```
GET    /api/v1/anomaly-reports/configs         # 获取报告配置列表
GET    /api/v1/anomaly-reports/configs/:id     # 获取单个配置
POST   /api/v1/anomaly-reports/configs         # 创建配置
PUT    /api/v1/anomaly-reports/configs/:id     # 更新配置
DELETE /api/v1/anomaly-reports/configs/:id     # 删除配置
POST   /api/v1/anomaly-reports/configs/:id/test # 测试发送
POST   /api/v1/anomaly-reports/configs/:id/run  # 手动执行
```

#### 3. 节点健康度评分系统

**评分体系**:

| 分数范围 | 等级 | 说明 |
|---------|------|------|
| 90-100 | 优秀 | 节点运行稳定，极少出现异常 |
| 75-89 | 良好 | 偶尔出现异常，但恢复迅速 |
| 60-74 | 一般 | 存在一定数量的异常，需关注 |
| 40-59 | 较差 | 异常频繁或恢复缓慢，建议检查 |
| 0-39 | 很差 | 严重问题，需要立即处理 |

**影响因素**:

- 异常频率（权重 30%）
- 恢复速度（权重 30%）
- 异常严重性（权重 20%）
- 稳定性/复发率（权重 20%）

**前端组件**:

- `NodeHealthCard.vue` - 节点健康度评分卡
- 显示健康度评分、等级、影响因素分解
- 支持近7天健康度趋势图

#### 4. 新增前端 API 方法

**文件**: `frontend/src/api/anomaly.js`

新增 17 个 API 方法：

- 高级统计相关（10个）
- 报告配置管理（7个）

#### 5. 新增系统配置页面

**导航路径**: 系统配置 → 分析报告

**功能**:

- 报告配置列表（表格展示）
- 启用/禁用、编辑、删除、测试、手动执行
- 新增/编辑报告配置对话框
- Cron 表达式验证和说明

### 🔧 配置变更

#### 后端配置

**文件**: `backend/configs/config.yaml.example`

新增配置项：

```yaml
monitoring:
  enabled: true
  interval: 60
  report_scheduler_enabled: true  # 新增：启用报告调度器
```

#### 数据库迁移

**新增表**: `anomaly_report_configs`

```sql
CREATE TABLE anomaly_report_configs (
    id                SERIAL PRIMARY KEY,
    enabled           BOOLEAN DEFAULT FALSE,
    report_name       VARCHAR(100) NOT NULL,
    schedule          VARCHAR(50),
    frequency         VARCHAR(20),
    cluster_ids       TEXT,
    feishu_enabled    BOOLEAN DEFAULT FALSE,
    feishu_webhook    VARCHAR(500),
    email_enabled     BOOLEAN DEFAULT FALSE,
    email_recipients  TEXT,
    last_run_time     TIMESTAMP,
    next_run_time     TIMESTAMP,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 前端路由

新增路由：

- `/analytics-report-settings` - 分析报告配置页面

### 📦 新增依赖

#### 后端

- `github.com/robfig/cron/v3 v3.0.1` - Cron 任务调度

### 📁 新增/修改文件

#### 后端新增

1. `backend/internal/service/anomaly/statistics_extended.go` - 扩展统计方法
2. `backend/internal/service/anomaly/statistics_visualization.go` - 可视化数据方法
3. `backend/internal/service/anomaly/report.go` - 报告生成和配置管理
4. `backend/internal/handler/anomaly/statistics_handler.go` - 统计接口处理器
5. `backend/internal/handler/anomaly/report_handler.go` - 报告配置处理器
6. `backend/migrations/002_add_anomaly_analytics.sql` - 数据库迁移

#### 后端修改

1. `backend/internal/model/anomaly.go` - 新增数据结构和报告配置表
2. `backend/internal/model/migrate.go` - 注册新表迁移
3. `backend/internal/service/anomaly/anomaly.go` - 新增统计方法
4. `backend/internal/service/services.go` - 注册 AnomalyReport 服务
5. `backend/internal/handler/handlers.go` - 注册 AnomalyReport handler
6. `backend/internal/config/config.go` - 新增配置定义
7. `backend/configs/config.yaml.example` - 新增配置示例
8. `backend/cmd/main.go` - 启动/停止报告调度器

#### 前端新增

1. `frontend/src/views/analytics/ReportSettings.vue` - 报告配置管理页面
2. `frontend/src/components/analytics/NodeHealthCard.vue` - 节点健康度评分卡

#### 前端修改

1. `frontend/src/api/anomaly.js` - 新增 17 个 API 方法
2. `frontend/src/router/index.js` - 新增路由
3. `frontend/src/components/layout/Sidebar.vue` - 新增菜单项

#### 文档新增

1. `docs/analytics-advanced-features.md` - 高级功能文档
2. `docs/CHANGELOG.md` - 更新变更日志（本文件）

### 🎯 功能亮点

1. **数据库驱动配置** - 报告配置通过 UI 管理，无需修改配置文件
2. **灵活的调度机制** - 支持 Cron 表达式，可配置日/周/月报告
3. **全面的统计分析** - 覆盖角色、集群、节点多维度分析
4. **健康度评分系统** - 综合多指标评估节点健康状况
5. **可扩展的报告系统** - 支持飞书、邮件等多渠道推送（邮件预留）

### 📊 性能优化

1. **数据库索引优化** - 为常用查询字段添加索引
2. **缓存策略** - 使用 PostgreSQL 缓存表或内存缓存
3. **异步处理** - 报告生成使用后台任务

### 📚 文档更新

- ✅ 新增《统计分析高级功能文档》
- ✅ Cron 表达式说明和示例
- ✅ API 接口详细说明
- ✅ 使用指南和常见问题

### ⚠️ 注意事项

1. 升级后需要执行数据库迁移（GORM 自动迁移）
2. 如需启用报告调度器，请确保配置 `monitoring.report_scheduler_enabled: true`
3. 飞书推送需要在群聊中添加自定义机器人并配置 Webhook URL
4. 邮件推送功能预留，将在下一版本实现

---

## [v2.12.8] - 2025-10-27

### 🐛 Bug 修复

#### 1. WebSocket 频繁重连问题修复

**问题描述**：
- 批量操作完成后 WebSocket 连接频繁断开重连（每秒一次）
- 产生大量日志，影响系统可读性和性能
- 根本原因：前端在完成后仍在等待 `complete` 消息，形成重连循环

**修复内容**：

**前端优化** (`frontend/src/components/common/ProgressDialog.vue`):
- ✅ 添加重连限制：最多重连 5 次，防止无限循环
- ✅ 递增延迟策略：1秒 → 2秒 → 3秒，减少服务器压力
- ✅ 完成时立即关闭：收到 `complete` 消息后立即关闭 WebSocket
- ✅ 智能重连判断：任务完成或出错后不再重连
- ✅ 状态同步：连接成功后重置重连计数器

**后端优化** (`backend/internal/service/progress/progress.go`):
- ✅ 减少日志输出：移除非必要的连接/断开日志
- ✅ 静默关闭连接：正常关闭不记录日志
- ✅ 只记录异常：仅在真正的异常情况下记录错误日志

**修复效果**：
- 正常操作不产生日志，只在异常时记录
- 最多重连 5 次，避免无限循环
- 任务完成后立即关闭连接，不再重连

**相关文档**：
- `docs/websocket-reconnect-optimization.md`

---

#### 2. 标签/污点应用后不刷新问题修复

**问题描述**：
- 标签或污点应用到节点后，界面没有自动刷新
- 用户看不到最新的标签/污点信息
- 需要手动刷新页面才能看到更新

**根本原因**：
- ≤5 个节点的同步操作：完成后没有调用刷新函数
- \>5 个节点的批量操作：只刷新模板列表，未刷新节点数据

**修复内容**：

**标签管理页面** (`frontend/src/views/labels/LabelManage.vue`):
- ✅ 同步操作（≤5 个节点）：应用成功后添加 `refreshData(true)`
- ✅ 批量操作（>5 个节点）：进度完成后改为 `refreshData(true)`

**污点管理页面** (`frontend/src/views/taints/TaintManage.vue`):
- ✅ 同步操作（≤5 个节点）：应用成功后添加 `refreshData(true)`
- ✅ 批量操作（>5 个节点）：进度完成后改为 `refreshData(true)`

**修复效果**：
- 应用标签/污点后自动刷新节点数据
- 界面立即显示最新的标签/污点信息
- 提供流畅的用户体验

**影响文件**：
- `frontend/src/views/labels/LabelManage.vue`
- `frontend/src/views/taints/TaintManage.vue`

---

### 📝 文件变更清单

#### 前端修改
1. `frontend/src/components/common/ProgressDialog.vue` - WebSocket 重连优化
2. `frontend/src/views/labels/LabelManage.vue` - 添加刷新逻辑
3. `frontend/src/views/taints/TaintManage.vue` - 添加刷新逻辑

#### 后端修改
1. `backend/internal/service/progress/progress.go` - 优化日志输出

#### 新增文档
1. `docs/websocket-reconnect-optimization.md` - WebSocket 重连优化文档

---

## [v2.11.0] - 2025-10-22

### 🔧 Bug 修复

#### 统计分析数据类型错误修复

**问题描述**：
统计接口返回 500 错误：
```
sql: Scan error on column index 4, name "average_duration": 
converting driver.Value type string ("1669.6000000000000000") to a int64: invalid syntax
```

**修复内容**：
- ✅ 修复 `AnomalyStatistics.AverageDuration` 类型：`int64` → `float64`
- ✅ 修复 `AnomalyTypeStatistics` 字段命名一致性
- ✅ 更新相关 SQL 查询和排序逻辑

**影响文件**：
- `backend/internal/model/anomaly.go`
- `backend/internal/service/anomaly/anomaly.go`

---

### ♻️ 重构优化

#### 1. 统计分析页面重构

**优化内容**：
- ✅ 将统计分析页面重构为 Tab 分栏结构
  - **数据概览**：统计卡片展示
  - **趋势分析**：ECharts 图表展示
  - **异常记录**：异常列表和详情
- ✅ 删除对比分析功能（简化用户界面）

**改进效果**：
- 更清晰的信息层次
- 更好的用户体验
- 更快的页面加载速度

**变更文件**：
- `frontend/src/views/analytics/Analytics.vue` - 完全重构
- `frontend/src/components/analytics/CompareAnalysis.vue` - 已删除

---

#### 2. 图表数据修复

**修复问题**：
- ✅ 修复节点异常 Top 10 数据显示问题
- ✅ 修复异常类型分布数据映射
- ✅ 优化图表数据聚合逻辑

**技术实现**：
- 从 `anomalies` prop 直接聚合节点统计
- 添加空数据处理逻辑
- 优化图表渲染性能

**变更文件**：
- `frontend/src/components/analytics/TrendCharts.vue`

---

#### 3. 异常详情页面简化

**优化内容**：
- ✅ 删除处理建议模块（减少冗余信息）
- ✅ 保留核心信息：基本信息、时间线、节点快照、历史记录

**优化理由**：
- 处理建议多为通用内容，实际参考价值有限
- 简化页面结构，提升加载速度
- 聚焦于数据展示而非指导性内容

**变更文件**：
- `frontend/src/views/analytics/AnomalyDetail.vue`

---

### 📊 功能对比

#### 修改前 vs 修改后

| 功能模块 | 修改前 | 修改后 | 说明 |
|---------|-------|-------|------|
| 统计分析布局 | 单页面混合 | Tab 分栏结构 | ✅ 更清晰 |
| 对比分析 | ✓ 存在 | ✗ 已删除 | 简化功能 |
| 异常详情处理建议 | ✓ 存在 | ✗ 已删除 | 聚焦核心数据 |
| 节点异常 Top 10 | ✗ 数据错误 | ✅ 正常显示 | 已修复 |
| 异常类型分布 | ✗ 数据错误 | ✅ 正常显示 | 已修复 |
| 平均持续时间 | ✗ 类型错误 | ✅ 正常显示 | 已修复 |

---

### 🗂️ 文件变更清单

#### 后端修改
1. `backend/internal/model/anomaly.go` - 修复数据类型
2. `backend/internal/service/anomaly/anomaly.go` - 更新 SQL 查询

#### 前端修改
1. `frontend/src/views/analytics/Analytics.vue` - 完全重构（Tab 分栏）
2. `frontend/src/components/analytics/TrendCharts.vue` - 修复图表数据逻辑
3. `frontend/src/views/analytics/AnomalyDetail.vue` - 删除处理建议模块

#### 删除文件
1. `frontend/src/components/analytics/CompareAnalysis.vue` - 对比分析组件

---

### 📸 页面结构

#### 统计分析页面（重构后）

```
┌─────────────────────────────────────────┐
│  筛选器：集群、时间范围、手动检测         │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  Tab 1: 数据概览                         │
│  ├─ 总异常数                             │
│  ├─ 活跃异常                             │
│  ├─ 已恢复异常                           │
│  └─ 受影响节点                           │
├─────────────────────────────────────────┤
│  Tab 2: 趋势分析                         │
│  ├─ 异常趋势折线图                        │
│  ├─ 异常类型分布饼图                      │
│  ├─ 节点异常Top 10                       │
│  └─ 集群对比柱状图                        │
├─────────────────────────────────────────┤
│  Tab 3: 异常记录                         │
│  ├─ 过滤器（异常类型）                    │
│  ├─ 异常列表表格                         │
│  └─ 分页                                │
└─────────────────────────────────────────┘
```

#### 异常详情页面（精简后）

```
┌─────────────────────────────────────────┐
│  返回按钮    异常详情                     │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  基本信息                                │
│  ├─ 集群、节点、类型                      │
│  ├─ 持续时间、开始/结束时间               │
│  └─ 原因、详细消息                        │
├─────────────────────────────────────────┤
│  事件时间线                              │
│  ├─ 异常开始                             │
│  ├─ 状态变更                             │
│  └─ 异常恢复                             │
├─────────────────────────────────────────┤
│  节点状态快照                            │
│  ├─ 角色、版本、系统                      │
│  └─ CPU、内存、磁盘、Pod使用率            │
├─────────────────────────────────────────┤
│  历史异常记录                            │
│  └─ 该节点最近30天异常列表                │
└─────────────────────────────────────────┘
```

---

### ✅ 测试验证

#### API 测试

| 接口 | 测试结果 |
|------|---------|
| `GET /api/v1/anomalies/statistics` | ✅ 200 OK |
| `GET /api/v1/anomalies/type-statistics` | ✅ 200 OK |
| `GET /api/v1/anomalies/active` | ✅ 200 OK |
| `GET /api/v1/anomalies` | ✅ 200 OK |

#### 前端页面测试

| 页面功能 | 测试结果 |
|---------|---------|
| 统计分析 - 数据概览 Tab | ✅ 正常显示 |
| 统计分析 - 趋势分析 Tab | ✅ 图表正常 |
| 统计分析 - 异常记录 Tab | ✅ 列表正常 |
| 节点异常 Top 10 | ✅ 数据正确 |
| 异常类型分布 | ✅ 数据正确 |
| 异常详情页面 | ✅ 信息完整 |
| Tab 切换 | ✅ 流畅无卡顿 |

---

### 🚀 部署说明

#### 1. 数据库兼容性

**无需数据库迁移** ✅
- 只修改 Go 代码类型定义
- 数据库表结构不变
- 现有数据完全兼容

#### 2. 缓存处理

**自动过期，无需手动操作** ✅
- 旧缓存会在 TTL 过期后自动失效
- 建议 TTL: 5 分钟（默认配置）

#### 3. 部署步骤

```bash
# 1. 构建镜像
make docker-build

# 2. 推送镜像
docker push your-registry/kube-node-manager:v2.11.0

# 3. 更新 Kubernetes 部署
kubectl set image deployment/kube-node-manager \
  kube-node-manager=your-registry/kube-node-manager:v2.11.0

# 4. 监控滚动更新
kubectl rollout status deployment/kube-node-manager

# 5. 验证
kubectl logs -f deployment/kube-node-manager --tail=50
```

#### 4. 回滚方案

如需回滚：
```bash
kubectl rollout undo deployment/kube-node-manager
```

---

### 📝 注意事项

#### 前端兼容性

✅ **无影响**
- JavaScript 自动处理 float/int 转换
- API 响应格式保持不变
- 现有客户端无需升级

#### 性能影响

✅ **性能提升**
- Tab 分栏减少初始渲染内容
- 按需加载图表数据
- 删除冗余组件降低打包体积

#### 用户体验

✅ **体验优化**
- 更清晰的信息架构
- 更快的页面响应
- 更直观的操作流程

---

## 版本说明

### 版本格式

本项目遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/) 规范：

- **主版本号（MAJOR）**：不兼容的 API 修改
- **次版本号（MINOR）**：向下兼容的功能性新增
- **修订号（PATCH）**：向下兼容的问题修正

### 变更类型

- 🎉 **Added** - 新增功能
- 🔧 **Fixed** - Bug 修复
- ♻️ **Changed** - 功能变更
- ⚠️ **Deprecated** - 即将废弃的功能
- 🗑️ **Removed** - 已删除的功能
- 🔒 **Security** - 安全性修复
- 📝 **Docs** - 文档更新

---

## 贡献指南

在提交变更时，请遵循以下格式更新 CHANGELOG：

```markdown
## [版本号] - YYYY-MM-DD

### 变更类型

#### 简短描述

**问题描述**：（如果是修复）
- 问题现象

**修复/新增内容**：
- ✅ 具体变更 1
- ✅ 具体变更 2

**影响文件**：
- 文件路径 1
- 文件路径 2
```

---

**维护者**：Kube Node Manager Team  
**最后更新**：2025-10-27

