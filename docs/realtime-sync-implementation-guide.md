# Kube-Node-Manager 实时同步实施指南

## 📋 概述

本文档描述了完整的实时同步方案，包括 K8s Informer、智能缓存和 WebSocket 实时推送的集成方案。

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
└────────────────────────┬────────────────────────────────────┘
                         │ Watch API
                         ▼
┌─────────────────────────────────────────────────────────────┐
│               Informer Service (内存 Watch)                  │
│  • SharedInformerFactory                                     │
│  • 自动同步节点变化                                          │
│  • 事件过滤和变化检测                                        │
└────────────────────────┬────────────────────────────────────┘
                         │ Node Events (Add/Update/Delete)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   SmartCache (智能缓存层)                    │
│  静态属性 (1小时 TTL): CPU、内存容量、OS信息                 │
│  动态属性 (实时更新): Labels、Taints、Schedulable、Status    │
└────────────────────────┬────────────────────────────────────┘
                         │
          ┌──────────────┴──────────────┐
          ▼                             ▼
┌─────────────────────┐       ┌─────────────────────┐
│  WebSocket Hub      │       │   K8s Service       │
│  • 连接管理          │       │   (查询层)           │
│  • 房间/订阅管理     │       │   读取 SmartCache   │
│  • 实时推送          │       └─────────────────────┘
└──────────┬──────────┘
           │ WebSocket Push
           ▼
┌─────────────────────┐
│  前端 WebSocket     │
│  • 自动重连          │
│  • 实时更新 UI       │
│  • 无需手动刷新      │
└─────────────────────┘
```

## 📦 已创建的组件

### 1. Informer Service
**文件**: `backend/internal/informer/informer.go`

**功能**:
- 使用 K8s SharedInformerFactory 监听节点变化
- 检测关键字段变化 (Labels、Taints、Schedulable、Status)
- 过滤无关事件 (只推送有意义的变化)
- 支持多集群管理

**关键方法**:
```go
// 启动集群的 Informer
func (s *Service) StartInformer(clusterName string, clientset *kubernetes.Clientset) error

// 停止集群的 Informer
func (s *Service) StopInformer(clusterName string)

// 注册事件处理器
func (s *Service) RegisterHandler(handler NodeEventHandler)
```

### 2. SmartCache (智能缓存)
**文件**: `backend/internal/smartcache/smart_cache.go`

**功能**:
- 实现 `NodeEventHandler` 接口，自动接收 Informer 事件
- 自动更新缓存，无需手动 invalidate
- 区分静态和动态属性
- 线程安全的并发访问

**关键方法**:
```go
// 获取单个节点
func (sc *SmartCache) GetNode(clusterName, nodeName string) (*corev1.Node, bool)

// 获取集群所有节点
func (sc *SmartCache) GetNodes(clusterName string) ([]*corev1.Node, bool)

// 设置节点 (用于初始化)
func (sc *SmartCache) SetNodes(clusterName string, nodes []*corev1.Node)
```

### 3. WebSocket Hub
**文件**: `backend/internal/websocket/hub.go`

**功能**:
- 实现 `NodeEventHandler` 接口，自动接收 Informer 事件
- 管理所有 WebSocket 客户端连接
- 基于集群的订阅/广播机制
- 心跳检测和自动断线处理

**关键方法**:
```go
// 启动 Hub
func (h *Hub) Run()

// 广播消息
func (h *Hub) Broadcast(message Message)

// 向特定集群订阅者推送
func (h *Hub) SendNodeUpdate(clusterName, nodeName string, changes []string, data interface{})
```

### 4. WebSocket Handler
**文件**: `backend/internal/handler/websocket/websocket.go`

**功能**:
- HTTP 升级到 WebSocket 协议
- 客户端连接管理
- 认证和授权
- 统计信息接口

**API Endpoints**:
```
GET  /api/v1/ws/nodes?cluster=xxx&token=xxx  - WebSocket 连接
GET  /api/v1/ws/stats                        - 获取统计信息
POST /api/v1/ws/test                         - 测试消息推送
GET  /api/v1/ws/ping                         - 健康检查
```

### 5. Realtime Manager
**文件**: `backend/internal/realtime/manager.go`

**功能**:
- 统一管理所有实时同步组件
- 集群注册和注销
- 自动连接 Informer → SmartCache → WebSocket
- 状态监控和查询

## 🔧 集成步骤

### 步骤 1: 修改 services.go

在 `backend/internal/service/services.go` 中添加实时管理器：

```go
import (
    "kube-node-manager/internal/realtime"
    // ... 其他导入
)

