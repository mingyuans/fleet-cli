# Fleet 使用说明

Fleet 是一个多仓库工作区管理工具，使用 Go 编写。它通过 manifest XML 文件声明式管理多个 Git 仓库，支持 GitHub Fork 工作流下的批量 clone、sync、分支管理、push、PR 创建和跨仓库命令执行。

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
├── fleet.xml              # 团队共享配置（提交到 Git）
├── local_fleet.xml       # 个人本地配置（不提交）
└── services/
    ├── user-service/
    ├── order-service/
    └── ...
```

### 2. 配置 Manifest

**fleet.xml** — 团队共享配置：

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

**local_fleet.xml** — 个人 Fork 配置：

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
  -g, --group <expr>    按 group 表达式过滤项目
  -h, --help            显示帮助信息
```

所有子命令均支持 `-g` 参数，用于按项目的 `groups` 属性过滤。

**Group 过滤表达式**支持 `,` 表示 OR、`+` 表示 AND：

| 表达式 | 含义 |
|--------|------|
| `core,web` | 属于 `core` **或** `web` 组的项目 |
| `core+backend` | 同时属于 `core` **和** `backend` 组的项目 |
| `core+backend,infra` | (`core` 且 `backend`) **或** `infra` |

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
  ▸ Local manifest: /path/to/local_fleet.xml

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
PROJECT              BRANCH              STATUS     AHEAD/BEHIND  FETCH              PUSH
─────────────────────────────────────────────────────────────────────────────────────────────
user-service         master              clean                    github/master      fork
order-service        feature/new-api     dirty      +3 -1        github/master      fork
payment-gateway      –                   not cloned               github/master      fork
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

### `fleet start`

在所有仓库中创建并切换到一个基于上游默认分支的新分支。

```bash
fleet start [-g <group>] <branch>
```

**行为：**

- 已在目标分支上 → 跳过
- 目标分支已存在 → 切换到该分支（`git checkout`）
- 目标分支不存在 → 从 remote fetch 最新代码，基于 `<remote>/<default-branch>` 创建（`git checkout -b`）
- 支持 `master-main-compat`：若 `master` 在 remote 上不存在，自动尝试 `main`（反之亦然）

**输出示例：**

```
Starting branch feature/new-api across 3 projects...
  ✓ [1/3] services/user-service (created from github/master)
  ✓ [2/3] services/order-service (switched)
  – [3/3] services/svc-a (skipped: not cloned)

1 created, 1 switched, 1 skipped
```

---

### `fleet finish`

在所有仓库中删除指定分支并切回默认分支。

```bash
fleet finish [-g <group>] [-r] <branch>
```

**选项：**

| 参数 | 说明 |
|------|------|
| `-r, --remote` | 同时删除 push remote 上的远程分支 |

**行为：**

- 本地不存在该分支 → 跳过
- 当前在目标分支上 → 先切回默认分支，再删除
- 使用 `-r` 参数 → 同时执行 `git push <push-remote> --delete <branch>`

**输出示例：**

```
Finishing branch feature/new-api across 3 projects...
  ✓ [1/3] services/user-service (finished)
  – [2/3] services/order-service (skipped: branch feature/new-api not found)
  – [3/3] services/svc-a (skipped: not cloned)

1 finished, 2 skipped
```

---

### `fleet pr`

推送当前分支并通过 `gh` CLI 创建 Pull Request。

```bash
fleet pr [-g <group>] [-t <title>]
```

**选项：**

| 参数 | 说明 |
|------|------|
| `-t, --title` | PR 标题（默认使用分支名） |

