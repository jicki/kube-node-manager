# Klog 日志格式配置说明

## 概述

本文档说明如何配置 Kubernetes 客户端库 (klog/v2) 的日志格式，使其输出格式与系统自定义 logger 保持一致。

## 问题背景

### 原始日志格式
Kubernetes 客户端库默认使用 klog 输出日志，格式如下：
```
I1104 17:43:39.589719       1 request.go:752] "Waited before sending request"
```

格式说明：
- `I` - 日志级别（I=Info, W=Warning, E=Error）
- `1104` - 月日（MMdd）
- `17:43:39.589719` - 时间（HH:mm:ss.微秒）
- `1` - Go routine ID
- `request.go:752` - 文件名:行号
- 消息内容

### 目标日志格式
希望改为与系统统一的格式：
```
INFO: 2025/11/04 17:43:38 logger.go message content
```

格式说明：
- `INFO:` - 日志级别前缀
- `2025/11/04` - 完整日期（YYYY/MM/DD）
- `17:43:38` - 时间（HH:mm:ss）
- `logger.go` - 文件名（短格式）
- 消息内容

## 解决方案

### 1. 配置 klog 基本参数

在 `main.go` 的 `main()` 函数开始处添加 klog 初始化配置：

```go
// 初始化 klog 配置
klog.InitFlags(nil)
flag.Set("logtostderr", "false")         // 不输出到 stderr
flag.Set("alsologtostderr", "false")     // 不同时输出到 stderr
flag.Set("stderrthreshold", "FATAL")     // 只有 FATAL 级别才输出到 stderr
flag.Set("v", "0")                       // 设置详细级别为 0（最小）
flag.Parse()
```

### 2. 实现 logr.LogSink 接口适配器

创建 `klogAdapter` 结构体，实现 `logr.LogSink` 接口，将 klog 的日志重定向到自定义 logger：

```go
// klogAdapter 实现 logr.Logger 接口，将 klog 的日志适配到自定义 logger
type klogAdapter struct {
	logger *logger.Logger
	name   string
	depth  int
}

func (k *klogAdapter) Init(info logr.RuntimeInfo) {
	k.depth = info.CallDepth
}

func (k *klogAdapter) Enabled(level int) bool {
	return true
}

func (k *klogAdapter) Info(level int, msg string, keysAndValues ...interface{}) {
	// 格式化键值对
	kvStr := formatKeyValues(keysAndValues)
	if kvStr != "" {
		k.logger.Infof("%s %s", msg, kvStr)
	} else {
		k.logger.Info(msg)
	}
}

func (k *klogAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	kvStr := formatKeyValues(keysAndValues)
	if err != nil {
		if kvStr != "" {
			k.logger.Errorf("%s: %v %s", msg, err, kvStr)
		} else {
			k.logger.Errorf("%s: %v", msg, err)
		}
	} else {
		if kvStr != "" {
			k.logger.Errorf("%s %s", msg, kvStr)
		} else {
			k.logger.Error(msg)
		}
	}
}

func (k *klogAdapter) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &klogAdapter{
		logger: k.logger,
		name:   k.name,
		depth:  k.depth,
	}
}

func (k *klogAdapter) WithName(name string) logr.LogSink {
	newName := k.name
	if len(newName) > 0 {
		newName += "."
	}
	newName += name
	return &klogAdapter{
		logger: k.logger,
		name:   newName,
		depth:  k.depth,
	}
}
```

### 3. 辅助函数

添加键值对格式化辅助函数：

```go
// formatKeyValues 格式化键值对为字符串
func formatKeyValues(keysAndValues ...interface{}) string {
	if len(keysAndValues) == 0 {
		return ""
	}
	
	// keysAndValues 是可变参数，直接遍历
	var parts []string
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			parts = append(parts, fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1]))
		}
	}
	
	if len(parts) > 0 {
		return "[" + strings.Join(parts, " ") + "]"
	}
	return ""
}
```

### 4. 设置 klog Logger

在创建自定义 logger 后，设置 klog 使用适配器：

```go
logger := logger.NewLogger()

// 配置 klog 使用自定义格式
klog.SetLogger(&klogAdapter{logger: logger})
```

## 依赖包

确保 `go.mod` 中包含以下依赖：

```go
github.com/go-logr/logr v1.4.2
k8s.io/klog/v2 v2.130.1
```

运行 `go mod tidy` 确保依赖正确。

## 验证

### 1. 编译测试

```bash
cd backend
go build -o bin/kube-node-manager ./cmd/main.go
```

### 2. 运行测试

启动应用并观察日志输出：

```bash
./bin/kube-node-manager
```

### 3. 预期输出

Kubernetes 客户端的日志应该以统一格式输出：

```
INFO: 2025/11/04 17:43:38 main.go Server starting on port 8080
INFO: 2025/11/04 17:43:39 request.go Waited before sending request
```

而不是原来的格式：

```
I1104 17:43:39.589719       1 request.go:752] "Waited before sending request"
```

## 优点

1. **统一性**：所有日志使用相同的格式，便于阅读和解析
2. **可维护性**：集中管理日志格式配置
3. **灵活性**：可以轻松切换到结构化 JSON 日志格式（通过 `LOG_FORMAT=json` 环境变量）
4. **兼容性**：不影响现有的日志记录功能

## 注意事项

1. `klog.InitFlags(nil)` 必须在 `flag.Parse()` 之前调用
2. 适配器必须在初始化任何 Kubernetes 客户端之前设置
3. 如果需要调整 klog 的详细级别，修改 `flag.Set("v", "0")` 中的数字（0-10，数字越大越详细）

## 参考资料

- [klog 官方文档](https://github.com/kubernetes/klog)
- [logr 接口定义](https://github.com/go-logr/logr)
- [Kubernetes 客户端日志最佳实践](https://kubernetes.io/docs/concepts/cluster-administration/logging/)

