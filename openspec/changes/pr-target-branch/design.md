## Context

`fleet pr` 当前的工作流程（`cmd/pr.go` 的 `prProject()` 函数）：

1. 调用 `pushPreflight()` 校验项目状态，获取当前分支和 push remote
2. `git push` 到 push remote
3. 解析 upstream repo（从 `proj.CloneURL` 提取 `owner/repo`）
4. 解析 base branch：默认使用 `proj.Revision`，通过 `resolveRevision()` 在 fetch remote 上确认分支存在，支持 master/main 兼容
5. 调用 `gh pr create --repo <upstream> --base <baseBranch> --head <head> --title <title>` 创建 PR

关键约束：
- `resolveRevision()` 使用 `git.RemoteRefExists(dir, remote+"/"+revision)` 检查远程分支存在性，依赖 `refs/remotes/<remote>/<branch>` 的本地 ref（需要先 fetch）
- 当前 `fleet pr` 在 push 之前没有 fetch fetch-remote，但 `fleet sync` 已经 fetch 过，本地 ref 通常是最新的
- fork 模式下 `--head` 需要使用 `pushOwner:branch` 格式

## Goals / Non-Goals

**Goals:**
- `fleet pr --base <candidates>` 让用户指定 PR 目标分支
- 支持 `|` 分隔的候选分支列表，按优先级从左到右尝试
- 候选分支不存在于 fetch remote 时自动跳过，所有候选都不存在时 skip 该 project
- 不指定 `--base` 时保持现有行为完全不变

**Non-Goals:**
- 不修改 manifest 格式（不在 fleet.xml 中增加 base 配置）
- 不改变 push 行为（push 仍然推送到 push remote）
- 不支持正则或通配符匹配分支名

## Decisions

### 1. Flag 设计：`--base` / `-b`

新增 `--base` flag 接收一个字符串值，支持 `|` 分隔多个候选分支。

```
fleet pr --base "testing-incy|testing"
fleet pr -b testing
```

**理由**：`-b` 与 `git checkout -b`、`gh pr create --base` 的习惯一致，用户无需记忆新的 flag 名。

**替代方案**：
- `--target`：语义更明确，但 `-t` 已被 `--title` 占用，会导致短 flag 冲突
- 使用位置参数而不是 flag：语义不够明确，且不同于现有 CLI 风格

### 2. 候选分支解析策略

在 `prProject()` 中，当 `--base` 被指定时：

1. 按 `|` 分割候选分支列表
2. 先 fetch fetch-remote 以确保 ref 是最新的
3. 遍历候选分支，检查 `refs/remotes/<fetchRemote>/<branch>` 是否存在
4. 找到第一个存在的分支作为 PR 的 base branch
5. 所有候选都不存在 → skip 该 project，输出 "no matching base branch" 信息

**理由**：
- fetch 确保分支检查基于最新的远程状态，避免因本地 ref 过期导致误判
- 使用现有的 `git.RemoteRefExists()` 函数，无需引入新的 git 调用

**替代方案**：
- 使用 `git ls-remote` 直接查询远程分支：更精确但需要网络请求，对多分支场景性能更差
- 不做 fetch 直接用本地 ref：用户如果已 `fleet sync` 过通常没问题，但存在过期风险。考虑到 `fleet pr` 前通常会 `fleet sync`，这个风险可接受。最终决定：**先 fetch fetch-remote 再检查**，保证准确性

### 3. `--base` 与 `masterMainCompat` 的交互

当指定 `--base` 时，**不**应用 `masterMainCompat` 逻辑。用户显式指定了目标分支，应严格按用户输入处理，`|` 操作符已经提供了回退机制。

`masterMainCompat` 仅在默认模式（未指定 `--base`）下继续生效。

### 4. 实现位置

所有变更集中在 `cmd/pr.go`：
- 新增 `prBase` 包级变量和 flag 注册
- 新增 `resolveBaseFromCandidates(dir, fetchRemote string, candidates []string) string` 辅助函数
- 修改 `prProject()` 中 base branch 解析段：先检查 `prBase` 是否非空，是则走候选解析逻辑

不需要修改 `internal/git/git.go`，现有的 `RemoteRefExists()` 和 `Fetch()` 已满足需求。

### 5. 测试方案

**单元测试**：
- 测试 `resolveBaseFromCandidates()` 函数：
  - 单个候选且存在 → 返回该分支
  - 多个候选，第一个不存在第二个存在 → 返回第二个
  - 所有候选都不存在 → 返回空字符串
  - 空输入 → 返回空字符串

**集成测试（手动）**：
- 准备一个测试 repo，有 `main` 和 `testing` 分支
- `fleet pr --base testing` → 应创建到 testing 的 PR
- `fleet pr --base "nonexist|testing"` → 应跳过 nonexist，创建到 testing 的 PR
- `fleet pr --base "nonexist1|nonexist2"` → 应 skip，提示 no matching base branch
- `fleet pr`（无 --base） → 行为不变，仍然用 manifest revision

**边界场景测试**：
- `--base "|testing"` → 空字符串候选应被忽略，使用 testing
- `--base "testing|"` → 同上
- `--base "  testing  "` → 应 trim 空格

## Risks / Trade-offs

- **Fetch 开销**：指定 `--base` 时会额外 fetch fetch-remote，增加网络延迟 → 可接受，确保分支状态准确比速度更重要；且 `fleet pr` 本身已经执行 push（网络操作），多一个 fetch 影响不大
- **`-b` 短 flag 未来冲突**：`-b` 可能与未来的 flag 冲突 → 当前 pr 命令只有 `-t`（title），`-b` 是最自然的选择，未来如有冲突再调整
