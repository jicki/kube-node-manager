# 多阶段构建 - 前端构建阶段
FROM reg.deeproute.ai/deeproute-public/node:18-alpine AS frontend-builder

WORKDIR /app/frontend

# 复制前端package文件
COPY frontend/package*.json ./

# 安装前端依赖
RUN npm ci

# 复制VERSION文件到父目录，供前端构建使用
COPY VERSION ../VERSION

# 复制前端源代码
COPY frontend/ .

# 设置构建时环境变量（可通过docker build --build-arg传入）
ARG VITE_API_BASE_URL=""
ARG VITE_ENABLE_LDAP="false"
ARG CACHEBUST=1

# 设置环境变量供构建使用
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_ENABLE_LDAP=$VITE_ENABLE_LDAP
ENV CACHEBUST=$CACHEBUST

# 构建前端应用（CACHEBUST 用于强制刷新缓存）
RUN echo "Building frontend with CACHEBUST=${CACHEBUST}..." && npm run build

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

# 构建应用 (禁用CGO使用纯Go SQLite驱动)
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -a -o main ./cmd

# 最终运行阶段
FROM reg.deeproute.ai/deeproute-public/zzh/alpine:3.21-plugin

# 安装 Ansible 和必要的运行时包
RUN apk --no-cache add \
    ansible \
    python3 \
    py3-pip \
    openssh-client \
    sshpass \
    ca-certificates \
    tzdata && \
    # 创建 Ansible 配置目录
    mkdir -p /etc/ansible && \
    # 配置 Ansible 默认设置
    echo "[defaults]" > /etc/ansible/ansible.cfg && \
    echo "host_key_checking = False" >> /etc/ansible/ansible.cfg && \
    echo "timeout = 30" >> /etc/ansible/ansible.cfg && \
    echo "gather_timeout = 30" >> /etc/ansible/ansible.cfg && \
    # 清理缓存
    rm -rf /var/cache/apk/*

WORKDIR /app

# 复制构建的二进制文件
COPY --from=backend-builder /app/main .

# 复制VERSION文件
COPY VERSION .

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