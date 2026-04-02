#!/bin/bash

# ============================================
# 公众号 Markdown 编辑器 - 一键部署脚本
# ============================================

set -e  # 遇到错误立即退出

# 从 .env 文件加载配置
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
else
    echo "错误: 未找到 .env 配置文件"
    echo "请复制 .env.example 为 .env 并填写你的配置"
    exit 1
fi

LOCAL_DIR="$SCRIPT_DIR"

# 检查关键环境变量
require_env() {
    local var_name="$1"
    local value="${!var_name:-}"
    if [ -z "$value" ]; then
        print_error "环境变量 $var_name 未设置，请检查 .env"
        exit 1
    fi
}

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_title() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

require_env "REMOTE_HOST"
require_env "REMOTE_DIR"
require_env "PUBLIC_URL"

# 检查 SSH 连接
check_connection() {
    print_info "检查 SSH 连接..."
    if ! ssh -q -o BatchMode=yes -o ConnectTimeout=5 "$REMOTE_HOST" exit 2>/dev/null; then
        print_error "无法连接到服务器 $REMOTE_HOST"
        print_info "请确保:"
        print_info "  1. SSH 服务正在运行"
        print_info "  2. 已配置 SSH 密钥"
        print_info "  3. ~/.ssh/config 中配置了 $REMOTE_HOST 别名"
        exit 1
    fi
    print_success "SSH 连接正常"
}

# 备份远程代码
backup_remote() {
    print_title "1. 备份远程代码"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_DIR="$REMOTE_DIR/backup/$TIMESTAMP"

    ssh "$REMOTE_HOST" "
        mkdir -p $BACKUP_DIR
        if [ -d $REMOTE_DIR/frontend ]; then
            cp -r $REMOTE_DIR/frontend $BACKUP_DIR/
            echo '前端代码已备份'
        fi
        if [ -d $REMOTE_DIR/server ]; then
            cp -r $REMOTE_DIR/server $BACKUP_DIR/
            echo '后端代码已备份'
        fi
        # 备份 Nginx 配置
        if [ -f /etc/nginx/sites-available/md-editor ]; then
            cp /etc/nginx/sites-available/md-editor $BACKUP_DIR/nginx.conf
            echo 'Nginx 配置已备份'
        fi
        echo '$BACKUP_DIR'
    "

    print_success "备份完成: $BACKUP_DIR"
}

