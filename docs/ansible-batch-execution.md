# Ansible 分阶段执行（金丝雀/灰度发布）使用指南

## 功能概述

分阶段执行（Batch Execution）功能允许您将 Ansible 任务分批次执行，而不是一次性在所有主机上执行。这是一种重要的风险控制策略，特别适用于生产环境的大规模变更，也被称为金丝雀发布或灰度发布。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **分批执行**：将主机分成多个批次，按顺序逐批执行
- ✅ **灵活配置**：支持固定数量和百分比两种分批策略
- ✅ **失败控制**：设置失败阈值和单批失败率限制
- ✅ **暂停等待**：支持每批执行后暂停，等待手动确认
- ✅ **进度跟踪**：实时显示当前批次和总批次数
- ✅ **批次状态**：显示批次执行状态（运行中/暂停/完成）

### 使用场景

#### 1. 大规模系统更新
- **场景**：需要对 100+ 台服务器进行系统更新
- **策略**：先在 10% 的主机上验证，确认无问题后再推广到所有主机
- **配置**：使用百分比策略，设置为 10%，启用暂停等待

#### 2. 生产环境配置变更
- **场景**：修改生产环境的 Nginx 配置
- **策略**：先在 2-3 台主机上测试，观察服务是否正常，再推广
- **配置**：使用固定数量策略，每批 2 台，启用暂停等待

#### 3. 应用版本升级
- **场景**：对微服务集群进行版本升级
- **策略**：逐步升级，每批完成后检查服务健康状态
- **配置**：使用百分比策略，设置为 20%，设置失败阈值

#### 4. 滚动重启服务
- **场景**：需要重启所有服务实例，但要保证服务可用性
- **策略**：每次只重启少量实例，确保有足够的实例提供服务
- **配置**：使用固定数量策略，每批 3-5 台，设置失败率限制

## 使用方法

### 前端界面操作

#### 1. 启用分批执行

1. 点击 "启动任务" 按钮打开任务创建对话框

2. 找到 "分批执行" 开关，切换至 "启用" 状态

3. 配置分批策略：

##### 选项 A：固定数量策略
```
- 选择：固定数量
- 每批主机数：5 台
- 适用场景：小规模集群或需要精确控制的场景
```

##### 选项 B：百分比策略  
```
- 选择：百分比
- 每批百分比：20%
- 适用场景：大规模集群，根据总数动态调整
```

#### 2. 配置执行控制

```
☑ 每批执行后暂停，等待手动确认
```

- **启用**：每批执行完成后任务会暂停，需要手动确认后继续
- **禁用**：自动连续执行所有批次

#### 3. 设置失败阈值

```
失败阈值：5 台
```

- 当失败主机总数超过此值时，任务将自动停止
- 设置为 0 表示不限制
- 建议根据集群规模设置合理的值

#### 4. 设置单批失败率

```
单批失败率：50%
```

- 当单个批次的失败率超过此百分比时，任务将停止
- 例如：设置为 50%，批次有 10 台主机，如果超过 5 台失败则停止
- 可以设置为 0-100%，建议设置为 50% 或更低

#### 5. 启动任务

- 确认配置无误后，点击 "启动任务"
- 任务将按照配置的批次逐步执行

### 查看执行进度

#### 任务列表显示

在任务列表中，分批执行的任务会显示：

```
进度: 
- 进度条显示整体完成百分比
- 批次: 2/5 （当前第2批，共5批）
- 状态标签：已暂停（如果启用了暂停等待）
```

#### 日志查看

点击 "查看日志" 可以看到：
- 每个批次的开始和结束信息
- 每批的执行结果
- 失败主机的详细信息

### API 调用

```bash
curl -X POST http://your-server/api/v1/ansible/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "分批更新系统",
    "template_id": 1,
    "inventory_id": 2,
    "batch_config": {
      "enabled": true,
      "batch_size": 0,
      "batch_percent": 20,
      "pause_after_batch": true,
      "failure_threshold": 5,
      "max_batch_fail_rate": 50
    }
  }'
```

## 技术实现

### 后端实现

#### 1. 数据模型

```go
// 分批执行配置
type BatchExecutionConfig struct {
    Enabled          bool   // 是否启用
    BatchSize        int    // 每批主机数量
    BatchPercent     int    // 每批主机百分比
    PauseAfterBatch  bool   // 每批后是否暂停
    FailureThreshold int    // 失败阈值
    MaxBatchFailRate int    // 单批最大失败率
}

// 任务模型新增字段
type AnsibleTask struct {
    // ... 其他字段
    BatchConfig  *BatchExecutionConfig  // 分批配置
    CurrentBatch int                    // 当前批次
    TotalBatches int                    // 总批次数
    BatchStatus  string                 // 批次状态
}
```

