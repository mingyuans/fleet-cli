## 1. internal/selfupdate 核心逻辑

- [x] 1.1 创建 `internal/selfupdate` 包，定义仓库常量（`mingyuans/fleet-cli`）与资产命名约定
- [x] 1.2 实现版本解析与比较 `CompareVersions`（容忍 `v` 前缀，三段数字比较，`dev`/不可解析返回"无法比较"）
- [x] 1.3 实现平台→资产名映射 `AssetName(goos, goarch)`，校验仅支持 darwin/linux × amd64/arm64
- [x] 1.4 实现 `checksums.txt` 解析与 SHA-256 校验 `VerifyChecksum`
- [x] 1.5 实现 tar.gz 解包，提取出可执行文件 `ExtractBinary`

## 2. 远程版本与下载

- [x] 2.1 实现 `LatestVersion()`：请求 GitHub Releases API 解析 `tag_name`，处理非 2xx / 网络错误
- [x] 2.2 实现资产与 `checksums.txt` 下载到临时文件（同二进制目录或临时目录）

## 3. 原地替换

- [x] 3.1 通过 `os.Executable()` 定位当前二进制，实现"同目录临时文件 + chmod + os.Rename"原子替换 `ReplaceBinary`
- [x] 3.2 处理无写权限场景，返回明确错误（提示使用 sudo）

## 4. cmd/update.go 命令编排

- [x] 4.1 创建 `update` cobra 命令并注册到 rootCmd，定义 `--check` / `--force` 标志
- [x] 4.2 编排流程：获取远程版本 → 版本比较/dev 与 force 处理 → check 模式提前返回 → 下载校验 → 替换 → 输出结果（中文）
- [x] 4.3 失败路径全部以非零退出码退出并输出清晰中文错误，确保旧二进制不被破坏

## 5. 测试

- [x] 5.1 为 `CompareVersions` 编写单元测试（更高/更低/相同/带前缀/dev/非法格式）
- [x] 5.2 为 `AssetName` 编写单元测试（支持组合与不支持组合）
- [x] 5.3 为 `VerifyChecksum` 编写单元测试（匹配/不匹配/资产不在清单）
- [x] 5.4 为 `ExtractBinary` 编写单元测试（用内存构造的 tar.gz）
- [x] 5.5 运行 `make test` 与 `go vet ./...`，确保全部通过

## 6. 文档

- [x] 6.1 在 README 增加 `fleet update`（含 `--check` / `--force`）使用说明