# 部署前端代码
deploy_frontend() {
    print_title "2. 部署前端代码"

    print_info "同步前端文件到服务器..."

    # 使用 rsync 同步前端文件
    rsync -avz --progress \
        --exclude='.DS_Store' \
        "$LOCAL_DIR/frontend/" "$REMOTE_HOST:$REMOTE_DIR/frontend/"

    # 设置权限
    ssh "$REMOTE_HOST" "
        chown -R www-data:www-data $REMOTE_DIR/frontend/ 2>/dev/null || chown -R root:root $REMOTE_DIR/frontend/
        chmod -R 644 $REMOTE_DIR/frontend/*.html $REMOTE_DIR/frontend/*.js $REMOTE_DIR/frontend/*.css 2>/dev/null || true
    "

    print_success "前端部署完成"
}

# 部署后端代码
deploy_backend() {
    print_title "3. 部署后端代码"

    # 检查本地是否有 server 目录
    if [ ! -d "$LOCAL_DIR/server" ]; then
        print_warning "本地 server 目录不存在，跳过后端部署"
        return
    fi

    if [ ! -f "$LOCAL_DIR/docker-compose.yml" ]; then
        print_error "缺少 docker-compose.yml，无法部署后端"
        exit 1
    fi

    # 确保远端目录存在
    ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR $REMOTE_DIR/server $REMOTE_DIR/frontend"

    print_info "同步后端文件到服务器..."

    # 同步 server 目录（保留 data 目录）
    rsync -avz --progress \
        --exclude='data' \
        --exclude='.git' \
        "$LOCAL_DIR/server/" "$REMOTE_HOST:$REMOTE_DIR/server/"

    # 同步前端静态文件到 frontend 目录
    rsync -avz --progress \
        --exclude='.DS_Store' \
        "$LOCAL_DIR/frontend/" "$REMOTE_HOST:$REMOTE_DIR/frontend/"

    # 同步 .env 文件到服务器（后端通过 env_file 加载）
    scp "$LOCAL_DIR/.env" "$REMOTE_HOST:$REMOTE_DIR/.env"

    # 同步 Docker Compose 配置
    rsync -avz --progress \
        "$LOCAL_DIR/docker-compose.yml" "$REMOTE_HOST:$REMOTE_DIR/docker-compose.yml"

    print_success "后端文件同步完成"

    # 重新构建并启动容器
    print_info "重新构建 Docker 容器..."
    ssh "$REMOTE_HOST" "
        set -e
        cd $REMOTE_DIR

        # 兼容 docker compose / docker-compose
        if docker compose version >/dev/null 2>&1; then
            DC='docker compose'
        elif command -v docker-compose >/dev/null 2>&1; then
            DC='docker-compose'
        else
            echo '错误: 未找到 docker compose 或 docker-compose'
            exit 1
        fi

        # 停止旧容器
        \$DC down 2>/dev/null || true

        # 重新构建并启动（通过 env_file 加载 .env）
        \$DC up -d --build 2>&1 | tail -20

        # 等待服务启动
        sleep 3

        # 检查状态
        \$DC ps
    "

    print_success "后端部署完成"
}

# 部署 Nginx 配置
deploy_nginx() {
    print_title "部署 Nginx 配置"

    require_env "PUBLIC_DOMAIN"

    # 检查本地是否有 nginx 配置文件
    if [ ! -f "$LOCAL_DIR/nginx/md-editor.conf" ]; then
        print_warning "本地 nginx/md-editor.conf 不存在，跳过 Nginx 部署"
        return
    fi

    print_info "从模板生成 Nginx 配置..."

    rendered_conf="$(mktemp /tmp/md-editor.conf.XXXXXX)"

    # 当 SSL_CERT_PATH 为空时，生成纯 HTTP 配置
    if [ -z "${SSL_CERT_PATH:-}" ]; then
        print_warning "SSL_CERT_PATH 为空，将生成纯 HTTP 配置（不启用 HTTPS）"
        cat > "$rendered_conf" <<EOF
server {
    listen 80;
    server_name ${PUBLIC_DOMAIN};

    location / {
        root ${REMOTE_DIR}/frontend;
        index index.html;
        try_files \$uri \$uri/ =404;
    }

    location ~ ^/(s|api)/ {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location = /list {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    else
        # 从模板生成 HTTPS 配置（替换占位符）
        sed -e "s|{{PUBLIC_DOMAIN}}|${PUBLIC_DOMAIN}|g" \
            -e "s|{{SSL_CERT_PATH}}|${SSL_CERT_PATH}|g" \
            -e "s|{{REMOTE_DIR}}|${REMOTE_DIR}|g" \
            "$LOCAL_DIR/nginx/md-editor.conf" > "$rendered_conf"
    fi

    print_info "同步 Nginx 配置到服务器..."

    # 同步配置
    scp "$rendered_conf" "$REMOTE_HOST:/tmp/md-editor.conf.new"

    # 测试并重载
    ssh "$REMOTE_HOST" "
        set -e
        BACKUP_FILE='/etc/nginx/sites-available/md-editor.backup.\$(date +%Y%m%d_%H%M%S)'

        # 备份当前配置
        if [ -f /etc/nginx/sites-available/md-editor ]; then
            cp /etc/nginx/sites-available/md-editor \$BACKUP_FILE
        fi

        # 应用新配置
        mv /tmp/md-editor.conf.new /etc/nginx/sites-available/md-editor

        # 测试配置语法
        if nginx -t >/tmp/nginx-test.log 2>&1; then
            nginx -s reload
            echo 'Nginx 配置已更新并重载'
        else
            echo 'Nginx 配置语法错误，正在回滚'
            if [ -f \$BACKUP_FILE ]; then
                mv \$BACKUP_FILE /etc/nginx/sites-available/md-editor
                nginx -t >/dev/null 2>&1 && nginx -s reload || true
            else
                rm -f /etc/nginx/sites-available/md-editor
            fi
            cat /tmp/nginx-test.log
            exit 1
        fi
    "

    # 清理临时文件
    rm -f "$rendered_conf"

    print_success "Nginx 配置部署完成"
}

# 验证部署
verify_deployment() {
    print_title "4. 验证部署"

    print_info "检查服务状态..."

    # 检查后端服务
    if ssh "$REMOTE_HOST" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/api/share/test" | grep -q "404"; then
        print_success "后端服务运行正常"
    else
        print_error "后端服务可能异常"
        ssh "$REMOTE_HOST" "
            cd $REMOTE_DIR
            if docker compose version >/dev/null 2>&1; then
                docker compose logs --tail=10
            elif command -v docker-compose >/dev/null 2>&1; then
                docker-compose logs --tail=10
            else
                echo '未找到 docker compose 命令'
            fi
        "
    fi

    # 检查前端访问
    if curl -s -o /dev/null -w "%{http_code}" "${PUBLIC_URL}" | grep -q "200"; then
        print_success "前端访问正常 (${PUBLIC_URL})"
    else
        print_error "前端访问异常"
    fi

    print_success "部署验证完成"
}

# 显示使用信息
show_usage() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  all       部署前端+后端+Nginx（默认）"
    echo "  frontend  仅部署前端"
    echo "  backend   仅部署后端"
    echo "  nginx     仅部署 Nginx 配置"
    echo "  verify    仅验证部署状态"
    echo "  backup    仅备份远程代码"
    echo "  help      显示此帮助"
    echo ""
    echo "示例:"
    echo "  $0              # 完整部署"
    echo "  $0 frontend     # 仅更新前端"
    echo "  $0 backend      # 仅更新后端"
    echo "  $0 nginx        # 仅更新 Nginx 配置"
}

# 主函数
main() {
    print_title "公众号 Markdown 编辑器 - 部署脚本"

    # 检查参数
    case "${1:-all}" in
        all)
            check_connection
            backup_remote
            deploy_frontend
            deploy_backend
            deploy_nginx
            verify_deployment
            ;;
        frontend)
            check_connection
            backup_remote
            deploy_frontend
            verify_deployment
            ;;
        backend)
            check_connection
            backup_remote
            deploy_backend
            verify_deployment
            ;;
        nginx)
            check_connection
            deploy_nginx
            ;;
        verify)
            check_connection
            verify_deployment
            ;;
        backup)
            check_connection
            backup_remote
            ;;
        help|--help|-h)
            show_usage
            exit 0
            ;;
        *)
            print_error "未知参数: $1"
            show_usage
            exit 1
            ;;
    esac

    print_title "部署完成！"
    echo "访问地址："
    echo "  - 主页: ${PUBLIC_URL}"
    echo ""
    echo "管理命令："
    echo "  ssh $REMOTE_HOST 'cd $REMOTE_DIR && docker compose logs -f'"
}

# 执行主函数
main "$@"
