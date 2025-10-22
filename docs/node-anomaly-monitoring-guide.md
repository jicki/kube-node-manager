# 节点异常监控功能使用指南

## 功能概述

节点异常监控功能自动检测和记录 Kubernetes 集群中的节点异常状态，包括：
- **NotReady**: 节点未就绪
- **MemoryPressure**: 内存压力
- **DiskPressure**: 磁盘压力
- **PIDPressure**: PID 压力
- **NetworkUnavailable**: 网络不可用

## 功能特性

### 后端功能

1. **自动监控**
   - 后台定时检测所有活跃集群的节点状态
   - 可配置监控周期（默认 60 秒）
   - 自动记录异常的开始和结束时间

2. **异常生命周期管理**
   - 检测到异常时自动创建 Active 状态记录
   - 持续异常时更新最后检查时间
   - 异常恢复时自动标记为 Resolved 并记录持续时长

3. **查询和统计**
   - 支持多条件过滤（集群、节点、异常类型、状态、时间范围）
   - 支持按天/周维度统计
   - 提供异常类型统计
   - 分页查询

4. **手动触发检测**
   - 管理员可手动触发一次检测

### 前端功能

1. **统计卡片**
   - 总异常次数
   - 活跃异常数
   - 已恢复异常数
   - 受影响节点数

2. **过滤和查询**
   - 集群选择
   - 异常类型选择
   - 时间范围选择
   - 统计维度切换（按天/按周）

3. **异常记录列表**
   - 详细的异常记录表格
   - 支持排序和分页
   - 持续时长自动计算
   - 导出为 CSV 功能

## 配置说明

### 1. 启用监控功能

编辑配置文件 `backend/configs/config.yaml`:

```yaml
monitoring:
  enabled: true       # 是否启用节点异常监控
  interval: 60        # 监控周期（秒），建议 60-300 秒
```

**建议配置：**
- 开发环境：60-120 秒
- 生产环境：120-300 秒（避免对集群造成过多压力）

### 2. 数据库迁移

首次启动时，系统会自动创建 `node_anomalies` 表。

## API 接口

### 1. 获取异常记录列表

```
GET /api/v1/anomalies
```

**查询参数：**
- `cluster_id`: 集群ID（可选）
- `node_name`: 节点名称（可选）
- `anomaly_type`: 异常类型（可选）
- `status`: 异常状态 Active/Resolved（可选）
- `start_time`: 开始时间 RFC3339 格式（可选）
- `end_time`: 结束时间 RFC3339 格式（可选）
- `page`: 页码，默认 1
- `page_size`: 每页数量，默认 20

**示例：**
```bash
curl -X GET "http://localhost:8080/api/v1/anomalies?cluster_id=1&page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. 获取统计数据

```
GET /api/v1/anomalies/statistics
```

**查询参数：**
- `cluster_id`: 集群ID（可选）
- `anomaly_type`: 异常类型（可选）
- `start_time`: 开始时间（可选）
- `end_time`: 结束时间（可选）
- `dimension`: 统计维度 day/week，默认 day

**示例：**
```bash
curl -X GET "http://localhost:8080/api/v1/anomalies/statistics?dimension=day" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. 获取活跃异常

```
GET /api/v1/anomalies/active
```

**查询参数：**
- `cluster_id`: 集群ID（可选）

### 4. 获取异常类型统计

```
GET /api/v1/anomalies/type-statistics
```

**查询参数：**
- `cluster_id`: 集群ID（可选）
- `start_time`: 开始时间（可选）
- `end_time`: 结束时间（可选）

### 5. 手动触发检测（仅管理员）

```
POST /api/v1/anomalies/check
```

## 使用步骤

### 1. 启动服务

```bash
cd backend
go run cmd/main.go
```

服务启动后，监控服务会自动开始工作。

### 2. 访问前端页面

打开浏览器访问：`http://localhost:8080/analytics`

### 3. 查看异常统计

- 页面顶部显示统计卡片，实时展示异常概况
- 使用过滤条件筛选特定集群或时间范围的异常
- 切换按天/按周维度查看统计趋势

### 4. 查看异常详情

- 异常记录列表显示所有异常的详细信息
- 点击列标题进行排序
- 使用分页功能浏览大量记录

### 5. 导出数据

点击"导出"按钮将当前页面的异常记录导出为 CSV 文件。

## 测试步骤

### 1. 测试自动监控

1. 启动服务，确保配置中 `monitoring.enabled = true`
2. 检查日志，应该看到类似信息：
   ```
   Starting node anomaly monitoring with interval: 1m0s
   ```
3. 等待监控周期后，查看数据库或通过 API 确认是否有记录

### 2. 测试异常检测

如果有测试集群，可以人为创建异常：

```bash
# 在测试集群中禁止调度一个节点（模拟异常）
kubectl cordon node-name

# 等待监控周期后，查看是否记录了异常
curl -X GET "http://localhost:8080/api/v1/anomalies/active" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. 测试异常恢复

```bash
# 恢复节点
kubectl uncordon node-name

# 等待监控周期后，异常应该被标记为 Resolved
```

### 4. 测试手动触发

```bash
# 以管理员身份登录
curl -X POST "http://localhost:8080/api/v1/anomalies/check" \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

### 5. 测试前端页面

1. 访问 `http://localhost:8080/analytics`
2. 验证统计卡片显示正确
3. 测试过滤功能
4. 测试分页功能
5. 测试导出功能

## 性能考虑

1. **监控周期设置**
   - 过短：增加集群 API 压力
   - 过长：可能遗漏短暂的异常
   - 推荐：60-300 秒

2. **数据库索引**
   系统已自动创建以下索引：
   - `idx_cluster_node`: (cluster_id, node_name)
   - `idx_anomaly_type`: anomaly_type
   - `idx_status`: status
   - `idx_start_time`: start_time

3. **数据清理**
   建议定期清理旧的已恢复异常记录（如 90 天前的记录）

## 故障排查

### 监控服务未启动

**症状：** 日志中没有监控相关信息

**解决：**
1. 检查配置文件中 `monitoring.enabled` 是否为 `true`
2. 检查配置文件路径是否正确

### 没有检测到异常

**症状：** 集群有异常节点但未记录

**解决：**
1. 检查集群状态是否为 Active
2. 检查集群 kubeconfig 是否有效
3. 检查日志是否有错误信息
4. 手动触发检测并查看日志

### 前端页面加载失败

**症状：** 访问 /analytics 返回 404

**解决：**
1. 确认前端代码已编译
2. 检查路由配置是否正确
3. 清除浏览器缓存

## 数据库表结构

```sql
CREATE TABLE node_anomalies (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    cluster_id BIGINT NOT NULL,
    cluster_name VARCHAR(255) NOT NULL,
    node_name VARCHAR(255) NOT NULL,
    anomaly_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'Active',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NULL,
    duration BIGINT DEFAULT 0,
    reason TEXT,
    message TEXT,
    last_check TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_cluster_node (cluster_id, node_name),
    INDEX idx_anomaly_type (anomaly_type),
    INDEX idx_status (status),
    INDEX idx_start_time (start_time)
);
```

## 未来改进方向

1. 添加异常告警功能（邮件、Webhook）
2. 支持更多异常类型（自定义条件）
3. 添加趋势分析和预测
4. 集成 Prometheus 监控指标
5. 支持异常记录导出为 PDF 报告

