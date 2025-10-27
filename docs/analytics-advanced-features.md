# 统计分析高级功能文档

> **版本**: v2.13.0  
> **更新日期**: 2025-10-27  
> **作者**: Kube Node Manager Team

---

## 📋 目录

1. [功能概述](#功能概述)
2. [新增功能列表](#新增功能列表)
3. [高级统计API](#高级统计api)
4. [定时报告配置](#定时报告配置)
5. [节点健康度评分](#节点健康度评分)
6. [使用指南](#使用指南)
7. [Cron表达式说明](#cron表达式说明)
8. [常见问题](#常见问题)

---

## 功能概述

基于 v2.12.8 的节点异常监控和统计分析功能，v2.13.0 版本新增了多项高级统计和分析能力，提供更全面的节点健康状况洞察和自动化报告功能。

### 核心改进

- ✅ **多维度统计聚合** - 支持按集群、节点角色、时间维度聚合异常统计
- ✅ **健康度评分系统** - 综合评估节点健康状况（0-100分）
- ✅ **SLA & MTTR 指标** - 计算SLA可用性、平均恢复时间等关键指标
- ✅ **数据库驱动的报告配置** - 通过UI界面管理定时报告，无需修改配置文件
- ✅ **飞书自动推送** - 定时生成异常报告并推送到飞书群聊
- ✅ **高级可视化** - 支持热力图、日历图等新图表类型（开发中）

---

## 新增功能列表

### 后端数据能力增强

| 功能 | 描述 | API 端点 |
|------|------|---------|
| 按角色聚合统计 | 统计不同节点角色的异常分布 | `GET /api/v1/anomalies/role-statistics` |
| 按集群聚合统计 | 统计各集群的异常情况 | `GET /api/v1/anomalies/cluster-aggregate` |
| 单节点历史趋势 | 查看单个节点的异常变化趋势 | `GET /api/v1/anomalies/node-trend` |
| MTTR 统计 | 平均恢复时间统计 | `GET /api/v1/anomalies/mttr` |
| SLA 可用性 | 计算节点/集群的SLA可用性百分比 | `GET /api/v1/anomalies/sla` |
| 恢复率和复发率 | 统计异常恢复率和复发率 | `GET /api/v1/anomalies/recovery-metrics` |
| 节点健康度评分 | 综合评估节点健康状况 | `GET /api/v1/anomalies/node-health` |
| 热力图数据 | 时间 × 节点矩阵的异常分布 | `GET /api/v1/anomalies/heatmap` |
| 日历图数据 | 按日期聚合的异常数量 | `GET /api/v1/anomalies/calendar` |
| Top不健康节点 | 健康度最低的节点列表 | `GET /api/v1/anomalies/top-unhealthy-nodes` |

### 定时报告配置管理

| 功能 | 描述 | API 端点 |
|------|------|---------|
| 获取报告配置列表 | 查看所有报告配置 | `GET /api/v1/anomaly-reports/configs` |
| 创建报告配置 | 新增定时报告 | `POST /api/v1/anomaly-reports/configs` |
| 更新报告配置 | 修改报告设置 | `PUT /api/v1/anomaly-reports/configs/:id` |
| 删除报告配置 | 移除报告 | `DELETE /api/v1/anomaly-reports/configs/:id` |
| 测试报告发送 | 测试报告推送渠道 | `POST /api/v1/anomaly-reports/configs/:id/test` |
| 手动执行报告 | 立即生成并发送报告 | `POST /api/v1/anomaly-reports/configs/:id/run` |

---

## 高级统计API

### 1. 按角色聚合统计

**端点**: `GET /api/v1/anomalies/role-statistics`

**参数**:
- `cluster_id` (可选): 集群ID，留空表示所有集群
- `start_time` (可选): 开始时间，RFC3339格式
- `end_time` (可选): 结束时间，RFC3339格式

**响应示例**:
```json
{
  "code": 0,
  "data": [
    {
      "role": "master",
      "total_anomalies": 45,
      "active_anomalies": 3,
      "resolved_anomalies": 42,
      "affected_nodes": 3
    },
    {
      "role": "worker",
      "total_anomalies": 128,
      "active_anomalies": 8,
      "resolved_anomalies": 120,
      "affected_nodes": 15
    }
  ]
}
```

### 2. MTTR 统计

**端点**: `GET /api/v1/anomalies/mttr`

**参数**:
- `entity_type` (必填): 实体类型，可选值：`node` | `cluster`
- `entity_name` (可选): 实体名称（节点名或集群名）
- `cluster_id` (可选): 集群ID
- `start_time` (可选): 开始时间
- `end_time` (可选): 结束时间

**计算公式**:
```
MTTR = AVG(anomaly_duration) WHERE status = 'Resolved'
```

**响应示例**:
```json
{
  "code": 0,
  "data": [
    {
      "entity_type": "cluster",
      "entity_name": "production",
      "total_resolved": 156,
      "total_recovery_time": 234560,
      "avg_recovery_time": 1503,
      "min_recovery_time": 120,
      "max_recovery_time": 7200
    }
  ]
}
```

### 3. SLA 可用性

**端点**: `GET /api/v1/anomalies/sla`

**计算公式**:
```
SLA% = (总时间 - 异常累计时长) / 总时间 × 100%
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "entity_type": "node",
    "entity_name": "node-1",
    "total_time": 604800,
    "downtime": 3600,
    "uptime": 601200,
    "sla_percentage": 99.40
  }
}
```

### 4. 节点健康度评分

**端点**: `GET /api/v1/anomalies/node-health`

**参数**:
- `node_name` (必填): 节点名称
- `cluster_id` (可选): 集群ID
- `start_time` (可选): 开始时间
- `end_time` (可选): 结束时间

**评分算法**:
```
健康度评分 = 100 - (异常频率影响 × 0.3 + 恢复速度影响 × 0.3 + 严重性影响 × 0.2 + 复发率影响 × 0.2)
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "node_name": "node-1",
    "cluster_name": "production",
    "node_roles": ["worker"],
    "health_score": 87.5,
    "total_anomalies": 12,
    "active_anomalies": 1,
    "avg_recovery_time": 1200,
    "max_duration": 3600,
    "recurrence_rate": 8.3
  }
}
```

---

## 定时报告配置

### 配置管理界面

导航至：**系统配置 → 分析报告**

### 配置项说明

| 配置项 | 说明 | 必填 |
|--------|------|------|
| 报告名称 | 报告的显示名称，例如"每日异常报告" | ✅ |
| 启用状态 | 是否启用该报告的定时执行 | ✅ |
| 报告频率 | 生成频率：每日/每周/每月 | ✅ |
| Cron表达式 | 具体的执行时间，使用Cron语法 | ✅ |
| 目标集群 | 报告涵盖的集群，留空表示全部集群 | ❌ |
| 飞书推送 | 是否启用飞书推送 | ❌ |
| Webhook URL | 飞书机器人的Webhook地址 | 🔶 (启用飞书时必填) |
| 邮件推送 | 是否启用邮件推送 (功能预留) | ❌ |

### 报告内容

定时报告包含以下内容：

1. **统计摘要**
   - 总异常数
   - 活跃异常数
   - 已恢复异常数
   - 受影响节点数
   - 涉及集群数

2. **异常趋势**
   - 按天统计的异常数量变化

3. **异常类型分布**
   - NotReady、MemoryPressure、DiskPressure等类型占比

4. **Top 10 异常节点**
   - 节点名称
   - 集群
   - 异常次数
   - 健康度评分

5. **关键指标**
   - 平均恢复时间 (MTTR)
   - SLA 可用性

### 操作说明

#### 创建报告配置

1. 点击"新增报告"按钮
2. 填写报告名称
3. 选择报告频率（每日/每周/每月）
4. 系统自动生成 Cron 表达式（也可手动修改）
5. （可选）选择目标集群
6. 配置飞书推送：
   - 在飞书群聊中添加自定义机器人
   - 复制 Webhook URL
   - 粘贴到配置中
7. 保存配置

#### 测试报告发送

在报告配置列表中点击"测试"按钮，系统将生成一份测试报告并发送到配置的飞书群聊。

#### 手动执行报告

在报告配置列表中点击"立即执行"按钮，系统将立即生成并发送报告，不影响定时计划。

---

## 节点健康度评分

### 评分体系

节点健康度评分是一个 0-100 的综合指标，评分越高表示节点越健康。

### 评分等级

| 分数范围 | 等级 | 说明 |
|---------|------|------|
| 90-100 | 优秀 | 节点运行稳定，极少出现异常 |
| 75-89 | 良好 | 偶尔出现异常，但恢复迅速 |
| 60-74 | 一般 | 存在一定数量的异常，需关注 |
| 40-59 | 较差 | 异常频繁或恢复缓慢，建议检查 |
| 0-39 | 很差 | 严重问题，需要立即处理 |

### 影响因素

1. **异常频率** (权重30%)
   - 异常次数越多，得分越低
   - 基准：50次异常以下为正常

2. **恢复速度** (权重30%)
   - 恢复时间越短，得分越高
   - 基准：1小时内恢复为优秀

3. **异常严重性** (权重20%)
   - 基于异常类型和持续时间
   - 长时间异常影响更大

4. **稳定性** (权重20%)
   - 复发率越低，得分越高
   - 反复出现的异常会降低得分

---

## 使用指南

### 场景1：查看集群整体健康状况

1. 访问"统计分析"页面
2. 选择目标集群
3. 查看统计摘要卡片
4. 查看异常趋势图

### 场景2：分析特定节点健康度

```bash
# 使用API查询
curl -X GET "http://your-server/api/v1/anomalies/node-health?node_name=node-1&cluster_id=1" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 场景3：配置每日异常报告

1. 导航至"系统配置 → 分析报告"
2. 点击"新增报告"
3. 填写配置：
   - 报告名称：每日异常报告
   - 频率：每日
   - Cron表达式：`0 9 * * *`（每天9:00）
   - 启用飞书推送
   - 输入飞书 Webhook URL
4. 保存配置
5. 点击"测试"验证配置

### 场景4：查询按角色聚合的统计

```javascript
// 前端调用示例
import { getRoleStatistics } from '@/api/anomaly'

const data = await getRoleStatistics({
  cluster_id: 1,
  start_time: '2025-10-20T00:00:00Z',
  end_time: '2025-10-27T00:00:00Z'
})
```

---

## Cron表达式说明

### 格式

```
分  时  日  月  周
*   *   *   *   *
```

### 常用示例

| 描述 | Cron表达式 |
|------|-----------|
| 每天 9:00 | `0 9 * * *` |
| 每天 18:00 | `0 18 * * *` |
| 每周一 9:00 | `0 9 * * 1` |
| 每周五 17:00 | `0 17 * * 5` |
| 每月1号 9:00 | `0 9 1 * *` |
| 每月15号 12:00 | `0 12 15 * *` |
| 每6小时 | `0 */6 * * *` |
| 每周一到周五 9:00 | `0 9 * * 1-5` |

### 在线工具

推荐使用在线工具生成和验证 Cron 表达式：
- [crontab.guru](https://crontab.guru)

---

## 常见问题

### Q1: 报告配置保存后没有自动执行？

**A**: 请检查以下几点：
1. 确认"启用状态"已开启
2. 确认 Cron 表达式正确
3. 查看"下次执行时间"是否正确
4. 检查后端配置 `monitoring.report_scheduler_enabled` 是否为 `true`

### Q2: 测试报告发送失败？

**A**: 可能的原因：
1. 飞书 Webhook URL 不正确
2. 飞书机器人被禁用
3. 网络连接问题
4. 后端日志中查看详细错误信息

### Q3: 健康度评分为什么突然下降？

**A**: 健康度评分综合考虑多个因素：
1. 检查节点是否出现新的异常
2. 查看异常恢复时间是否变长
3. 确认是否有复发的异常
4. 可以通过健康度卡片查看具体影响因素

### Q4: 如何设置报告只包含特定集群？

**A**: 在报告配置中的"目标集群"下拉框中选择需要的集群。如果留空，则报告将包含所有集群。

### Q5: 可以同时配置多个报告吗？

**A**: 可以。您可以创建多个报告配置，例如：
- 每日简报（所有集群）
- 每周详细报告（生产集群）
- 每月总结报告（所有集群）

### Q6: 报告内容可以自定义吗？

**A**: 当前版本的报告内容是固定的。未来版本将支持报告模板自定义功能。

### Q7: 如何停止某个定时报告？

**A**: 在报告配置列表中，将对应报告的"启用状态"切换为"禁用"即可。也可以直接删除该配置。

### Q8: 邮件推送功能什么时候上线？

**A**: 邮件推送功能正在开发中，预计在下一个版本（v2.14.0）中上线。

---

## 技术实现细节

### 数据库表结构

#### anomaly_report_configs

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

CREATE INDEX idx_report_configs_enabled ON anomaly_report_configs(enabled);
```

### 缓存策略

| 数据类型 | TTL | 说明 |
|---------|-----|------|
| 统计数据 | 5分钟 | 基础统计信息 |
| 活跃异常 | 30秒 | 实时性要求高 |
| 集群列表 | 10分钟 | 变化频率低 |
| 类型统计 | 5分钟 | 中等实时性 |

### 性能优化

1. **数据库索引**：为常用查询字段添加索引
2. **缓存机制**：使用 PostgreSQL 缓存表或内存缓存
3. **分页查询**：大数据量采用分页
4. **异步处理**：报告生成使用后台任务

---

## 版本历史

### v2.13.0 (2025-10-27)

**新增功能**:
- ✅ 按角色/集群聚合统计
- ✅ 单节点历史趋势分析
- ✅ MTTR、SLA、恢复率统计
- ✅ 节点健康度评分系统
- ✅ 定时报告配置管理（数据库驱动）
- ✅ 飞书自动推送报告

**API 变更**:
- 新增 10+ 统计分析API
- 新增 6+ 报告配置管理API

**配置变更**:
- 新增 `monitoring.report_scheduler_enabled` 配置项
- 新增"系统配置 → 分析报告"管理页面

### v2.12.8 (之前版本)

- 基础异常监控和统计
- 异常列表和详情查看
- 简单的统计图表

---

## 联系支持

如有问题或建议，请通过以下方式联系我们：

- GitHub Issues: [kube-node-manager/issues](https://github.com/your-org/kube-node-manager/issues)
- 邮箱: support@example.com

---

**文档最后更新**: 2025-10-27  
**适用版本**: v2.13.0 及以上

