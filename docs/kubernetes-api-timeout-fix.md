# Kubernetes API 超时问题修复方案

## 问题概述

### 现象描述

在处理大规模 Kubernetes 集群时，频繁出现 `context deadline exceeded` 错误：

```
E1103 15:16:07.156613 request.go:1196] "Unexpected error when reading response body" err="context deadline exceeded"
WARNING: 2025/11/03 15:16:07 logger.go:59: Failed to list pods for cluster jobsscz-k8s-cluster: unexpected error when reading response body. Please retry. Original error: context deadline exceeded
```

### 影响范围

**受影响集群：**
- `jobsscz-k8s-cluster`（主要）：104 个节点，83 个 GPU 节点，872 个 GPU
- 其他大规模集群在高负载时可能也会受影响

**受影响操作：**
1. 列出集群所有 Pod
2. 获取节点上的 Pod 数量
3. 节点指标enrichment

**受影响节点示例：**
- 10-16-10-110.maas
- 10-16-10-111.maas
- 10-16-10-116.maas
- 10-16-10-117.maas
- 10-16-10-118.maas
- 10-16-10-119.maas
- 10-16-10-120.maas
- 10-16-10-121.maas

## 根本原因分析

### 1. 超时配置不足

**原配置：**
```go
config.Timeout = 30 * time.Second           // Kubernetes 客户端配置
context.WithTimeout(..., 30*time.Second)    // 列出节点
context.WithTimeout(..., 15*time.Second)    // 批量获取 Pod 数量
context.WithTimeout(..., 10*time.Second)    // 单节点 Pod 数量
```

**问题：**
- 对于拥有数千个 Pod 的大规模集群，15 秒内列出所有 Pod 不够
- K8s API 服务器在高负载时响应变慢
- 网络延迟可能导致超时

### 2. 性能瓶颈

**代码位置：** `backend/internal/service/k8s/k8s.go`

```go
// 批量获取所有 Pods（所有命名空间）
podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
```

当集群有 100+ 节点和数千个 Pod 时：
- API 响应体可能达到数 MB
- 网络传输时间增加
- JSON 反序列化耗时

### 3. 节点状态频繁变化

某些节点（如 10-16-10-114.maas, 10-16-10-115.maas）的 conditions 频繁更新，可能表明：
- 节点健康状态不稳定
- kubelet 与 API server 通信异常
- 网络抖动

## 已实施的解决方案

### 方案 1：调整超时配置（已完成）✅

**修改文件：** `backend/internal/service/k8s/k8s.go`

#### 1.1 增加 Kubernetes 客户端超时

```go
// 设置超时 - 针对大规模集群增加超时时间
config.Timeout = 60 * time.Second  // 从 30s 增加到 60s
```

#### 1.2 增加节点列表操作超时

```go
// 针对大规模集群增加超时时间
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)  // 从 30s 增加到 60s
```

#### 1.3 增加 Pod 批量获取超时

```go
// 针对大规模集群增加超时时间到 30 秒
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)  // 从 15s 增加到 30s
```

#### 1.4 增加单节点 Pod 获取超时

```go
// 针对大规模集群增加超时时间到 20 秒
ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)  // 从 10s 增加到 20s
```

### 预期效果

- ✅ 减少 `context deadline exceeded` 错误发生频率
- ✅ 提高大规模集群的稳定性
- ✅ 允许更长的网络响应时间
- ⚠️ 可能会略微增加请求响应时间

## 进一步优化建议

### 方案 2：实现分页查询（推荐）

**优点：**
- 减少单次 API 调用的数据量
- 降低内存使用
- 提高响应速度

**实现示例：**

```go
func (s *Service) getNodesPodCountsPaginated(clusterName string, nodeNames []string) map[string]int {
    client, err := s.getClient(clusterName)
    if err != nil {
        s.logger.Warningf("Failed to get client for cluster %s: %v", clusterName, err)
        return make(map[string]int)
    }

    podCounts := make(map[string]int)
    for _, node := range nodeNames {
        podCounts[node] = 0
    }

    // 使用分页查询，每次获取 500 个 Pod
    continueToken := ""
    for {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        
        podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
            Limit:    500,
            Continue: continueToken,
        })
        cancel()

        if err != nil {
            s.logger.Warningf("Failed to list pods for cluster %s: %v", clusterName, err)
            return podCounts
        }

        // 统计此批次的 Pod
        for _, pod := range podList.Items {
            if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
                if _, exists := podCounts[pod.Spec.NodeName]; exists {
                    podCounts[pod.Spec.NodeName]++
                }
            }
        }

        // 检查是否还有更多数据
        if podList.Continue == "" {
            break
        }
        continueToken = podList.Continue
    }

    return podCounts
}
```

### 方案 3：增强缓存策略

**优化点：**

1. **缓存 Pod 数量信息**
   ```go
   // 缓存 5 分钟，减少 API 调用
   cacheKey := fmt.Sprintf("pod-counts-%s", clusterName)
   if cached, found := s.cache.Get(cacheKey); found {
       return cached.(map[string]int)
   }
   ```

2. **使用增量更新**
   - 监听 Pod 事件，增量更新计数
   - 避免每次都重新获取所有 Pod

### 方案 4：使用 Informer 机制

**优点：**
- 实时监听资源变化
- 本地缓存，无需频繁查询 API
- 大幅降低 API 服务器负载

**实现参考：**

```go
import (
    "k8s.io/client-go/informers"
    "k8s.io/client-go/tools/cache"
)

// 初始化 Pod Informer
func (s *Service) initPodInformer(clusterName string) {
    client, _ := s.getClient(clusterName)
    
    factory := informers.NewSharedInformerFactory(client, 30*time.Second)
    podInformer := factory.Core().V1().Pods().Informer()
    
    podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            // 处理 Pod 添加
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            // 处理 Pod 更新
        },
        DeleteFunc: func(obj interface{}) {
            // 处理 Pod 删除
        },
    })
    
    go factory.Start(wait.NeverStop)
}
```

