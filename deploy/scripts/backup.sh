#!/bin/bash

# 数据备份脚本
set -e

BACKUP_DIR="backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="kube-node-manager_${DATE}"

echo "🗄️  开始备份 Kubernetes 节点管理器数据..."

# 创建备份目录
mkdir -p ${BACKUP_DIR}

# 备份数据库
echo "📂 备份数据库..."
if [ -f "data/kube-node-manager.db" ]; then
    cp data/kube-node-manager.db ${BACKUP_DIR}/${BACKUP_NAME}.db
    echo "✅ 数据库备份完成: ${BACKUP_DIR}/${BACKUP_NAME}.db"
else
    echo "⚠️  数据库文件不存在，跳过"
fi

# 备份配置文件
echo "⚙️  备份配置文件..."
tar -czf ${BACKUP_DIR}/${BACKUP_NAME}_configs.tar.gz \
    .env \
    configs/ \
    deploy/docker/ \
    2>/dev/null || echo "⚠️  部分配置文件可能不存在"

echo "✅ 配置文件备份完成: ${BACKUP_DIR}/${BACKUP_NAME}_configs.tar.gz"

# 显示备份信息
echo ""
echo "✅ 备份完成！"
echo "📍 备份文件位置:"
echo "   数据库: ${BACKUP_DIR}/${BACKUP_NAME}.db"
echo "   配置:   ${BACKUP_DIR}/${BACKUP_NAME}_configs.tar.gz"
echo ""
echo "💡 恢复方法:"
echo "   cp ${BACKUP_DIR}/${BACKUP_NAME}.db data/kube-node-manager.db"
echo "   tar -xzf ${BACKUP_DIR}/${BACKUP_NAME}_configs.tar.gz"