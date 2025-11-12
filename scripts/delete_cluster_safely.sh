#!/bin/bash

# 安全删除集群脚本（处理外键约束）
# 解决: ERROR: update or delete on table "clusters" violates foreign key constraint

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
${BLUE}安全删除集群脚本${NC}
用于删除被外键约束保护的集群记录

${YELLOW}使用方法:${NC}
    $0 <集群ID或集群名称> [选项]

${YELLOW}选项:${NC}
    -h, --host HOST         数据库主机 (默认: localhost)
    -p, --port PORT         数据库端口 (默认: 5432)
    -d, --database NAME     数据库名称 (默认: kube_node_manager)
    -u, --user USER         数据库用户 (默认: postgres)
    -w, --password PASS     数据库密码
    --preview               仅预览，不执行删除
    --keep-audit            保留审计日志（推荐）
    --help                  显示帮助信息

${YELLOW}示例:${NC}
    # 通过 ID 删除集群
    $0 7

    # 通过名称删除集群
    $0 my-cluster

    # 预览将要删除的内容
    $0 7 --preview

    # 保留审计日志
    $0 7 --keep-audit

    # 自定义数据库连接
    $0 7 -h db.example.com -u admin -w secret123

${YELLOW}环境变量:${NC}
    DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD

${YELLOW}注意事项:${NC}
    - 默认会保留审计日志（解除关联但不删除）
    - 节点异常记录会被删除
    - Ansible 清单会解除集群关联
    - 所有操作在事务中执行，失败会自动回滚
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

# 通过名称或 ID 查找集群
find_cluster() {
    local identifier=$1
    local PSQL_CMD=$(build_psql_cmd)
    
    # 尝试作为 ID 查询
    if [[ "$identifier" =~ ^[0-9]+$ ]]; then
        local result=$($PSQL_CMD -c "SELECT id, name FROM clusters WHERE id = $identifier;")
        if [ -n "$result" ]; then
            echo "$result"
            return 0
        fi
    fi
    
    # 作为名称查询
    local result=$($PSQL_CMD -c "SELECT id, name FROM clusters WHERE name = '$identifier';")
    if [ -n "$result" ]; then
        echo "$result"
        return 0
    fi
    
    return 1
}

# 预览要删除的内容
preview_deletion() {
    local cluster_id=$1
    local cluster_name=$2
    local PSQL_CMD=$(build_psql_cmd)
    
    echo -e "${BLUE}===== 删除预览 =====${NC}"
    echo -e "${YELLOW}集群信息:${NC}"
    echo "  ID: $cluster_id"
    echo "  名称: $cluster_name"
    echo
    
    echo -e "${YELLOW}相关记录统计:${NC}"
    
    # 审计日志
    local audit_count=$($PSQL_CMD -c "SELECT COUNT(*) FROM audit_logs WHERE cluster_id = $cluster_id;" | xargs)
    echo "  审计日志: $audit_count 条"
    
    # 节点异常
    local anomaly_count=$($PSQL_CMD -c "SELECT COUNT(*) FROM node_anomalies WHERE cluster_id = $cluster_id;" | xargs)
    echo "  节点异常: $anomaly_count 条"
    
    # Ansible 清单
    local inventory_count=$($PSQL_CMD -c "SELECT COUNT(*) FROM ansible_inventories WHERE cluster_id = $cluster_id;" | xargs)
    echo "  Ansible 清单: $inventory_count 个"
    
    echo
    echo -e "${YELLOW}将执行的操作:${NC}"
    if [ "$KEEP_AUDIT" = true ]; then
        echo "  ✓ 审计日志: 解除关联（保留记录）"
    else
        echo "  ✗ 审计日志: 删除"
    fi
    echo "  ✗ 节点异常: 删除"
    echo "  ✓ Ansible 清单: 解除关联（保留记录）"
    echo "  ✗ 集群记录: 删除"
}

