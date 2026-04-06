# 微信公众号 Markdown 编辑器 - Agent 安装指南

> **目标读者**: AI Agent（Claude Code 等）
> 
> **本文档用途**: 教 Agent 如何安装和配置 Skill 包及 CLI 工具

---

## 快速安装（给 Agent 的指令模板）

当用户说"安装微信 Markdown 编辑器工具"或类似请求时，执行以下步骤：

### 步骤 1: 安装 CLI 工具（NPM 方式）

```bash
# 全局安装（推荐）
npm install -g @foolgry/wxmd-cli

# 或使用 pnpm
pnpm add -g @foolgry/wxmd-cli

# 或使用 yarn
yarn global add @foolgry/wxmd-cli
```

**验证安装**:
```bash
wxmd-cli --version
# 应输出: 1.0.0
```

### 步骤 2: 安装 Skill 包（可选，用于 Claude Code）

如果用户在使用 Claude Code，将本 Skill 安装到 Claude Code 的技能目录：

```bash
npx skills add foolgry/editor -g --all
```

---

## 完整安装流程（详细版）

### 前提条件检查

在安装前，检查以下条件：

```bash
# 检查 Node.js 版本（需要 >= 18）
node --version

# 检查 NPM/PNPM
npm --version
# 或
pnpm --version
```

如果 Node.js 版本过低，提示用户升级。

### 安装 CLI 工具

#### 方式 1: NPM 全局安装（推荐）

```bash
npm install -g @foolgry/wxmd-cli
```

#### 方式 2: 本地项目安装

```bash
# 在用户的项目目录
cd /path/to/user/project
npm install @foolgry/wxmd-cli

# 使用 npx 运行
npx wxmd-cli --help
```

#### 方式 3: 从源码安装（开发/测试）

```bash
# 克隆仓库
git clone https://github.com/foolgry/editor.git
cd editor/wxmd-cli

# 安装依赖
pnpm install

# 链接到全局（可选）
pnpm link --global
```

### 验证安装

```bash
# 检查版本
wxmd-cli --version

# 查看帮助
wxmd-cli --help

# 检查环境
doctor
wxmd-cli doctor

# 列出可用样式
wxmd-cli styles list
```

### 配置环境变量（可选）

如果用户需要连接自定义服务器：

```bash
# 添加到 shell 配置文件
echo 'export WXMD_API_URL=http://your-server:8080' >> ~/.bashrc
echo 'export WXMD_API_TIMEOUT=30000' >> ~/.bashrc

# 立即生效
source ~/.bashrc
```

---

## 故障排查

### 问题 1: 命令未找到 (command not found)

**症状**: `wxmd-cli: command not found`

**解决方案**:

```bash
# 检查 npm 全局安装路径
npm config get prefix

# 确保路径在 PATH 中
export PATH="$PATH:$(npm config get prefix)/bin"

# 或者使用 npx
npx wxmd-cli --help
```

### 问题 2: 权限错误 (EACCES)

**症状**: `npm ERR! Error: EACCES: permission denied`

**解决方案**:

```bash
# 方式 1: 使用 npx（无需全局安装）
npx @foolgry/wxmd-cli --help

# 方式 2: 修改 npm 默认目录权限
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
npm install -g @foolgry/wxmd-cli
```

### 问题 3: Node.js 版本过低

**症状**: `SyntaxError` 或 `ERR_REQUIRE_ESM`

**解决方案**: 提示用户升级 Node.js 到 18+

```bash
# 使用 nvm 升级
nvm install 18
nvm use 18

# 或使用 n
n install 18
```

### 问题 4: 网络/代理问题

**症状**: 安装超时或连接失败

**解决方案**:

```bash
# 使用国内镜像
npm install -g @foolgry/wxmd-cli --registry=https://registry.npmmirror.com

# 或使用代理
npm install -g @foolgry/wxmd-cli --proxy=http://proxy.example.com:8080
```

---

## 更新 CLI 工具

```bash
# 更新到最新版本
npm update -g @foolgry/wxmd-cli

# 或重新安装
npm uninstall -g @foolgry/wxmd-cli
npm install -g @foolgry/wxmd-cli
```

---

## 卸载 CLI 工具

```bash
npm uninstall -g @foolgry/wxmd-cli
```

---


## 相关文档

- **Skill 使用文档**: [SKILL.md](./SKILL.md)
- **CLI 详细文档**: [../../wxmd-cli/README.md](../../wxmd-cli/README.md)
- **项目主页**: https://github.com/foolgry/editor
- **NPM 包页**: https://www.npmjs.com/package/@foolgry/wxmd-cli

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2026-04-06 | 初始版本，包含 typeset, share, styles, doctor 命令 |
