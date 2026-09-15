---
name: wechat-markdown-editor
description: 公众号 Markdown 编辑器线上发布技能。当用户需要把 Markdown 文章发布为排版精美的微信公众号格式、生成可分享的在线链接时使用。纯线上模式：通过 HTTP 请求把内容写入线上编辑器（默认 https://md.foolgry.top），返回分享 URL，本地图片自动上传。发布需要令牌，令牌放在技能目录下的 .env 文件里（脚本自动读取，用户可手动编辑，agent 可用 set-token 子命令写入），无令牌时到 <API 地址>/apply 申请。支持纯文本直接发送和本地文件（含本地图片）发布；支持项目（Project）聚合多篇文章（日报、巡检报告等持续产出场景），对外只发一个项目链接。
---

# 公众号 Markdown 编辑器 - 线上发布

## 概述

把 Markdown 内容发布到线上公众号排版编辑器（`https://md.foolgry.top`），立即获得一个排版好的分享链接。不做任何本地渲染，排版由线上页面完成。

核心脚本：`scripts/publish.py`（Python 3 标准库实现，**零依赖**，无需安装任何东西）。

**发布需要令牌**：本站已关闭匿名发布，`publish`（含其中的图片上传）和所有项目操作都必须提供令牌。令牌**只需配置一次**——写在技能目录下的 `.env` 文件里，脚本每次运行自动读取；配置方法见下一节。未配置时脚本直接报错并给出申请入口，不会上传到一半才失败。

**项目（Project）**是分享的命名集合：把持续产出的文章（日报、服务器巡检报告等）挂到同一个项目名下，对外只发一个聚合链接 `/p/<pid>`（打开默认显示最新一篇，左侧目录可翻历史），单篇深链为 `/p/<pid>/<sid>`。项目按**名字**引用，不存在时自动创建（无人值守的定时任务友好）；一篇分享最多属于一个项目，也可以不属于任何项目。

## 令牌配置（首次使用必读）

首次使用只有一件事：把令牌写进 `<skill目录>/.env`。配好之后，下文所有命令都不需要再带任何令牌。

### 1. 申请令牌

打开 `<API 地址>/apply`（默认 <https://md.foolgry.top/apply>）提交申请，站长签发后会给你一串 `wmt_` 开头的令牌。

### 2. 写入 .env

`<skill目录>/.env` 就是放令牌的地方，脚本启动时自动读取它。该文件已被 `.gitignore` 忽略，不会进仓库。文件里已有带注释的模板，照着填即可。

**用户手动设置**：用编辑器打开 `<skill目录>/.env`，找到 `WXMD_TOKEN` 那一行，去掉行首的 `#` 并把令牌填进去：

```
WXMD_TOKEN=wmt_你的令牌
```

**Agent 代为设置**：执行下面任一命令，脚本会把令牌写进同一个 `.env`（并顺带把文件权限设为 600）：

```bash
# 直接把令牌作为参数
python3 scripts/publish.py set-token wmt_你的令牌

# 或从管道传入（令牌不会留在 shell 历史里，推荐）
echo 'wmt_你的令牌' | python3 scripts/publish.py set-token
```

`set-token` 是幂等的：重复执行会原地替换 `WXMD_TOKEN` 的值，不会堆出多行。

这个文件除了令牌还可以放 `WXMD_LIST_PASSWORD`（`list` / `delete` 用）和 `WXMD_API_URL`（自建服务时改地址），写法同上。

### 3. 确认配置生效

```bash
python3 scripts/publish.py token-status
```

输出会告诉你令牌来源（`.env` 还是环境变量）、掩码后的令牌值和令牌申请入口。`"ready": true` 即表示可以发布。**这是排查"令牌没生效"的第一手段**：若 `publishToken.source` 显示 `环境变量`，说明有个环境变量把 `.env` 顶掉了，改环境变量或先 `unset` 它。

### 4. 优先级与临时覆盖

