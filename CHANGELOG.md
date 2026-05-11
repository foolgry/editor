# Changelog

本文件汇总近期重要变更，便于对照发布说明与功能演进。完整功能说明与使用方式见 [README.md](README.md)。以下内容依据**最近 20 条** Git 提交整理，并与 README 中的能力描述对齐。

## wxmd-cli（`@foolgry/wxmd-cli`）

### 1.1.1 — 2026-04-30

- 依赖与版本维护：将 wxmd-cli 升级至 **1.1.1**（与 README 中 Agent CLI / NPM 包版本一致）。

### 1.1.0 — 2026-04-07

- **新增 `format` 命令**：集成 [@huacnlee/autocorrect](https://github.com/huacnlee/autocorrect)，对 Markdown 做中英文间距、标点等自动修正（与 README「自动修复空格和标点」一致）。
- **注册与文档**：在主程序中注册 `format`；补充 `format` 用法说明（对应 `wxmd-cli/README.md`）。
- **稳定性**：通过纯 JS 实现、动态 ESM 引入、切换 esm.sh CDN、缓存版本 bump、重试加载等，避免 WASM 加载失败导致的排版修复不可用问题。

## Web 编辑器与分享服务

### 2026-05-11

- **Markdown 宽表格**：用表格容器包裹宽表，支持横向滚动，窄屏下可左右滑动查看（与 README「响应式设计」场景一致）。

### 2026-04-30

- **Kami 主题**：新增 **Kami** 样式，并同步前端与 CLI 文档（README / wxmd-cli 主题表中均已列出）。

### 2026-04-27

- **分享按钮**：重命名分享相关类名，降低浏览器扩展误拦截的概率。

### 2026-04-09 ~ 2026-04-12

- **布局与滚动**：工具栏移至顶部；编辑器与预览区**双向滚动同步**；滚动条显示策略优化（编辑区隐藏、预览区按需显示等）。
- **分享体验**：分享页统一使用 `renderCore.renderMarkdown` 渲染；本地图片转为 Data URL；修正 highlight.js CDN；多项分享渲染与滚动行为改进。
- **站点图标**：使用 SVG favicon（替换错误的 JPG），并同步前端与分享页引用。
- **样式**：新增 **Claude Song** 主题（README「20 种样式主题」之一）。
- **主题管理**：设置弹窗内支持主题**拖拽排序**、**显示/隐藏**（配置存 `localStorage`）；右键菜单快速收藏/隐藏；优化设置 UI；修复分享 `shareServerUrl` 缺失与新增主题时排序校验等问题。
- **文案**：移除「Created by 花生」类署名；主页标题调整为「内容排版及分享」。

### 2026-04-07

- **文档**：补充 README 中对新增功能的描述（与本仓库当前 README 结构一致）。

---

### 说明

- **CLI 渲染一致性**：README 写明 `wxmd-cli typeset`、首页预览与 `/s/:id` 分享页共用 `frontend/render-core.js`；变更记录中的主题与渲染相关项均受此内核影响。
- **分享与数据**：分享依赖后端 Go + SQLite（`server/data/shares.db`），详见 README「分享功能」章节。
