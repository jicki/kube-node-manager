# 迁移文件路径智能检测

## 问题描述

在不同的启动场景下，应用的工作目录可能不同：

- **场景 1**：在 `backend/` 目录下启动
  ```bash
  cd backend
  ./bin/kube-node-manager
  ```
  工作目录：`backend/`，迁移文件在 `./migrations`

- **场景 2**：在项目根目录启动
  ```bash
  cd /path/to/kube-node-manager
  ./backend/bin/kube-node-manager
  ```
  工作目录：项目根目录，迁移文件在 `./backend/migrations`

- **场景 3**：在容器中启动
  ```bash
  docker run kube-node-manager
  ```
  工作目录：`/app/`，迁移文件可能在 `/app/migrations` 或 `/app/backend/migrations`

如果使用固定的相对路径（如 `./migrations`），在某些场景下会找不到迁移文件。

## 解决方案

实现了**智能路径检测**功能，按优先级尝试多个可能的路径：

```go
func detectMigrationsPath() string {
    possiblePaths := []string{
        "./migrations",                    // 当前目录下的 migrations
        "./backend/migrations",            // 项目根目录下的 backend/migrations
        "../migrations",                   // 父目录下的 migrations
        "/app/migrations",                 // 容器中的绝对路径
        "/app/backend/migrations",         // 容器中的另一个可能路径
    }

    for _, path := range possiblePaths {
        if _, err := os.Stat(path); err == nil {
            log.Printf("Found migrations directory at: %s", path)
            return path
        }
    }

    log.Println("Warning: migrations directory not found, using default path: ./migrations")
    return "./migrations"
}
```

## 工作原理

1. **按顺序检查**：从第一个路径开始，依次检查每个可能的路径
2. **存在性验证**：使用 `os.Stat()` 检查目录是否存在
3. **返回第一个找到的**：一旦找到存在的目录，立即返回并使用
4. **兜底默认值**：如果所有路径都不存在，返回 `./migrations`（让迁移管理器处理）

## 路径优先级说明

### 1. `./migrations` - 最高优先级

**适用场景**：在 `backend/` 目录下启动

```bash
cd /path/to/kube-node-manager/backend
./bin/kube-node-manager
```

**目录结构**：
```
backend/
  ├── bin/kube-node-manager
  └── migrations/
      ├── 001_xxx.sql
      └── 021_xxx.sql
```

### 2. `./backend/migrations`

**适用场景**：在项目根目录启动

```bash
cd /path/to/kube-node-manager
./backend/bin/kube-node-manager
```

**目录结构**：
```
kube-node-manager/
  └── backend/
      ├── bin/kube-node-manager
      └── migrations/
          ├── 001_xxx.sql
          └── 021_xxx.sql
```

### 3. `../migrations`

**适用场景**：可执行文件在 `backend/bin/` 目录，工作目录在 `backend/bin/`

```bash
cd /path/to/kube-node-manager/backend/bin
./kube-node-manager
```

**目录结构**：
```
backend/
  ├── bin/
  │   └── kube-node-manager  (当前目录)
  └── migrations/            (../ 可达)
```

### 4. `/app/migrations`

**适用场景**：容器中的绝对路径（Dockerfile 将迁移文件复制到 `/app/migrations`）

```dockerfile
COPY backend/migrations /app/migrations
```

### 5. `/app/backend/migrations`

**适用场景**：容器中保持原有目录结构

```dockerfile
COPY backend /app/backend
```

## 日志输出

### 成功找到迁移目录

```
Found migrations directory at: ./backend/migrations
Starting database migration check...
```

### 未找到迁移目录

```
Warning: migrations directory not found, using default path: ./migrations
Starting database migration check...
Migration directory ./migrations does not exist, skipping migration
No migration files found, skipping migration
```

## 使用场景示例

### 场景 1：开发环境（在 backend/ 目录下）

```bash
cd /path/to/kube-node-manager/backend
go run cmd/main.go
```

**检测结果**：
- 检查 `./migrations` ✅ **找到！**
- 使用路径：`./migrations`

### 场景 2：开发环境（在项目根目录）

```bash
cd /path/to/kube-node-manager
go run backend/cmd/main.go
```

**检测结果**：
- 检查 `./migrations` ❌ 不存在
- 检查 `./backend/migrations` ✅ **找到！**
- 使用路径：`./backend/migrations`

### 场景 3：生产环境（编译后的二进制）

```bash
cd /path/to/kube-node-manager/backend
./bin/kube-node-manager
```

**检测结果**：
- 检查 `./migrations` ✅ **找到！**
- 使用路径：`./migrations`

### 场景 4：容器环境

```bash
docker run -it kube-node-manager /app/bin/kube-node-manager
```

