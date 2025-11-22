# SSH 密钥迁移及系统配置清理总结

## 概述

本次任务完成了以下主要工作：
1. 全面删除系统配置中的"分析报告"功能
2. 将 Ansible 模块中的 SSH 密钥迁移到系统级配置
3. 检查并修改了所有相关的关联关系

## 一、删除分析报告功能

### 1.1 已删除的文件

#### 前端
- `/frontend/src/views/analytics/ReportSettings.vue` - 分析报告配置页面

#### 后端
- `/backend/internal/handler/anomaly/report_handler.go` - 报告处理器
- `/backend/internal/service/anomaly/report.go` - 报告服务
- `/backend/migrations/002_add_anomaly_analytics.sql` - 数据库迁移文件

### 1.2 已修改的文件

#### 前端修改
1. **`frontend/src/router/index.js`**
   - 删除了 `/analytics-report-settings` 路由

2. **`frontend/src/components/layout/Sidebar.vue`**
   - 删除了"分析报告"菜单项
   - 更新了路由检查逻辑

#### 后端修改
1. **`backend/internal/model/anomaly.go`**
   - 删除了 `ReportFrequency` 类型
   - 删除了 `AnomalyReportConfig` 结构体

2. **`backend/internal/model/migrate.go`**
   - 从模型列表中移除 `AnomalyReportConfig`

3. **`backend/internal/service/services.go`**
   - 删除了 `AnomalyReport` 服务字段
   - 删除了报告服务的初始化代码

4. **`backend/internal/handler/handlers.go`**
   - 删除了 `AnomalyReport` 处理器字段

5. **`backend/cmd/main.go`**
   - 删除了报告调度器的启动代码
   - 删除了报告调度器的停止代码
   - 删除了 `/api/v1/anomaly-reports` 路由组

## 二、SSH 密钥系统级迁移

### 2.1 新增的文件

#### 后端
1. **`backend/internal/model/sshkey.go`**
   - 定义了 `SystemSSHKey` 模型（替代 `AnsibleSSHKey`）
   - 定义了相关的请求/响应结构
   - 表名：`system_ssh_keys`

2. **`backend/internal/service/sshkey/sshkey.go`**
   - 系统级 SSH 密钥服务
   - 提供完整的 CRUD 操作
   - 支持 AES-256-GCM 加密

3. **`backend/internal/handler/sshkey/sshkey.go`**
   - 系统级 SSH 密钥 HTTP 处理器
   - 支持列表、创建、更新、删除操作

4. **`backend/migrations/024_migrate_ansible_ssh_keys_to_system.sql`**
   - 数据迁移脚本
   - 自动将 `ansible_ssh_keys` 表数据迁移到 `system_ssh_keys`
   - 更新 `ansible_inventories` 表中的 SSH 密钥引用
   - 保留原表以便回滚

#### 前端
1. **`frontend/src/views/system/SSHKeyManage.vue`**
   - 从 `ansible/SSHKeyManage.vue` 复制并调整
   - API 路径更新为 `/api/v1/ssh-keys`

### 2.2 已修改的文件

#### 后端修改
1. **`backend/internal/model/migrate.go`**
   - 添加了 `SystemSSHKey` 到模型列表

2. **`backend/internal/service/services.go`**
   - 添加了 `SSHKey *sshkey.Service` 字段
   - 初始化系统级 SSH 密钥服务
   - 支持 `SSH_ENCRYPTION_KEY` 环境变量

3. **`backend/internal/handler/handlers.go`**
   - 添加了 `SSHKey *sshkey.Handler` 字段
   - 初始化 SSH 密钥处理器

4. **`backend/cmd/main.go`**
   - 添加了 `/api/v1/ssh-keys` 路由组
   - 包含列表、获取、创建、更新、删除接口

#### 前端修改
1. **`frontend/src/router/index.js`**
   - 添加了 `/ssh-keys` 路由（系统配置菜单）

2. **`frontend/src/components/layout/Sidebar.vue`**
   - 在"系统配置"子菜单中添加"SSH 密钥"项
   - 更新了路由检查逻辑以包含 `/ssh-keys`

## 三、关键变更说明

### 3.1 SSH 密钥模型变更

#### 原模型（AnsibleSSHKey）
```go
type AnsibleSSHKey struct {
    ID          uint
    Name        string
    // ... 其他字段
}
// 表名: ansible_ssh_keys
```

