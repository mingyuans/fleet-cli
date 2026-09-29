## 1. 补齐 manifest 合并缺失的 worktree 字段

- [x] 1.1 在 `internal/manifest/merge.go` 的 `mergeDefault` 中补 `if local.WorktreeBase != "" { result.WorktreeBase = local.WorktreeBase }`，位置紧随 `MasterMainCompat` 分支之后，保持与既有分支一致的写法
- [x] 1.2 在 `mergeDefault` 中补 `if local.WorktreeCopy != "" { result.WorktreeCopy = local.WorktreeCopy }`
- [x] 1.3 在 `mergeProjects` 的同名 project 覆盖块中补 `if lp.WorktreeCopy != "" { bp.WorktreeCopy = lp.WorktreeCopy }`

## 2. 合并逻辑测试

- [x] 2.1 在 `internal/manifest/merge_test.go` 增加用例：base 无 `<default>` / local 有 `worktree-base` → 合并结果取 local 值
- [x] 2.2 增加用例：base 与 local 都有 `worktree-base` → 取 local 值；local 未声明 → 保留 base 值
- [x] 2.3 增加用例：`default/worktree-copy` 的覆盖与保留两种情形
- [x] 2.4 增加用例：同名 project 的 `worktree-copy` 覆盖与保留；以及 local 仅覆盖 `revision` 时 `worktree-copy` 不被清空
- [x] 2.5 增加表驱动的「防漏 merge」测试：对 `Default` 与 `Project` 的每个可覆盖属性各构造一次「仅该属性在 local 非空」的合并，断言均生效（新增字段忘记 merge 时该测试失败）
- [x] 2.6 在 `internal/manifest/resolve_test.go` 或 `integration_test.go` 增加用例：project 级 `worktree-copy` 非空时优先于 `<default>`，为空时回退到 default

## 3. 路径解析归一化

- [x] 3.1 在 `internal/workspace/workspace.go` 新增导出函数 `ResolvePath(base, path string) string`：空串原样返回；`~` 前缀走 `ExpandHome`；`filepath.IsAbs` 为真原样返回；否则 `filepath.Join(base, path)`。保留 `ExpandHome` 签名与行为不变
- [x] 3.2 修改 `Load`：把 `ws.WorktreeBase = ExpandHome(worktreeBase)` 改为 `ws.WorktreeBase = ResolvePath(root, worktreeBase)`，使 `ws.WorktreeBase` 对外保证为绝对路径（或空）
- [x] 3.3 在 `internal/workspace/workspace_test.go` 增加 `ResolvePath` 单元测试：相对路径、`./` 前缀、绝对路径、`~` 前缀、空串
- [x] 3.4 增加 `Load` 层测试：在临时 workspace 中以相对 `worktree-base` 加载，断言 `ws.WorktreeBase` 为 `<root>/worktrees/fleet`；并在切换进程 CWD 到子目录后重复断言，验证与 CWD 无关

## 4. worktree 命令路径统一

- [x] 4.1 修改 `cmd/worktree.go` 的 `runWorktree`：`--dest` 分支由 `workspace.ExpandHome(worktreeDest)` 改为 `workspace.ResolvePath(ws.Root, worktreeDest)`
- [x] 4.2 在 `worktreeProject` 中对 `wtPath` 做绝对化兜底（已绝对则为 no-op），确保传给 `git.WorktreeAdd` / `git.WorktreeAddNew` 的始终是绝对路径，与 `os.Stat` / `os.MkdirAll` / `worktree-copy` 拷贝目标共用同一路径
- [x] 4.3 更新 `--dest` flag 的说明文案，写明相对路径以 workspace root 为基准

## 5. worktree 端到端回归测试

- [x] 5.1 新增 `cmd/worktree_test.go`，搭建临时 workspace：两个本地 bare 上游仓库 + root project（path `.`）与非根 project（path `services/api`），`fleet.xml` 中 `worktree-base="./worktrees/fleet"`
- [x] 5.2 用例：执行 worktree 创建后，断言非根 project 的 worktree 位于 `<root>/worktrees/fleet/<name>/services/api`，且 `<root>/services/api` 仓库内不存在任何新建的 `worktrees` 目录
- [x] 5.3 用例：root project 的 worktree 位于 `<root>/worktrees/fleet/<name>`
- [x] 5.4 用例：重复执行同一 worktree 名 → 全部 skipped，不出现 failed
- [x] 5.5 用例：进程 CWD 切换到 `services/api` 子目录后执行，产出路径与在 root 下执行一致
- [x] 5.6 用例：配置 `worktree-copy` 后，被拷贝文件落在 `<root>/worktrees/fleet/<name>/services/api/` 下
- [x] 5.7 用例：`--dest` 传相对路径时，worktree 根目录为 `<root>/<dest>`

## 6. 文档

- [x] 6.1 更新 `docs/usage-zh.md` 与 `docs/usage-en.md` 的 worktree 小节：说明 `worktree-base` 支持 `~`、绝对路径与相对路径，相对路径以 workspace root（`fleet.xml` 所在目录）为基准；`--dest` 同规则
- [x] 6.2 在文档中补充：`worktree-base`、`worktree-copy` 可在 `local_fleet.xml` 中覆盖
- [x] 6.3 在文档中提示：若 worktree base 位于 workspace 内部，建议将该目录加入 `.gitignore`；此前用相对路径跑过的使用者需检查并用 `git worktree remove` 清理业务仓库内的残留 worktree
- [x] 6.4 视需要更新 `docs/example-fleet.xml` 中 `worktree-base` 的注释说明

## 7. 验证

- [x] 7.1 运行 `make test`，全部通过
- [x] 7.2 运行 `make lint`（若配置可用），无新增告警
- [x] 7.3 在临时沙箱 workspace 手工验证：`worktree-base="./worktrees/fleet"` 仅写在 `local_fleet.xml` 中时命令可正常工作，目录结构正确
