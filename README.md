# 公众号 Markdown 编辑器

<div align="center">
  <img src="frontend/logo.svg" width="120" height="120" alt="公众号 Markdown 编辑器">

  一个专为微信公众号设计的 Markdown 编辑器

  [![在线体验](https://img.shields.io/badge/在线体验-md.foolgry.top-0066FF?style=for-the-badge)](https://md.foolgry.top/)
  [![GitHub](https://img.shields.io/badge/GitHub-源代码-000?style=for-the-badge&logo=github)](https://github.com/foolgry/editor)
</div>

> 本项目 Fork 自 [alchaincyf/huasheng_editor](https://github.com/alchaincyf/huasheng_editor)，在原项目基础上增加了以下功能：
> - 修复多项 Bug（列表渲染、零宽字符、模块加载等）
> - 增加 Mermaid 图表渲染支持
> - 增加分享功能（可将文章生成链接分享给他人查看）

## 功能

### 纯前端功能（无需后端）

- 13 种样式主题（公众号、杂志、纽约时报、金融时报、Apple 极简、Claude 等）
- 实时预览 + 一键复制到公众号
- 智能图片处理：粘贴/拖拽图片、自动压缩、IndexedDB 本地存储、复制时转 Base64
- 多图网格布局（类似朋友圈）
- 代码高亮（macOS 风格）
- Mermaid 图表渲染
- 智能粘贴（支持飞书、Notion、Word 等富文本）
- 样式收藏、.md 文件上传
- 响应式设计（桌面/平板/手机）

### 分享功能（需要后端）

- 将文章生成短链接，发送给他人查看
- 保留当前主题样式
- 分享管理列表（`/list`，需密码）
- **注意**：分享内容保存在服务器 SQLite 数据库中（`server/data/shares.db`）

## 快速开始

```bash
git clone https://github.com/foolgry/editor.git
cd editor

cp .env.example .env     # 编辑 .env 填写配置
./start.sh               # 启动（需要本地安装 Go 1.21+）

# 访问 http://localhost:8080
```

Go 服务同时提供前端页面和后端 API，一个进程就够了。

## 部署

详见 [DEPLOY.md](DEPLOY.md)。核心步骤：

1. 复制 `.env.example` 为 `.env` 并填写配置
2. `./deploy.sh` 一键部署

仓库内已包含 `docker-compose.yml`，`deploy.sh` 会同步并在服务器执行 `docker compose up -d --build`。

## 技术栈

- Vue 3 + Markdown-it + Highlight.js + Mermaid
- IndexedDB（图片存储）+ Canvas API（图片压缩）+ Turndown（智能粘贴）
- Go + SQLite（后端分享服务）
- 纯 CSS，无需构建工具

## 开源协议

基于 [MIT License](LICENSE) 开源，原始项目作者：[花生](https://github.com/alchaincyf)。