# 执行删除
execute_deletion() {
    local cluster_id=$1
    local cluster_name=$2
    local PSQL_CMD=$(build_psql_cmd)
    
    echo -e "${BLUE}===== 开始删除操作 =====${NC}"
    
    # 生成 SQL
    local sql="BEGIN;"
    
    if [ "$KEEP_AUDIT" = true ]; then
        sql+="
UPDATE audit_logs SET cluster_id = NULL WHERE cluster_id = $cluster_id;
"
        echo -e "${GREEN}✓ 准备解除审计日志关联${NC}"
    else
        sql+="
DELETE FROM audit_logs WHERE cluster_id = $cluster_id;
"
        echo -e "${GREEN}✓ 准备删除审计日志${NC}"
    fi
    
    sql+="
DELETE FROM node_anomalies WHERE cluster_id = $cluster_id;
"
    echo -e "${GREEN}✓ 准备删除节点异常记录${NC}"
    
    sql+="
UPDATE ansible_inventories SET cluster_id = NULL WHERE cluster_id = $cluster_id;
"
    echo -e "${GREEN}✓ 准备解除 Ansible 清单关联${NC}"
    
    sql+="
DELETE FROM clusters WHERE id = $cluster_id;
COMMIT;
"
    echo -e "${GREEN}✓ 准备删除集群记录${NC}"
    
    echo
    echo -e "${YELLOW}执行删除...${NC}"
    
    # 执行 SQL
    if echo "$sql" | $PSQL_CMD > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 集群 '$cluster_name' (ID: $cluster_id) 删除成功${NC}"
        echo
        echo -e "${BLUE}验证删除结果:${NC}"
        local remaining=$($PSQL_CMD -c "SELECT COUNT(*) FROM clusters WHERE id = $cluster_id;" | xargs)
        if [ "$remaining" = "0" ]; then
            echo -e "${GREEN}✓ 集群记录已完全删除${NC}"
        else
            echo -e "${RED}✗ 删除可能未完成，请检查${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ 删除失败，事务已回滚${NC}"
        return 1
    fi
}

# 主函数
main() {
    local cluster_identifier=""
    local PREVIEW_ONLY=false
    local KEEP_AUDIT=true  # 默认保留审计日志
    
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
            --preview)
                PREVIEW_ONLY=true
                shift
                ;;
            --keep-audit)
                KEEP_AUDIT=true
                shift
                ;;
            --help)
                usage
                ;;
            *)
                if [ -z "$cluster_identifier" ]; then
                    cluster_identifier="$1"
                fi
                shift
                ;;
        esac
    done
    
    # 检查参数
    if [ -z "$cluster_identifier" ]; then
        echo -e "${RED}❌ 错误: 请指定集群 ID 或名称${NC}"
        echo
        usage
    fi
    
    # 检查 psql
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}❌ 错误: 未找到 psql 命令${NC}"
        echo -e "${YELLOW}请安装 PostgreSQL 客户端${NC}"
        exit 1
    fi
    
    # 查找集群
    echo -e "${BLUE}查找集群: $cluster_identifier${NC}"
    cluster_info=$(find_cluster "$cluster_identifier")
    
    if [ -z "$cluster_info" ]; then
        echo -e "${RED}❌ 未找到集群: $cluster_identifier${NC}"
        exit 1
    fi
    
    # 解析集群信息
    cluster_id=$(echo "$cluster_info" | awk '{print $1}' | xargs)
    cluster_name=$(echo "$cluster_info" | awk '{print $2}' | xargs)
    
    echo -e "${GREEN}✓ 找到集群: $cluster_name (ID: $cluster_id)${NC}"
    echo
    
    # 预览
    preview_deletion "$cluster_id" "$cluster_name"
    
    if [ "$PREVIEW_ONLY" = true ]; then
        echo
        echo -e "${YELLOW}这是预览模式，没有执行任何删除操作${NC}"
        echo -e "要执行删除，请运行: $0 $cluster_identifier"
        exit 0
    fi
    
    # 确认
    echo
    echo -e "${RED}警告: 此操作将永久删除集群记录！${NC}"
    read -p "确认删除吗？输入集群名称以确认: " confirm
    
    if [ "$confirm" != "$cluster_name" ]; then
        echo -e "${RED}❌ 名称不匹配，操作已取消${NC}"
        exit 1
    fi
    
    # 执行删除
    echo
    execute_deletion "$cluster_id" "$cluster_name"
}

# 运行主函数
main "$@"

