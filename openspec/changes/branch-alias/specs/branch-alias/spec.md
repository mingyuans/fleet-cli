## ADDED Requirements

### Requirement: Configure branch alias groups in manifest
manifest SHALL 支持零或多个 `<branch-alias>` 元素，每个元素通过其下的 `<branch>` 子元素逐个列出组成员，定义一组**对等等价**的别名分支（同组成员互为别名，无主从）。系统 SHALL 去除每个 `<branch>` 成员的首尾空白并忽略空白成员；有效成员数量少于 2 的别名组 SHALL 被忽略（单成员无别名意义）。

#### Scenario: Define a single alias group
- **WHEN** manifest 包含 `<branch-alias><branch>testing</branch><branch>testing-incy</branch></branch-alias>`
- **THEN** 系统 SHALL 将 `testing` 与 `testing-incy` 视为同一别名组，二者互为对等别名

#### Scenario: Trim whitespace and ignore blank members
- **WHEN** manifest 包含 `<branch-alias><branch> testing-incy </branch><branch>  </branch><branch>testing</branch></branch-alias>`
- **THEN** 系统 SHALL 规范化为成员 `["testing-incy", "testing"]`（trim 首尾空白、丢弃空白成员）

#### Scenario: Ignore single-member group
- **WHEN** manifest 包含 `<branch-alias><branch>testing</branch></branch-alias>`（仅一个有效成员）
- **THEN** 系统 SHALL 忽略该别名组，不产生任何回退行为

### Requirement: Branch alias fallback resolution
当系统需要在某 remote 上解析一个分支时，若该分支对应的远端 ref 存在则 SHALL 直接使用该分支；否则当该分支属于某别名组时，系统 SHALL 在组内回退到第一个在该 remote 上存在的别名成员（被请求的分支自身优先，其余成员按声明顺序作确定性 tie-break）；若组内成员均不存在，或该分支不属于任何别名组，则解析 SHALL 失败（返回空）。

#### Scenario: Configured branch exists
- **WHEN** 解析分支 `testing-incy` 且 `remote/testing-incy` 存在
- **THEN** 系统 SHALL 返回 `testing-incy`

#### Scenario: Fallback to alias member
- **WHEN** 解析分支 `testing-incy`，`remote/testing-incy` 不存在但同组的 `remote/testing` 存在
- **THEN** 系统 SHALL 返回 `testing`

#### Scenario: No alias member exists
- **WHEN** 解析分支 `testing-incy`，其别名组成员在 remote 上均不存在
- **THEN** 解析 SHALL 失败（返回空）

#### Scenario: Branch not in any alias group
- **WHEN** 解析分支 `feature-x` 且 `remote/feature-x` 不存在，`feature-x` 不属于任何别名组
- **THEN** 解析 SHALL 失败（返回空），不进行任何回退

### Requirement: `fleet start` resolves base from alias group
执行 `fleet start <branch>`：若本地已存在 `<branch>`，系统 SHALL 直接切换到该分支（既有行为）。否则当 `<branch>` 属于某别名组时，新建分支的 base 起点 SHALL 通过别名回退解析，候选顺序为 `<branch>` 自身排在最前、其后为组内其余成员（按声明顺序，去重）；新建的本地分支名 SHALL 始终为用户输入的 `<branch>`。若 `<branch>` 不属于任何别名组，base 起点 SHALL 仍由 `revision` 解析（既有行为保持不变）。

#### Scenario: Target alias branch exists on remote
- **WHEN** 用户运行 `fleet start testing-incy`，且某 repo 的 `remote/testing-incy` 存在
- **THEN** 系统 SHALL 基于 `remote/testing-incy` 创建本地分支 `testing-incy`

#### Scenario: Target alias branch missing, fallback member exists
- **WHEN** 用户运行 `fleet start testing-incy`，某 repo 没有 `remote/testing-incy` 但有 `remote/testing`，二者属于同一别名组
- **THEN** 系统 SHALL 基于 `remote/testing` 创建本地分支 `testing-incy`

