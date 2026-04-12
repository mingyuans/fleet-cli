# Fleet

一款面向 GitHub Fork 工作流的多仓库工作区管理工具，灵感来源于 [Google 的 `repo` 工具](https://gerrit.googlesource.com/git-repo/)。

与 `repo` 类似，Fleet 使用 XML manifest 文件声明式管理多个 Git 仓库——但专为 **GitHub Fork + Pull Request** 工作流设计，而非 Gerrit。支持批量 clone、sync、分支管理、push、创建 PR，以及跨所有仓库执行命令。

[English Documentation](README.md)

## 安装

```bash
# 一键安装（macOS / Linux）
curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# 或指定版本 / 安装目录
FLEET_VERSION=v0.1.0 FLEET_INSTALL_DIR=~/.local/bin \
  curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# 或从源码构建
git clone https://github.com/mingyuans/fleet-cli.git
cd fleet-cli
make install
```

## 快速开始

**1. 创建包含 `fleet.xml` 的工作区：**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:my-org/" />
  <default remote="github" revision="master" sync-j="4" />

  <project name="user-service"    path="services/user-service"  groups="core" />
  <project name="order-service"   path="services/order-service" groups="commerce" />
</manifest>
```

**2. 添加个人 `local_fleet.xml` 配置 Fork：**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:your-username/" />
  <default push="fork" />
</manifest>
```

**3. 克隆所有仓库：**

```bash
fleet init
```

## 命令列表

| 命令 | 说明 |
|------|------|
| `fleet init` | 克隆仓库并配置 remote |
| `fleet sync` | 从上游拉取最新代码（默认分支 rebase，feature 分支 fetch）|
| `fleet status` | 显示所有仓库的分支、脏/干净状态及领先/落后情况 |
| `fleet start <branch>` | 在所有仓库基于上游 HEAD 创建 feature 分支 |
| `fleet finish <branch>` | 删除分支并切回默认分支（`-r` 同时删除远程分支）|
| `fleet push` | 将当前分支推送到 fork（`--all` 包含默认分支）|
| `fleet pr` | 推送并通过 `gh` CLI 创建 PR（`-t` 设置标题）|
| `fleet forall -c "cmd"` | 在所有仓库中执行 shell 命令 |
| `fleet ide-setup idea` | 生成 IntelliJ/GoLand VCS 映射配置 |

所有命令均支持 `-g <expr>` 按 group 过滤（`,` 表示 OR，`+` 表示 AND）。

## 典型工作流

```bash
fleet init                          # 克隆所有仓库
fleet sync                          # 同步上游最新代码
fleet start feature/my-feature      # 在所有仓库创建 feature 分支
# ... 进行开发 ...
fleet push                          # 推送到 fork
fleet pr -t "feat: my feature"      # 创建 PR
fleet finish feature/my-feature     # 合并后清理分支
```

## Manifest 配置

Fleet 在工作区根目录使用两个 XML 文件：

| 文件 | 用途 |
|------|------|
| `fleet.xml` | 团队共享配置（提交到 Git）|
| `local_fleet.xml` | 个人本地覆盖——fork remote、私有仓库（添加到 .gitignore）|

当两个文件同时存在时，`local_fleet.xml` 会合并到 `fleet.xml` 中：

- **Remote** — 同名替换；新增追加
- **Default** — 逐属性覆盖
- **Project** — 同名逐属性覆盖；新增追加

完整注释示例见 [docs/example-fleet.xml](docs/example-fleet.xml)。

### 关键属性

| 元素 | 属性 |
|------|------|
| `<remote>` | `name`、`fetch`、`review` |
| `<default>` | `remote`、`revision`、`sync-j`、`push`、`master-main-compat` |
| `<project>` | `name`、`path`、`groups`、`remote`、`revision`、`push` |

在 `<default>` 上设置 `master-main-compat="true"` 可在 `master` 和 `main` 之间自动回退。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `FLEET_MANIFEST` | 主 manifest 文件路径 | `<workspace>/fleet.xml` |
| `FLEET_LOCAL_MANIFEST` | 本地 manifest 文件路径 | `<workspace>/local_fleet.xml` |

## 文档

- [English User Guide](docs/usage-en.md)
- [中文使用说明](docs/usage-zh.md)
- [Manifest 示例](docs/example-fleet.xml)

## 许可证

MIT