#### 新模型（SystemSSHKey）
```go
type SystemSSHKey struct {
    ID          uint
    Name        string
    // ... 相同的字段
}
// 表名: system_ssh_keys
```

### 3.2 API 路由变更

#### 原路由（Ansible 模块专用）
- `GET /api/v1/ansible/ssh-keys`
- `POST /api/v1/ansible/ssh-keys`
- `PUT /api/v1/ansible/ssh-keys/:id`
- `DELETE /api/v1/ansible/ssh-keys/:id`

#### 新路由（系统级）
- `GET /api/v1/ssh-keys`
- `POST /api/v1/ssh-keys`
- `PUT /api/v1/ssh-keys/:id`
- `DELETE /api/v1/ssh-keys/:id`

### 3.3 环境变量变更

#### 新增环境变量
```bash
# SSH 密钥加密密钥（推荐32字节用于AES-256）
SSH_ENCRYPTION_KEY=your-32-byte-encryption-key-here

# 向后兼容旧变量名
# ANSIBLE_ENCRYPTION_KEY=your-32-byte-encryption-key-here
```

## 四、数据库迁移

### 4.1 新表结构

```sql
CREATE TABLE system_ssh_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    username VARCHAR(255) NOT NULL,
    private_key TEXT,
    passphrase TEXT,
    password TEXT,
    port INTEGER DEFAULT 22,
    is_default BOOLEAN DEFAULT FALSE,
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
```

### 4.2 迁移步骤

1. **自动迁移**（通过 024_migrate_ansible_ssh_keys_to_system.sql）
   - 创建 `system_ssh_keys` 表
   - 将 `ansible_ssh_keys` 数据复制到 `system_ssh_keys`
   - 更新 `ansible_inventories.ssh_key_id` 引用
   - 保留 `ansible_ssh_keys` 表（标记为废弃）

2. **手动验证**
   ```sql
   -- 检查迁移结果
   SELECT COUNT(*) FROM ansible_ssh_keys WHERE deleted_at IS NULL;
   SELECT COUNT(*) FROM system_ssh_keys WHERE deleted_at IS NULL;
   
   -- 验证引用更新
   SELECT ai.id, ai.name, ai.ssh_key_id, ssk.name as ssh_key_name
   FROM ansible_inventories ai
   LEFT JOIN system_ssh_keys ssk ON ai.ssh_key_id = ssk.id
   WHERE ai.ssh_key_id IS NOT NULL;
   ```

## 五、向后兼容性

### 5.1 保留的功能
- `ansible_ssh_keys` 表保留（标记为废弃）
- Ansible 模块仍可正常工作
- 现有的 Inventory 引用自动更新

### 5.2 废弃的功能
- ⚠️ `GET /api/v1/ansible/ssh-keys` 路由（使用 `/api/v1/ssh-keys` 代替）
- ⚠️ 异常报告配置功能（已完全移除）
- ⚠️ `/analytics-report-settings` 前端路由

## 六、影响范围

### 6.1 影响的模块
1. **系统配置**
   - 删除：分析报告配置
   - 新增：SSH 密钥管理

2. **Ansible 模块**
   - SSH 密钥引用自动更新
   - 功能保持不变

3. **数据库**
   - 新表：`system_ssh_keys`
   - 保留：`ansible_ssh_keys`（废弃）
   - 更新：`ansible_inventories.ssh_key_id` 引用

### 6.2 不影响的功能
- 节点管理
- 集群管理
- 标签和污点管理
- 异常监控（删除了报告功能，保留监控功能）
- Ansible 任务执行
- 其他系统配置（GitLab、飞书）

## 七、部署指南

### 7.1 部署前准备
```bash
# 1. 备份数据库
pg_dump -U postgres kube_node_manager > backup_$(date +%Y%m%d).sql

# 2. 设置环境变量
export SSH_ENCRYPTION_KEY="your-32-byte-encryption-key-change-in-production!!"

# 3. 检查当前SSH密钥数量
psql -U postgres -d kube_node_manager -c "SELECT COUNT(*) FROM ansible_ssh_keys WHERE deleted_at IS NULL;"
```

