#!/bin/bash

# 公众号排版器后端服务启动脚本

echo "🚀 正在启动公众号排版器后端服务..."

# 检查是否安装了 Go
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 请先安装 Go 语言环境"
    echo "下载地址: https://go.dev/dl/"
    exit 1
fi

# 检查 go.mod 是否存在
if [ ! -f "go.mod" ]; then
    echo "📦 初始化 Go 模块..."
    go mod init huasheng-editor-server 2>/dev/null || true
fi

# 下载依赖
echo "📥 下载依赖..."
go mod tidy

# 创建数据目录
mkdir -p data

# 获取本机 IP
IP=$(hostname -I | awk '{print $1}')
if [ -z "$IP" ]; then
    IP="localhost"
fi

echo ""
echo "=========================================="
echo "📝 服务信息:"
echo "   本地访问: http://localhost:8080"
echo "   局域网访问: http://${IP}:8080"
echo "=========================================="
echo ""

# 启动服务
go run main.go
