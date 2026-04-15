## Why

当前 `fleet pr` 命令创建 PR 时，目标分支固定为 manifest 中配置的 `revision`（通常是 `main` 或 `master`）。在实际开发流程中，有些场景需要先将 PR 合并到 `testing` 等中间分支进行验证，而非直接合并到主分支。需要让 `fleet pr` 支持灵活指定 PR 的目标分支，并支持按优先级尝试多个候选分支。

## What Changes

- `fleet pr` 命令新增 `--base` flag，允许用户指定 PR 的目标分支名称
- 支持 `|` 分隔符指定多个候选分支，按从左到右的优先级依次尝试（如 `testing-incy|testing`）
- 对每个 project 的 fetch remote，检查候选分支是否存在：存在则向该分支创建 PR；所有候选均不存在则 skip 该 project 的 PR 创建
- 未指定 `--base` 时，保持现有行为不变（使用 manifest revision 作为目标分支）

## Capabilities

### New Capabilities
- `pr-target-branch`: 支持 `fleet pr --base <branch|branch|...>` 指定 PR 目标分支，带 `|` 分隔的回退机制

### Modified Capabilities

## Impact

- **代码变更**：
  - `cmd/pr.go`：新增 `--base` flag 定义；修改 `prProject()` 中 base branch 解析逻辑，增加候选分支遍历和 remote ref 存在性检查
  - `internal/git/git.go`：可能需要新增或复用 `RemoteRefExists()` 来检查 fetch remote 上的分支是否存在（现有函数已可用）
- **CLI 接口**：新增 `--base` / `-b` flag，不影响现有默认行为
- **依赖**：无新依赖
- **向后兼容**：完全向后兼容，未指定 `--base` 时行为不变
