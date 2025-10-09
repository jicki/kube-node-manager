#!/bin/bash
#
# GitLab 分页功能测试脚本
# 用于验证 GitLab Runners 和 Pipelines 的分页修复是否生效
#
# 使用方法:
#   ./scripts/test-gitlab-pagination.sh <API_BASE_URL> <TOKEN>
#
# 示例:
#   ./scripts/test-gitlab-pagination.sh http://localhost:8080 your-token-here
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查参数
if [ $# -lt 2 ]; then
    echo -e "${RED}错误: 缺少必需参数${NC}"
    echo "使用方法: $0 <API_BASE_URL> <TOKEN>"
    echo "示例: $0 http://localhost:8080 your-token-here"
    exit 1
fi

API_BASE_URL="$1"
TOKEN="$2"

# 去除末尾的斜杠
API_BASE_URL="${API_BASE_URL%/}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}GitLab 分页功能测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "API 地址: $API_BASE_URL"
echo "Token: ${TOKEN:0:10}..."
echo ""

# 测试函数
test_api() {
    local endpoint=$1
    local description=$2
    
    echo -e "${YELLOW}测试: $description${NC}"
    echo "端点: $endpoint"
    
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "${API_BASE_URL}${endpoint}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" == "200" ]; then
        # 计算返回的数据数量
        count=$(echo "$body" | jq 'length' 2>/dev/null || echo "无法解析")
        
        echo -e "${GREEN}✓ 成功${NC}"
        echo "HTTP 状态码: $http_code"
        echo "返回数据数量: $count"
        
        # 显示前3条数据的ID
        if [ "$count" != "无法解析" ] && [ "$count" -gt 0 ]; then
            echo "前3条数据ID:"
            echo "$body" | jq -r '.[0:3] | .[] | .id' 2>/dev/null | sed 's/^/  - /'
        fi
    else
        echo -e "${RED}✗ 失败${NC}"
        echo "HTTP 状态码: $http_code"
        echo "错误信息:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    fi
    
    echo ""
    return 0
}

# 1. 测试 Runners API
echo -e "${BLUE}----------------------------------------${NC}"
echo -e "${BLUE}1. 测试 GitLab Runners API${NC}"
echo -e "${BLUE}----------------------------------------${NC}"
test_api "/api/v1/gitlab/runners" "获取所有 Runners"

# 2. 测试 Runners API 带筛选条件
echo -e "${BLUE}----------------------------------------${NC}"
echo -e "${BLUE}2. 测试 Runners API (在线状态)${NC}"
echo -e "${BLUE}----------------------------------------${NC}"
test_api "/api/v1/gitlab/runners?status=online" "获取在线 Runners"

# 3. 测试 Runners API 带类型筛选
echo -e "${BLUE}----------------------------------------${NC}"
echo -e "${BLUE}3. 测试 Runners API (Instance 类型)${NC}"
echo -e "${BLUE}----------------------------------------${NC}"
test_api "/api/v1/gitlab/runners?type=instance_type" "获取 Instance 类型 Runners"

# 4. 检查后端日志（如果是 Docker 环境）
echo -e "${BLUE}----------------------------------------${NC}"
echo -e "${BLUE}4. 检查后端日志${NC}"
echo -e "${BLUE}----------------------------------------${NC}"

if command -v docker &> /dev/null; then
    echo "尝试从 Docker 容器获取日志..."
    
    # 查找包含 kube-node-manager 的后端容器
    container_name=$(docker ps --format '{{.Names}}' | grep -i 'backend\|kube-node-manager' | head -n 1)
    
    if [ -n "$container_name" ]; then
        echo -e "${GREEN}找到容器: $container_name${NC}"
        echo "最近的 GitLab 相关日志:"
        docker logs "$container_name" 2>&1 | grep -i "fetched total" | tail -n 5 || echo "没有找到相关日志"
    else
        echo -e "${YELLOW}未找到后端容器，跳过日志检查${NC}"
    fi
elif command -v kubectl &> /dev/null; then
    echo "尝试从 Kubernetes Pod 获取日志..."
    
    # 查找 kube-node-manager pod
    pod_name=$(kubectl get pods -l app=kube-node-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -n "$pod_name" ]; then
        echo -e "${GREEN}找到 Pod: $pod_name${NC}"
        echo "最近的 GitLab 相关日志:"
        kubectl logs "$pod_name" | grep -i "fetched total" | tail -n 5 || echo "没有找到相关日志"
    else
        echo -e "${YELLOW}未找到 kube-node-manager Pod，跳过日志检查${NC}"
    fi
else
    echo -e "${YELLOW}Docker 和 kubectl 都未安装，无法检查日志${NC}"
    echo "请手动检查后端日志，查找类似以下的输出:"
    echo '  INFO: Fetched total XXX runners from GitLab'
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试完成${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}验证要点:${NC}"
echo "1. 检查 Runners 数量是否与 GitLab 后台一致（应为 171）"
echo "2. 确认后端日志中有 'Fetched total XXX runners' 的记录"
echo "3. 验证筛选功能是否正常工作"
echo "4. 确认前端页面显示的数量与 API 返回一致"
echo ""
echo -e "${GREEN}建议:${NC}"
echo "- 在浏览器中打开前端页面，手动验证 Runners 列表"
echo "- 使用不同的筛选条件测试功能完整性"
echo "- 观察页面加载时间，确保性能可接受"
echo ""