**环境变量 > `.env`**。临时用另一个令牌发一篇，不必改文件：

```bash
WXMD_TOKEN=wmt_另一个令牌 python3 scripts/publish.py publish --file article.md
```

### 5. 令牌无效怎么办

返回 401 且提示"凭证无效或已被吊销"，说明令牌写错或已被站长吊销；重新申请后覆盖 `.env` 里的 `WXMD_TOKEN` 即可（重跑一次 `set-token` 即可）。

## 使用方式

本技能的脚本路径为 `<skill目录>/scripts/publish.py`，以下统一用 `publish.py` 指代。所有示例都假定令牌已在 `.env` 里配好（见上一节），因此命令里不再出现令牌；没配的话命令会直接失败并提示去哪申请。

### 1. 发布本地 Markdown 文件（最常用）

```bash
# 默认使用 kami-slides 样式（无需传 --style）
python3 publish.py publish --file article.md

# 或指定其他样式
python3 publish.py publish --file article.md --style latepost-depth
```

- 文件中的**本地图片会自动上传到服务器**，引用自动改写为线上 URL，无需手动处理；图片上传与发布共用同一个令牌。
- 图片相对路径按 Markdown 文件所在目录解析；已是 http(s):// 或 data: 的图片原样保留。
- 输出 JSON：`{"id", "url", "style", "uploadedImages"}`（另有 `projectId` 等项目字段，见第 4 节），其中 `url` 即分享链接。

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

### 4. 发布到项目（--project）

```bash
# 核心路径：发布并归入项目（项目不存在时自动创建）
python3 publish.py publish --file report.md --project 服务器巡检
```

- `--project` 按**名字**引用项目，不存在时自动创建，适合 Agent 定时任务无人值守发布。
- 同样需要令牌（未配置 `WXMD_TOKEN` 时回退 `WXMD_LIST_PASSWORD`，见配置项一节）。
- 输出 JSON 在原有字段上增加 `projectId` / `projectUrl` / `deepUrl`：

```json
{
  "id": "abc123",
  "url": "https://md.foolgry.top/s/abc123",
  "style": "kami-slides",
  "uploadedImages": 0,
  "projectId": "proj_x9",
  "projectUrl": "https://md.foolgry.top/p/proj_x9",
  "deepUrl": "https://md.foolgry.top/p/proj_x9/abc123"
}
```

- 未指定 `--project` 时这三个字段为 `null`（字段始终存在，向后兼容）。
- 一篇分享最多属于一个项目；发布后可随时用 `attach` / `detach` 调整归属。

### 5. 项目管理命令（需要令牌）

```bash
# 列出自己名下的项目
python3 publish.py projects

# 显式创建项目（发布带 --project 时通常无需手动创建）
python3 publish.py project-create 服务器巡检

# 重命名项目（警告：仍用旧名发布的定时任务之后会自动创建一个同名新项目，文章将分流）
python3 publish.py project-rename 旧名 新名
# 存在多个同名项目时改用项目 ID 消歧（ID 见 projects 输出）
python3 publish.py project-rename <project-id> 新名

# 把已有分享挂载/移动到项目（输出更新后的分享 JSON，含 deepUrl）
python3 publish.py attach <share-id> --project 服务器巡检
# 跨创建者同名项目时按 ID 精确挂载
python3 publish.py attach <share-id> --project-id <project-id>

# 把分享移出项目（变回独立单篇，/s/<id> 链接保持有效）
python3 publish.py detach <share-id>
```

### 6. 其他子命令

