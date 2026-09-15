# 公众号 Markdown 编辑器启动与管理

# 默认列出可用命令
default:
    @just --list

# 启动本地服务
start:
    cd server && go run .

# 启动开发服务器（别名）
dev:
    cd server && go run .

# 整理后端 Go 依赖
tidy:
    cd server && go mod tidy

# 验证前端脚本语法
lint:
    node -c frontend/styles.js
    node -c frontend/app.js
    node -c frontend/render-core.js
    node -c frontend/outline.js
