---
name: wechat-markdown-editor
description: 公众号 Markdown 编辑器 CLI 工具技能。当用户需要将 Markdown 排版为微信公众号格式、使用 wxmd-cli 进行文档排版、创建分享链接、列出样式主题或检查环境时使用。支持 typeset 排版、share 分享、styles 样式管理和 doctor 环境诊断等命令。
---

# wxmd-cli SKILL

## 概述

`wxmd-cli` 是公众号 Markdown 编辑器的 Agent-First CLI 工具，专为 AI Agent 设计，提供原子化命令和结构化 JSON 输出。

## 命令选择

### 1. typeset - Markdown 排版
将 Markdown 转换为微信公众号可用的 HTML。

**使用场景：**
- 本地排版 Markdown 文件
- 批量处理文档
- CI/CD 集成

**参数模板：**
```bash
wxmd-cli typeset --input <file> --style <style> --output <format> [--out <file>]
```

**示例：**
```bash
# 从文件读取，输出 HTML 到 stdout
wxmd-cli typeset --input article.md --style wechat-tech

# 从 stdin 读取
# echo "# Hello World" | wxmd-cli typeset --style wechat-default

# 输出到文件
wxmd-cli typeset --input article.md --style wechat-elegant --out output.html
```

### 2. share create - 创建分享
将内容分享到服务器，生成可访问的链接。

**使用场景：**
- 发布文章到分享服务
- 生成临时预览链接

**参数模板：**
```bash
wxmd-cli share create --input <file> --style <style> --output <format>
```

**示例：**
```bash
wxmd-cli share create --input article.md --style wechat-default
# 返回: { id, url, createdAt, style }
```

### 3. share get - 获取分享
根据 ID 获取分享内容。

**使用场景：**
- 验证分享是否创建成功
- 获取分享原始内容

**参数模板：**
```bash
wxmd-cli share get <id> --output <format>
```

**示例：**
```bash
wxmd-cli share get abc12345
# 返回: { id, content, style, createdAt, updatedAt }
```

### 4. styles list - 列出样式
查看所有可用的排版样式。

**使用场景：**
- 选择合适的样式主题
- 了解可用选项

**参数模板：**
```bash
wxmd-cli styles list [--fields <field1,field2>]
```

**示例：**
```bash
# 列出所有样式
wxmd-cli styles list

# 只显示特定字段
wxmd-cli styles list --fields key,name
```

### 5. doctor - 环境检查
检查本地环境和 API 连接状态。

**使用场景：**
- 初次使用前验证环境
- 排查连接问题
- CI 环境检查

**参数模板：**
```bash
wxmd-cli doctor
```

**示例：**
```bash
wxmd-cli doctor
# 返回: { healthy, summary: { ok, warning, error }, checks: [...] }
```

## 恢复路径

### 1. 依赖未安装

**症状：** 命令执行失败，提示 `Cannot find module`

**恢复步骤：**
```bash
cd wxmd-cli
pnpm install
```

### 2. API 连接失败

**症状：** share create/get 返回 `NETWORK_ERROR` 或 `ECONNREFUSED`

**恢复步骤：**
1. 检查服务器是否运行：`curl http://localhost:8080`
2. 检查环境变量：`echo $WXMD_API_URL`
3. 设置正确的 API URL：`export WXMD_API_URL=http://your-server:8080`
4. 重新运行 doctor 验证：`wxmd-cli doctor`

### 3. 样式不存在

**症状：** typeset/share 返回 `INVALID_STYLE`

**恢复步骤：**
1. 查看可用样式：`wxmd-cli styles list`
2. 使用正确的样式名称
3. 默认样式为 `wechat-default`

### 4. 输入文件不存在

**症状：** 返回 `READ_ERROR` 或 `File not found`

**恢复步骤：**
1. 检查文件路径：`ls -la <file>`
2. 使用绝对路径
3. 或使用 stdin：`cat file.md | wxmd-cli typeset`

## 最小可运行示例

### 场景 1：排版单篇文档
```bash
cd /Users/wangzhi/project/2026-02-17-alchaincyf-huasheng_editor

# 创建测试文档
cat > /tmp/test.md << 'EOF'
# 测试文章

这是一篇**测试文章**。

## 特性

- 自动排版
- 样式美化
- 一键分享
EOF

# 排版
./wxmd-cli/src/index.js typeset --input /tmp/test.md --style wechat-tech
```

### 场景 2：创建分享并获取
```bash
cd /Users/wangzhi/project/2026-02-17-alchaincyf-huasheng_editor

# 确保服务器在运行（另开终端）
# cd server && go run main.go

# 创建分享
result=$(./wxmd-cli/src/index.js share create --input /tmp/test.md --style wechat-default)
share_id=$(echo $result | grep -o '"id": "[^"]*"' | head -1 | cut -d'"' -f4)

# 获取分享
./wxmd-cli/src/index.js share get $share_id
```

### 场景 3：批量处理
```bash
# 批量排版多个文件
for file in *.md; do
  ./wxmd-cli/src/index.js typeset --input "$file" --out "${file%.md}.html"
done
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `WXMD_API_URL` | `http://localhost:8080` | API 服务器地址 |
| `WXMD_API_TIMEOUT` | `30000` | 请求超时时间（毫秒） |

## 输出格式

### JSON 格式（默认）
```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "cliVersion": "1.0.0",
    "timestamp": "2026-04-06T10:30:00.000Z"
  }
}
```

### 错误格式
```json
{
  "ok": false,
  "error": {
    "code": "ERROR_CODE",
    "type": "ERROR_TYPE",
    "message": "Human readable message",
    "retryable": false,
    "actionHint": "Suggested fix"
  },
  "meta": { ... }
}
```

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

## 注意事项

1. **默认输出到 stdout**，使用 `--out` 写入文件
2. **支持 stdin 输入**，适合管道操作
3. **API 操作需要服务器运行**，使用 doctor 检查
4. **样式名称需完全匹配**，使用 styles list 查看可用选项
