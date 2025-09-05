# kubectl-node-mgr 插件实现总结

## 项目概述

本文档总结了为 kube-node-manager 项目添加的 kubectl 插件功能。该插件是一个独立的工具，不影响原有的 Web 服务，专门用于命令行管理 Kubernetes 节点。

## 功能特性

### 1. 节点调度状态查看功能
- **主要功能**: 展示节点调度状态、禁止调度原因和时间
- **支持格式**: 表格、JSON、YAML 输出
- **高级功能**: 
  - 标签选择器过滤
  - 单节点或批量查看
  - 显示 cordon 详细信息

### 2. 节点标签查看功能
- **主要功能**: 展示节点的 `deeproute.cn/user-type` 标签归属
- **支持格式**: 表格、JSON、YAML 输出
- **高级功能**: 
  - 标签选择器过滤
  - 显示所有标签选项
  - 单节点或批量查看

### 3. 智能 Cordon 管理
- **主要功能**: 对节点执行 cordon 操作并添加详细说明 annotations
- **支持的 annotations**:
  - `deeproute.cn/kube-node-mgr`: Cordon 原因
  - `deeproute.cn/kube-node-mgr-timestamp`: 操作时间

### 4. Uncordon 功能
- **主要功能**: 取消节点的 cordon 状态
- **自动清理**: 自动清理相关的说明 annotations

## 项目结构

```
kubectl-plugin/
├── README.md              # 插件使用文档
├── INSTALL.md             # 安装指南
├── EXAMPLES.md            # 使用示例
├── Makefile              # 构建脚本
├── go.mod                # Go 模块文件
├── go.sum                # Go 依赖锁定
├── main.go               # 主入口文件
├── test.sh               # 测试脚本
├── cmd/                  # 命令实现
│   ├── root.go          # 根命令
│   ├── labels.go        # 标签查看命令
│   ├── cordon.go        # Cordon 命令
│   └── uncordon.go      # Uncordon 命令
└── pkg/                  # 工具包
    └── k8s/
        └── client.go     # Kubernetes 客户端
```

## 技术实现

### 依赖项
- **Go 版本**: 1.21+
- **主要依赖**:
  - `github.com/spf13/cobra`: CLI 框架
  - `k8s.io/client-go`: Kubernetes 客户端
  - `k8s.io/api`: Kubernetes API 类型
  - `sigs.k8s.io/yaml`: YAML 处理

### 核心组件

#### 1. 命令行接口 (CLI)
使用 Cobra 框架构建，提供：
- 清晰的命令层次结构
- 丰富的帮助信息
- 参数验证
- 自动补全支持

#### 2. Kubernetes 客户端
- 支持集群内和集群外配置
- 自动发现 kubeconfig 文件
- 标准的 kubectl 配置兼容性

#### 3. 输出格式化
- 表格格式：适合人类阅读
- JSON 格式：适合程序处理
- YAML 格式：适合配置文件

## 命令参考

### labels 命令
```bash
kubectl node-mgr labels [NODE_NAME] [flags]

标志:
  -o, --output string     输出格式 (table|json|yaml)
  -l, --selector string   标签选择器
      --show-all          显示所有标签
```

### cordon 命令
```bash
kubectl node-mgr cordon NODE_NAME[,NODE_NAME...] [flags]

标志:
      --reason string           Cordon 原因 (必需)
      --operator string         操作人员 (必需)
      --scheduled-time string   预计维护时间
      --contact string          联系方式
```

### uncordon 命令
```bash
kubectl node-mgr uncordon NODE_NAME[,NODE_NAME...]
```

### cordon list 命令
```bash
kubectl node-mgr cordon list [flags]

标志:
  -o, --output string   输出格式 (table|json|yaml)
```

## 安装方式

### 使用项目 Makefile
```bash
# 构建插件
make build-plugin

# 安装到系统路径
make install-plugin

# 安装到用户目录
make install-plugin-user
```

### 手动安装
```bash
cd kubectl-plugin
go build -o kubectl-node-mgr main.go
sudo mv kubectl-node-mgr /usr/local/bin/
```

## 配置和上下文支持

插件支持标准的 kubectl 配置选项：

- `--kubeconfig`: 指定 kubeconfig 文件路径
- `--context`: 指定要使用的 kubeconfig 上下文名称
- `--namespace`: 指定命名空间范围

这使得插件可以轻松地在多个 Kubernetes 集群和环境之间切换。

## 使用示例

### 查看节点调度状态
```bash
# 查看所有节点调度状态
kubectl node-mgr get

# 查看特定节点调度状态
kubectl node-mgr get node1

# 使用特定上下文
kubectl node-mgr get --context=production

# JSON 格式输出
kubectl node-mgr get -o json
```

### 查看节点标签归属
```bash
# 查看所有节点
kubectl node-mgr labels

# 查看特定节点
kubectl node-mgr labels node1

# JSON 格式输出
kubectl node-mgr labels -o json
```

### 节点维护操作
```bash
# Cordon 节点
kubectl node-mgr cordon node1 --reason "系统维护"

# 查看已 cordon 的节点
kubectl node-mgr cordon list

# 维护完成后 uncordon
kubectl node-mgr uncordon node1
```

## 测试验证

插件包含完整的测试套件：

### 自动化测试
- 二进制文件完整性检查
- 命令行参数验证
- 帮助信息测试
- 输出格式验证
- 代码质量检查

### 运行测试
```bash
cd kubectl-plugin
./test.sh
```

## 权限要求

插件需要以下 Kubernetes 权限：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nodemanager-plugin
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "patch", "update"]
```

## 设计原则

### 1. 不影响原服务
- 插件完全独立，不依赖现有的 Web 服务
- 可以单独安装和使用
- 不影响现有的数据库和配置

### 2. 标准兼容性
- 遵循 kubectl 插件标准
- 使用标准的 kubeconfig 配置
- 兼容 kubectl 的使用习惯

### 3. 用户友好
- 清晰的命令结构
- 丰富的帮助信息
- 多种输出格式
- 详细的错误信息

### 4. 可扩展性
- 模块化的代码结构
- 易于添加新功能
- 清晰的接口定义

## 维护和升级

### 版本管理
- 语义化版本控制
- Git 标签管理
- 构建时版本信息嵌入

### 更新流程
```bash
# 拉取最新代码
git pull

# 重新构建和安装
make build-plugin
make install-plugin
```

## 故障排除

### 常见问题
1. **插件未找到**: 检查二进制文件名和 PATH 配置
2. **权限错误**: 配置正确的 RBAC 权限
3. **连接错误**: 检查 kubeconfig 配置
4. **构建错误**: 运行 `go mod tidy` 更新依赖

### 调试方法
- 使用 `--help` 查看命令帮助
- 检查 kubeconfig 文件
- 验证集群连接：`kubectl cluster-info`

## 总结

kubectl-node-mgr 插件成功实现了以下目标：

1. ✅ **功能完整**: 实现了节点标签查看和 cordon 管理功能
2. ✅ **独立部署**: 不影响原有的 Web 服务
3. ✅ **标准兼容**: 遵循 kubectl 插件标准
4. ✅ **用户友好**: 提供清晰的命令接口和帮助信息
5. ✅ **测试完善**: 包含完整的测试套件
6. ✅ **文档齐全**: 提供详细的安装和使用文档

该插件为 Kubernetes 节点管理提供了强大的命令行工具，特别适合运维人员在维护场景中使用。通过详细的 annotations 记录，可以有效跟踪节点的维护历史和状态变化。
