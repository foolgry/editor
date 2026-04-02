#!/bin/bash

# ============================================
# 公众号 Markdown 编辑器 - 本地启动脚本
# ============================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 加载 .env（如果存在）
if [ -f .env ]; then
    set -a
    source .env
    set +a
else
    echo "未找到 .env，使用默认配置"
    echo "如需自定义，请: cp .env.example .env"
fi

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "错误: 请先安装 Go 1.21+"
    echo "下载地址: https://go.dev/dl/"
    exit 1
fi

# 安装依赖
cd server
go mod tidy

# 创建数据目录
mkdir -p data

# 设置默认端口
export PORT=${PORT:-8080}
export LIST_PAGE_PASSWORD=${LIST_PAGE_PASSWORD:-local-dev-password}
export CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:${PORT},http://localhost:3000}

echo ""
echo "  公众号 Markdown 编辑器"
echo "  访问: http://localhost:${PORT}"
echo "  按 Ctrl+C 停止"
if [ "${LIST_PAGE_PASSWORD}" = "local-dev-password" ]; then
    echo "  提示: 当前使用默认 LIST_PAGE_PASSWORD=local-dev-password（仅建议本地开发）"
fi
echo ""

go run main.go
