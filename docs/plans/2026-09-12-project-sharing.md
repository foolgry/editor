# 项目化分享（多文章聚合分享）需求设计

日期：2026-09-12
状态：已与需求方确认，待实现
相关 ADR：[0001](../../adr/0001-project-as-optional-container.md) [0002](../../adr/0002-multi-token-auth.md) [0003](../../adr/0003-project-referenced-by-name.md)
术语表：见根目录 `CONTEXT.md`

## 1. 背景与目标

当前分享功能每篇文章生成一个独立链接（`/s/<id>`）。对于"日报"、"服务器巡检"这类持续产出的场景，链接分散、不成体系。本需求引入**项目（Project）**：一个可挂载多篇分享的命名容器，拥有聚合链接，读者打开后左侧目录切换、右侧阅读。

### 目标场景

1. **日报聚合**：每天发布的日报挂到"日报"项目，对外只发一个项目链接，读者可翻历史。
2. **Agent 定时产出**：巡检 Agent 每次运行通过 skill 发布文档到"服务器巡检"项目，通知系统携带链接推送到手机。
3. **熟人协作**：少数熟人持有各自令牌，可发布文章、管理自己名下的项目。

### 非目标

- 不做用户注册/登录系统（用多令牌替代，见 ADR-0002）。
- 不做目录手动排序、按月分组（将来可纯前端增强）。
- 不做项目级统一排版样式（各篇保留各自 style）。
- 不做"删除项目时级联删除文章"的快捷入口。

## 2. 领域模型

| 概念 | 说明 |
|---|---|
| 分享 Share | 原子内容单位，`/s/<id>`，现状不变 |
| 项目 Project | 分享的命名集合，可选容器，有独立链接 `/p/<pid>` |
| 令牌 Token | 站长签发的命名凭证，所有写操作的认证，可吊销 |
| 创建者 Creator | 每篇分享、每个项目归属创建它的令牌 |

核心规则：

- 分享可不属于任何项目；**一篇分享最多属于一个项目**（ADR-0001）。
- 项目名在**同一创建者内部唯一**；不同创建者可重名（ADR-0003）。
- 读操作（看分享、看项目）完全公开；**一切写操作需要令牌**，发布独立单篇除外（保持现状匿名公开）。

> 后续修订（2026-09-14）：上面最后一句已作废，见 [ADR-0004](../adr/0004-token-required-for-publishing.md)——现在**任何发布都需要令牌**，独立单篇不再匿名公开。

## 3. 数据模型

### `projects` 表（新增）

| 字段 | 说明 |
|---|---|
| id | 项目 ID（生成规则同 share id） |
| name | 项目名 |
| creator_token_id | 创建者令牌，外键 → tokens.id |
| created_at / updated_at | 时间戳 |

唯一约束：`(creator_token_id, name)`。

### `tokens` 表（新增）

| 字段 | 说明 |
|---|---|
| id | 令牌 ID |
| name | 令牌名（如"巡检 Agent"、"张三"） |
| token | 令牌串（随机生成，存哈希或明文由实现定，建议存哈希） |
| revoked | 是否已吊销 |
| created_at | 时间戳 |

### `shares` 表（变更）

- 新增 `project_id`（可空，外键 → projects.id）。
- 新增 `creator_token_id`（可空——站长用主密码发布的单篇没有创建者令牌）。
- 预留 `sort_order` 字段位（不实现 UI，为将来手动排序留余地）。

### 迁移

现有 shares 全部 `project_id = NULL`、`creator_token_id = NULL`，行为不变。现有 `X-List-Password` 主密码保留，视作站长主凭证。

## 4. 读者侧交互

### 项目页 `/p/<pid>`

- 打开默认显示**最新一篇** + 左侧目录。
- 目录按文章创建时间**倒序**；每项显示「标题 + 日期」两行（标题沿用现有 Markdown 提取逻辑）。
- 目录可滚动，不做分组折叠。
- 空项目显示"该项目暂无文章"占位页。

### 深链 `/p/<pid>/<sid>`

- 显示目录 + 指定篇。
- 目录中切换文章时地址栏 pushState 同步为当前篇深链（读者随时复制地址栏即得深链）。
- **兜底**：若 `<sid>` 已不属于 `<pid>`（被移出/移动），跳转 `/s/<sid>`（链接存活原则），不 404。

### 响应式

- 桌面/宽屏：左目录右文章固定布局。
- 手机（断点与主站一致，约 768px）：目录默认隐藏，顶部标题栏显示项目名 + ☰ 按钮，抽屉滑出，选中后自动收起。第一眼必须看到内容本身。

### 样式

各篇保留各自 style 渲染；目录 UI 用中性样式，不受文章 style 影响。

