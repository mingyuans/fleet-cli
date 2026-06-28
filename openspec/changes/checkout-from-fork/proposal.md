## Why

在多人协作的 Fork 工作流中，经常需要拉取**其他协作者** fork 仓库上的某个 feature 分支到本地，以便复现问题、调试代码或做 Code Review。当前 fleet 只建模了两个 remote：fetch remote（团队 upstream）和 push remote（使用者自己的 fork），没有任何命令能跨仓库添加并切换到「第三方协作者 fork」上的分支。使用者只能用 `fleet forall` 手动拼 URL、加 remote、fetch、checkout，过程繁琐且易错。

## What Changes

- 新增 `fleet checkout <branch> [--from <user>]` 命令
- `--from <user>` 为**可选** flag：
  - **不带 `--from`** 时，行为与 `fleet start <branch>` 一致（从 upstream 默认分支创建/切换本地分支）
  - **带 `--from <user>`** 时，为每个 project 添加（或复用）指向协作者 `<user>` fork 的 git remote，fetch 后切换到 `<user>/<branch>`
- 协作者各仓库与主仓库**同名**，fork remote 的 URL 按 fetch remote URL 替换 owner 段 + 保留原协议拼接得到
- 临时添加的协作者 fork remote **保留不清理**，使用者后续可通过 `fleet sync` 等命令继续跟踪
- 切换是否已完成的判定**基于当前分支的 upstream 来源**，而非仅分支名：当前在 `origin/testing` 要切到 `<user>/testing` 时，名同源不同，仍会切换
- 本地已存在同名但跟踪不同来源的分支时，使用 fork 限定本地分支名 `<user>/<branch>` 切换，避免覆盖本地原有分支
- 对于在协作者 fork 中**不存在该分支**的仓库，跳过处理（skip），不影响其它仓库，也不整体报错
- 支持 `-g <group>` 分组过滤，与其它 fleet 命令行为一致

## Capabilities

### New Capabilities
- `checkout-from-fork`: 支持 `fleet checkout <branch> [--from <user>]`——可选地跨仓库添加协作者 fork remote 并切换到其指定分支，缺省时退化为 start 行为，带分组过滤、基于 upstream 的切换判定与缺失分支跳过机制

### Modified Capabilities

## Impact

- **代码变更**：
  - `cmd/checkout.go`（新增）：定义 `checkout` 命令、可选 `--from` flag；为空时复用 `cmd/start.go` 的 `startProject`，非空时走协作者 fork 路径
  - `internal/git/git.go`：复用 `RemoteAdd` / `RemoteExists` / `RemoteSetURL` / `Fetch` / `RemoteRefExists` / `BranchExists` / `CheckoutBranch` / `CreateBranchFrom`；新增 `DeriveForkURL`（派生 fork URL）与 `BranchUpstream`（读取本地分支 upstream）
- **CLI 接口**：新增 `checkout` 子命令与可选 `--from` flag；不改动任何现有命令默认行为
- **依赖**：无新依赖
- **向后兼容**：完全向后兼容，纯新增命令
