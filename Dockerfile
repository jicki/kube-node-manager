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

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

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

# 构建应用 (启用CGO以支持SQLite)
ENV CGO_ENABLED=1
ENV GOOS=linux
RUN go build -a -o main ./cmd

# 最终运行阶段
FROM reg.deeproute.ai/deeproute-public/zzh/alpine:3.21-plugin

# # 安装必要的运行时包
# RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# 复制构建的二进制文件
COPY --from=backend-builder /app/main .

# 暴露端口
EXPOSE 8080

# 健康检查 (使用内置命令，避免依赖外部工具)
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD nc -z localhost 8080 || exit 1

# 设置环境变量
ENV GIN_MODE=release \
    DATABASE_DSN=./data/kube-node-manager.db

# 清除基础镜像的入口点并设置我们的启动命令
ENTRYPOINT []
CMD ["./main"]