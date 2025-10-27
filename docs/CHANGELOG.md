# 变更日志 (CHANGELOG)

本文档记录了 Kube Node Manager 的所有版本变更历史。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

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

