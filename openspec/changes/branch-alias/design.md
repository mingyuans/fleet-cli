## Context

fleet 当前通过 `cmd/revision.go` 的 `resolveRevision` + `masterMainPeer` 实现 `master`↔`main` 的回退：当 `master-main-compat="true"` 且配置的 `revision` 不存在时，回退到硬编码的 peer。该机制：

- 仅支持 `master`/`main` 一对，且写死在代码里，无法扩展到 `testing`/`testing-incy` 等其它环境分支组。
- 只作用于 `revision`（建分支 base），不覆盖 `fleet start <branch>` 中 target 分支本身的回退。

`fleet pr` 已有 `--base "a|b"` 形式的运行时有序回退（`parseBranchCandidates` / `resolveBaseFromCandidates`），是本设计可复用的先例，但它是命令行级别、一次性的，不是 manifest 级别的持久化配置。

`resolveRevision` 的调用方：`cmd/start.go`、`cmd/sync.go`、`cmd/pr.go`、`cmd/finish.go`、`cmd/prune.go`、`cmd/worktree.go`。

约束：用户明确要求"一组别名一条配置"，即每个别名组对应独立的配置项，不把多组挤进单个属性。

## Goals / Non-Goals

**Goals:**
- 在 manifest 中以"一组一条" `<branch-alias>`（成员为 `<branch>` 子元素）的形式声明任意多个别名组。
- 提供统一的别名回退解析函数，供 `start`（target 分支）与各命令（revision）复用。
- `fleet start <branch>`：target 属于别名组时基于别名组解析 base 起点；非别名分支行为零变化。
- 完全向后兼容 `master-main-compat`，无破坏性变更。

**Non-Goals:**
- 不改变 `fleet pr --base` 的语义（`--base` 显式指定时不叠加别名回退）。
- 不引入别名组的远端自动发现/推断；别名组完全由配置驱动。
- 不支持别名组的嵌套或传递闭包合并（同一分支落入多组时按确定顺序取第一组，不做并集）。

## Decisions

### 决策 1：配置形态 —— 顶层 `<branch-alias>`，组成员用 `<branch>` 子元素

```xml
<manifest>
  <remote name="github" fetch="..." />
  <default remote="github" revision="master" master-main-compat="true" />
  <branch-alias>
    <branch>testing</branch>
    <branch>testing-incy</branch>
  </branch-alias>
  <branch-alias>
    <branch>staging</branch>
    <branch>staging-incy</branch>
  </branch-alias>
  <project ... />
</manifest>
```

**语义定位：别名组是"对等等价类"，而非有方向的优先级列表。** 同一个 `<branch-alias>` 内的所有 `<branch>` 互为等价别名（与 `master`/`main` 一样无主从）。回退时的"优先"只来自**当前被请求/配置的那个分支自身优先**；组内成员之间对等，声明顺序仅在「同一次请求需要在多个其它成员间做选择」时充当确定性 tie-break，不表达业务优先级。

- 每个成员是独立的 `<branch>` 子元素，解析为 `Branches []string`：天然结构化、免去逗号 split 与空段/空白处理的脆弱性、可读性强、易扩展到 3+ 成员。
- 解析时 trim 每个成员并丢弃空白成员；有效成员 < 2 的组忽略。
- 放在顶层 `<manifest>`（与 `<remote>`/`<project>` 同级）：别名组是全局概念且"一组一条"，与现有顶层列表元素风格一致，也避免 `<default>` 属性膨胀。

**Alternatives considered:**
- `<branch-alias branches="testing-incy,testing" />`（逗号字符串属性）：**被否决**。逗号顺序会误导读者以为是有方向的 fallback 优先级，掩盖"对等等价"的真实语义；且需二次解析、对空段/空白脆弱。
- `<default branch-aliases="master,main;testing,testing-incy" />`：被用户否决（多组挤在一个属性，可读性差）。
- `<branch-alias canonical="testing"><alias>testing-incy</alias></branch-alias>`（规范分支 + 别名的主从模型）：表达力更强，但引入"谁是规范分支"的决策负担，且与 `master`/`main` 的对等语义不符，故不采用。

### 决策 2：数据模型与解析

