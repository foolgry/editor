# 微信公众号 Markdown 编辑器 - Agent 安装指南

> **目标读者**: AI Agent（Claude Code、ZCode 等）
>
> **本文档用途**: 教 Agent 如何安装 Skill 包。Skill 为纯线上模式，核心是一个零依赖的 Python 脚本，**无需安装任何 CLI 或 npm 包**。

---

## 快速安装（给 Agent 的指令模板）

当用户说"安装微信 Markdown 编辑器工具"或类似请求时，执行以下步骤：

### 步骤 1: 确认 Python 3 可用

```bash
python3 --version
```

macOS / Linux 一般自带。没有则提示用户先安装 Python 3。

### 步骤 2: 安装 Skill 包

```bash
# 克隆仓库
git clone https://github.com/foolgry/editor.git /tmp/editor

# 确定技能目录（按用户使用的 Agent 选择）
SKILLS_DIR="$HOME/.claude/skills"   # Claude Code
# SKILLS_DIR="$HOME/.agents/skills" # ZCode / 通用

# 复制 Skill
mkdir -p "$SKILLS_DIR"
cp -r /tmp/editor/skills/wechat-markdown-editor "$SKILLS_DIR/"

# 验证安装
ls "$SKILLS_DIR/wechat-markdown-editor/SKILL.md"
ls "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py"
```

### 步骤 3: 验证可用

```bash
python3 "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py" styles
# 应输出 18 个样式的 JSON 列表

python3 "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py" publish --text "# 安装测试"
# 应输出包含 url 的 JSON
```

---

## 配置环境变量（可选）

默认连接 `https://md.foolgry.top`。如需连接自建服务器：

```bash
# 添加到 shell 配置文件
echo 'export WXMD_API_URL=https://your-server.example.com' >> ~/.zshrc
source ~/.zshrc
```

| 变量 | 说明 | 默认值 |
|---|---|---|
| `WXMD_API_URL` | API 服务器地址 | `https://md.foolgry.top` |
| `WXMD_API_TIMEOUT` | 请求超时（秒） | `30` |
| `WXMD_LIST_PASSWORD` | 列表/删除操作的管理密码 | 无 |

---

## 常见使用场景（给 Agent 的参考）

### 场景 1: 发布 Markdown 文件并生成分享链接

```bash
python3 "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py" publish --file article.md
```

文件中的本地图片会自动上传到服务器并改写引用。

### 场景 2: 直接把对话中写好的内容发布出去

```bash
python3 "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py" publish --text "$(cat <<'EOF'
# 标题
正文内容
EOF
)"
```

### 场景 3: 发布后打开预览

```bash
python3 "$SKILLS_DIR/wechat-markdown-editor/scripts/publish.py" publish --file article.md --open
```

---

## 故障排查

### 问题 1: python3 不存在

提示用户安装 Python 3（macOS: `brew install python3`，或 Xcode Command Line Tools 自带）。

### 问题 2: 无法连接服务器

```bash
curl -s -o /dev/null -w '%{http_code}' https://md.foolgry.top
# 应返回 200；否则检查网络或 WXMD_API_URL
```

### 问题 3: list/delete 返回 401

需要设置正确的 `WXMD_LIST_PASSWORD` 环境变量。

---

## 相关文档

- **Skill 使用文档**: [../skills/wechat-markdown-editor/SKILL.md](../skills/wechat-markdown-editor/SKILL.md)
- **项目主页**: https://github.com/foolgry/editor
