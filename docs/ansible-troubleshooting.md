# Ansible SSH 认证问题排查指南

## 问题现象

直接使用 SSH Key 可以正常登录主机，但通过 Ansible 任务中心执行任务时出现认证失败：

```
fatal: [host]: UNREACHABLE! => {"msg": "Permission denied (publickey,password).", "unreachable": true}
```

从日志中看到 Ansible 尝试使用默认的 SSH 密钥路径：
```
debug1: Will attempt key: /root/.ssh/id_rsa 
debug1: Will attempt key: /root/.ssh/id_ecdsa 
...
debug1: No more authentication methods to try.
```

## 根因分析

问题的根本原因是 Ansible 任务执行时没有使用数据库中存储的 SSH 密钥。可能的原因包括：

1. ✅ **使用了旧版本的代码**（v2.18.7 之前的版本）
2. ✅ **使用了旧的 Inventory**（创建时没有关联 SSH 密钥）
3. ✅ **SSH 密钥没有正确保存到数据库**

## 排查步骤

### 步骤 1：确认应用版本

检查当前运行的应用版本：

```bash
# 方法 1: 查看 Pod 的镜像版本
kubectl get deployment kube-node-manager -n your-namespace -o jsonpath='{.spec.template.spec.containers[0].image}'

# 方法 2: 查看应用内的版本文件
kubectl exec -it deployment/kube-node-manager -n your-namespace -- cat /app/VERSION
```

**期望输出**：`v2.18.7` 或更高版本

**如果版本低于 v2.18.7**：
```bash
# 重新部署应用
kubectl rollout restart deployment kube-node-manager -n your-namespace

# 等待部署完成
kubectl rollout status deployment kube-node-manager -n your-namespace
```

### 步骤 2：检查数据库中的 Inventory

连接到数据库，检查 `ansible_inventories` 表：

```sql
-- 查看所有 Inventory 及其关联的 SSH 密钥
SELECT 
    id,
    name,
    description,
    source_type,
    ssh_key_id,
    created_at,
    updated_at
FROM ansible_inventories
ORDER BY created_at DESC;
```

**期望结果**：`ssh_key_id` 列应该有值（不为 NULL）

**如果 `ssh_key_id` 为 NULL**：
- 这是旧的 Inventory（在修复之前创建的）
- 需要重新创建或更新 Inventory

### 步骤 3：检查 SSH 密钥是否存在

```sql
-- 查看所有 SSH 密钥
SELECT 
    id,
    name,
    description,
    type,
    username,
    port,
    is_default,
    created_at
FROM ansible_ssh_keys
ORDER BY created_at DESC;
```

**期望结果**：应该看到至少一个 SSH 密钥记录

**如果没有 SSH 密钥**：
- 需要在前端创建 SSH 密钥
- 路径：Ansible 任务中心 → SSH 密钥管理 → 添加密钥

### 步骤 4：检查后端日志

查看应用日志，寻找关键信息：

```bash
# 实时查看日志
kubectl logs -f deployment/kube-node-manager -n your-namespace | grep -E "Task.*SSH|Inventory.*ssh_key"

# 查看最近的日志
kubectl logs --tail=200 deployment/kube-node-manager -n your-namespace | grep -i "ssh\|ansible"
```

**关键日志信息**：

#### ✅ 正常情况（有 SSH 密钥）
```
INFO: Task 123: Using SSH key ID 1 for inventory 5 (test-inventory)
INFO: Task 123: Retrieved SSH key 1 (devops key) - Type: private_key, Username: devops
INFO: Task 123: Created SSH key file: /tmp/ansible/ssh-key-123-1234567890.pem (size: 1675 bytes)
INFO: Task 123: Ansible will use SSH key file: /tmp/ansible/ssh-key-123-1234567890.pem
INFO: Task 123: Executing command: ansible-playbook -i /tmp/inventory-123.ini /tmp/playbook-123.yml -v --private-key /tmp/ansible/ssh-key-123-1234567890.pem
```

#### ⚠️ 异常情况（没有 SSH 密钥）
```
WARN: Task 123: Inventory 5 (test-inventory) has no SSH key associated, Ansible will use default authentication
WARN: Task 123: No SSH key file provided, Ansible will use default authentication
INFO: Task 123: Executing command: ansible-playbook -i /tmp/inventory-123.ini /tmp/playbook-123.yml -v
```

注意：如果看到 **WARN** 日志，说明 Inventory 没有关联 SSH 密钥！

### 步骤 5：更新现有 Inventory（如果需要）

如果您想保留现有的 Inventory 并添加 SSH 密钥关联：

#### 方式 A：通过前端界面更新

1. 进入 **Ansible 任务中心** → **主机清单管理**
2. 找到目标 Inventory，点击 **"编辑"**
3. 在 **"SSH 密钥"** 下拉框中选择对应的密钥
4. 点击 **"保存"**

#### 方式 B：直接更新数据库（仅限紧急情况）

```sql
-- 1. 查找 SSH 密钥的 ID
SELECT id, name FROM ansible_ssh_keys;

-- 假设得到 ssh_key_id = 1

-- 2. 更新 Inventory
UPDATE ansible_inventories 
SET ssh_key_id = 1, updated_at = NOW() 
WHERE id = 你的_inventory_id;

-- 3. 验证更新
SELECT id, name, ssh_key_id FROM ansible_inventories WHERE id = 你的_inventory_id;
```

### 步骤 6：重新创建 Inventory（推荐）

为确保万无一失，建议重新创建 Inventory：

#### 方式 A：从集群生成