### 方案 5：添加重试机制

**实现建议：**

```go
import "github.com/cenkalti/backoff/v4"

func (s *Service) getNodesPodCountsWithRetry(clusterName string, nodeNames []string) map[string]int {
    var result map[string]int
    
    operation := func() error {
        result = s.getNodesPodCounts(clusterName, nodeNames)
        if len(result) == 0 {
            return fmt.Errorf("failed to get pod counts")
        }
        return nil
    }
    
    // 指数退避重试：初始 1s，最大 30s，最多重试 3 次
    exponentialBackOff := backoff.NewExponentialBackOff()
    exponentialBackOff.InitialInterval = 1 * time.Second
    exponentialBackOff.MaxInterval = 30 * time.Second
    exponentialBackOff.MaxElapsedTime = 2 * time.Minute
    
    err := backoff.Retry(operation, backoff.WithMaxRetries(exponentialBackOff, 3))
    if err != nil {
        s.logger.Errorf("Failed to get pod counts after retries: %v", err)
        return make(map[string]int)
    }
    
    return result
}
```

### 方案 6：添加监控告警

**推荐指标：**

1. **API 调用延迟监控**
   ```go
   // 记录 API 调用耗时
   start := time.Now()
   podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
   duration := time.Since(start)
   
   s.logger.Infof("List pods for cluster %s took %v", clusterName, duration)
   
   // 如果超过阈值，记录警告
   if duration > 10*time.Second {
       s.logger.Warningf("Slow API response for cluster %s: %v", clusterName, duration)
   }
   ```

2. **超时错误计数器**
   ```go
   // 统计超时错误次数
   if strings.Contains(err.Error(), "context deadline exceeded") {
       s.timeoutErrorCount++
       if s.timeoutErrorCount > 10 {
           s.logger.Errorf("Cluster %s has too many timeout errors: %d", clusterName, s.timeoutErrorCount)
           // 触发告警
       }
   }
   ```

## 集群健康检查建议

### 检查 jobsscz-k8s-cluster 集群

**1. 检查 API Server 负载**
```bash
# 检查 API Server 日志
kubectl logs -n kube-system kube-apiserver-xxx --tail=100

# 检查 API Server 指标
kubectl top pods -n kube-system | grep apiserver
```

**2. 检查节点状态**
```bash
# 检查频繁更新的节点
kubectl describe node 10-16-10-114.maas
kubectl describe node 10-16-10-115.maas

# 检查节点事件
kubectl get events --field-selector involvedObject.name=10-16-10-114.maas
```

**3. 检查 kubelet 日志**
```bash
# SSH 到节点
ssh 10-16-10-114.maas

# 查看 kubelet 日志
journalctl -u kubelet -f --since "1 hour ago"
```

**4. 检查网络连接**
```bash
# 测试到 API Server 的网络延迟
time kubectl get nodes

# 检查 DNS 解析
nslookup kubernetes.default.svc.cluster.local
```

## 配置建议

### 针对不同规模集群的超时配置

**小型集群（< 10 节点）**
```go
config.Timeout = 30 * time.Second
podListTimeout = 15 * time.Second
```

**中型集群（10-50 节点）**
```go
config.Timeout = 45 * time.Second
podListTimeout = 20 * time.Second
```

**大型集群（50-200 节点）** ✅ 当前配置
```go
config.Timeout = 60 * time.Second
podListTimeout = 30 * time.Second
```

**超大型集群（> 200 节点）**
```go
config.Timeout = 90 * time.Second
podListTimeout = 60 * time.Second
// 强烈建议实施分页和 Informer 方案
```

## 部署步骤

### 1. 编译更新

```bash
cd backend
go build -o bin/kube-node-manager cmd/main.go
```

### 2. 测试验证

```bash
# 启动服务
./bin/kube-node-manager

# 监控日志，观察是否还有超时错误
tail -f logs/app.log | grep "deadline exceeded"
```

### 3. 生产部署

```bash
# 使用 Docker 部署
cd deploy/docker
docker-compose down
docker-compose up -d --build

# 或使用 Kubernetes 部署
kubectl apply -f deploy/k8s/
kubectl rollout status deployment/kube-node-manager
```

### 4. 监控观察

部署后持续监控以下指标：
- API 超时错误频率
- 请求响应时间
- 内存和 CPU 使用率
- Pod 计数准确性

## 回滚计划

如果新配置导致问题：

```bash
# 方案 1：回滚到之前的版本
git revert <commit-hash>
git push

# 方案 2：手动恢复超时配置
# 编辑 backend/internal/service/k8s/k8s.go
# 将超时配置改回原值，重新构建部署
```

## 总结

### 已完成 ✅
- 增加 Kubernetes API 客户端超时配置
- 增加节点列表操作超时
- 增加 Pod 批量获取超时
- 增加单节点 Pod 获取超时

### 推荐后续优化 🔧
1. 实现分页查询（高优先级）
2. 增强缓存策略（中优先级）
3. 实施 Informer 机制（高优先级，长期方案）
4. 添加重试机制（中优先级）
5. 添加监控告警（高优先级）

### 注意事项 ⚠️
- 增加超时时间会略微延长用户等待时间
- 建议对特定集群进行针对性优化
- 定期检查 Kubernetes 集群健康状况
- 考虑升级 Kubernetes 版本以获得性能改进

---

**文档版本：** v1.0  
**创建日期：** 2025-11-03  
**更新日期：** 2025-11-03  
**作者：** AI Assistant  
**状态：** 已实施（方案 1）

