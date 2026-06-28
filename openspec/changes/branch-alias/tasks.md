## 1. 配置模型与解析

- [x] 1.1 在 `internal/manifest/types.go` 新增 `BranchAlias` 结构（成员 `Branches []string` 映射 `<branch>` 子元素，tag `xml:"branch"`），并在 `Manifest` 增加 `BranchAliases []BranchAlias` 字段（`xml:"branch-alias"`）
- [x] 1.2 在 `ResolvedProject` 增加 `AliasGroups [][]string` 字段（规范化后的别名组，组内成员为对等等价别名）
- [x] 1.3 在 `internal/manifest/resolve.go` 实现别名组规范化：trim 每个 `<branch>` 成员、丢弃空白成员、忽略有效成员 < 2 的组
- [x] 1.4 在 `resolve.go` 中当 `master-main-compat="true"` 且无显式组覆盖 master/main 时注入内置组 `["master","main"]`，并将 `AliasGroups` 注入每个 `ResolvedProject`
- [x] 1.5 在 `internal/manifest/merge.go` 新增 `mergeBranchAliases`（以成员集合为 key：同 key 覆盖、新 key 追加），并接入 `Merge`

## 2. 统一别名回退解析

- [x] 2.1 在 `cmd/revision.go` 新增 `resolveBranchWithAliases(dir, remote, branch, groups)`：先试 branch 本身，否则按其所属别名组顺序（branch 优先在前、去重）回退到第一个存在的远端分支
- [x] 2.2 重构 `resolveRevision` 签名为基于 `groups` 回退（移除 `masterMainCompat bool` 参数），并删除/替换 `masterMainPeer`
- [x] 2.3 新增 `branchAliasGroup(branch, groups)` 辅助函数：返回包含该分支的第一个别名组（确定性）

## 3. `fleet start` target 别名行为

- [x] 3.1 在 `cmd/start.go` 的 `startProject` 中，当 `branch`（target）属于某别名组时，base 起点改用 `resolveBranchWithAliases`；否则保持 `resolveRevision(revision)`
- [x] 3.2 处理失败提示：别名组成员均不存在时输出已尝试的候选分支；新建本地分支名始终为用户输入的 `branch`

## 4. 各调用方透传别名组

- [x] 4.1 更新 `cmd/start.go`、`cmd/sync.go`、`cmd/finish.go`、`cmd/prune.go`、`cmd/worktree.go` 调用 `resolveRevision` 处，传入 `proj.AliasGroups`
- [x] 4.2 更新 `cmd/pr.go`：默认路径（无 `--base`）透传 `AliasGroups`；确认 `--base` 显式指定时**不**叠加别名回退

## 5. 测试

- [x] 5.1 `internal/manifest` 单测：`<branch-alias>` 解析、规范化（trim/空段/单成员忽略）、master-main-compat 注入、多 manifest 合并
- [x] 5.2 `cmd/revision` 单测：`resolveBranchWithAliases` 各分支（命中本体/回退成员/全不存在/不在任何组）
- [x] 5.3 `cmd/start` 测试：target 别名命中、回退、全缺失、非别名分支行为不变、本地已存在分支直接切换（参考 `cmd/pr_test.go` 的本地仓库构造方式）
- [x] 5.4 `cmd/pr` 回归：`--base` 显式指定时不自动回退到同组成员
- [x] 5.5 运行 `make test`（或 `go test ./...`）确保全绿

## 6. 文档与示例

- [x] 6.1 更新 `docs/usage-zh.md`、`docs/usage-en.md`：`<branch-alias>` 配置说明、`fleet start` 别名行为、与 `master-main-compat` 关系
- [x] 6.2 在 `fleet.xml` / `docs/example-fleet.xml` 增加 `<branch-alias>` 示例
- [x] 6.3 更新 `README.md` 相关章节