1. 进入 **Ansible 任务中心** → **主机清单管理**
2. 点击 **"从集群生成"**
3. 填写信息：
   - **清单名称**：`new-test-inventory`
   - **描述**：`测试清单（带SSH密钥）`
   - **选择集群**：选择你的 K8s 集群
   - **选择 SSH 密钥**：选择 `devops key` ⭐ **关键步骤**
4. 点击 **"生成"**

#### 方式 B：手动创建

1. 点击 **"手动创建"**
2. 填写清单内容：

```ini
[all]
10-9-9-84.vm.pd.sz.deeproute.ai ansible_host=10.9.9.84 ansible_user=devops ansible_ssh_port=6688

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

3. **选择 SSH 密钥**：选择 `devops key` ⭐ **关键步骤**
4. 点击 **"保存"**

### 步骤 7：执行测试任务

1. 进入 **Ansible 任务中心**
2. 点击 **"创建任务"**
3. 选择：
   - **任务名称**：`SSH Test`
   - **模板**：`Ping`
   - **主机清单**：`new-test-inventory`（刚创建的）
4. 点击 **"启动任务"**
5. 点击 **"查看日志"** 按钮

**期望结果**：

```
PLAY [Ping Test] *****************************************************

TASK [Gathering Facts] ***********************************************
ok: [10-9-9-84.vm.pd.sz.deeproute.ai]

TASK [Ping 测试节点] *************************************************
ok: [10-9-9-84.vm.pd.sz.deeproute.ai]

PLAY RECAP ***********************************************************
10-9-9-84.vm.pd.sz.deeproute.ai : ok=2    changed=0    unreachable=0    failed=0
```

## 常见问题 FAQ

### Q1: 为什么我的 SSH 密钥在前端创建后，Inventory 还是没有关联？

**A**: 这是因为 Inventory 是在 SSH 密钥创建之前就已经存在的。需要：
1. 编辑现有 Inventory，选择 SSH 密钥
2. 或者删除旧 Inventory，重新创建

### Q2: 我如何知道我的 Inventory 是否关联了 SSH 密钥？

**A**: 有两种方式：
1. **前端界面**：在主机清单列表中查看（未来版本会显示关联的 SSH 密钥名称）
2. **数据库查询**：执行上面的 SQL 查询，检查 `ssh_key_id` 列

### Q3: 为什么日志中显示 "No SSH key file provided"？

**A**: 可能的原因：
1. Inventory 没有关联 SSH 密钥（`ssh_key_id` 为 NULL）
2. 关联的 SSH 密钥类型是 `password`（密码认证，不需要密钥文件）
3. SSH 密钥在数据库中不存在或已被删除

### Q4: 我直接 SSH 可以登录，为什么 Ansible 不行？

**A**: 常见原因：
1. **用户名不匹配**：直接 SSH 使用的用户名和 Inventory 中配置的 `ansible_user` 不一致
2. **端口不匹配**：直接 SSH 使用的端口和 Inventory 中的 `ansible_ssh_port` 不一致
3. **SSH 密钥不匹配**：直接 SSH 使用的密钥和数据库中存储的密钥不一致

**解决方法**：
```ini
# 确保 Inventory 中的配置与直接 SSH 命令一致
[all]
10-9-9-84.vm.pd.sz.deeproute.ai ansible_host=10.9.9.84 ansible_user=devops ansible_ssh_port=6688
#                                                        ^^^^^^^^^^^^         ^^^^^^^^^^^^^^^^^^
#                                                        用户名必须匹配        端口必须匹配
```

### Q5: 如何验证数据库中的 SSH 密钥内容是否正确？

**A**: 
```sql
-- 查看 SSH 密钥的部分内容（已加密）
SELECT 
    id,
    name,
    type,
    username,
    SUBSTRING(private_key, 1, 50) as key_preview,
    LENGTH(private_key) as key_length
FROM ansible_ssh_keys;
```

注意：`private_key` 字段存储的是加密后的内容，无法直接查看原始密钥。

### Q6: 我更新了 Inventory 的 SSH 密钥，但任务还是失败？

**A**: 可能需要：
1. 确认应用版本是 v2.18.7 或更高
2. 重启应用以确保使用最新代码
3. 检查后端日志，确认 SSH 密钥文件已创建
4. 验证 SSH 密钥内容与目标主机的 `authorized_keys` 匹配

## 快速诊断检查清单

使用以下检查清单快速定位问题：

- [ ] **应用版本** ≥ v2.18.7
- [ ] **SSH 密钥已创建** 在前端 SSH 密钥管理中
- [ ] **SSH 密钥内容正确** 与目标主机的 authorized_keys 匹配
- [ ] **Inventory 已关联 SSH 密钥** `ssh_key_id` 不为 NULL
- [ ] **Inventory 配置正确**
  - [ ] `ansible_host` 正确
  - [ ] `ansible_user` 正确
  - [ ] `ansible_ssh_port` 正确（如果不是 22）
- [ ] **后端日志正常** 看到 "Using SSH key ID" 和 "Created SSH key file"
- [ ] **Ansible 命令包含** `--private-key` 参数

## 联系支持

如果按照上述步骤操作后问题仍未解决，请收集以下信息：

1. **应用版本**
   ```bash
   kubectl exec -it deployment/kube-node-manager -n your-namespace -- cat /app/VERSION
   ```

2. **数据库查询结果**
   ```sql
   SELECT id, name, ssh_key_id FROM ansible_inventories;
   SELECT id, name, type, username FROM ansible_ssh_keys;
   ```

3. **任务执行日志**（最后 200 行，包含错误信息）
   ```bash
   kubectl logs --tail=200 deployment/kube-node-manager -n your-namespace
   ```

4. **Ansible 错误日志**（从前端查看日志按钮获取）

将这些信息提供给技术支持团队以获得进一步帮助。