type Services struct {
    // ... 现有字段
    Realtime  *realtime.Manager
    WSHub     *websocket.Hub  // 导出供 handler 使用
}

func NewServices(db *gorm.DB, logger *logger.Logger, cfg *config.Config) *Services {
    // ... 现有代码

    // 创建实时管理器
    realtimeMgr := realtime.NewManager(logger)
    realtimeMgr.Start()

    return &Services{
        // ... 现有字段
        Realtime: realtimeMgr,
        WSHub:    realtimeMgr.GetWebSocketHub(),
    }
}
```

### 步骤 2: 修改 K8s Service

在 `backend/internal/service/k8s/k8s.go` 中集成 SmartCache：

```go
type Service struct {
    logger         *logger.Logger
    clients        map[string]*kubernetes.Clientset
    metricsClients map[string]*metricsclientset.Clientset
    mu             sync.RWMutex
    smartCache     *smartcache.SmartCache  // 替换旧的 K8sCache
    realtimeMgr    *realtime.Manager       // 引用实时管理器
}

// 修改 NewService
func NewService(logger *logger.Logger, realtimeMgr *realtime.Manager) *Service {
    return &Service{
        logger:         logger,
        clients:        make(map[string]*kubernetes.Clientset),
        metricsClients: make(map[string]*metricsclientset.Clientset),
        smartCache:     realtimeMgr.GetSmartCache(),
        realtimeMgr:    realtimeMgr,
    }
}

// 修改 CreateClient
func (s *Service) CreateClient(clusterName, kubeconfig string) error {
    // ... 现有创建 clientset 的代码

    // 注册到实时管理器
    if err := s.realtimeMgr.RegisterCluster(clusterName, clientset); err != nil {
        return err
    }

    // 初始化缓存：首次加载所有节点
    nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
    if err == nil {
        s.smartCache.SetNodes(clusterName, convertToNodePointers(nodes.Items))
    }

    return nil
}

// 修改 ListNodes - 从 SmartCache 读取
func (s *Service) ListNodes(clusterName string) ([]NodeInfo, error) {
    nodes, ok := s.smartCache.GetNodes(clusterName)
    if !ok {
        // 缓存未命中，从 API 加载
        return s.fetchNodesFromAPI(clusterName)
    }

    // 转换为 NodeInfo
    return s.convertNodesToNodeInfo(nodes), nil
}

// 修改 GetNode - 从 SmartCache 读取
func (s *Service) GetNode(clusterName, nodeName string) (*NodeInfo, error) {
    node, ok := s.smartCache.GetNode(clusterName, nodeName)
    if !ok {
        // 缓存未命中，从 API 加载
        return s.fetchNodeFromAPI(clusterName, nodeName)
    }

    nodeInfo := s.nodeToNodeInfo(node)
    return &nodeInfo, nil
}
```

### 步骤 3: 移除旧的缓存逻辑

在以下文件中删除 `InvalidateNode` 和 `InvalidateClusterCache` 调用：

**backend/internal/service/k8s/k8s.go**:
- 删除 `s.cache.InvalidateNode()` (行 492, 597, 635, 702, 744, 773)
- 删除 `s.cache.InvalidateCluster()` (行 1466)

**backend/internal/service/node/node.go**:
- 删除 `s.k8sSvc.InvalidateClusterCache()` (行 529, 582, 692, 859, 899, 939)

**backend/internal/service/label/label.go**:
- 删除 `s.k8sSvc.InvalidateClusterCache()` (行 286)

**backend/internal/service/taint/taint.go**:
- 删除 `s.k8sSvc.InvalidateClusterCache()` (行 305, 1113)

**原因**: Informer 会自动更新 SmartCache，不再需要手动清除缓存

### 步骤 4: 注册 WebSocket 路由

在 `backend/internal/handler/handlers.go` 中添加 WebSocket 路由：

```go
import (
    wshandler "kube-node-manager/internal/handler/websocket"
)

func RegisterRoutes(router *gin.Engine, services *service.Services, logger *logger.Logger) {
    // ... 现有路由

    // WebSocket 路由
    wsHandler := wshandler.NewHandler(services.WSHub, logger)
    wsHandler.RegisterRoutes(api)
}
```

### 步骤 5: 更新 main.go

在 `backend/cmd/main.go` 中添加优雅关闭：

```go
// 捕获中断信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

