#!/bin/bash

# Ansible 数据库级联删除修复应用脚本
# 用于快速应用 004_fix_ansible_foreign_keys.sql 迁移

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取脚本目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATION_FILE="$BACKEND_DIR/migrations/004_fix_ansible_foreign_keys.sql"

print_info "Ansible 数据库级联删除修复脚本"
echo "=========================================="

# 检查迁移文件是否存在
if [ ! -f "$MIGRATION_FILE" ]; then
    print_error "迁移文件不存在: $MIGRATION_FILE"
    exit 1
fi

print_info "迁移文件: $MIGRATION_FILE"

# 从环境变量或参数读取数据库连接信息
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-kube_node_manager}"
DB_PASSWORD="${DB_PASSWORD}"

# 如果没有设置密码，提示输入
if [ -z "$DB_PASSWORD" ]; then
    print_warn "请输入数据库密码:"
    read -s DB_PASSWORD
    echo
fi

# 检查 PostgreSQL 客户端
if ! command -v psql &> /dev/null; then
    print_error "未找到 psql 客户端，请先安装 PostgreSQL 客户端工具"
    exit 1
fi

print_info "数据库连接信息:"
echo "  主机: $DB_HOST"
echo "  端口: $DB_PORT"
echo "  用户: $DB_USER"
echo "  数据库: $DB_NAME"
echo

# 测试数据库连接
print_info "测试数据库连接..."
export PGPASSWORD="$DB_PASSWORD"
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
    print_error "数据库连接失败，请检查连接信息"
    exit 1
fi
print_info "数据库连接成功"

# 备份当前外键约束
print_info "备份当前外键约束..."
BACKUP_FILE="$BACKEND_DIR/migrations/backup_constraints_$(date +%Y%m%d_%H%M%S).sql"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF > "$BACKUP_FILE"
-- 导出当前外键约束
SELECT 'ALTER TABLE ' || tc.table_name || 
       ' DROP CONSTRAINT IF EXISTS ' || tc.constraint_name || ';'
FROM information_schema.table_constraints AS tc
WHERE tc.table_name IN ('ansible_logs', 'ansible_tasks', 'ansible_inventories')
  AND tc.constraint_type = 'FOREIGN KEY';

SELECT 'ALTER TABLE ' || tc.table_name ||
       ' ADD CONSTRAINT ' || tc.constraint_name ||
       ' FOREIGN KEY (' || kcu.column_name || ')' ||
       ' REFERENCES ' || ccu.table_name || '(' || ccu.column_name || ')' ||
       CASE rc.delete_rule
         WHEN 'CASCADE' THEN ' ON DELETE CASCADE'
         WHEN 'SET NULL' THEN ' ON DELETE SET NULL'
         WHEN 'RESTRICT' THEN ' ON DELETE RESTRICT'
         ELSE ''
       END || ';'
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
JOIN information_schema.referential_constraints AS rc
  ON rc.constraint_name = tc.constraint_name
WHERE tc.table_name IN ('ansible_logs', 'ansible_tasks', 'ansible_inventories')
  AND tc.constraint_type = 'FOREIGN KEY';
EOF

print_info "备份文件: $BACKUP_FILE"

# 确认执行
print_warn "即将执行数据库迁移，继续吗? (y/N)"
read -r CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    print_info "已取消"
    exit 0
fi

# 执行迁移
print_info "执行数据库迁移..."
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$MIGRATION_FILE"; then
    print_info "迁移执行成功！"
else
    print_error "迁移执行失败！"
    print_info "可以使用备份文件恢复: $BACKUP_FILE"
    exit 1
fi

# 验证迁移结果
print_info "验证迁移结果..."

# 检查外键约束
print_info "检查外键约束..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    tc.table_name as "表名", 
    tc.constraint_name as "约束名",
    kcu.column_name as "列名",
    ccu.table_name as "引用表",
    rc.delete_rule as "删除规则"
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
JOIN information_schema.referential_constraints AS rc
    ON rc.constraint_name = tc.constraint_name
WHERE tc.table_name IN ('ansible_logs', 'ansible_tasks', 'ansible_inventories')
    AND tc.constraint_type = 'FOREIGN KEY'
ORDER BY tc.table_name, tc.constraint_name;
EOF

# 检查唯一索引
print_info "检查唯一索引..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    tablename as "表名",
    indexname as "索引名"
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name%'
ORDER BY tablename;
EOF

print_info "=========================================="
print_info "修复完成！"
print_info ""
print_info "后续步骤:"
print_info "1. 重启应用程序以加载新的模型定义"
print_info "2. 测试删除功能是否正常"
print_info "3. 如有问题，可使用备份文件恢复: $BACKUP_FILE"

# 清理密码环境变量
unset PGPASSWORD

