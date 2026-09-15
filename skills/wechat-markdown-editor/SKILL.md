---
name: wechat-markdown-editor
description: 公众号 Markdown 编辑器线上发布技能。当用户需要把 Markdown 文章发布为排版精美的微信公众号格式、生成可分享的在线链接时使用。纯线上模式：通过 HTTP 请求把内容写入线上编辑器（默认 https://md.foolgry.top），返回分享 URL，本地图片自动上传。发布需要令牌，令牌放在技能目录下的 .env 文件里（脚本自动读取，用户可手动编辑，agent 可用 set-token 子命令写入），无令牌时到 <API 地址>/apply 申请。支持纯文本直接发送和本地文件（含本地图片）发布；支持项目（Project）聚合多篇文章（日报、巡检报告等持续产出场景），对外只发一个项目链接。
---

# 公众号 Markdown 编辑器 - 线上发布

## 概述

把 Markdown 内容发布到线上公众号排版编辑器（`https://md.foolgry.top`），立即获得一个排版好的分享链接。不做任何本地渲染，排版由线上页面完成。

核心脚本：`scripts/publish.py`（Python 3 标准库实现，**零依赖**，无需安装任何东西）。

**发布需要令牌**：本站已关闭匿名发布，`publish`（含其中的图片上传）和所有项目操作都必须提供令牌。令牌写在技能目录下的 `.env` 文件里，**配好一次即可**——脚本每次运行自动读取，命令里不用带任何令牌。

**直接发布就行，不要预先检查 `.env`。** 不要为了"保险"在每次发布前跑 `token-status`、`cat .env` 或做存在性判断：令牌配好了发布自然成功，没配好脚本会立刻报错，错误信息里已经带上 `.env` 路径、`set-token` 命令和令牌申请入口。**只有看到那个报错时**，才去处理令牌（见下一节）——这样正常路径少一次无谓的检查和一轮往返。

**项目（Project）**是分享的命名集合：把持续产出的文章（日报、服务器巡检报告等）挂到同一个项目名下，对外只发一个聚合链接 `/p/<pid>`（打开默认显示最新一篇，左侧目录可翻历史），单篇深链为 `/p/<pid>/<sid>`。项目按**名字**引用，不存在时自动创建（无人值守的定时任务友好）；一篇分享最多属于一个项目，也可以不属于任何项目。

## 令牌：报错后再看这一节

### 报错长什么样

没配令牌时，脚本会立刻失败，不会上传到一半才断：

```json
{"error": "需要令牌：请在 <skill目录>/.env 里设置 WXMD_TOKEN=wmt_xxxx，或执行 python3 <skill目录>/scripts/publish.py set-token wmt_xxxx；申请令牌：https://md.foolgry.top/apply"}
```

拿到这句才需要往下走。

### 写入 .env

`<skill目录>/.env` 是令牌唯一的存放位置，文件里已有带注释的模板，照着填即可。该文件已被 `.gitignore` 忽略，不会进仓库。

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

`set-token` 是幂等的：重复执行会原地替换 `WXMD_TOKEN` 的值，不会堆出多行。换令牌也走它。

这个文件除了令牌还可以放 `WXMD_LIST_PASSWORD`（`list` / `delete` 用）和 `WXMD_API_URL`（自建服务时改地址），写法同上。

### 还没有令牌

打开 `<API 地址>/apply`（默认 <https://md.foolgry.top/apply>）提交申请，站长签发后会给你一串 `wmt_` 开头的令牌。

### 仍不通时的排查

- 报错说「凭证无效或已被吊销」：令牌写错或已被站长吊销，重新申请后重跑一次 `set-token` 覆盖即可。
- 写入后仍报「需要令牌」：确认 `.env` 位置对不对——它必须与 `SKILL.md` 同级（`<skill目录>/.env`），放到 `scripts/` 里不会被读取。用 `python3 scripts/publish.py token-status` 可以看到脚本实际读的是哪个文件、读到了什么（令牌只显示掩码）。

## 使用方式

