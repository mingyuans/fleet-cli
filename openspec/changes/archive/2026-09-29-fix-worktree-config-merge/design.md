## Context

`fleet worktree` 依赖两段配置链路，两段都有缺陷。

**链路一：manifest 合并。** `workspace.Load` 读取 `fleet.xml` 与可选的 `local_fleet.xml`，经 `manifest.Merge` 合并后交给 `manifest.Resolve`。`Merge` 中 `mergeDefault`（`internal/manifest/merge.go:82-106`）与 `mergeProjects`（同文件 `:107-140`）采用「逐属性覆盖」——为每个可覆盖属性写一个 `if local.X != "" { result.X = local.X }` 分支。worktree 相关的三个属性在 `internal/manifest/types.go` 有字段定义（`:35` `WorktreeBase`、`:36` `Default.WorktreeCopy`、`:47` `Project.WorktreeCopy`），但在 merge 中没有对应分支，因此 local manifest 的取值被静默丢弃。这是纯粹的遗漏，不涉及设计取舍。

**链路二：路径解析。** `workspace.Load`（`internal/workspace/workspace.go:64`）仅对 `worktreeBase` 调用 `ExpandHome`，而 `ExpandHome`（`:93-101`）只处理 `~` 前缀，相对路径原样返回。`cmd/worktree.go:77` 用 `filepath.Join(ws.WorktreeBase, name)` 拼出的 `worktreeRoot` 因而可能是相对路径，`:131` 的 `wtPath` 随之相对。此后：

- Go 侧 `os.Stat`（`:135`）、`os.MkdirAll`（`:139`）、拷贝目标（`:181`）按**进程 CWD** 解析；
- git 侧 `git.WorktreeAdd` / `WorktreeAddNew` 经 `internal/git/git.go:266` 的 `run` 设置 `cmd.Dir = projDir`，由 git 按**该 project 的仓库目录**解析。

两个基准不同，非根 project 的 worktree 被创建进业务仓库内部，而 Go 侧只在 CWD 下造了一串空目录。首次执行报告成功，重跑则 git 报 `already exists`。已实测复现。

约束：不引入新依赖；不改 CLI 接口与 XML schema；保持既有绝对路径 / `~` 路径的行为不变。

## Goals / Non-Goals

**Goals:**
- `local_fleet.xml` 能覆盖 `default/worktree-base`、`default/worktree-copy`、`project/worktree-copy`，语义与同元素的其余属性完全一致
- worktree 目标路径的解析有唯一、明确的基准（workspace root），且与进程 CWD 无关
- Go 侧与 git 侧使用同一个绝对路径，消除双基准
- 上述行为有回归测试覆盖

**Non-Goals:**
- 不修 `mergeRemotes` 的整体替换语义（local `<remote>` 省略 `fetch` 会清空 base 的 `fetch`）
- 不修 `<branch-alias>` 组无法增删成员、只能整组替换的问题
- 不新增「worktree 列表 / 删除」等能力
- 不为 worktree base 落在 workspace 内部时产生的 git untracked 噪音做额外处理（属使用者配置选择）

## Decisions

### 决策一：相对路径以 workspace root 为基准，而非进程 CWD

`worktree-base` 来自 manifest，manifest 是 workspace 的描述文件；同一份配置在 workspace 内任何目录执行都应产出同一结果。以 CWD 为基准会让 `fleet worktree feat-x` 的结果随 shell 当前位置漂移——这正是当前 bug 的表现之一，不能作为修复后的语义。workspace root 取 `filepath.Dir(manifestPath)`，`Load` 中已有该值（`Root` 字段）。

**替代方案：**
- *以 CWD 为基准* —— 被否。与 manifest 的 workspace 级语义矛盾，且结果不可复现。
- *禁止相对路径，直接报错* —— 被否。`./worktrees/fleet` 是合理且常见的诉求（把 worktree 放在 workspace 旁边），报错只是把问题推给使用者。

