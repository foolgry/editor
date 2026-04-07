# @foolgry/wxmd-cli

微信公众号 Markdown 编辑器 CLI - Agent-First 命令行工具。

将 Markdown 转换为微信公众号可用的 HTML，支持 18 种精美样式主题。

渲染内核与首页编辑器预览、分享页预览统一，减少本地与线上预览差异。

[![npm version](https://badge.fury.io/js/@foolgry%2Fwxmd-cli.svg)](https://badge.fury.io/js/@foolgry%2Fwxmd-cli)

## 安装

```bash
# 全局安装（推荐）
npm install -g @foolgry/wxmd-cli

# 使用 pnpm
pnpm add -g @foolgry/wxmd-cli

# 使用 yarn
yarn global add @foolgry/wxmd-cli
```

## 快速开始

```bash
# 排版 Markdown 文件（输出 JSON）
wxmd-cli typeset --input article.md --style wechat-tech

# 推荐：提取 HTML 并写入文件（Agent/脚本友好）
wxmd-cli typeset --input article.md --style wechat-default | jq -r '.data' > article.html

# 使用 --out 参数输出到文件
wxmd-cli typeset --input article.md --style wechat-elegant --out article.html

# 从 stdin 读取
echo "# Hello World" | wxmd-cli typeset --style wechat-elegant

# 查看所有可用样式
wxmd-cli styles list

# 检查环境
wxmd-cli doctor
```

## 命令

### `typeset` - Markdown 排版

将 Markdown 转换为样式化的 HTML。

```bash
wxmd-cli typeset --input <file> --style <style> [options]

选项：
  -i, --input <file>    输入文件（默认从 stdin 读取）
  -s, --style <name>    样式主题（默认: wechat-default）
  -o, --output <format> 输出格式: json, html, text（默认: json）
      --out <file>      输出到文件（默认输出到 stdout）
```

**示例：**

```bash
# 基础用法（输出 JSON）
wxmd-cli typeset --input article.md

# 指定样式
wxmd-cli typeset --input article.md --style wechat-tech

# 推荐：使用 jq 提取 HTML 并重定向到文件（Agent/CI 友好）
wxmd-cli typeset --input article.md --style wechat-elegant | jq -r '.data' > article.html

# 或使用 --out 参数输出到文件
wxmd-cli typeset --input article.md --style wechat-elegant --out article.html

# 管道输入并提取 HTML
cat article.md | wxmd-cli typeset --style wechat-nyt | jq -r '.data'

# 查看完整 JSON 响应
wxmd-cli typeset --input article.md --style wechat-tech | jq .
```

### `format` - 自动修复空格和标点

使用 AutoCorrect 自动修复 Markdown 中的 CJK（中日韩）空格和标点问题。

```bash
wxmd-cli format --input <file> [options]

选项：
  -i, --input <file>    输入文件（默认从 stdin 读取）
      --out <file>      输出到文件（默认输出到 stdout）
```

**功能说明：**
- 自动在中英文之间添加空格（如 `你好World` → `你好 World`）
- 纠正标点符号（中文内容使用全角标点）
- 修复全宽字符和半宽字符问题

**示例：**

```bash
# 修复文件并输出到新文件
wxmd-cli format --input article.md --out fixed.md

# 从 stdin 读取并输出到 stdout
echo "hello世界，你好World" | wxmd-cli format
# 输出: hello 世界，你好 World

# JSON 格式输出（包含修复统计）
wxmd-cli format --input article.md --output json
# 输出: { "data": "...", "meta": { "changes": 5, "inputSize": 100, "outputSize": 105 } }
```

**使用场景：**
- 批量修复现有 Markdown 文件的空格问题
- 配合 `typeset` 命令使用：先修复，再排版
- CI/CD 流水线中文案规范化检查

### `share create` - 创建分享

创建在线分享链接（需要后端服务器）。

```bash
wxmd-cli share create --input <file> --style <style>
```

**示例：**

```bash
wxmd-cli share create --input article.md --style wechat-default
# 输出: { "id": "abc123", "url": "http://...", "createdAt": "..." }
```

### `share get` - 获取分享

根据 ID 获取分享内容。

```bash
wxmd-cli share get <id>
```

### `styles list` - 列出样式

查看所有可用的 18 种排版样式。

```bash
wxmd-cli styles list
# 输出样式列表: wechat-default, wechat-tech, wechat-elegant, wechat-nyt, ...
```

### `doctor` - 环境检查

检查本地环境和 API 连接状态。

```bash
wxmd-cli doctor
```

## 支持的样式主题

| 样式键名 | 名称 | 特点 |
|---------|------|------|
| `wechat-default` | 默认公众号风格 | 经典微信风格，清晰易读 |
| `wechat-tech` | 技术风格 | 代码友好，蓝色主题 |
| `wechat-elegant` | 优雅简约 | 宋体，文艺气息 |
| `wechat-nyt` | 纽约时报 | 报纸排版风格 |
| `wechat-apple` | Apple 极简 | 苹果设计美学 |
| `latepost-depth` | 晚点风格 | 深红色调，深度报道 |
| `wechat-ft` | 金融时报 | 米黄色背景，财经风格 |
| `wechat-anthropic` | Claude | 渐变色标题，现代感 |
| `wechat-jonyive` | Jony Ive | 苹果前设计师风格 |
| `wechat-deepread` | 深度阅读 | GitHub 风格，技术文档 |
| `wechat-medium` | Medium 长文 | 博客平台风格 |
| `kenya-emptiness` | 原研哉·空 | 留白设计，日式极简 |
| `hische-editorial` | Hische·编辑部 | 衬线字体，编辑风格 |
| `ando-concrete` | 安藤·清水 | 混凝土质感，建筑美学 |
| `gaudi-organic` | 高迪·有机 | 彩色曲线，有机设计 |
| `guardian` | Guardian 卫报 | 媒体网站风格 |
| `nikkei` | Nikkei 日经 | 日本经济新闻风格 |
| `lemonde` | Le Monde 世界报 | 法国报纸风格 |

## 输出格式

### JSON 格式（默认）

```json
{
  "ok": true,
  "data": "<html>...</html>",
  "meta": {
    "cliVersion": "1.0.0",
    "timestamp": "2026-04-06T10:30:00Z"
  }
}
```

### HTML 格式

直接使用 `--output html` 获取纯 HTML：

```bash
wxmd-cli typeset --input article.md --output html
```

### 错误格式

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_STYLE",
    "type": "INVALID_ARGUMENTS",
    "message": "Style 'xxx' not found",
    "retryable": false,
    "actionHint": "Run 'wxmd-cli styles list' to see available styles"
  },
  "meta": { ... }
}
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `WXMD_API_URL` | `http://localhost:8080` | API 服务器地址（仅 share 命令需要） |
| `WXMD_API_TIMEOUT` | `30000` | 请求超时时间（毫秒） |

