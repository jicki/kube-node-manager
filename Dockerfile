# 多阶段构建 - 前端构建阶段
FROM reg.deeproute.ai/deeproute-public/node:18-alpine AS frontend-builder

WORKDIR /app/frontend

# 复制前端package文件
COPY frontend/package*.json ./

# 安装前端依赖
RUN npm ci

# 复制前端源代码
COPY frontend/ .

# 构建前端应用
RUN npm run build

# 多阶段构建 - 后端构建阶段
FROM reg.deeproute.ai/deeproute-public/zzh/golang:1.24-alpine-plugin AS backend-builder

WORKDIR /app

# # 安装必要的包
# RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

# # 安装statik工具
# RUN go install github.com/rakyll/statik@latest

# 复制go.mod和go.sum
COPY backend/go.mod backend/go.sum ./

# 下载Go依赖
RUN go mod download

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist ./web

# 生成静态文件嵌入代码
RUN statik -src=./web -dest=. -f

# 复制后端源代码
COPY backend/ .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o main ./cmd

# 最终运行阶段
FROM reg.deeproute.ai/deeproute-public/zzh/alpine:3.21-plugin

# # 安装必要的运行时包
# RUN apk --no-cache add ca-certificates tzdata wget

# 创建非root用户
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 复制构建的二进制文件
COPY --from=backend-builder /app/main .

# 创建数据目录
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# 设置环境变量
ENV GIN_MODE=release \
    DATABASE_DSN=./data/kube-node-manager.db

# 运行应用
CMD ["./main"]