#### Scenario: No alias member exists on remote
- **WHEN** 用户运行 `fleet start testing-incy`，某 repo 别名组成员在 remote 上均不存在
- **THEN** 该 repo SHALL 失败，并提示已尝试过的候选分支

#### Scenario: Non-alias branch keeps existing behavior
- **WHEN** 用户运行 `fleet start my-feature`，`my-feature` 不属于任何别名组
- **THEN** 系统 SHALL 基于 `revision`（含 master-main-compat 逻辑）创建本地分支 `my-feature`，行为与现状一致

#### Scenario: Local branch already exists
- **WHEN** 用户运行 `fleet start testing-incy`，本地已存在 `testing-incy` 分支
- **THEN** 系统 SHALL 直接切换到本地 `testing-incy`，不解析别名

### Requirement: Revision resolution reuses alias groups
`sync` / `finish` / `prune` / `worktree` 以及 `fleet pr`（未指定 `--base` 时）对 manifest `revision` 的解析 SHALL 复用别名组回退逻辑：当 `revision` 在 remote 上不存在时，在其所属别名组内回退到第一个存在的成员。

#### Scenario: Revision falls back via alias group
- **WHEN** 项目 `revision` 配置为 `testing`，某 repo 仅存在 `remote/testing-incy`，且 `testing` 与 `testing-incy` 属于同一别名组
- **THEN** 相关命令 SHALL 将该 repo 的有效 revision 解析为 `testing-incy`

### Requirement: Backward compatibility with master-main-compat
`master-main-compat="true"` SHALL 继续生效，其语义等价于内置一组别名 `master`/`main`，并可与显式 `<branch-alias>` 配置并存。当 `master-main-compat` 未启用且没有显式别名组覆盖 `master`/`main` 时，`master` 与 `main` 之间 SHALL NOT 自动回退。

#### Scenario: master-main-compat still works
- **WHEN** `master-main-compat="true"`，项目 `revision` 为 `master`，某 repo 仅存在 `remote/main`
- **THEN** 系统 SHALL 将有效 revision 回退为 `main`，与现状行为一致

#### Scenario: Explicit alias group coexists with master-main-compat
- **WHEN** manifest 同时启用 `master-main-compat="true"` 并声明 `<branch-alias><branch>testing</branch><branch>testing-incy</branch></branch-alias>`
- **THEN** `master`/`main` 与 `testing`/`testing-incy` 两组别名 SHALL 同时生效

#### Scenario: No implicit master-main fallback when disabled
- **WHEN** `master-main-compat` 未启用且无任何别名组包含 `master`/`main`，`revision` 为 `master`，某 repo 仅存在 `remote/main`
- **THEN** 解析 SHALL 失败，`master` 与 `main` 之间不自动回退

### Requirement: `pr --base` does not stack alias fallback
当用户显式使用 `fleet pr --base` 时，别名组回退 SHALL NOT 自动叠加到 `--base` 的候选解析上；`--base` 仍仅按其自身的 `|` 候选从左到右解析。该约束保持 `pr-target-branch` 既有行为不变。

#### Scenario: --base with alias member does not auto-fallback
- **WHEN** 用户运行 `fleet pr --base testing-incy`，某 repo 没有 `remote/testing-incy` 但有同组的 `remote/testing`
- **THEN** 该 repo 的 PR 创建 SHALL 被跳过，不自动回退到 `testing`（用户可显式使用 `--base "testing-incy|testing"`）

### Requirement: Alias groups merge across manifests
当存在多个 manifest（例如本地 manifest 覆盖上游）时，各 manifest 声明的 `<branch-alias>` 元素 SHALL 被合并，使所有声明的别名组共同生效。若同一分支被多个别名组包含，系统 SHALL 以确定的顺序选取其中第一个匹配的别名组进行回退。

#### Scenario: Local manifest adds an alias group
- **WHEN** 上游 manifest 声明 `<branch-alias><branch>master</branch><branch>main</branch></branch-alias>`，本地 manifest 声明 `<branch-alias><branch>testing</branch><branch>testing-incy</branch></branch-alias>`
- **THEN** 合并后两组别名 SHALL 同时生效
