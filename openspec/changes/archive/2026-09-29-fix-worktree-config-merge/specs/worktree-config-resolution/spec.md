## ADDED Requirements

### Requirement: Local manifest overrides worktree defaults
合并 `local_fleet.xml` 与 `fleet.xml` 时，系统 SHALL 对 `<default>` 的 `worktree-base` 与 `worktree-copy` 属性执行与其余 `<default>` 属性一致的逐属性覆盖：local manifest 中该属性非空时 SHALL 覆盖 base manifest 的取值；为空或未声明时 SHALL 保留 base manifest 的取值。当 base manifest 没有 `<default>` 元素而 local manifest 有时，local 的这两个属性 SHALL 同样生效。

#### Scenario: Local sets worktree-base when base manifest has none
- **WHEN** `fleet.xml` 的 `<default>` 未声明 `worktree-base`，`local_fleet.xml` 的 `<default>` 声明 `worktree-base="./worktrees/fleet"`
- **THEN** 解析结果的 worktree base SHALL 为 `./worktrees/fleet` 所对应的值，而非空

#### Scenario: Local overrides base worktree-base
- **WHEN** `fleet.xml` 声明 `worktree-base="~/worktrees/team"`，`local_fleet.xml` 声明 `worktree-base="~/worktrees/mine"`
- **THEN** 解析结果的 worktree base SHALL 来自 local manifest 的 `~/worktrees/mine`

#### Scenario: Local omits worktree-base
- **WHEN** `fleet.xml` 声明 `worktree-base="~/worktrees/team"`，`local_fleet.xml` 的 `<default>` 未声明 `worktree-base`
- **THEN** 解析结果的 worktree base SHALL 保留 `~/worktrees/team`

#### Scenario: Local overrides default worktree-copy
- **WHEN** `fleet.xml` 声明 `worktree-copy=".env"`，`local_fleet.xml` 声明 `worktree-copy=".env,.env.*,config.local.yaml"`
- **THEN** 所有未单独声明 `worktree-copy` 的 project SHALL 使用 local manifest 的模式列表

#### Scenario: Base manifest has no default element
- **WHEN** `fleet.xml` 不含 `<default>` 元素，`local_fleet.xml` 的 `<default>` 声明 `worktree-base` 与 `worktree-copy`
- **THEN** 解析结果 SHALL 采用 local manifest 声明的这两个取值

### Requirement: Local manifest overrides project-level worktree-copy
合并同名 `<project>` 时，系统 SHALL 对 `worktree-copy` 属性执行与 `path` / `groups` / `remote` / `revision` / `push` 一致的逐属性覆盖：local manifest 中该属性非空时 SHALL 覆盖 base manifest 中同名 project 的取值；为空或未声明时 SHALL 保留 base manifest 的取值。project 级取值非空时 SHALL 优先于 `<default>` 的 `worktree-copy`。

#### Scenario: Local overrides an existing project's worktree-copy
- **WHEN** `fleet.xml` 中 project `api.git` 声明 `worktree-copy=".env"`，`local_fleet.xml` 中同名 project 声明 `worktree-copy=".env,secrets.local"`
- **THEN** 该 project 的拷贝模式 SHALL 为 `.env` 与 `secrets.local`

#### Scenario: Local omits project worktree-copy
- **WHEN** `fleet.xml` 中 project `api.git` 声明 `worktree-copy=".env"`，`local_fleet.xml` 中同名 project 仅覆盖 `revision`
- **THEN** 该 project 的拷贝模式 SHALL 保留 `.env`

#### Scenario: Project-level value wins over default
- **WHEN** 合并后 `<default>` 的 `worktree-copy` 为 `.env`，project `api.git` 的 `worktree-copy` 为 `config.local.yaml`
- **THEN** 该 project 的拷贝模式 SHALL 为 `config.local.yaml`，不包含 `.env`

### Requirement: Worktree base path resolves against workspace root
系统 SHALL 将 `worktree-base` 解析为绝对路径：以 `~` 开头的路径 SHALL 展开为用户主目录；已是绝对路径的 SHALL 原样保留；其余相对路径 SHALL 以 **workspace root**（即所采用的 `fleet.xml` 所在目录）为基准解析。解析结果 SHALL NOT 依赖执行命令时的进程工作目录。`fleet worktree --dest` 传入的路径 SHALL 遵循同一解析规则。

#### Scenario: Relative worktree-base resolves against workspace root
- **WHEN** workspace root 为 `/w`，`worktree-base` 为 `./worktrees/fleet`
- **THEN** 解析后的 worktree base SHALL 为 `/w/worktrees/fleet`

#### Scenario: Resolution is independent of current working directory
- **WHEN** workspace root 为 `/w`，`worktree-base` 为 `./worktrees/fleet`，且在子目录 `/w/services/api` 下执行 `fleet worktree feat-x`
- **THEN** 解析后的 worktree base SHALL 仍为 `/w/worktrees/fleet`，与在 `/w` 下执行的结果一致

#### Scenario: Absolute and tilde paths unchanged
- **WHEN** `worktree-base` 为 `/abs/worktrees` 或 `~/worktrees/fleet`
- **THEN** 解析结果 SHALL 分别为 `/abs/worktrees` 与主目录下的 `worktrees/fleet`，与既有行为一致

#### Scenario: Relative --dest resolves against workspace root
- **WHEN** workspace root 为 `/w`，在任意工作目录下执行 `fleet worktree --dest ./tmp/wt -b feat-x`
- **THEN** worktree 根目录 SHALL 为 `/w/tmp/wt`

### Requirement: Worktree target paths use a single absolute basis
为每个 project 创建 worktree 时，系统 SHALL 使用同一个绝对目标路径 `<worktree-root>/<project.path>` 完成全部操作：已存在判定、父目录创建、`git worktree add` 的目标实参、以及 `worktree-copy` 的拷贝目标。系统 SHALL NOT 将相对路径交给 git，以避免其按各 project 仓库目录二次解析而产生嵌套在仓库内部的 worktree。

#### Scenario: Non-root project worktree lands under worktree root
- **WHEN** workspace root 为 `/w`，`worktree-base` 为 `./worktrees/fleet`，project `api.git` 的 path 为 `services/api`，执行 `fleet worktree feat-x`
- **THEN** 该 project 的 worktree SHALL 位于 `/w/worktrees/fleet/feat-x/services/api`
- **AND** `/w/services/api` 仓库内部 SHALL NOT 出现任何新建的 worktree 目录

#### Scenario: Re-running an existing worktree is skipped
- **WHEN** `/w/worktrees/fleet/feat-x/services/api` 的 worktree 已存在，再次执行 `fleet worktree feat-x`
- **THEN** 该 project SHALL 被标记为 skipped，而非失败

#### Scenario: worktree-copy files land in the worktree
- **WHEN** project `api.git` 配置 `worktree-copy=".env"` 且 `/w/services/api/.env` 存在，执行 `fleet worktree feat-x`
- **THEN** 该文件 SHALL 被拷贝到 `/w/worktrees/fleet/feat-x/services/api/.env`

#### Scenario: Root project unaffected
- **WHEN** 存在 path 为 `.` 的 root project，执行 `fleet worktree feat-x`
- **THEN** 其 worktree SHALL 位于 `/w/worktrees/fleet/feat-x`，与修复前的正确行为一致
