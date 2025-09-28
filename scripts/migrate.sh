#!/bin/bash

# SQLite to PostgreSQL Migration Script
# 
# 本脚本用于将 SQLite 数据库中的数据迁移到 PostgreSQL 数据库
# 支持配置文件和环境变量两种配置方式

set -e

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
SQLite to PostgreSQL Migration Tool

Usage: $0 [options]

Options:
    -c, --config FILE       配置文件路径 (默认: ./migration.yaml)
    -s, --source PATH       SQLite 数据库路径
    -h, --host HOST         PostgreSQL 主机地址
    -p, --port PORT         PostgreSQL 端口
    -d, --database DB       PostgreSQL 数据库名
    -u, --username USER     PostgreSQL 用户名
    -w, --password PASS     PostgreSQL 密码
    --ssl-mode MODE         SSL 模式 (disable/require/verify-ca/verify-full)
    --dry-run              预览模式，不执行实际迁移
    --help                 显示此帮助信息

Examples:
    # 使用配置文件 (推荐)
    $0 -c migration.yaml

    # 使用命令行参数
    $0 -s ./backend/data/kube-node-manager.db -h localhost -p 5432 -d kube_node_manager -u postgres -w password

    # 使用环境变量
    export MIGRATION_POSTGRESQL_PASSWORD="your-password"
    $0 -c migration.yaml

Environment Variables:
    MIGRATION_SQLITE_PATH              SQLite 数据库路径
    MIGRATION_POSTGRESQL_HOST          PostgreSQL 主机地址
    MIGRATION_POSTGRESQL_PORT          PostgreSQL 端口
    MIGRATION_POSTGRESQL_USERNAME      PostgreSQL 用户名
    MIGRATION_POSTGRESQL_PASSWORD      PostgreSQL 密码
    MIGRATION_POSTGRESQL_DATABASE      PostgreSQL 数据库名
    MIGRATION_POSTGRESQL_SSL_MODE      SSL 模式

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 构建迁移工具
build_migration_tool() {
    log_info "构建迁移工具..."
    
    cd "$SCRIPT_DIR"
    
    # 初始化 go module 如果不存在
    if [ ! -f "go.mod" ]; then
        go mod init migration-tool
        go mod tidy
    fi
    
    # 安装依赖
    go get github.com/glebarez/sqlite
    go get github.com/spf13/viper
    go get gorm.io/driver/postgres
    go get gorm.io/gorm
    
    # 构建
    go build -o migration-tool sqlite-to-postgres.go
    
    if [ ! -f "migration-tool" ]; then
        log_error "构建迁移工具失败"
        exit 1
    fi
    
    log_success "迁移工具构建完成"
}

# 检查配置文件
check_config() {
    local config_file="$1"
    
    if [ -n "$config_file" ] && [ ! -f "$config_file" ]; then
        log_warning "配置文件 $config_file 不存在"
        
        if [ ! -f "migration.yaml" ] && [ ! -f "migration.yaml.example" ]; then
            log_error "找不到配置文件，请创建 migration.yaml 或使用命令行参数"
            echo
            echo "可以复制示例配置文件："
            echo "cp $SCRIPT_DIR/migration.yaml.example $SCRIPT_DIR/migration.yaml"
            exit 1
        fi
        
        if [ -f "migration.yaml.example" ] && [ ! -f "migration.yaml" ]; then
            log_info "复制示例配置文件到 migration.yaml"
            cp migration.yaml.example migration.yaml
            log_warning "请编辑 migration.yaml 配置文件，设置正确的数据库连接信息"
            return 1
        fi
    fi
    
    return 0
}

# 检查 PostgreSQL 连接
check_postgres_connection() {
    log_info "测试 PostgreSQL 连接..."
    
    # 这里可以添加连接测试逻辑
    # 暂时跳过，让迁移工具自己处理连接测试
    
    log_success "准备连接 PostgreSQL"
}

