## Why

`fleet` 当前只能通过 `install.sh` 脚本（`curl ... | sh`）来安装和升级，用户无法在已安装的 CLI 内部直接升级到最新版本。用户需要记住安装命令、手动判断是否有新版本，体验割裂。提供一个内置的 `fleet update` 自更新命令，可以让用户一条命令完成"检查远程是否有新版本并自动升级"。

## What Changes

- 新增 `fleet update` 子命令：自动从 GitHub Releases 检测最新版本，与当前版本比较，若有更新则下载对应平台资产、校验 checksum 后原地替换正在运行的二进制文件。
- 版本比较：解析当前内置 `version` 与远程 `tag_name`，仅在远程版本更高时执行升级。
- 支持 `--check` 标志：仅检查并报告是否有新版本，不执行实际升级。
- 支持 `--force` 标志：跳过版本比较，强制重新安装最新版本。
- 当当前为开发版本（`version == "dev"`）时给出明确提示，默认不阻断（配合 `--force` 可强制安装）。
- 升级过程复用 `install.sh` 既有逻辑：平台/架构检测（darwin/linux × amd64/arm64）、资产命名约定（`fleet-<os>-<arch>.tar.gz`）、`checksums.txt` 校验。

## Capabilities

### New Capabilities
- `self-update`: `fleet update` 命令的行为契约——版本检测、远程版本解析、平台资产下载、checksum 校验、原地二进制替换，以及 `--check` / `--force` 标志的语义。

### Modified Capabilities
<!-- 无既有 spec 需要修改 -->

## Impact

- **新增代码**：`cmd/update.go`（命令定义与升级流程）、对应单元测试 `cmd/update_test.go`。
- **可能新增内部包**：`internal/selfupdate`（版本比较、平台检测、下载与替换的可测试逻辑）。
- **依赖**：仅使用 Go 标准库（`net/http`、`archive/tar`、`compress/gzip`、`crypto/sha256`、`os`），不引入第三方库。
- **外部接口**：依赖 GitHub Releases API（`https://api.github.com/repos/mingyuans/fleet-cli/releases/latest`）与 release 资产下载地址。
- **文档**：README 增加 `fleet update` 使用说明。
- **运行时约束**：需要对二进制安装目录有写权限（通常 `/usr/local/bin`），无权限时给出清晰错误提示。
