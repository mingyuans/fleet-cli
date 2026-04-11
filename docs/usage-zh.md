# Fleet 使用说明

Fleet 是一个多仓库工作区管理工具，使用 Go 编写。它通过 manifest XML 文件声明式管理多个 Git 仓库，支持 GitHub Fork 工作流下的批量 clone、sync、status、push 和跨仓库命令执行。

## 安装

```bash
# 从源码安装
go install github.com/mingyuans/fleet-cli@latest

# 或本地构建
git clone https://github.com/mingyuans/fleet-cli.git
cd fleet-cli
make build
# 二进制文件生成在 bin/fleet
```

## 快速开始

### 1. 准备工作区

工作区是一个包含 manifest 文件和多个 Git 仓库的目录：

```
workspace/
├── default.xml              # 团队共享配置（提交到 Git）
├── local_manifest.xml       # 个人本地配置（不提交）
└── services/
    ├── user-service/
    ├── order-service/
    └── ...
```

### 2. 配置 Manifest

**default.xml** — 团队共享配置：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github"
          fetch="git@github.com:my-org/"
          review="https://github.com/my-org/" />

  <default remote="github" revision="master" sync-j="4" />

  <project name="user-service"
           path="services/user-service"
           groups="core,backend" />

  <project name="order-service"
           path="services/order-service"
           groups="commerce" />
</manifest>
```

**local_manifest.xml** — 个人 Fork 配置：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:your-username/" />
  <default push="fork" />
</manifest>
```

### 3. 初始化仓库

```bash
fleet init
```

## 命令详解

### 全局选项

```
fleet <command> [options]

选项:
  -g, --group <group>   按 group 过滤项目
  -h, --help            显示帮助信息
```

所有子命令均支持 `-g` 参数，用于按项目的 `groups` 属性过滤。

---

### `fleet init`

克隆仓库并配置 fetch / push remote。

```bash
fleet init [-g <group>]
```

**行为：**

- **仓库不存在** — 执行 `git clone`，并配置 push remote（如有）
- **仓库已存在** — 幂等检查并修复 remote 配置（URL 不匹配时自动修正）

**输出示例：**

```
  ▸ Default push remote: fork
  ▸ Local manifest: /path/to/local_manifest.xml

Initializing 5 projects...
  ✓ [1/5] services/user-service (cloned)
  ✓ [2/5] services/order-service (cloned)
  ✓ [3/5] services/svc-a (configured)
  – [4/5] services/svc-b (skipped)
  ✗ [5/5] services/svc-c: clone failed: ...

3 cloned, 1 configured, 1 skipped
```

`fleet init` 可安全重复执行。已正确配置的仓库会被跳过。

---

### `fleet sync`

从上游 fetch remote 拉取最新代码。

```bash
fleet sync [-g <group>]
```

**行为：**

- 当前分支 == 默认分支 → `git pull --rebase`
- 当前分支 != 默认分支（feature branch）→ `git fetch`（仅 fetch，不 merge）
- 仓库未 clone → 跳过并警告
- 支持 remote fallback：若 manifest 配置的 remote 不存在，自动 fallback 到 `origin`

**输出示例：**

```
Syncing 3 projects...
  ✓ [1/3] services/user-service (rebased)
  ✓ [2/3] services/order-service (fetched)
  – [3/3] services/svc-a (skipped)

1 rebased, 1 fetched, 1 skipped
```

---

### `fleet status`

表格式展示所有仓库的状态。

```bash
fleet status [-g <group>]
```

**输出示例：**

```
PROJECT                  BRANCH              STATUS     AHEAD/BEHIND
──────────────────────────────────────────────────────────────────────
user-service             master              clean
order-service            feature/new-api     dirty      +3 -1
payment-gateway          –                   not cloned
```

**颜色规则：**

- 分支名：与默认分支不同时显示黄色
- 状态：`clean` 绿色，`dirty` 黄色
- 未 clone：整行灰色

