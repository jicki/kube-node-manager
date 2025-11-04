# kubectl-node_mgr 插件

这是一个 kubectl 插件，用于管理 Kubernetes 节点的归属标签和 cordon 操作。

## 功能特性

1. **节点归属查看** - 展示节点的 `deeproute.cn/user-type` 标签归属
2. **智能 Cordon** - 对节点执行 cordon 操作并添加详细说明 annotations

## 安装

### 方式一：直接安装二进制文件
```bash
# 构建插件
cd kubectl-plugin
go build -o kubectl-node-mgr main.go

# 将插件移动到 PATH 中
sudo mv kubectl-node-mgr /usr/local/bin/

# 验证安装
kubectl node_mgr --help
```

### 方式二：使用 Makefile
```bash
make build-plugin
make install-plugin
```

## 使用方法

### 1. 查看节点调度状态

查看所有节点的调度状态，包括是否被 cordon、禁止调度的原因和时间：
```bash
kubectl node_mgr get
```

查看特定节点的调度状态：
```bash
kubectl node_mgr get node1
```

输出格式化选项：
```bash
# 表格格式（默认）
kubectl node_mgr get -o table

# JSON 格式
kubectl node_mgr get -o json

# YAML 格式
kubectl node_mgr get -o yaml
```

### 2. 查看节点归属标签

查看所有节点的 `deeproute.cn/user-type` 标签：
```bash
kubectl node_mgr labels
```

查看特定节点的归属：
```bash
kubectl node_mgr labels node1
```

输出格式化选项：
```bash
# 表格格式（默认）
kubectl node_mgr labels -o table

# JSON 格式
kubectl node_mgr labels -o json

# YAML 格式
kubectl node_mgr labels -o yaml
```

### 3. Cordon 节点管理

对节点执行 cordon 并添加说明：
```bash
kubectl node_mgr cordon node1 --reason "维护升级"
```

批量 cordon 多个节点：
```bash
kubectl node_mgr cordon node1,node2,node3 --reason "集群维护"
```

查看已 cordon 的节点及其说明：
```bash
kubectl node_mgr cordon list
```

取消 cordon（uncordon）：
```bash
kubectl node_mgr uncordon node1
```

## 命令参考

### get 子命令
```bash
kubectl node_mgr get [NODE_NAME] [flags]

Flags:
  -o, --output string   输出格式 (table|json|yaml) (default "table")
  -l, --selector string 标签选择器
```

### labels 子命令
```bash
kubectl node_mgr labels [NODE_NAME] [flags]

Flags:
  -o, --output string   输出格式 (table|json|yaml) (default "table")
  -l, --selector string 标签选择器
      --show-all        显示所有标签，不仅仅是 deeproute.cn/user-type
```

### cordon 子命令
```bash
kubectl node_mgr cordon NODE_NAME[,NODE_NAME...] [flags]

Flags:
      --reason string     Cordon 原因 (required)
```

### uncordon 子命令
```bash
kubectl node_mgr uncordon NODE_NAME[,NODE_NAME...] [flags]
```

### list 子命令
```bash
kubectl node_mgr cordon list [flags]

Flags:
  -o, --output string   输出格式 (table|json|yaml) (default "table")
```

## 配置

插件使用标准的 kubectl 配置文件（`~/.kube/config`）来连接 Kubernetes 集群。

### 支持的全局参数

- `--kubeconfig`: 指定 kubeconfig 文件路径（默认为 `$HOME/.kube/config`）
- `--context`: 指定要使用的 kubeconfig 上下文名称
- `--namespace` / `-n`: 指定命名空间范围

### 使用不同的上下文

```bash
# 使用特定的上下文
kubectl node_mgr get --context=production

# 使用特定的 kubeconfig 文件
kubectl node_mgr labels --kubeconfig=/path/to/config

# 组合使用
kubectl node_mgr cordon node1 --reason "维护" --context=staging --kubeconfig=/path/to/config
```

### 多集群配置支持

插件完全支持 kubectl 的多集群配置方式：

```bash
# 使用 KUBECONFIG 环境变量合并多个配置文件
export KUBECONFIG=~/.kube/config:~/.kube/staging:~/.kube/production
kubectl node_mgr get

# 查看合并后的所有上下文
kubectl config get-contexts

# 使用特定上下文
kubectl node_mgr get --context=staging-cluster
```

### 故障排除

如果遇到 "file name too long" 或其他配置相关的错误，可以启用调试模式：

```bash
# 启用调试模式查看详细的配置加载信息
KUBECTL_NODE_MGR_DEBUG=1 kubectl node_mgr get

# 调试特定的 kubeconfig 文件
KUBECTL_NODE_MGR_DEBUG=1 kubectl node_mgr get --kubeconfig=/path/to/your/config
```

调试模式会显示：
- 使用的 kubeconfig 文件路径
- 文件路径长度
- KUBECONFIG 环境变量的处理过程
- 多个配置文件的合并信息

## 示例

### 查看节点调度状态示例
```bash
$ kubectl node_mgr get
NAME     STATUS   ROLES    AGE    SCHEDULABLE          CORDON-REASON   CORDON-TIME
node1    Ready    master   10d    Schedulable          -               -
node2    Ready    worker   10d    SchedulingDisabled   系统维护         2024-01-15 10:30:00
node3    Ready    worker   10d    Schedulable          -               -
```

### 查看节点归属示例
```bash
$ kubectl node_mgr labels
NAME     STATUS   ROLES    AGE   USER-TYPE
node1    Ready    master   10d   admin
node2    Ready    worker   10d   developer
node3    Ready    worker   10d   tester
```

### Cordon 操作示例
```bash
$ kubectl node_mgr cordon node2 --reason "系统升级"
Node node2 cordoned successfully with reason: 系统升级

$ kubectl node_mgr cordon list
NAME     STATUS                     REASON      CORDON-TIME
node2    SchedulingDisabled        系统升级     2024-01-15 10:30:00
```

## 注意事项

1. 确保有足够的权限操作 Kubernetes 节点
2. Cordon 操作会阻止新的 Pod 调度到该节点
3. 插件会自动添加以下 annotations：
   - `deeproute.cn/kube-node-mgr`: Cordon 原因
   - `deeproute.cn/kube-node-mgr-timestamp`: 操作时间
4. **自动清理机制**：
   - 如果使用 `kubectl node_mgr cordon` 进行 cordon，建议也使用 `kubectl node_mgr uncordon` 进行 uncordon
   - 如果混用原生 `kubectl uncordon`，系统会自动检测并清理遗留的 annotations（需要 backend 服务运行）
   - 详见：[自动清理 Annotations 功能文档](../docs/auto-cleanup-annotations.md)

## 故障排除

### 权限问题
如果遇到权限错误，确保当前用户有以下权限：
- 读取节点信息
- 修改节点状态和 annotations

### 连接问题
检查 kubeconfig 文件是否正确配置：
```bash
kubectl config view
kubectl cluster-info
```