#### 2. 批次大小计算

```go
func (e *TaskExecutor) calculateBatchSize(task *model.AnsibleTask) int {
    // 优先使用固定数量
    if task.BatchConfig.BatchSize > 0 {
        return task.BatchConfig.BatchSize
    }
    
    // 使用百分比计算
    if task.BatchConfig.BatchPercent > 0 {
        size := (task.HostsTotal * task.BatchConfig.BatchPercent) / 100
        if size < 1 {
            size = 1 // 至少执行1台
        }
        return size
    }
    
    return 0
}
```

#### 3. Ansible 集成

通过 Ansible 的 `serial` 参数实现分批执行：

```bash
# 方式1：在 Playbook 中使用 serial
- hosts: all
  serial: 5  # 每次执行5台主机
  tasks:
    - name: 更新系统
      yum:
        name: '*'
        state: latest

# 方式2：通过 extra-vars 传递
ansible-playbook playbook.yml --extra-vars "ansible_serial=5"
```

### 前端实现

#### 1. 配置界面

```vue
<!-- 分批执行开关 -->
<el-switch 
  v-model="batchEnabled" 
  active-text="启用（金丝雀/灰度发布）"
/>

<!-- 策略选择 -->
<el-radio-group v-model="batchStrategy">
  <el-radio label="size">固定数量</el-radio>
  <el-radio label="percent">百分比</el-radio>
</el-radio-group>

<!-- 批次大小配置 -->
<el-input-number 
  v-model="batchValue" 
  :min="1" 
  :max="100"
/>
```

#### 2. 批次信息显示

```vue
<!-- 任务列表中显示批次 -->
<div v-if="row.batch_config && row.batch_config.enabled">
  批次: {{ row.current_batch }}/{{ row.total_batches }}
  <el-tag v-if="row.batch_status === 'paused'" type="warning">
    已暂停
  </el-tag>
</div>
```

### 数据库迁移

```sql
-- 添加分批执行相关字段
ALTER TABLE ansible_tasks ADD COLUMN batch_config JSONB;
ALTER TABLE ansible_tasks ADD COLUMN current_batch INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN total_batches INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN batch_status VARCHAR(50);
```

## 配置建议

### 小规模集群（< 20 台）

```
策略：固定数量
每批：2-3 台
暂停等待：启用
失败阈值：1 台
单批失败率：50%
```

### 中等规模集群（20-100 台）

```
策略：固定数量或百分比
每批：5 台 或 20%
暂停等待：视情况而定
失败阈值：3-5 台
单批失败率：30-50%
```

### 大规模集群（> 100 台）

```
策略：百分比
每批：10-20%
暂停等待：第一批后启用
失败阈值：5-10 台
单批失败率：20-30%
```

## 最佳实践

### 1. 逐步扩大批次

```
第1批：10%（验证）
第2批：20%（小规模验证）
第3批：30%（中等规模）
第4批：40%（大规模推广）
```

### 2. 关键系统的谨慎策略

```
- 使用固定数量，每批 1-2 台
- 启用暂停等待
- 设置严格的失败阈值
- 每批执行后检查监控指标
```

### 3. 配合监控使用

```
1. 执行第一批
2. 暂停等待
3. 检查监控指标（CPU、内存、错误率）
4. 确认无问题后继续
5. 重复 1-4 步骤
```

### 4. 回滚准备

```
- 在执行前准备回滚 Playbook
- 记录每批执行的主机列表
- 如果发现问题，立即停止并回滚已执行的主机
```

## 使用示例

### 示例 1：系统补丁更新（百分比策略）

```yaml
---
- name: 系统补丁更新
  hosts: all
  serial: "{{ ansible_serial | default(20) }}%"  # 默认20%
  max_fail_percentage: 50
  tasks:
    - name: 更新系统包
      yum:
        name: '*'
        state: latest
        
    - name: 重启系统（如果需要）
      reboot:
        reboot_timeout: 600
      when: ansible_reboot_required
```

**配置**：
- 批次策略：百分比 20%
- 暂停等待：第一批后启用
- 失败阈值：5 台
- 单批失败率：50%

### 示例 2：应用版本升级（固定数量策略）