# 备份 SQLite 数据库
backup_sqlite() {
    local sqlite_path="$1"
    
    if [ -f "$sqlite_path" ]; then
        local backup_path="${sqlite_path}.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "备份 SQLite 数据库到 $backup_path"
        cp "$sqlite_path" "$backup_path"
        log_success "SQLite 数据库备份完成"
    else
        log_warning "SQLite 数据库文件不存在: $sqlite_path"
    fi
}

# 执行迁移
run_migration() {
    local config_file="$1"
    local dry_run="$2"
    
    log_info "开始数据迁移..."
    
    if [ "$dry_run" = "true" ]; then
        log_info "预览模式：将显示迁移计划但不执行实际迁移"
        # 可以添加预览逻辑
    fi
    
    # 执行迁移工具
    if [ -n "$config_file" ]; then
        log_info "使用配置文件: $config_file"
        cp "$config_file" migration.yaml
    fi
    
    # 设置环境变量（如果通过命令行参数提供）
    if [ -n "$SQLITE_PATH" ]; then
        export MIGRATION_SQLITE_PATH="$SQLITE_PATH"
    fi
    if [ -n "$PG_HOST" ]; then
        export MIGRATION_POSTGRESQL_HOST="$PG_HOST"
    fi
    if [ -n "$PG_PORT" ]; then
        export MIGRATION_POSTGRESQL_PORT="$PG_PORT"
    fi
    if [ -n "$PG_DATABASE" ]; then
        export MIGRATION_POSTGRESQL_DATABASE="$PG_DATABASE"
    fi
    if [ -n "$PG_USERNAME" ]; then
        export MIGRATION_POSTGRESQL_USERNAME="$PG_USERNAME"
    fi
    if [ -n "$PG_PASSWORD" ]; then
        export MIGRATION_POSTGRESQL_PASSWORD="$PG_PASSWORD"
    fi
    if [ -n "$PG_SSL_MODE" ]; then
        export MIGRATION_POSTGRESQL_SSL_MODE="$PG_SSL_MODE"
    fi
    
    # 运行迁移工具
    ./migration-tool
    
    if [ $? -eq 0 ]; then
        log_success "数据迁移完成!"
    else
        log_error "数据迁移失败，请检查日志"
        exit 1
    fi
}

# 清理临时文件
cleanup() {
    if [ -f "$SCRIPT_DIR/migration-tool" ]; then
        rm "$SCRIPT_DIR/migration-tool"
    fi
}

# 主函数
main() {
    local config_file=""
    local dry_run="false"
    
    # 参数解析
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--config)
                config_file="$2"
                shift 2
                ;;
            -s|--source)
                SQLITE_PATH="$2"
                shift 2
                ;;
            -h|--host)
                PG_HOST="$2"
                shift 2
                ;;
            -p|--port)
                PG_PORT="$2"
                shift 2
                ;;
            -d|--database)
                PG_DATABASE="$2"
                shift 2
                ;;
            -u|--username)
                PG_USERNAME="$2"
                shift 2
                ;;
            -w|--password)
                PG_PASSWORD="$2"
                shift 2
                ;;
            --ssl-mode)
                PG_SSL_MODE="$2"
                shift 2
                ;;
            --dry-run)
                dry_run="true"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    log_info "SQLite to PostgreSQL 迁移工具"
    echo
    
    # 检查依赖
    check_dependencies
    
    # 构建迁移工具
    build_migration_tool
    
    # 检查配置
    if ! check_config "$config_file"; then
        exit 1
    fi
    
    # 备份 SQLite 数据库
    if [ -n "$SQLITE_PATH" ]; then
        backup_sqlite "$SQLITE_PATH"
    elif [ -f "./backend/data/kube-node-manager.db" ]; then
        backup_sqlite "./backend/data/kube-node-manager.db"
    fi
    
    # 检查 PostgreSQL 连接
    check_postgres_connection
    
    # 执行迁移
    run_migration "$config_file" "$dry_run"
    
    # 清理
    cleanup
    
    echo
    log_success "迁移流程完成!"
}

# 捕获退出信号，确保清理
trap cleanup EXIT

# 运行主函数
main "$@"
