## ADDED Requirements

### Requirement: 检查远程最新版本

`fleet update` 命令 SHALL 从 GitHub Releases 查询仓库 `mingyuans/fleet-cli` 的最新发布版本（`releases/latest`），并解析其 `tag_name` 作为远程最新版本号。

#### Scenario: 成功获取远程最新版本
- **WHEN** 用户执行 `fleet update` 且网络可达
- **THEN** 命令通过 GitHub Releases API 获取最新 release 的 `tag_name`
- **AND** 将其作为远程版本与当前版本进行比较

#### Scenario: 无法获取远程版本
- **WHEN** GitHub API 请求失败（网络错误、限流或非 2xx 响应）
- **THEN** 命令 SHALL 以非零退出码退出
- **AND** 输出清晰的中文错误信息说明无法获取最新版本

### Requirement: 版本比较与升级判定

命令 SHALL 比较当前内置版本与远程最新版本，仅当远程版本严格高于当前版本时才判定需要升级。版本号比较 SHALL 容忍可选的 `v` 前缀（如 `v1.2.3` 与 `1.2.3` 等价）。

#### Scenario: 已是最新版本
- **WHEN** 当前版本与远程最新版本相同或更高
- **THEN** 命令输出"已是最新版本"提示并以零退出码退出，不执行下载

#### Scenario: 存在更新版本
- **WHEN** 远程最新版本高于当前版本
- **THEN** 命令报告当前版本与目标版本，并继续执行升级流程

#### Scenario: 开发版本
- **WHEN** 当前版本为 `dev`（未注入正式版本）
- **THEN** 命令 SHALL 提示当前为开发版本无法可靠比较，默认不执行升级
- **AND** 仅当用户显式传入 `--force` 时才继续安装最新版本

### Requirement: 仅检查模式

命令 SHALL 支持 `--check` 标志，在该模式下只检查并报告是否有新版本，不下载也不替换二进制。

#### Scenario: check 模式发现新版本
- **WHEN** 用户执行 `fleet update --check` 且存在更新版本
- **THEN** 命令输出当前版本与可用的新版本，但不执行下载或替换

#### Scenario: check 模式已是最新
- **WHEN** 用户执行 `fleet update --check` 且已是最新版本
- **THEN** 命令输出"已是最新版本"并以零退出码退出

### Requirement: 强制重装

命令 SHALL 支持 `--force` 标志，跳过版本比较直接下载并安装远程最新版本（即使版本相同或当前为开发版本）。

#### Scenario: 强制重装最新版本
- **WHEN** 用户执行 `fleet update --force`
- **THEN** 命令跳过"已是最新"判定，直接下载最新版本并替换当前二进制

### Requirement: 平台资产下载与校验

命令 SHALL 根据运行平台（`runtime.GOOS`/`runtime.GOARCH`）选择对应的 release 资产 `fleet-<os>-<arch>.tar.gz`，下载后 SHALL 通过 release 中的 `checksums.txt` 校验 SHA-256，校验失败 MUST 中止升级且不替换二进制。仅支持 darwin/linux 与 amd64/arm64 组合。

#### Scenario: checksum 校验通过
- **WHEN** 下载的资产 SHA-256 与 `checksums.txt` 中记录一致
- **THEN** 命令解压并继续替换二进制

#### Scenario: checksum 校验失败
- **WHEN** 下载的资产 SHA-256 与记录不一致
- **THEN** 命令 SHALL 中止升级、删除临时文件并以非零退出码报错
- **AND** 当前已安装的二进制保持不变

#### Scenario: 不支持的平台
- **WHEN** 运行平台不在支持的 os/arch 组合内
- **THEN** 命令 SHALL 报错说明该平台不受支持

### Requirement: 原地替换正在运行的二进制

命令 SHALL 将下载并解压后的新二进制原子地替换当前正在运行的可执行文件（通过 `os.Executable()` 定位），替换 SHALL 保留可执行权限。

#### Scenario: 成功替换二进制
- **WHEN** 新二进制下载并校验通过
- **THEN** 命令将其安装到当前可执行文件路径并设置可执行权限
- **AND** 输出升级成功及新版本号

#### Scenario: 无写权限
- **WHEN** 当前可执行文件所在目录不可写
- **THEN** 命令 SHALL 以非零退出码退出并提示需要更高权限（如使用 sudo 重新运行）
- **AND** 当前二进制保持不变
