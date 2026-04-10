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
> - 增加中英文数字空格自动修复功能
> - 重构了整个项目结构

## 功能

### 纯前端功能（无需后端）

- 19 种样式主题（公众号、杂志、纽约时报、金融时报、Apple 极简、Claude、Claude Song 等）
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

详见 [DEPLOY.md](docs/DEPLOY.md)。核心步骤：

1. 复制 `.env.example` 为 `.env` 并填写配置
2. `./deploy.sh` 一键部署

仓库内已包含 `docker-compose.yml`，`deploy.sh` 会同步并在服务器执行 `docker compose up -d --build`。

## 技术栈

- Vue 3 + Markdown-it + Highlight.js + Mermaid
- IndexedDB（图片存储）+ Canvas API（图片压缩）+ Turndown（智能粘贴）
- Go + SQLite（后端分享服务）
- 纯 CSS，无需构建工具
- Node.js CLI（Agent-First 命令行工具）

## Agent CLI

专为 AI Agent 设计的命令行工具，支持本地排版 Markdown 和调用分享 API。

> 渲染一致性说明：`CLI typeset`、首页编辑器预览、`/s/:id` 分享页已统一复用 `frontend/render-core.js`。

[![NPM](https://img.shields.io/badge/NPM-@foolgry/wxmd--cli-CB3837?style=for-the-badge&logo=npm)](https://www.npmjs.com/package/@foolgry/wxmd-cli)

### 安装 Skill（推荐）

让 AI Agent 帮你一键安装：
```txt
请按照 https://github.com/foolgry/editor/blob/master/docs/INSTALL.md 文档帮我安装skills
```

命令行安装
```bash
npx skills add foolgry/editor -g --all
```

或手动安装到 Claude Code：

```bash
# 克隆仓库
git clone https://github.com/foolgry/editor.git /tmp/editor

# 复制 Skill 到 Claude Code 技能目录
cp -r /tmp/editor/skills/wechat-markdown-editor ~/.claude/skills/
```

### 安装 CLI 工具（可选）

> **注意**：一般情况下无需手动安装 CLI 工具，使用 Skill 时会自动通过 npx 运行。
> 
> 如需全局安装：

```bash
# 全局安装
npm install -g @foolgry/wxmd-cli

# 或使用 pnpm
pnpm add -g @foolgry/wxmd-cli
```

### 快速使用

```bash
# Markdown 排版（本地执行）
wxmd-cli typeset --input article.md --style wechat-tech

# 自动修复空格和标点
echo "hello世界" | wxmd-cli format
wxmd-cli format --input article.md --out fixed.md

# 创建分享（需服务器运行）
wxmd-cli share create --input article.md --style wechat-default

# 获取分享内容
wxmd-cli share get <share-id>

# 列出可用样式（19种主题）
wxmd-cli styles list

# 环境检查
wxmd-cli doctor
```

### 从源码安装（开发）

```bash
cd wxmd-cli
pnpm install
./src/index.js --help
```

### 环境变量

- `WXMD_API_URL` - API 服务器地址（默认：`http://localhost:8080`）
- `WXMD_API_TIMEOUT` - 请求超时（毫秒，默认：30000）

### 输出格式

默认 JSON 输出，面向 Agent 设计：

```json
{
  "ok": true,
  "data": { "html": "...", "file": "output.html" },
  "meta": { "cliVersion": "1.0.0", "timestamp": "2026-04-06T10:30:00Z" }
}
```

### 相关文档

- **CLI 详细文档**: [wxmd-cli/README.md](wxmd-cli/README.md)
- **Agent Skill 安装指南**: [docs/INSTALL.md](docs/INSTALL.md)
- **Skill 使用手册**: [skills/wechat-markdown-editor/SKILL.md](skills/wechat-markdown-editor/SKILL.md)

## 开源协议

基于 [MIT License](LICENSE) 开源，原始项目作者：[花生](https://github.com/alchaincyf)。
