# 部署指南

## 整体架构

```
用户浏览器
    ↓
Nginx (80/443)
    ├── /               → 前端静态文件（编辑器）
    ├── /s/*             → 分享页面（后端渲染）
    ├── /api/*           → API 接口
    └── /list            → 分享管理页
            ↓
    Docker: Go 后端 (127.0.0.1:3000)
            ↓
    SQLite: shares.db
```

前端是纯静态文件（HTML/JS/CSS），不需要构建。后端是 Go 服务，负责分享功能的数据存储和页面渲染。如果不使用分享功能，只需部署前端即可。仓库内自带 `docker-compose.yml`，部署脚本会自动同步并在服务器执行。

## 服务器要求

- Linux 服务器（Ubuntu 20.04+ / Debian 11+ / CentOS 8+）
- 至少 1GB 内存
- Docker + Docker Compose
- Nginx
- 一个域名（可选，用于 HTTPS）

## 部署步骤

### 第 1 步：配置 DNS

将你的域名 A 记录指向服务器 IP：

```
类型    名称          值
A      editor       你的服务器IP
```

例如你要用 `md.example.com` 访问编辑器，就添加一条 `md` 的 A 记录。

等 DNS 生效后验证：

```bash
ping md.example.com
# 应该返回你的服务器 IP
```

### 第 2 步：准备服务器

```bash
# SSH 登录服务器
ssh root@你的服务器IP

# 安装 Docker
curl -fsSL https://get.docker.com | sh
systemctl enable docker && systemctl start docker

# 安装 Nginx
apt install -y nginx

# 安装 certbot（用于 SSL 证书）
apt install -y certbot python3-certbot-nginx
```

### 第 3 步：配置 SSH 免密登录（推荐）

在你的本地电脑上操作，这样 deploy.sh 可以自动部署：

```bash
# 生成密钥（如果还没有的话）
ssh-keygen -t ed25519

# 复制公钥到服务器
ssh-copy-id root@你的服务器IP

# 测试免密登录
ssh root@你的服务器IP
# 应该不需要输入密码就能登录
```

然后在 `~/.ssh/config` 中添加别名：

```
Host my-editor-server
    HostName 你的服务器IP
    User root
    IdentityFile ~/.ssh/id_ed25519
```

测试：`ssh my-editor-server` 能直接登录即可。

### 第 4 步：克隆项目并配置

```bash
# 在本地电脑操作
git clone https://github.com/foolgry/editor.git
cd editor

# 复制配置模板
cp .env.example .env
```

编辑 `.env` 文件，填入你的实际配置：

```env
# ========== 部署配置 ==========

# SSH 别名（对应 ~/.ssh/config 中的 Host）或直接用服务器 IP
REMOTE_HOST=my-editor-server

# 服务器上的部署目录，前端和后端代码会同步到这里
REMOTE_DIR=/opt/md-editor

# 你的域名（不带 https://）
PUBLIC_DOMAIN=md.example.com

# 完整访问地址（带 https://）
PUBLIC_URL=https://md.example.com

# ========== 后端配置 ==========

# 分享管理列表的访问密码，自己设一个强密码
# 访问 /list 时需要输入这个密码
LIST_PAGE_PASSWORD=your-secure-password

# 允许跨域请求的来源地址，逗号分隔
# 必须包含你的前端域名，否则分享功能无法使用
CORS_ORIGINS=https://md.example.com,http://localhost:8080

# ========== Nginx 配置 ==========

# SSL 证书路径（Let's Encrypt 默认路径格式）
# 第 6 步申请证书后会自动创建
SSL_CERT_PATH=/etc/letsencrypt/live/md.example.com
```

每个变量说明：

| 变量 | 必填 | 说明 |
|------|------|------|
| `REMOTE_HOST` | 是 | deploy.sh 通过 SSH 连接服务器用的地址。可以是 SSH config 中的别名，也可以是 `root@1.2.3.4` |
| `REMOTE_DIR` | 是 | 服务器上存放项目的目录，不需要提前创建 |
| `PUBLIC_DOMAIN` | 是 | 你的域名，Nginx 配置和 SSL 证书都会用到 |
| `PUBLIC_URL` | 是 | 带协议的完整地址，用于部署后验证 |
| `LIST_PAGE_PASSWORD` | 是 | 后端启动必需。用于保护 `/list` 和删除接口 |
| `CORS_ORIGINS` | 是 | 跨域白名单，需要包含前端访问地址。本地开发加上 `http://localhost:8080` |
| `SSL_CERT_PATH` | 否 | HTTPS 证书路径。不用 HTTPS 则不需要 |

