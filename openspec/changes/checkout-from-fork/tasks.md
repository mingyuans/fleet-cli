## 1. git helper

- [x] 1.1 在 `internal/git/git.go` 新增 `DeriveForkURL(fetchURL, newOwner string) (string, bool)`：用 `ParseRepoOwner` 解析原 URL，取 `repo` 段与 `newOwner` 重建同协议（SSH/HTTPS）URL，解析失败返回 `ok=false`
- [x] 1.2 在 `internal/git/git.go` 新增 `BranchUpstream(dir, branch string) (string, bool)`：执行 `git rev-parse --abbrev-ref <branch>@{upstream}`，返回如 `origin/testing` 的 upstream 名；无 upstream 返回 `ok=false`
- [x] 1.3 为 `DeriveForkURL` 编写单元测试：SSH URL、HTTPS URL、无法解析的 URL 三类用例

## 2. checkout 命令骨架

- [x] 2.1 新增 `cmd/checkout.go`，定义 `checkoutCmd`（`Use: "checkout <branch>"`，`Args: cobra.ExactArgs(1)`），在 `init()` 中 `rootCmd.AddCommand`
- [x] 2.2 注册可选 `--from` flag（绑定包级变量 `checkoutFrom`）
- [x] 2.3 `runCheckout` 复用 `workspace.Load()`、`filterByGroup(ws.Projects)`、`groupFilter` 输出、`executor.Run(projects, ws.SyncJ, ...)`，并用 `output.Header` / `output.Summary`（状态列：`created` / `checked-out` / `switched` / `skipped` / `failed`）
- [x] 2.4 分流：`checkoutFrom == ""` 时对每个仓库调用 `cmd/start.go` 的 `startProject`；非空时调用新的 `checkoutForkProject`

## 3. 协作者 fork 处理逻辑（checkoutForkProject）

- [x] 3.1 仓库未 clone → skip（`not cloned`）
- [x] 3.2 用 `DeriveForkURL` 派生协作者 URL；解析失败 → failed（清晰原因）
- [x] 3.3 确保协作者 remote 存在：不存在则 `git.RemoteAdd(dir, user, forkURL)`；已存在但 URL 不一致则 `git.RemoteSetURL` 纠正（参考 `init.go` reconcile 风格）
- [x] 3.4 `git.Fetch(dir, user)`；失败 → failed（透传 git stderr）
- [x] 3.5 `git.RemoteRefExists(dir, user+"/"+branch)` 为 false → skip（`<user>/<branch> not found`）
- [x] 3.6 确定 `localBranch` 与切换方式（基于 upstream 判定）：
  - 本地无 `<branch>` → `localBranch = <branch>`，`CreateBranchFrom(localBranch, "refs/remotes/"+user+"/"+branch)`，结果 `checked-out`
  - 本地有 `<branch>` 且 `BranchUpstream == user+"/"+branch` → `localBranch = <branch>`，`CheckoutBranch`，结果 `switched`
  - 本地有 `<branch>` 且 upstream 不同 → `localBranch = user+"/"+branch`：已存在则 `CheckoutBranch`（`switched`），否则 `CreateBranchFrom(localBranch, "refs/remotes/"+user+"/"+branch)`（`checked-out`）
- [x] 3.7 收敛：当前分支已是 `localBranch` 且其 upstream 已等于 `user+"/"+branch` → skip（`already tracking`）

## 4. 单元测试

- [x] 4.1 为 `checkoutForkProject` / git helper 编写测试：`DeriveForkURL`（SSH/HTTPS/不可解析）、`BranchUpstream`（有/无 upstream）、未 clone 跳过（`TestCheckoutFromForkSkipsNotCloned`）。注：fork 成功 checkout 的端到端路径依赖 SSH/HTTPS 真实 URL，本地文件路径 remote 无法派生 fork URL，故由手动集成测试(6.x)覆盖
- [x] 4.2 验证不带 `--from` 时分流到 start 逻辑的行为（`TestCheckoutWithoutFromDelegatesToStart`）

## 5. 文档

- [x] 5.1 更新 `README.md` 命令表，新增 `fleet checkout <branch> [--from <user>]` 一行
- [x] 5.2 更新 `docs/usage-en.md` 与 `docs/usage-zh.md`，说明用途、可选 `--from`、同名假设、remote 保留行为、基于 upstream 的切换判定与缺失分支跳过语义

## 6. 手动集成测试

> 注：6.1 已由自动化测试 `TestCheckoutWithoutFromDelegatesToStart` 覆盖；命令/flag/分组注册已通过 `fleet checkout --help` 本地冒烟验证。6.2–6.6 需真实多仓库 + 协作者 SSH/HTTPS fork + 网络，留待使用者在真实 workspace 中验证。

- [x] 6.1 验证不带 `--from`：`fleet checkout feature/x` 与 `fleet start feature/x` 行为一致（自动化测试覆盖）
- [ ] 6.2 验证带 `--from`：`fleet checkout feature/x --from alice` 在各仓库添加 `alice` remote 并切换到 `alice/feature/x`（需真实 fork）
- [ ] 6.3 验证同名异源：当前在 `origin/testing` 时执行 `fleet checkout testing --from alice`，切到 fork 限定本地分支 `alice/testing`，原 `testing` 不受影响（需真实 fork）
- [ ] 6.4 验证缺失分支跳过：协作者 fork 无该分支的仓库被 skip，其它仓库正常 checkout（需真实 fork）
- [ ] 6.5 验证 remote 保留：命令结束后 `git remote -v` 仍含 `alice`（需真实 fork）
- [ ] 6.6 验证分组过滤：`fleet checkout feature/x --from alice -g core` 仅处理 core 分组（需真实 fork）