---

### `fleet push`

将当前分支推送到 push remote。

```bash
fleet push [-g <group>] [--all]
```

**行为：**

- 默认只推送 feature branch（非默认分支）
- `--all` 模式推送所有分支（包括默认分支）
- 以下情况自动跳过：未 clone、无 push remote、detached HEAD、push remote 不存在

**输出示例：**

```
  ▸ Push remote: fork

Pushing 3 projects...
  ✓ [1/3] services/user-service (pushed)
  – [2/3] services/order-service (skipped)
  – [3/3] services/svc-a (skipped)

1 pushed, 2 skipped
```

---

### `fleet forall`

在所有仓库中执行指定命令。

```bash
# shell 模式
fleet forall [-g <group>] -c "<command>"

# 直接执行模式
fleet forall [-g <group>] -- <command> [args...]
```

**环境变量：**

在命令执行上下文中，以下环境变量可用：

| 变量名 | 说明 |
|--------|------|
| `FLEET_PROJECT_NAME` | 项目名称 |
| `FLEET_PROJECT_PATH` | 相对路径 |
| `FLEET_PROJECT_GROUPS` | group 列表（逗号分隔） |
| `FLEET_PROJECT_REMOTE` | fetch remote 名称 |
| `FLEET_PROJECT_REVISION` | 默认分支 |
| `FLEET_PROJECT_PUSH_REMOTE` | push remote 名称 |

**示例：**

```bash
# 查看所有仓库的分支
fleet forall -c "git branch"

# 查看 commerce 组的最近提交
fleet forall -g commerce -- git log --oneline -5

# 批量执行 go mod tidy
fleet forall -c "go mod tidy"
```

单个项目的命令失败不会中断整体流程，会发出警告并继续执行。

---

## Manifest 配置

### 标签说明

| 标签 | 说明 |
|------|------|
| `<remote>` | 定义一个 Git remote 端点（`name`、`fetch`、`review`） |
| `<default>` | 所有项目的默认值（`remote`、`revision`、`sync-j`、`push`） |
| `<project>` | 定义一个 Git 仓库（`name`、`path`、`groups`、`remote`、`revision`、`push`） |

### 合并规则

当 `local_manifest.xml` 存在时，与 `default.xml` 按以下规则合并：

- **Remote** — 同名替换，新增追加
- **Default** — 逐属性覆盖（local 非空属性覆盖 default 对应属性）
- **Project** — 同名逐属性覆盖，新增追加

### URL 构建规则

```
clone_url = remote.fetch + project.name + ".git"
push_url  = push_remote.fetch + project.name + ".git"
```

仅当 push remote 与 fetch remote 不同时，才配置独立的 push remote。

---

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `FLEET_MANIFEST` | 主 manifest 文件路径 | `<workspace_root>/default.xml` |
| `FLEET_LOCAL_MANIFEST` | 本地覆盖 manifest 文件路径 | `<workspace_root>/local_manifest.xml` |

```bash
FLEET_MANIFEST=~/custom/default.xml fleet status
FLEET_LOCAL_MANIFEST=~/custom/local.xml fleet init
```

---

## 典型工作流

### 首次初始化

```bash
cd my-workspace
# 创建个人 local_manifest.xml，配置 fork remote
vim local_manifest.xml
# 初始化所有仓库
fleet init
# 或只初始化某个 group
fleet init -g core
```

### 日常开发

```bash
# 查看所有仓库状态
fleet status

# 同步上游代码
fleet sync

# 推送 feature branch 到 fork
fleet push

# 推送包括默认分支
fleet push --all
```

### 添加新服务

1. 在 `default.xml` 中添加 `<project>` 条目
2. 运行 `fleet init` — 新仓库被 clone，已有仓库不受影响

### 添加私有仓库

1. 在 `local_manifest.xml` 中添加 `<project>` 条目
2. 运行 `fleet init`
