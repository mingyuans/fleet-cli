## Context

fleet 现有的 remote 模型只有两类：fetch remote（团队 upstream，来自 `fleet.xml`）和 push remote（使用者自己的 fork，来自 `local_fleet.xml`）。`internal/manifest/resolve.go` 将其解析为 `ResolvedProject.CloneURL`（fetch 模板 + 仓库名）和 `PushURL`。

`cmd/start.go` 已经实现了「fetch remote → 解析 base → 创建/切换本地分支」的标准流程，并通过 `internal/executor` 做并发、`internal/output` 做聚合输出、`filterByGroup` 做分组过滤。本设计的命令复用这套既有基础设施：当不指定协作者时退化为 `start` 的行为；指定协作者时把「base 来源」换成「该协作者 fork 的指定分支」。

约束（已与用户确认）：协作者各仓库与主仓库同名；fork remote 保留不清理；支持 `-g` 分组过滤；缺失分支跳过；`--from` 为可选 flag。

## Goals / Non-Goals

**Goals:**
- 新增 `fleet checkout <branch> [--from <user>]`：
  - 不带 `--from` 时，行为与 `fleet start <branch>` 一致（从 upstream 默认分支创建/切换本地分支）
  - 带 `--from <user>` 时，跨仓库添加协作者 fork remote、fetch 并切换到 `<user>/<branch>`
- 复用现有 executor / output / filterByGroup / git 封装及 `start` 的处理逻辑，保持交互与输出风格一致
- fork remote 持久保留，使用者后续可用 `fleet sync` 等命令继续跟踪
- 缺失分支的仓库 skip，不阻断其它仓库
- skip 判定基于「当前分支的 upstream 是否已指向目标来源」，而非仅分支名

**Non-Goals:**
- 不支持协作者仓库与主仓库**异名**的映射（本期假设同名）
- 不自动清理协作者 remote（明确保留）
- 不修改 `start` / `sync` / `pr` 等现有命令的行为
- 不处理本地工作区脏状态的自动 stash（沿用 git 原生 checkout 行为）
- 不对本地已有分支做 `reset --hard` 等破坏性对齐（改用 fork 限定本地分支名规避覆盖）

## Decisions

### 决策 1：fork remote URL 的构造方式
带 `--from <user>` 时，基于 `ResolvedProject.CloneURL`（如 `git@github.com:my-org/user-service.git`）派生协作者 URL，把 owner 段替换为 `<user>`，保留原协议形式（SSH / HTTPS）。

- 复用 `internal/git/git.go` 的 `ParseRepoOwner` 解析出 host 与 `owner/repo`，取 `repo` 段，与 `<user>` 重新拼接为同协议 URL。
- 新增小 helper `DeriveForkURL(fetchURL, newOwner string) (string, bool)`，解析失败返回 `ok=false`。
- **理由**：保证协作者 fork 的协议与团队配置一致，避免凭证/协议不匹配；同名假设下 `repo` 段可直接复用。
- **备选**：直接用 manifest fetch 模板替换 owner——但模板里 owner 段位置不固定，解析既有 URL 更稳健。

### 决策 2：remote 命名与幂等
协作者 remote 名直接用 `<user>`（GitHub 用户名作为 remote 名）。

- 若 remote 不存在 → `RemoteAdd(dir, user, forkURL)`。
- 若 remote 已存在 → 视为可复用；URL 与期望不一致时用 `RemoteSetURL` 纠正（与 `init.go` 的 reconcile 风格一致）。
- **理由**：用户名作 remote 名直观、稳定、可被后续 `sync` 复用；幂等避免重复执行报错。

### 决策 3：`--from` 可选 + 基于 upstream 的切换语义
`--from` 为可选 flag。

**不带 `--from`（退化为 start）**：直接复用 `cmd/start.go` 的 `startProject` 逻辑，从 upstream 默认分支创建/切换本地分支，行为完全一致。

**带 `--from <user>`** 时单仓库处理流程：
1. 仓库未 clone → skip（`not cloned`）。
2. 确保协作者 remote 存在（决策 2）→ `Fetch(dir, user)`。
3. `RemoteRefExists(dir, user+"/"+branch)` 为 false → skip（`<user>/<branch> not found`）。
4. 确定本地分支名 `localBranch` 与是否需要切换——**关键：以 upstream 来源判定，而非仅分支名**：
   - 新增 helper `BranchUpstream(dir, branch) (string, bool)`（`git rev-parse --abbrev-ref <branch>@{upstream}`）。
   - 若本地不存在 `<branch>` → `localBranch = <branch>`，从 `refs/remotes/<user>/<branch>` 创建并切换（`checked out`）。
   - 若本地已存在 `<branch>`：
     - 其 upstream **等于** `<user>/<branch>` → 该本地分支已对应目标 fork，`localBranch = <branch>`，`CheckoutBranch` 切换（`switched`）。
     - 其 upstream **不等于** `<user>/<branch>`（例如 `origin/testing`，同名不同源）→ 为避免覆盖本地分支，使用 fork 限定本地分支名 `localBranch = <user>/<branch>`：已存在则切换（`switched`），否则从 `refs/remotes/<user>/<branch>` 创建并切换（`checked out`）。
5. skip 收敛条件：当前分支已是 `localBranch` 且其 upstream 已等于 `<user>/<branch>` → skip（`already tracking <user>/<branch>`）。

- **理由**：直接回应评审意见——「当前在 `origin/testing`，要切到 `mingyuans/testing`」时分支名相同但来源不同，必须切换；用 upstream 比较准确判定「是否已在目标」。fork 限定本地分支名让同名不同源场景也能真正切到协作者代码，且不破坏本地原有 `<branch>`（不丢未推送提交）。
- **备选**：对本地同名分支 `git checkout -B` 强制对齐到 fork ref——会丢弃本地未推送提交，风险高，弃用。

### 决策 4：命令编排
- `--from <user>` 为可选 flag；为空时分流到 `startProject`，非空时走协作者 fork 路径。
- 复用 `filterByGroup(ws.Projects)` 与全局 `groupFilter`、`executor.Run(projects, ws.SyncJ, ...)`、`output.Summary`。
- 结果状态分类：`checked out` / `switched` / `skipped` / `failed`。

## Risks / Trade-offs

- [协作者仓库异名] → 本期不支持，文档与错误信息中说明假设；异名映射留待后续迭代。
- [fork remote 残留累积] → 明确保留是需求；多个协作者会留下多个 remote，文档提示可按需手动 `git remote remove`。
- [URL 协议解析失败] → `DeriveForkURL` 返回 `ok=false` 时该仓库 `failed` 并给出清晰原因，不影响其它仓库。
- [本地同名分支与 fork 不同步] → 切换后看到的是 fork 分支当时 fetch 到的内容；使用者如需后续更新可 `fleet sync` 或 `git pull`，文档说明该行为。
- [fork 限定本地分支名与 remote-tracking ref 歧义] → 创建分支时显式以 `refs/remotes/<user>/<branch>` 作为 start point，checkout 时 git 优先本地 heads，规避歧义。
- [私有 fork 无访问权限] → fetch 失败该仓库 `failed`，错误信息透传 git stderr。
