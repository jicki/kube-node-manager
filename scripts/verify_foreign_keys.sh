#!/bin/bash

# 验证所有外键约束配置是否正确
# 用于确认数据库迁移已成功应用

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 数据库配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-kube_node_manager}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-}"

usage() {
    cat <<EOF
${BLUE}外键约束验证脚本${NC}

${YELLOW}使用方法:${NC}
    $0 [选项]

${YELLOW}选项:${NC}
    -h, --host HOST         数据库主机 (默认: localhost)
    -p, --port PORT         数据库端口 (默认: 5432)
    -d, --database NAME     数据库名称 (默认: kube_node_manager)
    -u, --user USER         数据库用户 (默认: postgres)
    -w, --password PASS     数据库密码
    --help                  显示帮助信息

${YELLOW}示例:${NC}
    # 使用默认配置
    $0

    # 自定义数据库连接
    $0 -h db.example.com -u admin -w secret123

${YELLOW}环境变量:${NC}
    DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD
EOF
    exit 1
}

# 构建 psql 命令
build_psql_cmd() {
    local cmd="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t"
    if [ -n "$DB_PASSWORD" ]; then
        export PGPASSWORD="$DB_PASSWORD"
    fi
    echo "$cmd"
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            DB_HOST="$2"
            shift 2
            ;;
        -p|--port)
            DB_PORT="$2"
            shift 2
            ;;
        -d|--database)
            DB_NAME="$2"
            shift 2
            ;;
        -u|--user)
            DB_USER="$2"
            shift 2
            ;;
        -w|--password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --help)
            usage
            ;;
        *)
            echo -e "${RED}未知参数: $1${NC}"
            usage
            ;;
    esac
done

# 检查 psql
if ! command -v psql &> /dev/null; then
    echo -e "${RED}❌ 错误: 未找到 psql 命令${NC}"
    echo -e "${YELLOW}请安装 PostgreSQL 客户端${NC}"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证外键约束配置${NC}"
echo -e "${BLUE}========================================${NC}"
echo

PSQL_CMD=$(build_psql_cmd)

# 定义期望的外键约束
declare -A EXPECTED_CONSTRAINTS
# 格式: "表名.约束名"="期望的delete_rule"

# clusters 相关
EXPECTED_CONSTRAINTS["audit_logs.fk_audit_logs_cluster"]="SET NULL"
EXPECTED_CONSTRAINTS["node_anomalies.fk_node_anomalies_cluster"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_task_history.fk_ansible_task_history_cluster"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_schedules.fk_ansible_schedules_cluster"]="SET NULL"

# users 相关
EXPECTED_CONSTRAINTS["audit_logs.fk_audit_logs_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["clusters.fk_clusters_creator"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_tasks.fk_ansible_tasks_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_templates.fk_ansible_templates_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_inventories.fk_ansible_inventories_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_ssh_keys.fk_ansible_ssh_keys_creator"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_schedules.fk_ansible_schedules_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_favorites.fk_ansible_favorites_user"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_task_history.fk_ansible_task_history_user"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_tags.fk_ansible_tags_user"]="RESTRICT"
EXPECTED_CONSTRAINTS["feishu_user_mappings.fk_feishu_mappings_user"]="CASCADE"
EXPECTED_CONSTRAINTS["progress_tasks.fk_progress_tasks_user"]="CASCADE"
EXPECTED_CONSTRAINTS["progress_messages.fk_progress_messages_user"]="CASCADE"

# 模板/清单相关
EXPECTED_CONSTRAINTS["ansible_schedules.fk_ansible_schedules_template"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_task_history.fk_ansible_task_history_template"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_schedules.fk_ansible_schedules_inventory"]="RESTRICT"
EXPECTED_CONSTRAINTS["ansible_task_history.fk_ansible_task_history_inventory"]="SET NULL"

# SSH Keys 相关
EXPECTED_CONSTRAINTS["ansible_inventories.fk_ansible_inventories_ssh_key"]="RESTRICT"

# 多对多关联
EXPECTED_CONSTRAINTS["ansible_task_tags.fk_ansible_task_tags_task"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_task_tags.fk_ansible_task_tags_tag"]="CASCADE"

# 已在其他迁移中配置的约束（004, 019）
EXPECTED_CONSTRAINTS["ansible_tasks.fk_ansible_tasks_template"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_tasks.fk_ansible_tasks_cluster"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_tasks.fk_ansible_tasks_inventory"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_tasks.fk_ansible_tasks_workflow_execution"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_logs.fk_ansible_logs_task"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_inventories.fk_ansible_inventories_cluster"]="SET NULL"
EXPECTED_CONSTRAINTS["ansible_workflows.fk_ansible_workflows_user"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_workflow_executions.fk_ansible_workflow_executions_workflow"]="CASCADE"
EXPECTED_CONSTRAINTS["ansible_workflow_executions.fk_ansible_workflow_executions_user"]="CASCADE"

echo -e "${BLUE}检查外键约束...${NC}"
echo

total=0
passed=0
failed=0
missing=0

for key in "${!EXPECTED_CONSTRAINTS[@]}"; do
    table_name="${key%%.*}"
    constraint_name="${key##*.}"
    expected_rule="${EXPECTED_CONSTRAINTS[$key]}"
    
    total=$((total + 1))
    
    # 查询实际的 delete_rule
    actual_rule=$($PSQL_CMD -c "
        SELECT rc.delete_rule 
        FROM information_schema.referential_constraints rc
        JOIN information_schema.table_constraints tc 
            ON rc.constraint_name = tc.constraint_name
        WHERE tc.table_name = '$table_name' 
        AND tc.constraint_name = '$constraint_name';
    " | xargs)
    
    if [ -z "$actual_rule" ]; then
        # 检查表是否存在
        table_exists=$($PSQL_CMD -c "
            SELECT EXISTS (
                SELECT 1 FROM information_schema.tables 
                WHERE table_name = '$table_name'
            );
        " | xargs)
        
        if [ "$table_exists" = "t" ]; then
            echo -e "${YELLOW}⚠️  $table_name.$constraint_name${NC}"
            echo -e "   状态: ${YELLOW}约束不存在${NC}"
            echo -e "   期望: ${expected_rule}"
            missing=$((missing + 1))
        fi
    elif [ "$actual_rule" = "$expected_rule" ]; then
        echo -e "${GREEN}✓ $table_name.$constraint_name${NC}"
        echo -e "   规则: ${GREEN}$actual_rule${NC}"
        passed=$((passed + 1))
    else
        echo -e "${RED}✗ $table_name.$constraint_name${NC}"
        echo -e "   期望: ${expected_rule}"
        echo -e "   实际: ${RED}$actual_rule${NC}"
        failed=$((failed + 1))
    fi
    echo
done

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证摘要${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "总计: $total"
echo -e "${GREEN}通过: $passed${NC}"
if [ $failed -gt 0 ]; then
    echo -e "${RED}失败: $failed${NC}"
fi
if [ $missing -gt 0 ]; then
    echo -e "${YELLOW}缺失: $missing${NC}"
fi
echo

# 输出修复建议
if [ $failed -gt 0 ] || [ $missing -gt 0 ]; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}修复建议${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo
    echo -e "请执行以下命令应用数据库迁移:"
    echo -e "  ${GREEN}cd backend${NC}"
    echo -e "  ${GREEN}go run tools/migrate.go up${NC}"
    echo
    echo -e "或手动执行 SQL:"
    echo -e "  ${GREEN}psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f backend/migrations/021_fix_all_foreign_keys.sql${NC}"
    echo
    exit 1
else
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ 所有外键约束配置正确！${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
fi