**前置条件：** 需要安装并认证 [GitHub CLI (`gh`)](https://cli.github.com/)。

**行为：**

- 将当前分支推送到 push remote
- 创建 PR，目标为上游默认分支（fetch remote）
- Fork 工作流下自动设置 `--head` 为 `<fork-owner>:<branch>`
- 跳过处于默认分支、detached HEAD 或无 push remote 的仓库

**输出示例：**

```
Creating PRs for 2 projects...
  ✓ [1/2] services/user-service (created https://github.com/my-org/user-service/pull/42)
  – [2/2] services/order-service (skipped: on default branch)

1 created, 1 skipped
```

---

### `fleet worktree`

为所有受管理仓库在共享基础目录下创建 git worktree，目录结构与原始工作区保持一致。

```bash
fleet worktree <name> [-b <branch>] [-r <revision>] [-d <dest>] [-g <group>]
```

**选项：**

| 参数 | 说明 |
|------|------|
| `-b, --branch` | 要创建或切换的分支名（默认：与 `name` 相同） |
| `-r, --revision` | 新分支的基础 upstream revision（默认：各项目在 fleet.xml 中配置的 `revision`） |
| `-d, --dest` | worktree 的目标目录，覆盖 `worktree-base/<name>` 路径拼接 |

**使用场景：** 在不切换主工作区分支的情况下，并行开发另一个需求。每个仓库的 worktree 创建在 `<worktree-base>/<name>/<proj.path>` 下，目录结构与原工作区完全对应。

指定 `--dest` 时，worktree 直接创建在 `<dest>/<proj.path>` 下，跳过 `worktree-base` 配置。适用于需要完全控制目标路径或未配置 `worktree-base` 的场景。

**配置** — 在 `fleet.xml` 的 `<default>` 中添加 `worktree-base`（未使用 `--dest` 时必填）和 `worktree-copy`（可选）：

```xml
<default remote="github"
         revision="main"
         sync-j="4"
         worktree-base="~/worktrees/myproject"
         worktree-copy=".env,.env.*" />
```

| 属性 | 说明 |
|------|------|
| `worktree-base` | 所有 worktree 的基础目录，支持 `~` 展开。未使用 `--dest` 时必填。 |
| `worktree-copy` | 逗号分隔的 glob 表达式，用于指定创建 worktree 后需要从原仓库复制的 gitignored 文件（如 `.env`）。所有项目默认继承此配置，单个 `<project>` 可通过自身的 `worktree-copy` 属性完全覆盖。 |

**行为：**

- Worktree 创建在 `<worktree-base>/<name>/<proj.path>` 路径下（使用 `--dest` 时为 `<dest>/<proj.path>`）
- 若分支在本地不存在，则基于 `<remote>/<revision>` 创建新分支
- `-r` 参数可覆盖各项目的 `revision` 字段，作为新分支的基础
- workspace 根项目（`path="."`）优先处理，避免与并行执行的 service 项目产生目录竞争
- 创建完成后，将 `worktree-copy` 匹配的文件从源仓库复制到新 worktree
- 目标路径的 worktree 已存在 → 跳过

**输出示例：**

```
Creating worktree JIRA-123 across 3 projects...
  ✓ [1/3] . (created)
  ✓ [2/3] services/user-service (created)
  ✓ [3/3] services/order-service (created)

3 created
```

创建后的 worktree 工作区与原始目录结构完全对应：

```
~/worktrees/myproject/
└── JIRA-123/
    ├── .                      ← workspace 仓库的 worktree
    └── services/
        ├── user-service/      ← 基于 JIRA-123 分支的 worktree
        └── order-service/     ← 基于 JIRA-123 分支的 worktree
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

### `fleet ide-setup idea`

为工作区中的所有仓库生成 IntelliJ IDEA / GoLand 的 VCS 目录映射。

```bash
fleet ide-setup idea [-g <group>]
```

**行为：**

- 在工作区根目录创建 `.idea/vcs.xml`
- 为根项目和每个已 clone 的子项目添加 VCS 映射
- 未 clone 的项目自动跳过

使用此命令后，IntelliJ 系列 IDE 打开工作区根目录即可识别所有仓库。

---

## Manifest 配置

### 标签说明

| 标签 | 说明 |
|------|------|
| `<remote>` | 定义一个 Git remote 端点（`name`、`fetch`、`review`） |
| `<default>` | 所有项目的默认值（`remote`、`revision`、`sync-j`、`push`、`master-main-compat`、`worktree-base`、`worktree-copy`） |
| `<project>` | 定义一个 Git 仓库（`name`、`path`、`groups`、`remote`、`revision`、`push`、`worktree-copy`） |

### `master-main-compat` 属性

当 `<default>` 上设置 `master-main-compat="true"` 时，Fleet 会在 remote 上找不到配置的 revision 分支时，自动在 `master` 和 `main` 之间回退。适用于工作区中部分仓库用 `master`、部分用 `main` 的场景。

```xml
<default remote="github" revision="master" sync-j="4" master-main-compat="true" />
```

### 合并规则

当 `local_fleet.xml` 存在时，与 `fleet.xml` 按以下规则合并：

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
| `FLEET_MANIFEST` | 主 manifest 文件路径 | `<workspace_root>/fleet.xml` |
| `FLEET_LOCAL_MANIFEST` | 本地覆盖 manifest 文件路径 | `<workspace_root>/local_fleet.xml` |

```bash
FLEET_MANIFEST=~/custom/fleet.xml fleet status
FLEET_LOCAL_MANIFEST=~/custom/local.xml fleet init
```

---

## 典型工作流

### 首次初始化

```bash
cd my-workspace
# 创建个人 local_fleet.xml，配置 fork remote
vim local_fleet.xml
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

# 在所有仓库创建并切换到新 feature 分支
fleet start feature/my-feature

# 推送 feature branch 到 fork
fleet push

# 推送并创建 PR
fleet pr -t "feat: my feature"

# 合并后清理分支
fleet finish feature/my-feature

# 推送包括默认分支
fleet push --all
```

### 添加新服务

1. 在 `fleet.xml` 中添加 `<project>` 条目
2. 运行 `fleet init` — 新仓库被 clone，已有仓库不受影响

### 添加私有仓库

1. 在 `local_fleet.xml` 中添加 `<project>` 条目
2. 运行 `fleet init`