### 7.2 部署步骤
```bash
# 1. 停止应用
kubectl scale deployment kube-node-manager --replicas=0

# 2. 运行数据库迁移
psql -U postgres -d kube_node_manager -f backend/migrations/024_migrate_ansible_ssh_keys_to_system.sql

# 3. 验证迁移
psql -U postgres -d kube_node_manager -c "
SELECT 
    (SELECT COUNT(*) FROM ansible_ssh_keys WHERE deleted_at IS NULL) as old_count,
    (SELECT COUNT(*) FROM system_ssh_keys WHERE deleted_at IS NULL) as new_count;
"

# 4. 重新部署应用
kubectl apply -f deploy/k8s/
kubectl scale deployment kube-node-manager --replicas=1
```

### 7.3 回滚方案
```sql
-- 如果需要回滚，执行以下SQL
BEGIN;

-- 1. 恢复 Inventory 引用到旧的 SSH 密钥ID
-- 注意：需要手动维护ID映射关系

-- 2. 删除系统SSH密钥表（可选）
DROP TABLE IF EXISTS system_ssh_keys;

COMMIT;
```

## 八、测试清单

### 8.1 功能测试
- [ ] 系统配置 - SSH 密钥列表显示正常
- [ ] 系统配置 - 创建新SSH密钥
- [ ] 系统配置 - 编辑SSH密钥
- [ ] 系统配置 - 删除SSH密钥
- [ ] 系统配置 - 设置默认SSH密钥
- [ ] Ansible - 创建 Inventory 并关联SSH密钥
- [ ] Ansible - 执行任务使用新的SSH密钥
- [ ] 分析报告菜单已移除
- [ ] 原分析报告路由返回404

### 8.2 数据一致性测试
- [ ] 所有原有SSH密钥已迁移
- [ ] Inventory 的 ssh_key_id 引用正确
- [ ] 加密/解密功能正常
- [ ] 软删除功能正常

### 8.3 性能测试
- [ ] SSH密钥列表加载时间 < 1s
- [ ] 创建SSH密钥响应时间 < 500ms
- [ ] Ansible 任务执行无延迟

## 九、注意事项

### 9.1 安全考虑
1. **加密密钥管理**
   - 必须设置 `SSH_ENCRYPTION_KEY` 环境变量
   - 密钥长度必须为32字节（AES-256）
   - 生产环境不要使用默认密钥

2. **权限控制**
   - SSH 密钥管理需要管理员权限
   - 敏感信息不会在 API 响应中返回

### 9.2 运维建议
1. 定期备份 `system_ssh_keys` 表
2. 监控 SSH 密钥使用情况
3. 定期审查未使用的 SSH 密钥
4. 保留 `ansible_ssh_keys` 表至少一个版本周期

### 9.3 已知限制
1. 不支持从系统SSH密钥迁移回 Ansible SSH密钥
2. 迁移过程中不能执行 Ansible 任务
3. 旧的 API 路由需要手动更新（如果有外部调用）

## 十、FAQ

### Q1: 为什么要将 SSH 密钥移到系统配置？
**A:** SSH 密钥是系统级的基础设施配置，不应该局限在 Ansible 模块中。这样可以：
- 统一管理所有系统级配置
- 支持未来其他模块使用相同的 SSH 密钥
- 提高代码的可维护性和可扩展性

### Q2: 旧的 ansible_ssh_keys 表什么时候可以删除？
**A:** 建议保留至少一个大版本周期（3-6个月），确保所有功能稳定后再考虑删除。

### Q3: 如何处理正在使用旧 API 的第三方集成？
**A:** 可以在 nginx 或 API Gateway 层添加路由重定向：
```nginx
location /api/v1/ansible/ssh-keys {
    rewrite ^/api/v1/ansible/ssh-keys(.*)$ /api/v1/ssh-keys$1 break;
    proxy_pass http://backend;
}
```

### Q4: 迁移失败怎么办？
**A:** 
1. 检查迁移脚本执行日志
2. 验证数据库连接和权限
3. 手动执行 SQL 语句进行调试
4. 如有必要，使用备份恢复数据库

## 十一、相关文档

- [系统架构文档](../README.md)
- [API 文档](./api-documentation.md)
- [数据库 Schema](./database-schema.md)
- [Ansible 模块使用指南](../README.md#ansible-自动化运维使用)

---

**变更日期**: 2025-11-22  
**变更人**: AI Assistant  
**审核状态**: 待审核  
**版本**: v2.35.0

