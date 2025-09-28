#!/bin/bash

# SQLite to PostgreSQL Migration Example
# 本脚本展示了如何使用迁移工具的各种方式

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}SQLite to PostgreSQL Migration Examples${NC}"
echo "=============================================="
echo

echo -e "${YELLOW}示例 1: 使用配置文件 (推荐方式)${NC}"
echo "1. 复制配置文件模板："
echo "   cp migration.yaml.example migration.yaml"
echo
echo "2. 编辑配置文件 migration.yaml："
cat << 'EOF'
sqlite:
  path: "./backend/data/kube-node-manager.db"

postgresql:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "your-password"
  database: "kube_node_manager"
  ssl_mode: "disable"
EOF
echo
echo "3. 执行迁移："
echo "   ./migrate.sh -c migration.yaml"
echo
echo "=============================================="
echo

echo -e "${YELLOW}示例 2: 使用命令行参数${NC}"
echo "./migrate.sh \\"
echo "  -s ./backend/data/kube-node-manager.db \\"
echo "  -h localhost \\"
echo "  -p 5432 \\"
echo "  -d kube_node_manager \\"
echo "  -u postgres \\"
echo "  -w your-password"
echo
echo "=============================================="
echo

echo -e "${YELLOW}示例 3: 使用环境变量${NC}"
echo "# 设置环境变量"
echo "export MIGRATION_SQLITE_PATH=\"./backend/data/kube-node-manager.db\""
echo "export MIGRATION_POSTGRESQL_HOST=\"localhost\""
echo "export MIGRATION_POSTGRESQL_PORT=\"5432\""
echo "export MIGRATION_POSTGRESQL_USERNAME=\"postgres\""
echo "export MIGRATION_POSTGRESQL_PASSWORD=\"your-password\""
echo "export MIGRATION_POSTGRESQL_DATABASE=\"kube_node_manager\""
echo "export MIGRATION_POSTGRESQL_SSL_MODE=\"disable\""
echo
echo "# 执行迁移"
echo "./migrate.sh"
echo
echo "=============================================="
echo

echo -e "${YELLOW}示例 4: Docker 环境下的 PostgreSQL${NC}"
echo "# 启动 PostgreSQL 容器"
echo "docker run --name postgres-migration \\"
echo "  -e POSTGRES_PASSWORD=mypassword \\"
echo "  -e POSTGRES_DB=kube_node_manager \\"
echo "  -p 5432:5432 \\"
echo "  -d postgres:13"
echo
echo "# 等待数据库启动"
echo "sleep 10"
echo
echo "# 执行迁移"
echo "./migrate.sh \\"
echo "  -s ./backend/data/kube-node-manager.db \\"
echo "  -h localhost \\"
echo "  -p 5432 \\"
echo "  -d kube_node_manager \\"
echo "  -u postgres \\"
echo "  -w mypassword"
echo
echo "=============================================="
echo

echo -e "${YELLOW}示例 5: 远程 PostgreSQL 服务器${NC}"
echo "./migrate.sh \\"
echo "  -s ./backend/data/kube-node-manager.db \\"
echo "  -h db.example.com \\"
echo "  -p 5432 \\"
echo "  -d kube_node_manager \\"
echo "  -u myuser \\"
echo "  -w mypassword \\"
echo "  --ssl-mode require"
echo
echo "=============================================="
echo

echo -e "${YELLOW}示例 6: 完整的迁移流程${NC}"
echo "# 1. 停止应用服务"
echo "systemctl stop kube-node-manager"
echo
echo "# 2. 备份当前数据"
echo "cp ./backend/data/kube-node-manager.db ./backup/sqlite-backup-\$(date +%Y%m%d).db"
echo
echo "# 3. 准备 PostgreSQL 数据库"
echo "createdb -h localhost -U postgres kube_node_manager"
echo
echo "# 4. 执行迁移"
echo "./migrate.sh -c migration.yaml"
echo
echo "# 5. 更新应用配置"
echo "# 编辑 backend/configs/config.yaml，设置 database.type = \"postgres\""
echo
echo "# 6. 启动应用服务"
echo "systemctl start kube-node-manager"
echo
echo "# 7. 验证迁移结果"
echo "curl http://localhost:8080/api/v1/health"
echo
echo "=============================================="
echo

echo -e "${GREEN}提示:${NC}"
echo "1. 迁移前请停止应用服务以确保数据一致性"
echo "2. 建议在低峰期执行迁移操作"
echo "3. 迁移工具会自动备份 SQLite 数据库"
echo "4. 如有问题，可查看详细文档: README_MIGRATION.md"
echo

echo -e "${BLUE}快速测试命令:${NC}"
echo "./migrate.sh --help"