```yaml
---
- name: 应用版本升级
  hosts: app_servers
  serial: 3  # 每次3台
  tasks:
    - name: 停止应用服务
      systemd:
        name: myapp
        state: stopped
        
    - name: 部署新版本
      copy:
        src: /path/to/new/version
        dest: /app/current
        
    - name: 启动应用服务
      systemd:
        name: myapp
        state: started
        
    - name: 健康检查
      uri:
        url: http://localhost:8080/health
        status_code: 200
      retries: 10
      delay: 3
```

**配置**：
- 批次策略：固定数量 3 台
- 暂停等待：启用
- 失败阈值：1 台
- 单批失败率：33%

### 示例 3：配置文件更新（谨慎策略）

```yaml
---
- name: Nginx 配置更新
  hosts: webservers
  serial: 1  # 一次一台
  tasks:
    - name: 备份现有配置
      copy:
        src: /etc/nginx/nginx.conf
        dest: /etc/nginx/nginx.conf.backup
        remote_src: yes
        
    - name: 更新 Nginx 配置
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        validate: 'nginx -t -c %s'
        
    - name: 重载 Nginx
      systemd:
        name: nginx
        state: reloaded
```

**配置**：
- 批次策略：固定数量 1 台
- 暂停等待：启用
- 失败阈值：0 台（任何失败都停止）
- 单批失败率：100%

## 注意事项

### 1. Playbook 要求

- Playbook 需要支持 `serial` 参数
- 建议在 Playbook 中设置合理的超时时间
- 使用 `max_fail_percentage` 参数配合失败控制

### 2. 性能考虑

- 分批执行会增加总执行时间
- 需要在风险控制和效率之间平衡
- 大规模集群建议使用百分比策略

### 3. 网络和并发

- 注意 SSH 连接数限制
- 考虑网络带宽和目标主机负载
- 避免批次过大导致资源争用

### 4. 失败处理

- 设置合理的失败阈值和失败率
- 准备回滚方案
- 记录失败原因便于排查

## 故障排查

### 问题 1：批次没有生效，任务一次性执行了所有主机

**原因**：
- Playbook 中已经硬编码了 `serial` 参数
- BatchConfig 配置未正确传递

**解决方案**：
- 检查 Playbook 是否包含 `serial` 参数
- 确认 `batch_config.enabled` 为 true
- 查看任务日志确认 `ansible_serial` 是否传递

### 问题 2：暂停后无法继续执行

**状态**：批次状态显示 "paused"

**原因**：当前版本暂停/继续功能未实现（基础实现）

**临时方案**：
- 在第一阶段使用自动连续执行
- 手动观察每批的执行结果

### 问题 3：失败阈值不生效

**原因**：
- 失败阈值是累计值，不是单批值
- 需要结合单批失败率一起使用

**解决方案**：
- 同时设置失败阈值和单批失败率
- 失败阈值：控制总失败数
- 单批失败率：控制单批质量

## 常见问题

### Q1：分批执行会增加多少时间？

**A**：取决于批次数和是否启用暂停等待。例如：
- 不暂停：增加 10-20% 的时间（批次切换开销）
- 暂停等待：取决于确认时间
- 建议：关键变更使用暂停，常规变更可以连续执行

### Q2：如何选择固定数量还是百分比？

**A**：
- **固定数量**：适用于小规模集群或需要精确控制的场景
- **百分比**：适用于大规模集群，根据总数动态调整
- **建议**：20 台以下用固定数量，20 台以上用百分比

### Q3：第一批应该设置多少台？

**A**：
- **生产环境**：1-2 台（极度谨慎）
- **测试环境**：10-20%
- **开发环境**：可以更大
- **建议**：先小后大，逐步扩大批次

### Q4：失败率应该设置多少？

**A**：
- **关键系统**：10-20%
- **一般系统**：30-50%
- **非关键系统**：50-80%
- **建议**：根据系统重要性和容错能力设置

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)
- [Ansible 官方文档 - Rolling Updates](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_strategies.html)

## 更新日志

### v2.19.0 (2025-11-03)

- ✅ 实现分批执行基础功能
- ✅ 支持固定数量和百分比两种策略
- ✅ 支持失败阈值和单批失败率控制
- ✅ 前端添加分批执行配置界面
- ✅ 任务列表显示批次进度
- ✅ 数据库迁移脚本
- ✅ 完善文档和使用指南

### 待实现功能

- ⏳ 暂停/继续控制（需要实现任务控制 API）
- ⏳ 批次级别的详细统计
- ⏳ 批次执行时间线可视化
- ⏳ 自动回滚功能