**Dockerfile 示例**：
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN cd backend && go build -o bin/kube-node-manager cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/backend/bin/kube-node-manager /app/bin/
COPY --from=builder /build/backend/migrations /app/migrations
CMD ["/app/bin/kube-node-manager"]
```

**检测结果**：
- 检查 `./migrations` ❌ 不存在
- 检查 `./backend/migrations` ❌ 不存在
- 检查 `../migrations` ❌ 不存在
- 检查 `/app/migrations` ✅ **找到！**
- 使用路径：`/app/migrations`

### 场景 5：Kubernetes 部署

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migrations
data:
  001_xxx.sql: |
    -- SQL content
  021_xxx.sql: |
    -- SQL content
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        image: kube-node-manager:latest
        volumeMounts:
        - name: migrations
          mountPath: /app/migrations
      volumes:
      - name: migrations
        configMap:
          name: migrations
```

**检测结果**：
- 检查 `/app/migrations` ✅ **找到！**
- 使用路径：`/app/migrations`

## 手动迁移工具

`tools/migrate.go` 也使用相同的路径检测逻辑：

```bash
# 在 backend/ 目录下
cd backend
go run tools/migrate.go -cmd status
# 输出: Found migrations directory at: ./migrations

# 在项目根目录
cd /path/to/kube-node-manager
go run backend/tools/migrate.go -cmd status
# 输出: Found migrations directory at: ./backend/migrations
```

## 最佳实践

### 1. 推荐的启动方式

**开发环境**：
```bash
cd backend
go run cmd/main.go
```

**生产环境**：
```bash
cd backend
./bin/kube-node-manager
```

### 2. Dockerfile 建议

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN cd backend && \
    go build -o bin/kube-node-manager cmd/main.go && \
    go build -o bin/migrate tools/migrate.go

FROM alpine:latest
WORKDIR /app

# 复制二进制文件
COPY --from=builder /build/backend/bin/kube-node-manager /app/bin/
COPY --from=builder /build/backend/bin/migrate /app/bin/

# 复制迁移文件到容器中的标准位置
COPY --from=builder /build/backend/migrations /app/migrations

# 设置工作目录为 /app（迁移文件在 /app/migrations）
WORKDIR /app

CMD ["/app/bin/kube-node-manager"]
```

### 3. 环境变量支持（未来增强）

可以考虑添加环境变量支持，允许手动指定迁移目录：

```bash
export MIGRATIONS_PATH=/custom/path/to/migrations
./bin/kube-node-manager
```

## 故障排查

### 问题：找不到迁移目录

**症状**：
```
Warning: migrations directory not found, using default path: ./migrations
Migration directory ./migrations does not exist, skipping migration
```

**排查步骤**：

1. **检查当前工作目录**
   ```bash
   pwd
   ```

2. **检查迁移目录是否存在**
   ```bash
   ls -la migrations/
   ls -la backend/migrations/
   ls -la /app/migrations/
   ```

3. **手动指定路径（临时方案）**
   ```bash
   # 创建符号链接
   ln -s /path/to/backend/migrations ./migrations
   ```

4. **使用正确的启动方式**
   ```bash
   # 确保在 backend/ 目录下启动
   cd backend
   ./bin/kube-node-manager
   ```

### 问题：容器中找不到迁移文件

**排查步骤**：

1. **检查 Dockerfile 是否正确复制迁移文件**
   ```dockerfile
   COPY backend/migrations /app/migrations
   ```

2. **进入容器检查文件**
   ```bash
   docker exec -it <container-id> sh
   ls -la /app/migrations/
   ```

3. **检查工作目录**
   ```bash
   docker exec -it <container-id> pwd
   ```

## 技术细节

### 文件系统检查

使用 `os.Stat()` 检查目录是否存在：

```go
if _, err := os.Stat(path); err == nil {
    // 目录存在
    return path
}
// 目录不存在，继续检查下一个
```

### 性能影响

路径检测只在应用启动时执行一次，对性能影响可忽略不计：

- 最多检查 5 个路径
- 每次检查只是一个文件系统 stat 调用
- 找到后立即返回，不继续检查

### 线程安全

路径检测在应用启动的主线程中执行，无需考虑线程安全问题。

## 相关文档

- [自动迁移功能说明](./auto-migration.md)
- [迁移工具使用指南](../tools/README.md)

## 总结

通过智能路径检测功能，应用可以：

- ✅ 在不同工作目录下正常启动
- ✅ 适应开发、测试、生产等多种环境
- ✅ 简化 Docker 和 Kubernetes 部署
- ✅ 提供清晰的日志输出便于排查问题

无论你在哪个目录启动应用，系统都能自动找到正确的迁移文件目录！🎉

