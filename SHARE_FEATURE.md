# 分享功能使用指南

## 功能简介

新增分享功能，可以将当前编辑的文章生成一个分享链接，分享给他人查看。

## 特点

- 🔗 生成短链接（8位随机字符）
- 🎨 保留当前选择的主题样式
- 📱 分享页面支持移动端访问
- 💾 数据存储在服务端 SQLite 数据库
- 🔒 链接长期有效

## 快速开始

### 1. 启动后端服务

```bash
cd server

# 安装依赖
go mod tidy

# 启动服务
./start.sh
# 或
go run main.go
```

服务启动后，会显示访问地址：
- 本地访问: http://localhost:8080
- 局域网访问: http://你的IP:8080

### 2. 配置前端

默认配置为 `http://localhost:8080`，如需修改，编辑 `app.js`：

```javascript
data() {
  return {
    // ...
    shareServerUrl: 'http://your-server:8080'
  }
}
```

### 3. 使用分享功能

1. 编辑 Markdown 内容
2. 选择喜欢的主题样式
3. 点击预览面板右上角的 **"分享"** 按钮
4. 等待链接生成
5. 复制链接并发送给朋友

## 技术架构

### 后端（Go）

```
server/
├── main.go          # 主程序
├── go.mod           # 模块配置
├── start.sh         # 启动脚本
├── Dockerfile       # Docker 配置
└── data/
    └── shares.db    # SQLite 数据库
```

### API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/share` | POST | 创建分享 |
| `/api/share/{id}` | GET | 获取分享内容 |
| `/api/share/{id}` | DELETE | 删除分享（需 Header 密码） |
| `/api/shares` | GET | 获取全部分享列表（需 Header 密码） |
| `/s/{id}` | GET | 分享页面 |
| `/list` | GET | 分享管理列表页面（输入密码后查看全部） |

列表接口鉴权 Header：

```http
X-List-Password: XOu6rt5uK9BIX
```

### 数据存储

```sql
CREATE TABLE shares (
    id TEXT PRIMARY KEY,      -- 分享ID (8位随机字符)
    content TEXT NOT NULL,     -- Markdown内容
    style TEXT NOT NULL,       -- 主题样式
    created_at DATETIME,       -- 创建时间
    updated_at DATETIME        -- 更新时间
);
```

## 部署方式

### 方式一：直接部署

```bash
cd server
go build -o huasheng-server
./huasheng-server
```

### 方式二：Docker 部署

```bash
cd server
docker build -t huasheng-editor-server .
docker run -d -p 8080:8080 -v $(pwd)/data:/app/data huasheng-editor-server
```

### 方式三：使用 Docker Compose

```yaml
version: '3'
services:
  server:
    build: ./server
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

## 分享页面

分享页面包含：
- 渲染后的 Markdown 内容（应用选中主题）
- 主题标识
- 复制内容按钮
- 返回编辑器链接
- 自适应移动端

访问地址：`http://your-server:8080/s/{share-id}`

## 注意事项

1. **图片处理**：分享页面只包含 Markdown 文本内容，本地图片（img:// 协议）不会显示
2. **数据备份**：定期备份 `server/data/shares.db` 文件
3. **跨域设置**：默认允许所有来源，生产环境建议限制域名
4. **端口占用**：确保 8080 端口未被占用

## 开发计划

- [ ] 分享链接有效期设置
- [ ] 分享内容加密
- [ ] 访问统计
- [ ] 分享内容管理后台
- [ ] 自定义分享页面标题

## 问题排查

### 后端启动失败

```bash
# 检查端口占用
lsof -i :8080

# 检查权限
ls -la server/data/

# 查看日志
cd server && go run main.go
```

### 前端无法连接后端

1. 检查后端服务是否运行
2. 检查防火墙设置
3. 确认 `shareServerUrl` 配置正确
4. 浏览器控制台查看网络请求

### 分享页面样式异常

1. 确认 styles.js 文件可正常访问
2. 检查浏览器控制台是否有 404 错误
3. 刷新页面重试