```bash
# 列出可用样式
python3 publish.py styles

# 获取分享内容
python3 publish.py get <id>

# 列出全部分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
python3 publish.py list

# 删除分享（需要 WXMD_LIST_PASSWORD 或 WXMD_TOKEN）
python3 publish.py delete <id>

# 写入令牌 / 查看当前生效的凭证
python3 publish.py set-token wmt_xxxx
python3 publish.py token-status
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

## 配置项（.env 或环境变量）

配置写在技能目录下的 `.env` 里（也可用同名环境变量覆盖）：

```
<skill目录>/.env
```

| 变量 | 说明 | 默认值 |
|---|---|---|
| `WXMD_API_URL` | API 地址 | `https://md.foolgry.top` |
| `WXMD_API_TIMEOUT` | 请求超时（秒） | `30` |
| `WXMD_TOKEN` | 发布令牌：`publish`（含图片上传）与全部项目操作都需要。从 `<API 地址>/apply` 申请 | 无 |
| `WXMD_LIST_PASSWORD` | 列表/删除的管理密码；发布与项目操作未设 `WXMD_TOKEN` 时回退用它（主密码视作站长凭证） | 无 |

**取值顺序：环境变量 > `.env`**。变量不存在或值为空都算未设置——`WXMD_TOKEN=` 空着不会挡住后面的回退。`.env` 缺失也不影响运行，只是凭证得靠环境变量提供。

`.env` 的写法是常见的 KEY=VALUE，支持 `export` 前缀、单/双引号、行尾 `#` 注释：

```
WXMD_TOKEN=wmt_xxxxxxxxxxxxxxxx
WXMD_LIST_PASSWORD=xxxxxxxx
```

凭证选择规则：优先 `WXMD_TOKEN`（`Authorization: Bearer`）；未设置时回退 `WXMD_LIST_PASSWORD`——发布与项目操作同样以 Bearer 发送（主密码视作站长凭证），`list` / `delete` 沿用 `X-List-Password` 头。两者都未设置时，`publish` 和项目操作都会报错并给出令牌申请入口，不会静默失败，也不会降级成匿名发布。

## 令牌安全

- `.env` 已被仓库的 `.gitignore` 忽略，不要为了"方便"把令牌写进 `SKILL.md`、代码或文章正文。
- `set-token` 会把 `.env` 权限设为 600（仅本人可读写）。
- 输出里只出现掩码后的令牌（如 `wmt_abcd…wxyz`），完整令牌不会被打日志。
- 令牌泄露或不再使用，请联系站长吊销后重新申请。

## 图片处理说明

- 支持 png / jpg / jpeg / gif / webp / svg，单张 ≤10MB。
- 本地图片通过 `POST /api/upload` 上传到服务器，文件存储在服务端 `data/uploads/`，经 `/uploads/<文件>` 访问。
- **上传需要令牌**：与发布同口径，缺少有效令牌会返回 401，此时不会产生任何上传文件。
- 同一张图片在同一篇文档中多次引用只上传一次。
- 某张图上传失败不会中断发布：原引用保留，并在输出 JSON 的 `warnings` 中列出。

## 故障排查

- **提示"需要令牌"**：`.env` 里还没填令牌。执行 `python3 publish.py set-token wmt_xxxx`（或手动编辑 `<skill目录>/.env`）后重试。
- **HTTP 401 / 凭证无效或已被吊销**：令牌不对或已被站长吊销。先跑 `python3 publish.py token-status` 确认当前读到的令牌是不是你以为的那个——`publishToken.source` 若是 `环境变量`，说明环境变量覆盖了 `.env`，这时改 `.env` 是无效的。确属令牌失效则重新申请后覆盖。
- **改了 .env 却不生效**：见上一条，先看 `token-status` 的 `source`；另外 `.env` 需与 `SKILL.md` 同级（`<skill目录>/.env`），放到 `scripts/` 里不会被读取。
- **无法连接服务器**：检查网络和 `WXMD_API_URL`；`curl -s -o /dev/null -w '%{http_code}' https://md.foolgry.top` 应返回 200。
- **HTTP 400 不支持的图片格式**：图片扩展名需在支持列表内。
- **HTTP 413**：图片超过 nginx 限制（20MB），先压缩图片再发布。
- **发布后图片不显示**：查看输出 JSON 的 `warnings`，确认图片是否上传成功。
