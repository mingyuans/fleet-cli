## Context

`fleet` 是一个基于 cobra 的 Go CLI，发布在 GitHub 仓库 `mingyuans/fleet-cli` 的 Releases 上。Release 资产命名为 `fleet-<os>-<arch>.tar.gz`，并附带 `checksums.txt`（`sha256sum *.tar.gz` 生成）。版本号通过 ldflags 在构建期注入 `main.version`，开发构建为 `dev`。

现有 `install.sh` 已实现完整的安装/升级逻辑：检测 os/arch、解析最新 tag、下载资产、校验 checksum、解压并替换 `/usr/local/bin/fleet`。`fleet update` 即把这套逻辑用 Go 重新实现为内置自更新命令，使用户无需依赖外部脚本。

## Goals / Non-Goals

**Goals:**
- 提供 `fleet update` 命令，自动检测远程最新版本并升级当前二进制。
- 复用 `install.sh` 的资产命名约定与 checksum 校验机制，保证一致性与安全性。
- 升级失败（下载错误、校验失败、无写权限）时不破坏已安装的二进制，给出清晰中文提示。
- 核心逻辑（版本比较、资产名解析、checksum 校验）可单元测试。

**Non-Goals:**
- 不支持 Windows（与 `install.sh` 现状一致，仅 darwin/linux）。
- 不实现后台自动更新或启动时自动检查（仅用户显式调用）。
- 不引入第三方自更新库，仅用 Go 标准库。
- 不处理通过包管理器（如 Homebrew）安装的场景的特殊回退。

## Decisions

### 决策 1：拆分 `internal/selfupdate` 包承载可测试逻辑
将纯逻辑（版本比较、平台→资产名映射、checksums.txt 解析、SHA-256 校验、tar.gz 解包）放入 `internal/selfupdate`，`cmd/update.go` 只负责 cobra 命令定义、flag 解析与流程编排。
- **理由**：纯函数易于单元测试，无需真实网络。命令层保持薄。
- **备选**：全部写在 `cmd/update.go` —— 难以测试，被否决。

### 决策 2：版本比较采用语义化比较，容忍 `v` 前缀
解析 `vX.Y.Z` 形式为三段数字比较；当前版本为 `dev` 或无法解析时视为"无法比较"，默认不升级（除非 `--force`）。
- **理由**：tag 形如 `v1.2.3`，内置 version 可能带或不带前缀，需归一化。
- **备选**：字符串直接比较 —— 无法正确判断 `v1.10.0 > v1.9.0`，被否决。
- **取舍**：暂不处理预发布后缀（如 `-rc1`），如出现按"无法比较"保守处理。

### 决策 3：通过 GitHub Releases API 获取最新版本
请求 `https://api.github.com/repos/mingyuans/fleet-cli/releases/latest`，解析 JSON 的 `tag_name`。资产与 checksums 通过 `https://github.com/<repo>/releases/download/<tag>/...` 直接下载（无需鉴权，public repo）。
- **理由**：与 `install.sh` 一致，匿名访问即可。
- **风险**：GitHub 匿名 API 有限流（60 次/小时/IP）。对单用户偶发调用足够；失败时给出明确提示。

### 决策 4：原地替换使用"同目录临时文件 + rename"
通过 `os.Executable()` 定位当前二进制，将新二进制写入同目录临时文件，`chmod` 后用 `os.Rename` 原子替换。
- **理由**：同文件系统内 `rename` 原子，避免替换到一半损坏。正在运行的进程在 Unix 上允许替换其可执行文件（inode 解耦）。
- **备选**：直接覆盖写入 —— 非原子，中途失败会损坏二进制，被否决。
- **取舍**：临时文件须与目标同目录（跨设备 rename 会失败），故不能用 `os.TempDir()`。

### 决策 5：flag 设计
`--check`（仅检查不升级）与 `--force`（跳过版本判定强制重装）。二者可独立使用；`--check` 优先级语义为只读，不与 `--force` 组合执行实际安装。

## Risks / Trade-offs

- [GitHub API 限流] → 失败时明确提示用户稍后重试或手动通过 `install.sh` 升级。
- [无写权限（/usr/local/bin 常需 sudo）] → 检测到权限错误时提示用户用 `sudo fleet update` 重新运行；不在程序内静默 sudo 提权。
- [网络中途失败 / 部分下载] → 全程在临时文件操作，仅校验通过后才 rename；任何失败都保证旧二进制完好。
- [版本号格式异常] → 无法解析时保守处理为"无法比较"，避免误判降级。
- [仓库地址硬编码 `mingyuans/fleet-cli`] → 与 `install.sh` 保持单一来源；定义为包内常量便于未来调整。

## Migration Plan

纯新增命令，无破坏性变更，无需迁移。发布包含该命令的新版本后，旧版本用户仍可用 `install.sh` 升级到含 `fleet update` 的版本，此后即可使用 `fleet update` 自更新。

## Open Questions

- 是否需要在其他命令运行时被动提示"有新版本可用"？当前 Non-Goal，留待后续。
- 预发布版本（`-rc`、`-beta`）的比较策略，当前保守跳过，后续如需可扩展。