### 决策二：在 `workspace.Load` 中做归一化，而非在 `cmd` 层

把「解析成绝对路径」放在 `Load` 出口，`ws.WorktreeBase` 对外即保证是绝对路径，所有消费方无需重复判断。这与现有 `ExpandHome` 的调用位置一致，改动面最小。

具体做法：把 `ExpandHome` 的职责扩展为「展开 `~` + 相对路径按给定 base 转绝对」，或新增一个 `resolvePath(base, p string) string` 并在 `Load` 中以 `root` 调用。选后者：`ExpandHome` 是导出函数且 `cmd/worktree.go:75` 也在用，扩展其签名会牵连调用方；新增函数语义更清晰，`ExpandHome` 保持不变可继续复用。

**替代方案：** *在 `cmd/worktree.go` 里 `filepath.Abs`* —— 被否。`filepath.Abs` 以 CWD 为基准，正是要避免的；且未来若有其他命令消费 `WorktreeBase`，同样的判断要再写一遍。

### 决策三：`--dest` 与 `worktree-base` 共用同一解析规则

`--dest` 当前只过 `ExpandHome`（`cmd/worktree.go:75`），有完全相同的缺陷。两者都是「worktree 根目录」的来源，规则应统一：`~` 展开、绝对路径保留、相对路径以 workspace root 为基准。

**替代方案：** *`--dest` 以 CWD 为基准（贴近一般 CLI 直觉）* —— 被否。两条来源规则不一致会制造新的认知负担；且 `--dest` 只在 workspace 内执行才有意义，以 workspace root 为基准同样自然。此点在文档中显式说明。

### 决策四：`worktreeProject` 内对目标路径做绝对化兜底

即便 `worktreeRoot` 在入口已绝对化，`worktreeProject` 仍是一个接收路径参数的独立函数，未来可能被其他调用方复用。在其内部对 `wtPath` 做一次绝对化（已是绝对则为 no-op），可从结构上杜绝「相对路径流向 git」这类问题再次出现，成本可忽略。

### 决策五：合并逻辑照抄既有模式，不做重构

三个缺失分支直接按 `if local.X != "" { result.X = local.X }` 补齐即可。当前有人会想顺手把 `mergeDefault` 改成反射或泛型驱动以杜绝「新增字段忘记 merge」，但这会牺牲可读性、且与 `mergeProjects` 的手写风格不一致。改为用测试守住：加一个断言「`Default` / `Project` 的每个可覆盖属性都能被 local 覆盖」的表驱动测试，新增字段漏 merge 时测试会失败。

## Risks / Trade-offs

- **既有用户的相对路径配置产出位置会改变** → 修复前相对路径产出的结构本就是错的（worktree 嵌在业务仓库里、伴随空目录），不存在需要保留的既有行为。修复说明中提示：若此前已用相对路径跑过，需手工 `git worktree remove` / 清理业务仓库内残留的 `worktree*` 目录。

- **`--dest` 改为以 workspace root 为基准，可能与使用者「相对 CWD」的直觉不符** → 影响面仅限传相对路径的场景（此前该场景本就是坏的）。在 `--dest` 的 flag 说明与 `docs/usage-*.md` 中显式写明基准。

- **worktree base 落在 workspace 内部会让 root 仓库出现 untracked 目录** → 非本次引入，属使用者的配置选择。在文档中提示可将该目录加入 `.gitignore`。

- **补齐 merge 后，此前「local 里写了但不生效」的配置会突然开始生效** → 这正是修复目的；但若有人依赖过这一 bug（例如 local 里留了废弃的 `worktree-copy`），行为会变化。风险极低，在变更说明中提及。

## Migration Plan

无数据迁移、无配置格式变更。发布即生效。回滚策略：还原提交即可，无残留状态。

建议在 release note 中包含一句清理提示：此前用相对 `worktree-base` 跑过 `fleet worktree` 的使用者，检查各业务仓库内是否有残留的 worktree 目录，用 `git worktree remove` 清理。

## Open Questions

无。
