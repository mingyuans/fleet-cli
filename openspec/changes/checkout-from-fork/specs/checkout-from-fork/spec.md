## ADDED Requirements

### Requirement: 命令与可选 --from 参数

系统 SHALL 提供 `fleet checkout <branch> [--from <user>]` 命令，其中 `<branch>` 为位置参数（必填），`--from <user>` 为可选 flag。当**未提供** `--from` 时，命令行为 MUST 与 `fleet start <branch>` 一致（从 upstream 默认分支创建/切换本地分支）。当**提供** `--from <user>` 时，命令 MUST 走协作者 fork 切换路径。

#### Scenario: 不带 --from 时退化为 start 行为
- **WHEN** 用户执行 `fleet checkout feature/x`（未提供 `--from`）
- **THEN** 系统对每个仓库执行与 `fleet start feature/x` 相同的逻辑，从 upstream 默认分支创建或切换本地分支

#### Scenario: 带 --from 时走协作者 fork 路径
- **WHEN** 用户执行 `fleet checkout feature/x --from alice`
- **THEN** 系统为每个仓库添加/复用 `alice` remote 并切换到 `alice/feature/x`

#### Scenario: 缺少 branch 位置参数
- **WHEN** 用户执行 `fleet checkout --from alice` 且未提供分支名
- **THEN** 系统返回参数错误，提示需要 `<branch>` 参数

### Requirement: 协作者 fork remote 的添加与复用

提供 `--from <user>` 时，系统 SHALL 为每个待处理仓库确保存在指向协作者 fork 的 git remote，remote 名为 `<user>`。fork remote 的 URL MUST 基于该仓库的 fetch remote URL，将 owner 段替换为 `<user>` 并保留原协议形式（SSH 或 HTTPS）得到。该 remote MUST 在命令结束后保留，不被清理。

#### Scenario: 协作者 remote 不存在时添加
- **WHEN** 仓库尚未配置名为 `<user>` 的 remote
- **THEN** 系统按 fetch remote URL 派生协作者 fork URL 并执行 `git remote add <user> <forkURL>`

#### Scenario: 协作者 remote 已存在时复用
- **WHEN** 仓库已存在名为 `<user>` 的 remote 且 URL 与期望一致
- **THEN** 系统直接复用该 remote，不重复添加

#### Scenario: 命令结束后保留 remote
- **WHEN** checkout 命令执行完成
- **THEN** 协作者 `<user>` remote 仍保留在各仓库中，供后续 `fleet sync` 等命令跟踪

#### Scenario: fetch remote URL 无法解析 owner
- **WHEN** 仓库 fetch remote URL 无法解析出 owner 段
- **THEN** 该仓库标记为 failed 并给出清晰原因，其它仓库继续处理

### Requirement: 基于 upstream 来源的分支切换

提供 `--from <user>` 时，系统 SHALL 对每个待处理仓库 fetch 协作者 remote 后，切换到目标来源 `<user>/<branch>`。是否「已在目标」MUST 依据**当前分支的 upstream 来源**判定，而非仅依据分支名。当本地已存在与 `<branch>` 同名但跟踪不同来源的分支时，系统 MUST 使用 fork 限定本地分支名 `<user>/<branch>` 进行切换，以避免覆盖本地原有分支。

#### Scenario: 本地无同名分支时创建并切换
- **WHEN** 仓库已 clone、本地无 `<branch>` 分支、且协作者 remote 上存在 `<branch>`
- **THEN** 系统基于 `refs/remotes/<user>/<branch>` 创建本地分支 `<branch>` 并切换，结果记为 checked out

#### Scenario: 本地同名分支已跟踪目标来源
- **WHEN** 本地已存在 `<branch>` 分支且其 upstream 等于 `<user>/<branch>`
- **THEN** 系统直接切换到该本地分支，结果记为 switched

#### Scenario: 同名分支但来源不同（如当前在 origin/testing 要切到 user/testing）
- **WHEN** 本地已存在 `<branch>` 分支但其 upstream 不等于 `<user>/<branch>`
- **THEN** 系统使用 fork 限定本地分支名 `<user>/<branch>`：若已存在则切换，否则基于 `refs/remotes/<user>/<branch>` 创建并切换，且不覆盖本地原有 `<branch>` 分支

#### Scenario: 已在目标来源分支
- **WHEN** 当前分支已是目标本地分支且其 upstream 已等于 `<user>/<branch>`
- **THEN** 系统跳过该仓库，结果记为 skipped（already tracking）

#### Scenario: 协作者 fork 中不存在该分支
- **WHEN** fetch 后协作者 remote 上不存在 `<branch>` 对应的远端引用
- **THEN** 系统跳过该仓库（skip），不报错，且不影响其它仓库的处理

#### Scenario: 仓库尚未 clone
- **WHEN** 仓库目录不存在
- **THEN** 系统跳过该仓库，结果记为 skipped（not cloned）

### Requirement: 分组过滤与批量执行

系统 SHALL 支持 `-g <expr>` 分组过滤，行为与其它 fleet 命令一致（`,` 表示 OR，`+` 表示 AND）。命令 MUST 跨所有匹配仓库并发执行，并在结束时按 `checked out` / `switched` / `skipped` / `failed` 分类输出汇总结果。

#### Scenario: 使用分组过滤
- **WHEN** 用户执行 `fleet checkout feature/x --from alice -g core`
- **THEN** 系统仅对属于 `core` 分组的仓库执行 checkout 操作

#### Scenario: 汇总输出
- **WHEN** checkout 命令在多个仓库上执行完成
- **THEN** 系统输出各状态（checked out / switched / skipped / failed）的计数汇总
