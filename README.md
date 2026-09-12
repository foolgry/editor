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

- 27 种样式主题（公众号、杂志、纽约时报、金融时报、Apple 极简、Claude、Claude Song 以及 8 款 Kami 出版级主题：简历、白底单页、财报研报、演讲文稿、产品简报、正式信函、更新日志、沉静画册等）
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
- 图片上传接口（`/api/upload`），供 Skill 发布本地图片
- **注意**：分享内容保存在服务器 SQLite 数据库中（`server/data/shares.db`），上传的图片保存在 `server/data/uploads/`

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

## Agent Skill（线上发布）

专为 AI Agent 设计的发布技能：**纯线上模式**，通过 HTTP 请求把 Markdown 写入线上编辑器并返回分享链接，不做本地渲染。核心是一个零依赖的 Python 脚本（`skills/wechat-markdown-editor/scripts/publish.py`，仅用标准库）。

### 安装 Skill

让 AI Agent 帮你一键安装：
```txt
请按照 https://github.com/foolgry/editor/blob/master/docs/INSTALL.md 文档帮我安装skills
```

命令行安装
```bash
npx skills add foolgry/editor -g --all
```

或手动复制技能目录到 Agent 的技能目录（如 `~/.claude/skills/` 或 `~/.agents/skills/`）：

```bash
git clone https://github.com/foolgry/editor.git /tmp/editor
cp -r /tmp/editor/skills/wechat-markdown-editor ~/.agents/skills/
```

### 快速使用

```bash
# 发布本地 Markdown 文件（本地图片自动上传）
python3 skills/wechat-markdown-editor/scripts/publish.py publish --file article.md --style wechat-default

# 发布纯文本
python3 skills/wechat-markdown-editor/scripts/publish.py publish --text "# 标题"

# 发布后用浏览器打开
python3 skills/wechat-markdown-editor/scripts/publish.py publish --file article.md --open

# 获取/列出/删除分享（list/delete 需要 WXMD_LIST_PASSWORD）
python3 skills/wechat-markdown-editor/scripts/publish.py get <share-id>
```

输出 JSON：`{"id", "url", "style", "uploadedImages"}`，`url` 即分享链接。

### 环境变量

- `WXMD_API_URL` - API 服务器地址（默认：`https://md.foolgry.top`）
- `WXMD_API_TIMEOUT` - 请求超时（秒，默认：30）
- `WXMD_LIST_PASSWORD` - 列表/删除操作的管理密码

### 相关文档

- **变更记录**: [CHANGELOG.md](CHANGELOG.md)（依据近期提交整理，便于查阅版本与功能演进）
- **Agent Skill 安装指南**: [docs/INSTALL.md](docs/INSTALL.md)
- **Skill 使用手册**: [skills/wechat-markdown-editor/SKILL.md](skills/wechat-markdown-editor/SKILL.md)

## 开源协议

基于 [MIT License](LICENSE) 开源，原始项目作者：[花生](https://github.com/alchaincyf)。
