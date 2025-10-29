# 批量操作缓存刷新修复

## 📋 问题描述

### 症状
用户在执行批量操作后，发现节点列表未立即更新：

**节点调度操作**：
- 批量禁止调度（Cordon）
- 批量解除调度（Uncordon）
- 批量驱逐（Drain）

**标签管理操作**：
- 批量添加/删除/替换标签
- 应用标签模板到节点

**污点管理操作**：
- 批量添加/删除/替换污点
- 批量复制污点到其他节点
- 应用污点模板到节点

### 共同表现
1. **节点列表未立即更新** - 操作完成后，前端显示的节点状态仍然是旧状态
2. **需要等待或手动刷新** - 用户需要等待约30秒（缓存过期时间）或手动刷新页面才能看到正确状态
3. **用户体验差** - 批量操作完成的提示与实际显示状态不一致，造成困惑

### 根本原因

#### 后端缓存机制
Kube Node Manager 实现了多层缓存以提升性能和减少对 Kubernetes API Server 的压力：

```
前端请求 → 后端 API → K8s Service → 缓存层 → Kubernetes API
                            ↓
                      检查缓存是否有效
                            ↓
                    有效：返回缓存数据
                    无效：请求 K8s API
```

缓存策略：
- **列表缓存 TTL**: 30秒
- **详情缓存 TTL**: 5分钟
- **异步刷新**: 缓存过期后异步更新

#### 问题所在
在批量操作中：
1. **单个节点操作会清除节点缓存**，但批量操作执行多个单节点操作时，缓存被多次清除和重建
2. **批量操作完成后没有统一清除集群缓存**，导致前端刷新时可能获取到过期的缓存数据
3. **带进度的批量操作**（异步执行）完成后也没有清除缓存

## 🔧 修复方案

### 核心策略
在所有批量操作完成后，**统一清除整个集群的缓存**，确保前端刷新时获取到最新数据。

### 修复实现

#### 1. BatchCordon - 批量禁止调度（同步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchCordon(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]string)
	successful := make([]string, 0)

	// 🔥 新增：批量操作完成后清除缓存
	defer func() {
		if len(successful) > 0 {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch cordon operation", req.ClusterName)
		}
	}()

	// ... 批量操作逻辑
	for _, nodeName := range req.Nodes {
		if err := s.Cordon(cordonReq, userID); err != nil {
			errors[nodeName] = err.Error()
		} else {
			successful = append(successful, nodeName)
		}
	}

	return results, nil
}
```

**关键点**：
- 使用 `defer` 确保操作完成后执行缓存清除
- 只有当至少有一个节点操作成功时才清除缓存
- 清除整个集群的缓存，而不是单个节点缓存

#### 2. BatchUncordon - 批量解除调度（同步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchUncordon(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]string)
	successful := make([]string, 0)

	// 🔥 新增：批量操作完成后清除缓存
	defer func() {
		if len(successful) > 0 {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch uncordon operation", req.ClusterName)
		}
	}()

	// ... 批量操作逻辑
	return results, nil
}
```

#### 3. BatchCordonWithProgress - 带进度的批量禁止调度（异步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchCordonWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	if s.progressSvc == nil {
		return fmt.Errorf("progress service not set")
	}

	// ... 设置处理器和并发数

	ctx := context.Background()
	err := s.progressSvc.ProcessBatchWithProgress(
		ctx,
		taskID,
		"batch_cordon",
		req.Nodes,
		userID,
		concurrency,
		processor,
	)

	// 🔥 新增：批量操作完成后清除缓存
	if err == nil {
		s.k8sSvc.InvalidateClusterCache(req.ClusterName)
		s.logger.Infof("Invalidated cache for cluster %s after batch cordon with progress", req.ClusterName)
	}

	return err
}
```

**关键点**：
- 等待异步批量操作完成后再清除缓存
- 只有在操作成功（err == nil）时才清除缓存

#### 4. BatchUncordonWithProgress - 带进度的批量解除调度（异步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchUncordonWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	// ... 类似 BatchCordonWithProgress 的实现
	
	err := s.progressSvc.ProcessBatchWithProgress(...)

	// 🔥 新增：批量操作完成后清除缓存
	if err == nil {
		s.k8sSvc.InvalidateClusterCache(req.ClusterName)
		s.logger.Infof("Invalidated cache for cluster %s after batch uncordon with progress", req.ClusterName)
	}

	return err
}
```