- `types.go` 新增：
  ```go
  type BranchAlias struct {
      Branches []string `xml:"branch"`  // 每个 <branch> 子元素一个成员
  }
  // Manifest 增加: BranchAliases []BranchAlias `xml:"branch-alias"`
  // ResolvedProject 增加: AliasGroups [][]string  // 规范化后的别名组（对等等价类）
  ```
- `resolve.go`：把 manifest 的 `BranchAliases` 规范化成 `[][]string`（trim 每个成员、去空、丢弃有效成员 < 2 的组）；若 `master-main-compat="true"` 且没有任何显式组包含 master/main，则**注入内置组** `["master","main"]`。每个 `ResolvedProject` 共享同一份别名组切片（别名是全局配置，与具体 project 无关）。

**Alternatives considered:** 把别名组放在 `Workspace` 而非每个 `ResolvedProject`。但现有命令以 `ResolvedProject` 为处理单元（`executor.Run` 的回调签名），随 project 携带改动面最小。

### 决策 3：统一回退函数

新增（`cmd/revision.go`）：
```go
// resolveBranchWithAliases 返回 remote 上实际可用的分支。
// 先尝试 branch 本身；不存在则按 branch 所属别名组顺序回退。
func resolveBranchWithAliases(dir, remote, branch string, groups [][]string) string
```
- 候选构造：`[branch] + (branch 所属组中除 branch 外的成员，按配置顺序)`，去重。这样无论配置内顺序如何，用户显式请求的 `branch` 永远优先。
- `resolveRevision(dir, remote, revision, groups)` 改为内部调用它（移除旧的 `masterMainCompat bool` 参数与 `masterMainPeer`，master/main 通过"内置组"承载）。
- 一个分支至多归属一个别名组：查找时返回第一个包含它的组（确定性由 `resolve.go` 内组的顺序保证）。

**Alternatives considered:** 保留 `masterMainPeer` 并在其上叠加别名查询。冗余且双轨，统一为别名组更清晰。

### 决策 4：`fleet start` 的 base 起点

`startProject` 在"本地分支不存在 → fetch 之后"分两路：
- `branch`（args[0]）属于某别名组 → `baseBranch = resolveBranchWithAliases(projDir, remote, branch, groups)`；失败提示尝试过的候选。
- 否则 → 维持现状 `baseBranch = resolveRevision(projDir, remote, proj.Revision, groups)`。

新建的本地分支名始终是用户输入的 `branch`，仅 base 起点受别名影响。

### 决策 5：调用方改造

各调用方将原 `proj.MasterMainCompat` 实参替换为 `proj.AliasGroups`（透传别名组）。`pr.go` 在 `--base` 显式指定时**不**调用别名回退（保持 `pr-target-branch` 行为）；未指定 `--base` 的默认路径仍走 `resolveRevision`，自动获得别名能力。

### 决策 6：多 manifest 合并

`merge.go` 新增 `mergeBranchAliases`：以规范化后的成员集合为 key，local 同 key 覆盖、新 key 追加（与 `mergeRemotes` 思路一致）。`master-main-compat` 仍按属性覆盖。

## Risks / Trade-offs

- [一个分支被配置进多个别名组导致回退歧义] → 规范：取第一个匹配组；在文档中说明，并在 `resolve.go` 保持稳定顺序。
- [向后兼容：旧 manifest 仅有 `master-main-compat`] → 通过"内置 master,main 组"注入，行为完全一致；新增针对该场景的测试。
- [`fleet start` target 别名改变了部分用户对"总是从 master 建分支"的预期] → 仅当 target **恰好**是别名组成员时才改变；普通 feature 分支不受影响，文档明确说明。
- [`--base` 与别名回退的叠加歧义] → 明确 Non-Goal：`--base` 不叠加别名，由 `|` 自行控制；新增回归场景验证。

## Migration Plan

- 纯增量、无破坏性：未声明 `<branch-alias>` 且未开启 `master-main-compat` 的 manifest 行为不变。
- 重构 `resolveRevision` 签名（去掉 bool、加 groups）属内部调用方同步修改，无对外接口。
- 回滚：移除 `<branch-alias>` 配置即恢复旧行为；代码层可独立 revert。

## Open Questions

- 是否需要为 `<branch-alias>` 增加可选 `name` 属性以增强可读性与合并去重？当前以成员集合为 key，暂不需要；若后续别名组增多可再加。
