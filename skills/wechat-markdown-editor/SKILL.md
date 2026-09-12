---
name: wechat-markdown-editor
description: 公众号 Markdown 编辑器线上发布技能。当用户需要把 Markdown 文章发布为排版精美的微信公众号格式、生成可分享的在线链接时使用。纯线上模式：通过 HTTP 请求把内容写入线上编辑器（默认 https://md.foolgry.top），返回分享 URL，本地图片自动上传。支持纯文本直接发送和本地文件（含本地图片）发布。
---

# 公众号 Markdown 编辑器 - 线上发布

## 概述

把 Markdown 内容发布到线上公众号排版编辑器（`https://md.foolgry.top`），立即获得一个排版好的分享链接。不做任何本地渲染，排版由线上页面完成。

核心脚本：`scripts/publish.py`（Python 3 标准库实现，**零依赖**，无需安装任何东西）。

## 使用方式

本技能的脚本路径为 `<skill目录>/scripts/publish.py`，以下统一用 `publish.py` 指代。

### 1. 发布本地 Markdown 文件（最常用）

```bash
# 默认使用 kami-slides 样式（无需传 --style）
python3 publish.py publish --file article.md

# 或指定其他样式
python3 publish.py publish --file article.md --style latepost-depth
```

- 文件中的**本地图片会自动上传到服务器**，引用自动改写为线上 URL，无需手动处理。
- 图片相对路径按 Markdown 文件所在目录解析；已是 http(s):// 或 data: 的图片原样保留。
- 输出 JSON：`{"id", "url", "style", "uploadedImages"}`，其中 `url` 即分享链接。

### 2. 发布纯文本内容

```bash
# 默认使用 kami-slides 样式
python3 publish.py publish --text "# 标题\n正文内容"

# 或管道输入并指定样式
cat article.md | python3 publish.py publish --style wechat-tech
```

### 3. 发布后用浏览器打开

```bash
python3 publish.py publish --file article.md --open
```

### 4. 其他子命令

```bash
# 列出可用样式
python3 publish.py styles

# 获取分享内容
python3 publish.py get <id>

# 列出全部分享（需要管理密码）
WXMD_LIST_PASSWORD=xxx python3 publish.py list

# 删除分享（需要管理密码）
WXMD_LIST_PASSWORD=xxx python3 publish.py delete <id>
```

## 排版样式（--style）

| key | 名称 |
|---|---|
| `wechat-default` | 默认公众号风格 |
| `latepost-depth` | 晚点风格 |
| `wechat-ft` | 金融时报 |
| `wechat-anthropic` | Claude |
| `wechat-claude-song` | Claude Song |
| `wechat-tech` | 技术风格 |
| `wechat-elegant` | 优雅简约 |
| `wechat-deepread` | 深度阅读 |
| `wechat-nyt` | 纽约时报 |
| `wechat-jonyive` | Jony Ive |
| `wechat-medium` | Medium 长文 |
| `wechat-apple` | Apple 极简 |
| `kenya-emptiness` | 原研哉·空 |
| `hische-editorial` | Hische·编辑部 |
| `ando-concrete` | 安藤·清水 |
| `gaudi-organic` | 高迪·有机 |
| `kami-resume` | Kami · 简历 |
| `kami-print` | Kami · 白底单页 |
| `kami-report` | Kami · 财报研报 |
| `kami-slides` | Kami · 演讲文稿 |
| `kami-product` | Kami · 产品简报 |
| `kami-letter` | Kami · 正式信函 |
| `kami-changelog` | Kami · 更新日志 |
| `kami-portfolio` | Kami · 沉静画册 |
| `guardian` | Guardian 卫报 |
| `nikkei` | Nikkei 日経 |
| `lemonde` | Le Monde 世界报 |

用户未指定时默认使用 `kami-slides`（Kami · 演讲文稿）；若用户明确指定了样式（或提出了特定的排版风格要求），则使用用户指定的样式。发布后用户也可在线上编辑器切换样式预览。

## 环境变量

| 变量 | 说明 | 默认值 |
|---|---|---|
| `WXMD_API_URL` | API 地址 | `https://md.foolgry.top` |
| `WXMD_API_TIMEOUT` | 请求超时（秒） | `30` |
| `WXMD_LIST_PASSWORD` | 列表/删除的管理密码 | 无 |

## 图片处理说明

- 支持 png / jpg / jpeg / gif / webp / svg，单张 ≤10MB。
- 本地图片通过 `POST /api/upload` 上传到服务器，文件存储在服务端 `data/uploads/`，经 `/uploads/<文件>` 访问。
- 同一张图片在同一篇文档中多次引用只上传一次。
- 某张图上传失败不会中断发布：原引用保留，并在输出 JSON 的 `warnings` 中列出。

## 故障排查

- **无法连接服务器**：检查网络和 `WXMD_API_URL`；`curl -s -o /dev/null -w '%{http_code}' https://md.foolgry.top` 应返回 200。
- **HTTP 401**：list/delete 需要正确的 `WXMD_LIST_PASSWORD`。
- **HTTP 400 不支持的图片格式**：图片扩展名需在支持列表内。
- **HTTP 413**：图片超过 nginx 限制（20MB），先压缩图片再发布。
- **发布后图片不显示**：查看输出 JSON 的 `warnings`，确认图片是否上传成功。