// 启动服务器
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatalf("Server failed: %v", err)
    }
}()

// 等待中断信号
<-quit
logger.Info("Shutting down server...")

// 关闭实时管理器
services.Realtime.Shutdown()

// 优雅关闭服务器
// ... 现有关闭代码
```

## 📡 前端集成

### WebSocket 连接

```javascript
// utils/websocket.js
class NodeWebSocket {
  constructor(clusterName, token) {
    this.clusterName = clusterName;
    this.token = token;
    this.ws = null;
    this.reconnectInterval = 5000;
    this.handlers = {
      node_add: [],
      node_update: [],
      node_delete: [],
      connected: [],
      error: []
    };
  }

  connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${window.location.host}/api/v1/ws/nodes?cluster=${this.clusterName}&token=${this.token}`;
    
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.emit('error', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket closed, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectInterval);
    };
  }

  handleMessage(message) {
    const { type, cluster_name, node_name, data, changes } = message;

    switch (type) {
      case 'node_add':
        this.emit('node_add', { clusterName: cluster_name, nodeName: node_name, node: data });
        break;
      case 'node_update':
        this.emit('node_update', { clusterName: cluster_name, nodeName: node_name, node: data, changes });
        break;
      case 'node_delete':
        this.emit('node_delete', { clusterName: cluster_name, nodeName: node_name });
        break;
      case 'connected':
        this.emit('connected', data);
        break;
      case 'ping':
        // 响应 pong
        this.send({ type: 'pong' });
        break;
    }
  }

  on(eventType, handler) {
    if (this.handlers[eventType]) {
      this.handlers[eventType].push(handler);
    }
  }

  emit(eventType, data) {
    if (this.handlers[eventType]) {
      this.handlers[eventType].forEach(handler => handler(data));
    }
  }

  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  subscribe(clusterName) {
    this.send({ type: 'subscribe', data: clusterName });
  }

  unsubscribe(clusterName) {
    this.send({ type: 'unsubscribe', data: clusterName });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

export default NodeWebSocket;
```

### Vue 组件集成示例

```vue
<template>
  <div>
    <div v-if="wsConnected" class="status-badge success">
      <i class="el-icon-connection"></i> 实时同步
    </div>
    <div v-else class="status-badge error">
      <i class="el-icon-warning"></i> 连接断开
    </div>

    <!-- 节点列表 -->
    <el-table :data="nodes" v-loading="loading">
      <!-- ... 表格列 -->
    </el-table>
  </div>
</template>

<script>
import NodeWebSocket from '@/utils/websocket';

export default {
  data() {
    return {
      nodes: [],
      loading: false,
      wsConnected: false,
      ws: null
    };
  },

  mounted() {
    this.loadNodes();
    this.setupWebSocket();
  },

  beforeDestroy() {
    if (this.ws) {
      this.ws.disconnect();
    }
  },

  methods: {
    async loadNodes() {
      this.loading = true;
      try {
        const response = await this.$api.node.list({ cluster_name: this.clusterName });
        this.nodes = response.data;
      } finally {
        this.loading = false;
      }
    },

    setupWebSocket() {
      const token = localStorage.getItem('token');
      this.ws = new NodeWebSocket(this.clusterName, token);

      // 连接成功
      this.ws.on('connected', () => {
        this.wsConnected = true;
        this.$message.success('实时同步已连接');
      });

      // 节点添加
      this.ws.on('node_add', ({ node }) => {
        this.nodes.push(this.convertNode(node));
        this.$message.info(`节点 ${node.metadata.name} 已添加`);
      });

      // 节点更新
      this.ws.on('node_update', ({ nodeName, node, changes }) => {
        const index = this.nodes.findIndex(n => n.name === nodeName);
        if (index !== -1) {
          this.$set(this.nodes, index, this.convertNode(node));
          
          // 显示变化提示
          if (changes.includes('labels')) {
            this.$message.info(`节点 ${nodeName} 标签已更新`);
          }
          if (changes.includes('taints')) {
            this.$message.info(`节点 ${nodeName} 污点已更新`);
          }
          if (changes.includes('schedulable')) {
            this.$message.info(`节点 ${nodeName} 调度状态已更新`);
          }
        }
      });

      // 节点删除
      this.ws.on('node_delete', ({ nodeName }) => {
        const index = this.nodes.findIndex(n => n.name === nodeName);
        if (index !== -1) {
          this.nodes.splice(index, 1);
          this.$message.warning(`节点 ${nodeName} 已删除`);
        }
      });

      // 连接错误
      this.ws.on('error', () => {
        this.wsConnected = false;
      });

      // 启动连接
      this.ws.connect();
    },

    convertNode(k8sNode) {
      // 转换 K8s Node 对象为前端需要的格式
      return {
        name: k8sNode.metadata.name,
        labels: k8sNode.metadata.labels,
        taints: k8sNode.spec.taints,
        schedulable: !k8sNode.spec.unschedulable,
        // ... 其他字段
      };
    }
  }
};
</script>

<style scoped>
.status-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  margin-bottom: 16px;
}

.status-badge.success {
  background-color: #67C23A;
  color: white;
}

.status-badge.error {
  background-color: #F56C6C;
  color: white;
}
</style>
```

## 🧪 测试指南

### 1. 测试 Informer 实时监听

```bash
# 终端 1: 启动后端服务
cd backend
go run cmd/main.go

# 终端 2: 使用 kubectl 修改节点
kubectl label node <node-name> test=value
kubectl taint node <node-name> key=value:NoSchedule
kubectl cordon <node-name>

# 查看后端日志，应该看到 Informer 事件
```

### 2. 测试 WebSocket 实时推送

```bash
# 使用 wscat 测试 WebSocket
npm install -g wscat
wscat -c "ws://localhost:8080/api/v1/ws/nodes?cluster=test&token=xxx"

# 修改节点后，应该实时收到消息：
{
  "type": "node_update",
  "cluster_name": "test",
  "node_name": "node-1",
  "changes": ["labels"],
  "timestamp": "2025-01-01T00:00:00Z"
}
```

### 3. 测试智能缓存

```bash
# 启动服务后立即查询节点列表
curl http://localhost:8080/api/v1/nodes?cluster_name=test

# 使用 kubectl 修改节点
kubectl label node <node-name> env=prod

# 立即再次查询（无需等待，应该看到最新数据）
curl http://localhost:8080/api/v1/nodes?cluster_name=test
```

## ⚠️ 注意事项

### 1. 性能考虑
- Informer 使用内存缓存，每个集群约占用 10-50MB
- WebSocket 连接数建议限制在 1000 以内
- 如果集群节点数超过 1000，考虑增加 resyncPeriod

### 2. 安全考虑
- WebSocket 连接必须经过认证
- 生产环境配置正确的 `CheckOrigin`
- 使用 WSS (WebSocket over TLS)

### 3. 高可用性
- Informer 在多副本环境下每个副本独立运行（无状态）
- WebSocket 连接会分散到不同副本（需要配置 sticky session 或使用 Redis Pub/Sub）
- SmartCache 是本地内存，多副本间不共享（Informer 会自动同步）

### 4. 监控指标
- Informer 同步延迟
- WebSocket 连接数
- 缓存命中率
- 事件处理速度

## 📊 性能对比

### 优化前（强制 forceRefresh）:
- 查询延迟: 100-500ms (直接调用 K8s API)
- API 请求数: 高频繁（每次查询）
- 数据一致性: 强一致

### 优化后（Informer + SmartCache）:
- 查询延迟: <5ms (内存读取)
- API 请求数: 极低（仅 Watch）
- 数据一致性: 最终一致 (延迟 <1秒)
- 实时推送: WebSocket 零延迟

## 🎯 下一步

1. **完成集成**: 按照本文档步骤完成所有代码集成
2. **测试验证**: 在开发环境完整测试所有功能
3. **前端开发**: 实现 WebSocket 客户端和 UI 实时更新
4. **性能调优**: 根据实际负载调整参数
5. **生产部署**: 分阶段灰度发布

## 📝 相关文件

- `backend/internal/informer/informer.go` - Informer 服务
- `backend/internal/smartcache/smart_cache.go` - 智能缓存
- `backend/internal/websocket/hub.go` - WebSocket Hub
- `backend/internal/handler/websocket/websocket.go` - WebSocket Handler
- `backend/internal/realtime/manager.go` - 实时管理器

---

**文档版本**: v1.0
**创建日期**: 2025-10-30
**最后更新**: 2025-10-30

