# 公众号排版器 - 后端服务

基于 Go + SQLite 的分享服务后端。

## 功能特性

- ✅ 创建分享链接
- ✅ SQLite 数据持久化
- ✅ 自动生成分享页面
- ✅ 跨域支持（CORS）
- ✅ 简洁的 REST API

## 快速开始

### 1. 环境要求

- Go 1.21 或更高版本

### 2. 安装依赖

```bash
cd server
go mod tidy
```

### 3. 启动服务

```bash
# 使用启动脚本
./start.sh

# 或直接运行
go run main.go
```

服务默认监听 `8080` 端口。

### 4. 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | `8080` |

## API 接口

### 创建分享

```http
POST /api/share
Content-Type: application/json

{
  "content": "# Markdown 内容",
  "style": "wechat-default"
}
```

响应：

```json
{
  "id": "abc12345",
  "content": "# Markdown 内容",
  "style": "wechat-default",
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

### 获取分享

```http
GET /api/share/{id}
```

响应：

```json
{
  "id": "abc12345",
  "content": "# Markdown 内容",
  "style": "wechat-default",
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

### 查看分享页面

浏览器访问：

```
GET /s/{id}
```

返回渲染好的 HTML 页面，包含：
- 应用主题的 Markdown 渲染内容
- 样式切换信息
- 复制内容功能
- 返回编辑器链接

## 数据存储

数据存储在 `./data/shares.db` 的 SQLite 数据库中。

表结构：

```sql
CREATE TABLE shares (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    style TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 项目结构

```
server/
├── main.go          # 主程序
├── go.mod           # Go 模块配置
├── go.sum           # 依赖校验
├── start.sh         # 启动脚本
├── README.md        # 说明文档
└── data/            # 数据目录
    └── shares.db    # SQLite 数据库
```

## 构建部署

### 本地构建

```bash
# 构建可执行文件
go build -o huasheng-server

# 运行
./huasheng-server
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
RUN mkdir -p data
EXPOSE 8080
CMD ["./server"]
```

### 前端配置

在前端 `app.js` 中配置后端地址：

```javascript
data() {
  return {
    // ...
    shareServerUrl: 'http://your-server:8080'
  }
}
```

## 注意事项

1. **数据持久化**：SQLite 数据库存储在 `./data/shares.db`，请确保该目录有写入权限
2. **跨域设置**：默认允许所有来源，生产环境建议配置具体域名
3. **图片处理**：分享页面只包含 Markdown 内容，本地图片需要额外处理
4. **备份建议**：定期备份 `data/shares.db` 文件

## 开发计划

- [ ] 分享链接有效期设置
- [ ] 分享内容加密
- [ ] 访问统计
- [ ] 分享内容管理后台