### 第 5 步：申请 SSL 证书

在服务器上操作：

```bash
# 确保域名 DNS 已经生效
ping md.example.com

# 申请 Let's Encrypt 免费证书
# --nginx 会自动配置 Nginx，但我们只需要证书，手动配置 Nginx
certbot certonly --nginx -d md.example.com

# 证书会保存在 /etc/letsencrypt/live/md.example.com/
# 自动续期已内置，无需额外配置
```

如果暂时不需要 HTTPS，可以跳过这步，并将：

- `SSL_CERT_PATH` 留空
- `PUBLIC_URL` 改成 `http://你的域名`

`deploy.sh nginx` 会自动生成纯 HTTP 的 Nginx 配置（不做 80→443 跳转）。

### 第 6 步：一键部署

回到本地电脑：

```bash
chmod +x deploy.sh

# 完整部署（前端 + 后端 + Nginx 配置）
./deploy.sh

# 也可以只部署某个部分
./deploy.sh frontend   # 仅前端
./deploy.sh backend    # 仅后端
./deploy.sh nginx      # 仅 Nginx 配置
./deploy.sh verify     # 验证部署状态
./deploy.sh backup     # 备份服务器代码
```

部署完成后访问你配置的 `PUBLIC_URL`，应该能看到编辑器页面。

### 第 7 步：验证

```bash
# 检查前端
curl -I http://md.example.com
# 应返回 200

# 检查后端
curl -I http://md.example.com/api/share
# 应返回 405（Method Not Allowed，说明后端在运行）

# 检查分享页面
curl -I http://md.example.com/s/test
# 应返回 404（说明路由正常）
```

如果你启用了 HTTPS，把上面 `http://` 改成 `https://`。

## 不用分享功能

如果只用前端编辑器，不需要后端：

1. 把前端文件放到任何 HTTP 服务器即可（Nginx、GitHub Pages、Vercel 等）
2. 不需要 `.env`、Docker、Go 后端
3. 编辑器所有功能（编辑、预览、复制、图片处理）都是纯前端，离线可用

## 不用 Nginx（纯 Docker）

如果不想配置 Nginx，可以直接用 Docker 运行：

```bash
# 本地运行
./start.sh

# 或用 Docker（需要同时挂载 frontend 目录）
cd server
docker build -t md-editor .
docker run -d -p 8080:8080 \
  --env-file ../.env \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/../frontend:/app/frontend:ro \
  md-editor

# 访问 http://localhost:8080
```

Go 服务同时提供前端页面和后端 API，不需要额外启动其他服务。

## Nginx 模板说明

`nginx/md-editor.conf` 是模板文件，使用 `{{占位符}}` 标记需要替换的值：

- `{{PUBLIC_DOMAIN}}` → 你的域名
- `{{SSL_CERT_PATH}}` → SSL 证书路径
- `{{REMOTE_DIR}}` → 部署目录

deploy.sh 部署时会自动从 `.env` 读取并替换。如果 `SSL_CERT_PATH` 为空，deploy.sh 会生成纯 HTTP 配置；如果不为空，会使用该模板生成 HTTPS 配置。

## 常见问题

### 后端启动失败

```bash
# 查看容器日志
ssh 你的服务器 'docker compose -f /opt/md-editor/docker-compose.yml logs'

# 检查端口占用
ssh 你的服务器 'ss -tlnp | grep 3000'

# 检查 .env 是否已传到服务器
ssh 你的服务器 'cat /opt/md-editor/.env'
```

### 前端无法使用分享功能

1. 检查 Nginx 是否正确代理了 `/api/` 和 `/s/` 路径
2. 检查 `CORS_ORIGINS` 是否包含前端域名
3. 打开浏览器开发者工具 → Network 查看请求状态

### SSL 证书问题

```bash
# 检查证书是否过期
ssh 你的服务器 'certbot certificates'

# 手动续期
ssh 你的服务器 'certbot renew'

# 检查 Nginx 配置中的证书路径是否正确
ssh 你的服务器 'nginx -t'
```