## 退出码

| 码 | 含义 |
|----|------|
| 0 | 成功 |
| 1 | 通用错误 |
| 2 | 参数错误 |
| 3 | 网络错误 |
| 4 | 服务器错误 |
| 5 | 未找到 |
| 6 | 权限拒绝 |
| 7 | 超时 |

## 使用 jq 解析输出（推荐）

CLI 默认输出 JSON，配合 `jq` 工具可以灵活提取所需内容：

```bash
# 提取 HTML 内容到文件（最常用）
wxmd-cli typeset --input article.md --style wechat-tech | jq -r '.data' > output.html

# 提取元数据
wxmd-cli typeset --input article.md | jq -r '.meta'

# 检查是否成功
wxmd-cli typeset --input article.md | jq -r '.ok'

# 保存完整响应用于调试
wxmd-cli typeset --input article.md | jq . > result.json

# 批量处理多个文件
for file in *.md; do
  wxmd-cli typeset --input "$file" | jq -r '.data' > "${file%.md}.html"
done
```

**提示：** `jq -r` 表示 raw 输出，去除 JSON 字符串的引号，适合直接写入文件。

## 面向 Agent 设计

本 CLI 专为 AI Agent 设计：

- **结构化 JSON 输出** - 便于程序解析
- **明确的错误类型** - `INVALID_ARGUMENTS`, `NETWORK_ERROR`, `NOT_FOUND` 等
- **可恢复错误提示** - 每个错误都包含 `actionHint` 建议
- **固定退出码** - 便于自动化脚本处理
- **支持 stdin/stdout** - 适合管道操作

## 渲染一致性回归测试

用于防止首页编辑器、CLI、分享页三端渲染漂移：

```bash
# 运行一致性测试（含快照校验）
pnpm --dir wxmd-cli test

# 当你有意修改渲染行为时，更新快照
UPDATE_SNAPSHOT=1 pnpm --dir wxmd-cli test
```

## 相关链接

- **完整项目**: https://github.com/foolgry/editor
- **在线使用**: https://md.foolgry.top
- **NPM 包**: https://www.npmjs.com/package/@foolgry/wxmd-cli
- **Issue 反馈**: https://github.com/foolgry/editor/issues

## 许可证

MIT License - 详见 [LICENSE](../LICENSE)