## 5. 发布者侧交互

### 编辑器主页

- 分享弹层增加"归入项目"区，**默认折叠**；未设令牌时显示"设置令牌后可归入项目"，分享按钮照常发独立单篇。
- 首次展开提示输入令牌，存 `localStorage`，之后自动携带；提供更换/清除入口。
- 已设令牌：项目下拉（仅列自己名下项目）+ 输入新名字即发布时自动创建。
- 匿名访客体验零变化。

> 后续修订（2026-09-14）：本节已作废，见 [ADR-0004](../adr/0004-token-required-for-publishing.md)——分享弹层现在无条件要求令牌，未设令牌时展示申请入口，"分享按钮照常发独立单篇"与"匿名访客体验零变化"不再成立。

### 管理页（`/list` 升级）

- 登录：站长主密码**或任一有效令牌**。令牌登录只见自己名下内容；主密码见全部。
- **项目标签页**：项目列表（名称、文章数、最新更新时间、创建者），新建、重命名、删除；点进项目可查看内含文章、移除、移动到其他项目。
- **分享标签页**：现有列表 + "所属项目"列，支持事后挂载/移动。
- **令牌管理区**：仅主密码可见，签发（命名）/吊销令牌。
- 删除项目确认文案："项目下的 N 篇文章将变为独立分享，链接保持有效"。
- 重命名项目确认文案：必须警告"使用旧名字发布的 Agent 将会自动创建一个新项目"（ADR-0003）。

## 6. 删除语义

| 动作 | 结果 |
|---|---|
| 从项目移除文章 | 变回独立单篇，`/s/<id>` 照常有效 |
| 删除项目 | 只删容器，文章全部变独立单篇，不级联 |
| 删除分享 | 彻底删除；目录中消失，其深链按兜底规则处理 |

## 7. API 设计（草案）

认证：写操作请求头携带 `Authorization: Bearer <token>` 或复用现有密码头（主密码视作站长凭证）。

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| POST | /api/share | 发布分享；body 可带 `project`（名字，不存在自动创建） | 带 project 时需令牌 |
| GET | /api/projects | 列出自己名下项目 | 令牌 |
| POST | /api/projects | 显式创建项目 | 令牌 |
| PATCH | /api/projects/<pid> | 重命名 | 令牌（本人名下） |
| DELETE | /api/projects/<pid> | 删除项目（不级联） | 令牌（本人名下） |
| POST | /api/share/<sid>/attach | 挂载/移动到项目 | 令牌 |
| POST | /api/share/<sid>/detach | 移出项目 | 令牌 |
| GET/POST/DELETE | /api/tokens | 令牌签发/列表/吊销 | 仅主密码 |
| GET | /p/<pid>，/p/<pid>/<sid> | 项目页/深链（HTML） | 公开 |
| GET | /api/projects/<pid> | 项目信息+文章列表（JSON，供项目页前端渲染） | 公开 |

## 8. skill（publish.py）变更

```bash
WXMD_TOKEN=xxx python3 publish.py publish --file report.md --project 服务器巡检   # 核心路径
python3 publish.py projects                                        # 列出项目
python3 publish.py project-create 服务器巡检
python3 publish.py project-rename 旧名 新名
python3 publish.py attach <share-id> --project 服务器巡检
python3 publish.py detach <share-id>
```

> 注（2026-09-14）：所有命令现在都要求 `WXMD_TOKEN`（见 [ADR-0004](../adr/0004-token-required-for-publishing.md)）。

- 认证：新增环境变量 `WXMD_TOKEN`；旧 `WXMD_LIST_PASSWORD` 继续兼容（视作站长主凭证）。
- publish 输出 JSON 增加字段：

```json
{
  "id": "abc123",
  "url": "https://md.foolgry.top/s/abc123",
  "projectId": "proj_x9",
  "projectUrl": "https://md.foolgry.top/p/proj_x9",
  "deepUrl": "https://md.foolgry.top/p/proj_x9/abc123"
}
```

未指定 `--project` 时 `projectId/projectUrl/deepUrl` 为 null，输出向后兼容。
SKILL.md 需同步更新：项目说明、创建、挂载等用法文档。

## 9. 实现拆分建议（供排期参考）

1. **后端数据层**：tokens/projects 表 + shares 迁移 + 令牌校验中间件。
2. **后端 API**：项目 CRUD、attach/detach、发布带 project。
3. **项目查看页**：`/p/` HTML 模板（目录 + 深链 + 响应式抽屉）。
4. **管理页升级**：双标签页 + 令牌管理区 + 双凭证登录。
5. **编辑器主页**：分享弹层"归入项目"区 + localStorage 令牌。
6. **skill**：publish.py 新命令 + SKILL.md 文档更新。
