## Why

`fleet worktree` 的配置在两个环节被破坏，导致该命令在常见配置下静默产出错误结果：

1. `local_fleet.xml` 中所有 worktree 相关属性（`default/worktree-base`、`default/worktree-copy`、`project/worktree-copy`）在 manifest 合并时被整体丢弃——`internal/manifest/merge.go` 的 `mergeDefault` / `mergeProjects` 没有为这三个字段写覆盖分支，而 `internal/manifest/types.go` 里字段是存在的。使用者在 local manifest 里配置个人 worktree 路径完全无效，且无任何提示。
2. `worktree-base` 只做了 `~` 展开（`internal/workspace/workspace.go` 的 `ExpandHome`），相对路径原样透传。后续 Go 侧的 `os.Stat` / `os.MkdirAll` 按**进程 CWD** 解析，git 侧的 `git worktree add` 按**各项目仓库目录**解析，两个基准不一致，非根项目的 worktree 被创建到该项目仓库内部，而命令仍报告成功。

## What Changes

- `mergeDefault` 补齐 `worktree-base`、`worktree-copy` 的逐属性覆盖，语义与既有字段一致：local 非空则覆盖 base
- `mergeProjects` 补齐 project 级 `worktree-copy` 的逐属性覆盖
- `worktree-base` 与 `--dest` 的相对路径统一以 **workspace root**（`fleet.xml` 所在目录）为基准解析为绝对路径，不再依赖进程 CWD
- `worktree` 命令内部的目标路径（存在性检查、`MkdirAll`、`worktree-copy` 拷贝目标、`git worktree add` 实参）统一使用同一个绝对路径，消除双基准
- 不改动任何 CLI 接口、flag 或 XML schema；完全向后兼容（原本能工作的绝对路径 / `~` 路径行为不变）

## Capabilities

### New Capabilities
- `worktree-config-resolution`: worktree 配置的解析契约——local manifest 对 `worktree-base` / `worktree-copy` 的覆盖规则，以及 worktree 目标路径相对 workspace root 的解析规则与 CWD 无关性

### Modified Capabilities

<!-- openspec/specs/ 当前为空，无既有 spec 需要改 -->

## Impact

- **代码变更**：
  - `internal/manifest/merge.go`：`mergeDefault` 增加 `WorktreeBase` / `WorktreeCopy` 覆盖分支；`mergeProjects` 增加 `WorktreeCopy` 覆盖分支
  - `internal/workspace/workspace.go`：新增相对路径解析能力（以 workspace root 为基准），`Load` 中对 `worktreeBase` 应用
  - `cmd/worktree.go`：`--dest` 走同一解析路径；`worktreeProject` 中目标路径保证为绝对路径
- **CLI 接口**：无变更
- **XML schema**：无变更
- **依赖**：无新依赖
- **向后兼容**：是。修复前相对路径产出的目录结构本就是错误的（worktree 落在业务仓库内部），不构成需要保留的既有行为；已有的绝对路径与 `~` 配置行为完全不变
- **不在本次范围**：`mergeRemotes` 的整体替换语义、`<branch-alias>` 组无法增删成员——另行处理
