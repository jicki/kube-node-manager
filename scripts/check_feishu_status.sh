#!/bin/bash

# 飞书长连接状态检查脚本

echo "=== 飞书长连接状态检查 ==="
echo ""

# 1. 检查数据库配置
echo "1. 检查数据库配置："
echo "-------------------"
sqlite3 data/kube-node-manager.db "SELECT id, enabled, app_id, bot_enabled, created_at FROM feishu_settings;" 2>/dev/null || \
mysql -u root -p -e "SELECT id, enabled, app_id, bot_enabled, created_at FROM feishu_settings;" 2>/dev/null || \
psql -U postgres -d kube_node_manager -c "SELECT id, enabled, app_id, bot_enabled, created_at FROM feishu_settings;" 2>/dev/null

echo ""

# 2. 检查进程
echo "2. 检查后端进程："
echo "-------------------"
ps aux | grep kube-node-manager | grep -v grep

echo ""

# 3. 检查最近的日志
echo "3. 检查最近的日志（飞书相关）："
echo "-------------------"
if [ -f "logs/app.log" ]; then
    echo "从 logs/app.log 中查找："
    tail -100 logs/app.log | grep -i feishu
elif [ -f "/var/log/kube-node-manager/app.log" ]; then
    echo "从 /var/log/kube-node-manager/app.log 中查找："
    tail -100 /var/log/kube-node-manager/app.log | grep -i feishu
else
    echo "未找到日志文件，尝试从 journalctl 查找："
    journalctl -u kube-node-manager -n 100 | grep -i feishu
fi

echo ""

# 4. 测试飞书 API 连接
echo "4. 测试飞书 API 连接："
echo "-------------------"
if command -v curl &> /dev/null; then
    curl -I -s --connect-timeout 5 https://open.feishu.cn | head -1
else
    echo "curl 未安装，跳过网络测试"
fi

echo ""
echo "=== 检查完成 ==="
echo ""
echo "📝 诊断建议："
echo "1. 如果配置为空或 bot_enabled=0，请在前端页面保存配置"
echo "2. 如果进程不存在，请启动后端服务"
echo "3. 如果日志中有错误，请根据错误信息排查"
echo "4. 如果 API 连接失败，检查网络和防火墙设置"
echo "5. 修改配置后，建议重启后端服务"

