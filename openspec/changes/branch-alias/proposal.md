## Why

目前 fleet 只内置了 `master` ↔ `main` 的等价处理（`master-main-compat` 布尔开关 + 硬编码 peer），无法支持团队中其它成对的环境分支。例如 `testing` 与 `testing-incy` 是两套并存的测试环境分支：不同 repo 只存在其中之一。执行 `fleet start testing-incy` 时，缺少 `testing-incy` 的 repo 会失败，无法自动回退到 `testing`。需要一种**可配置、可扩展**的分支别名机制，把任意一组分支视为"同名"并按顺序回退。

## What Changes

- **新增 manifest 配置 `<branch-alias>` 元素**：一组别名一条配置，组成员以 `<branch>` 子元素逐个列出（例如一个 `<branch-alias>` 内含 `<branch>testing</branch>` 与 `<branch>testing-incy</branch>`）。同组成员互为**对等等价别名**（无主从）。可配置多条，互不影响。
- **泛化分支回退解析**：当某分支在某个 repo 不存在时，查其所属别名组，回退到组内第一个真实存在的远端分支（被请求的分支自身优先，其余成员按声明顺序作确定性 tie-break）。
- **`fleet start <branch>` 增强**：当 target 分支属于某个别名组时，建分支的 base 起点从该别名组解析（如 `testing-incy` 缺失则用 `testing`），而非固定使用 `revision`。不属于任何别名组的普通分支行为**完全不变**（仍基于 `revision` 创建）。
- **统一回退入口**：`sync` / `pr` / `finish` / `prune` / `worktree` 中现有的 revision 解析改为复用别名组回退逻辑，保持全局一致。
- **向后兼容**：保留 `master-main-compat="true"`，其语义等价于内置一组 `master,main` 别名；与显式 `<branch-alias>` 配置并存。
- 更新文档与示例 manifest。

## Capabilities

### New Capabilities

- `branch-alias`: 在 manifest 中配置一组或多组分支别名，使组内分支被视为"同名"；在 `start` 建分支以及各命令解析 revision 时，按别名组顺序回退到存在的分支。涵盖配置解析/合并、回退解析算法、`fleet start` target 别名行为，以及与 `master-main-compat` 的兼容关系。

### Modified Capabilities

<!-- openspec/specs/ 当前为空，无已落档的 capability spec，故无需修改既有 spec。 -->

## Impact

- **配置与模型**：`internal/manifest/types.go`（新增 `<branch-alias>` 解析结构与 `ResolvedProject` 别名组字段）、`internal/manifest/resolve.go`（解析别名组并注入到每个 project）、`internal/manifest/merge.go`（多 manifest 合并时的别名组合并策略）。
- **解析逻辑**：`cmd/revision.go`（`resolveRevision` 泛化为基于别名组回退，`masterMainPeer` 改为内置别名组）。
- **命令行为**：`cmd/start.go`（target 属于别名组时的 base 起点解析）；`cmd/sync.go`、`cmd/pr.go`、`cmd/finish.go`、`cmd/prune.go`、`cmd/worktree.go`（透传别名组配置）。
- **文档/示例**：`docs/usage-zh.md`、`docs/usage-en.md`、`README.md`、`fleet.xml` / `docs/example-fleet.xml`。
- **兼容性**：保留 `master-main-compat` 属性，无破坏性变更。
