## 1. Flag 定义与候选分支解析

- [x] 1.1 在 `cmd/pr.go` 中新增 `prBase` 包级变量，注册 `--base` / `-b` flag
- [x] 1.2 新增 `parseBranchCandidates(input string) []string` 函数：按 `|` 分割、trim 空格、过滤空字符串
- [x] 1.3 新增 `resolveBaseFromCandidates(dir, fetchRemote string, candidates []string) string` 函数：遍历候选分支，用 `git.RemoteRefExists()` 检查存在性，返回第一个匹配的分支名或空字符串

## 2. prProject 逻辑修改

- [x] 2.1 在 `prProject()` 中，当 `prBase` 非空时，先 fetch fetch-remote 再调用 `resolveBaseFromCandidates()` 确定 base branch
- [x] 2.2 所有候选分支都不存在时，返回 `"skipped"` + StatusSkip + `"no matching base branch on <remote>: <candidates>"`
- [x] 2.3 候选分支匹配成功时，使用匹配到的分支替代原有的 `baseBranch`，后续创建 PR 逻辑不变

## 3. 单元测试

- [x] 3.1 为 `parseBranchCandidates()` 编写测试：正常分割、空元素过滤、空格 trim、空输入
- [x] 3.2 为 `resolveBaseFromCandidates()` 编写测试（可 mock `RemoteRefExists`，或在集成测试中验证）

## 4. 手动集成测试

- [ ] 4.1 在测试 repo 中验证：`fleet pr --base testing` 创建 PR 到 testing 分支
- [ ] 4.2 验证 fallback：`fleet pr --base "nonexist|testing"` 跳过 nonexist，创建 PR 到 testing
- [ ] 4.3 验证全部不存在：`fleet pr --base "nonexist1|nonexist2"` skip 并显示清晰提示
- [ ] 4.4 验证默认行为：`fleet pr`（不带 --base）行为不变