#### 5. BatchDrain - 批量驱逐（同步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchDrain(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	// 🔥 新增：批量操作完成后清除缓存
	defer func() {
		if len(successful) > 0 {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch drain operation", req.ClusterName)
		}
	}()

	// ... 批量操作逻辑
	return results, nil
}
```

#### 6. BatchDrainWithProgress - 带进度的批量驱逐（异步）

**修改文件**: `backend/internal/service/node/node.go`

```go
func (s *Service) BatchDrainWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	// ... 异步批量操作

	err := s.progressSvc.ProcessBatchWithProgress(...)

	// 🔥 新增：批量操作完成后清除缓存
	if err == nil {
		s.k8sSvc.InvalidateClusterCache(req.ClusterName)
		s.logger.Infof("Invalidated cache for cluster %s after batch drain with progress", req.ClusterName)
	}

	return err
}
```

#### 7. BatchUpdateLabels - 批量更新标签（同步/异步）

**修改文件**: `backend/internal/service/label/label.go`

```go
func (s *Service) BatchUpdateLabelsWithProgress(req BatchUpdateRequest, userID uint, taskID string) error {
	// 标记是否有成功的操作
	hasSuccess := false
	defer func() {
		if hasSuccess {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch label update", req.ClusterName)
		}
	}()

	// ... 批量操作逻辑
	// 根据是否使用进度推送决定同步或异步处理
	
	if taskID != "" && s.progressSvc != nil {
		// 异步处理
		err := s.progressSvc.ProcessBatchWithProgress(...)
		if err == nil {
			hasSuccess = true
		}
	} else {
		// 同步处理
		successCount := 0
		// ... 处理每个节点
		if successCount > 0 {
			hasSuccess = true
		}
	}

	return nil
}
```

**关键点**：
- 使用 `hasSuccess` 标记跟踪是否有成功操作
- 使用 `defer` 确保操作完成后清除缓存
- 同步和异步模式都支持缓存清除

#### 8. BatchUpdateTaints - 批量更新污点（同步/异步）

**修改文件**: `backend/internal/service/taint/taint.go`

```go
func (s *Service) BatchUpdateTaintsWithProgress(req BatchUpdateRequest, userID uint, taskID string) error {
	// 标记是否有成功的操作
	hasSuccess := false
	defer func() {
		if hasSuccess {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch taint update", req.ClusterName)
		}
	}()

	// ... 批量操作逻辑（类似标签更新）
	
	return nil
}
```

#### 9. BatchCopyTaints - 批量复制污点（同步/异步）

**修改文件**: `backend/internal/service/taint/taint.go`

```go
func (s *Service) BatchCopyTaintsWithProgress(req BatchCopyTaintsRequest, userID uint, taskID string) error {
	// 标记是否有成功的操作
	hasSuccess := false
	defer func() {
		if hasSuccess {
			s.k8sSvc.InvalidateClusterCache(req.ClusterName)
			s.logger.Infof("Invalidated cache for cluster %s after batch taint copy", req.ClusterName)
		}
	}()

	// 验证源节点并获取污点
	sourceNode, err := s.k8sSvc.GetNode(req.ClusterName, req.SourceNodeName)
	if err != nil {
		return err
	}

	// ... 批量复制逻辑
	
	return nil
}
```

#### 10. InvalidateClusterCache - 新增缓存清除方法

**修改文件**: `backend/internal/service/k8s/k8s.go`

```go
// InvalidateClusterCache 清除指定集群的所有缓存
func (s *Service) InvalidateClusterCache(clusterName string) {
	s.cache.InvalidateCluster(clusterName)
}
```

**说明**：
- 提供统一的集群缓存清除接口
- 封装了对底层缓存层的访问
- 便于其他服务（node、label、taint）调用

## ✅ 修复效果

### 用户体验改善
1. ✅ **立即看到结果** - 批量操作完成后，刷新页面立即显示最新状态
2. ✅ **无需等待** - 不再需要等待30秒缓存过期
3. ✅ **无需手动刷新** - 前端自动刷新获取到正确数据
4. ✅ **状态一致** - 操作完成提示与实际显示状态保持一致

### 技术改进
1. ✅ **所有批量操作统一处理** - 同步和异步批量操作都会清除缓存
2. ✅ **避免部分更新** - 清除整个集群缓存，确保完整更新
3. ✅ **日志记录** - 每次缓存清除都有日志记录，便于排查问题
4. ✅ **错误处理** - 只有成功的操作才会清除缓存

### 性能影响
- **缓存清除开销很小** - 只是删除内存中的 Map 条目
- **不影响并发性能** - 使用 defer 确保操作完成后清除
- **不增加 API 调用** - 前端刷新时重新获取数据是正常流程

## 🔍 测试验证

### 测试场景

#### 场景1: 少量节点批量操作（≤5个节点）
1. 选择 3 个节点
2. 执行批量禁止调度
3. 验证：操作完成后立即刷新，节点状态正确显示为 "Ready,SchedulingDisabled"

#### 场景2: 大量节点批量操作（>5个节点）
1. 选择 10 个节点
2. 执行批量解除调度（带进度条）
3. 验证：进度条完成后刷新，节点状态正确显示为 "Ready"

#### 场景3: 部分失败的批量操作
1. 选择 5 个节点（其中 2 个不存在）
2. 执行批量禁止调度
3. 验证：成功的 3 个节点状态正确更新

### 验证方法

#### 后端日志验证
```bash
# 查看缓存清除日志
grep "Invalidated cache for cluster" /var/log/kube-node-manager/app.log