本技能的脚本路径为 `<skill目录>/scripts/publish.py`，以下统一用 `publish.py` 指代。**直接执行即可，不需要任何前置检查**：`.env` 配好了命令就成功，没配好会在报错里说明怎么补（见上一节）。

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

## 配置项（.env）

配置**只**来自技能目录下的 `.env`（脚本不读环境变量）：

```
<skill目录>/.env
```

| 键 | 说明 | 默认值 |
|---|---|---|
| `WXMD_API_URL` | API 地址 | `https://md.foolgry.top` |
| `WXMD_API_TIMEOUT` | 请求超时（秒） | `30` |
| `WXMD_TOKEN` | 发布令牌：`publish`（含图片上传）与全部项目操作都需要。从 `<API 地址>/apply` 申请 | 无 |
| `WXMD_LIST_PASSWORD` | 列表/删除的管理密码；发布与项目操作未设 `WXMD_TOKEN` 时回退用它（主密码视作站长凭证） | 无 |

写法是常见的 `KEY=VALUE`，支持 `export` 前缀、单/双引号、行尾 `#` 注释：

```
WXMD_TOKEN=wmt_xxxxxxxxxxxxxxxx
WXMD_LIST_PASSWORD=xxxxxxxx
```

键不存在或值为空都算未设置（`WXMD_TOKEN=` 空着不会挡住后面的回退）。`.env` 整个缺失也不影响运行，只是凭证为空，发布时会报错提示去补。临时换 API 地址可以用 `--base-url` 参数，不必改文件。

凭证选择规则：优先 `WXMD_TOKEN`（`Authorization: Bearer`）；未设置时回退 `WXMD_LIST_PASSWORD`——发布与项目操作同样以 Bearer 发送（主密码视作站长凭证），`list` / `delete` 沿用 `X-List-Password` 头。两者都未设置时，`publish` 和项目操作都会报错并给出令牌申请入口，不会静默失败，也不会降级成匿名发布。

## 维护：同步到全局时不要覆盖 .env

本技能会被同步到 `~/.agents/skills/wechat-markdown-editor` 等全局位置，同步时**必须排除 `.env`**：

```bash
rsync -a --delete \
  --exclude '.env' \
  --exclude '__pycache__/' --exclude '*.pyc' --exclude '.DS_Store' \
  skills/wechat-markdown-editor/ ~/.agents/skills/wechat-markdown-editor/
```

- 项目仓库里的 `.env` 是**模板**（令牌那行是注释掉的），发布位置上的是**用户实际配置**。不加 `--exclude '.env'` 会用模板覆盖掉真令牌，表现为「昨天还能发，今天突然说需要令牌」。
- 只在目标位置**没有** `.env` 时才用模板补一份（`install -m 600 <源>/.env <目标>/.env`），已有就原样留着。
- 每个落地位置的 `.env` 各自独立：在 A 位置配了令牌，B 位置仍然要配。想确认某处配没配，跑那次目录里的 `token-status`。

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
- **HTTP 401 / 凭证无效或已被吊销**：令牌不对或已被站长吊销。重新申请后重跑 `set-token` 覆盖即可。
- **改了 .env 却不生效**：确认文件位置——`.env` 必须与 `SKILL.md` 同级（`<skill目录>/.env`），丢进 `scripts/` 里不会被读取。跑 `python3 publish.py token-status` 可以看到脚本实际读的是哪个文件、读到了什么。
- **之前能用，突然提示需要令牌**：多半是 `.env` 被覆盖了（例如同步技能到全局时没排除 `.env`，模板把真令牌冲掉了）。重新 `set-token` 一次。
- **无法连接服务器**：检查网络和 `.env` 里的 `WXMD_API_URL`；`curl -s -o /dev/null -w '%{http_code}' https://md.foolgry.top` 应返回 200。
- **HTTP 400 不支持的图片格式**：图片扩展名需在支持列表内。
- **HTTP 413**：图片超过 nginx 限制（20MB），先压缩图片再发布。
- **发布后图片不显示**：查看输出 JSON 的 `warnings`，确认图片是否上传成功。