# 示例输出
2025-10-29 10:30:15 INFO Invalidated cache for cluster prod-cluster after batch cordon operation
2025-10-29 10:30:45 INFO Invalidated cache for cluster prod-cluster after batch uncordon with progress
```

#### 前端测试步骤
1. 打开浏览器开发者工具（Network 标签）
2. 执行批量操作
3. 观察操作完成后的网络请求
4. 验证节点列表 API 返回的数据是最新的

## 📊 影响范围

### 修改的文件
- ✅ `backend/internal/service/node/node.go` - 6 个节点批量操作方法
- ✅ `backend/internal/service/label/label.go` - 2 个标签批量操作方法
- ✅ `backend/internal/service/taint/taint.go` - 3 个污点批量操作方法
- ✅ `backend/internal/service/k8s/k8s.go` - 新增 1 个缓存清除方法
- ✅ `docs/CHANGELOG.md` - 更新变更日志
- ✅ `docs/batch-operations-cache-fix.md` - 更新修复文档
- ✅ `VERSION` - 更新版本号到 v2.16.2

### 不受影响的部分
- ✅ 单节点操作（Cordon/Uncordon/Drain）- 已有缓存清除逻辑
- ✅ 单节点标签操作 - 已有缓存清除逻辑
- ✅ 单节点污点操作 - 已有缓存清除逻辑
- ✅ 节点列表查询 - 正常获取缓存或实时数据
- ✅ 前端代码 - 无需修改，继续使用现有刷新逻辑
- ✅ 缓存层实现 - 复用现有的 InvalidateCluster 方法

## 🚀 部署建议

### 升级步骤
1. 备份当前版本
2. 拉取最新代码（v2.16.2）
3. 编译后端服务
4. 重启后端服务
5. 验证批量操作功能

### 回滚方案
如果发现问题，可以快速回滚：
```bash
# 回滚到 v2.16.1
git checkout v2.16.1
make build
kubectl rollout undo deployment/kube-node-manager
```

### 监控要点
- 观察后端日志中的缓存清除记录
- 监控批量操作的执行时间（应无明显增加）
- 检查用户反馈，确认问题已解决

## 📚 相关文档

- [缓存实现文档](./cache-invalidation-audit.md)
- [批量操作优化](./batch-operations-optimization.md)
- [变更日志](./CHANGELOG.md)

## 🔗 相关 Issue

- Issue #N/A - 批量操作缓存刷新问题

---

**版本**: v2.16.2  
**修复日期**: 2025-10-29  
**修复人员**: AI Assistant  
**审核状态**: ✅ 已完成

