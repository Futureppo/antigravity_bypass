# Antigravity Bypass

[![Go](https://img.shields.io/badge/Go-%3E%3D1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-4caf50)](README.md#支持平台)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/Futureppo/antigravity_bypass?color=green)](https://github.com/Futureppo/antigravity_bypass/releases)

> Antigravity IDE 工具数量限制绕过工具 — 全版本自动适配，跨平台支持

Antigravity IDE 将 MCP 工具数量硬编码限制为 **100 个**，本工具通过修补前端 JS 和 Language Server 二进制文件，一键移除该限制。采用自动搜索 + 上下文评分机制定位补丁目标，无需随版本手动更新特征码。

## 特性

- **全自动 IDE 定位** — 候选路径 → 注册表查询 → 全盘扫描，无需手动指定路径
- **双重补丁** — 同时修补前端 JS 限制和 Language Server 二进制检查，确保完整绕过
- **版本自适应** — 正则匹配 + 上下文关键词评分，自动定位目标代码段，适配各版本
- **校验值同步** — 修补 JS 后自动重算 SHA-256 并更新 `product.json`，避免完整性校验失败
- **自动备份** — 修补二进制文件前自动创建 `.backup` 备份
- **跨平台** — 支持 Windows / Linux / macOS，x64 / ARM64 架构
- **零依赖** — 单文件可执行，下载即用

## 工作原理

本工具分两步完成补丁：

### 第一步：前端 JS 工具数量限制

定位 `workbench.desktop.main.js` 中形如 `x=50,y=100,z=class` 的赋值模式（其中 `100` 即为工具上限）。当匹配到多个候选时，通过上下文 ±500 字符范围内的关键词（`tool`、`exceed`、`limit`、`maximum` 等）评分，自动选择关联度最高的目标，将上限值修改为 `114514`。

修补完成后自动计算新文件的 SHA-256 校验值，回写到 `product.json` 的 `checksums` 字段中。

### 第二步：Language Server 二进制补丁

在 Language Server 可执行文件的 `.text` 段中搜索 x86-64 指令序列：

```
48 83 f9 64 0f 8e    cmp rcx, 0x64 (100) + jle near
```

将条件跳转 `jle` 修改为无条件跳转 `nop + jmp`，使数量检查永远通过：

```
48 83 f9 64 90 e9    cmp rcx, 0x64 (100) + nop + jmp
```

同时验证跳转偏移的合理性（正值且 < 0x2000），避免误补丁。支持解析 PE (Windows) 和 ELF64 (Linux) 格式，精确限定搜索范围。

## 使用方式

### 直接下载

前往 [Releases](https://github.com/Futureppo/antigravity_bypass/releases) 下载对应平台的可执行文件，双击运行即可。

### 从源码构建

```bash
git clone https://github.com/Futureppo/antigravity_bypass.git
cd antigravity_bypass

# 构建当前平台
go build -ldflags="-s -w" -o antigravity_bypass .

# 交叉编译其他平台
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o antigravity_bypass_linux_x64 .
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o antigravity_bypass_mac_arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o antigravity_bypass_win_x64.exe .
```

### 自定义 IDE 路径

如果自动定位失败，可通过环境变量指定 IDE 安装目录：

```bash
# 工具会自动拼接 resources/app 子路径
ANTIGRAVITY_DIR="/path/to/Antigravity" ./antigravity_bypass
```

> [!IMPORTANT]
> 运行前请**完全退出 Antigravity IDE 的所有进程**。
> 每次 IDE 更新后需重新执行本工具。

## IDE 自动定位策略

工具按以下优先级定位 IDE 安装路径（以 Windows 为例）：

| 优先级 | 策略 | 说明 |
| :---: | --- | --- |
| 1 | 环境变量 | 读取 `ANTIGRAVITY_DIR` 环境变量 |
| 2 | 候选路径 | 扫描 `%LOCALAPPDATA%`、`%APPDATA%`、`%PROGRAMFILES%` 等常见安装位置 |
| 3 | 注册表 | 查询 `App Paths` 和 `Uninstall` 注册表项 |
| 4 | 全盘扫描 | 遍历 C-Z 盘符下的 `Program Files` 等目录 |
| 5 | 手动输入 | 以上均失败时提示用户手动输入 `resources/app` 路径 |

> Linux / macOS 会扫描 `/usr/share`、`/opt`、`~/Applications` 等对应路径。

## 支持平台

| 平台    | 架构        | 状态   |
| ------- | ----------- | ------ |
| Windows | x64 / ARM64 | ✅ 已验证 |
| Linux   | x64 / ARM64 | 🔲 待测试 |
| macOS   | x64 / ARM64 | 🔲 待测试 |

CI 通过 GitHub Actions 自动构建全部 6 个平台产物，并使用 UPX 压缩（macOS 和 Windows ARM64 除外）。

## 常见问题

**Q: 提示"文件被占用"怎么办？**
> 请确保已完全退出 Antigravity IDE（包括托盘进程），或以管理员权限运行。

**Q: IDE 更新后工具限制又回来了？**
> 正常现象。IDE 更新会覆盖被修补的文件，重新运行本工具即可。

**Q: 提示"未找到工具限制特征"？**
> 可能是 IDE 新版本变更了代码结构。请提 Issue 并附上 IDE 版本号。

**Q: 二进制补丁安全吗？**
> 工具修补前会自动创建 `.backup` 文件。如需恢复，删除修补后的文件并将备份文件去掉 `.backup` 后缀即可。

## 免责声明

本项目仅供学习和研究用途。使用本工具产生的任何后果由使用者自行承担。

遇到问题请带着日志提 [Issue](https://github.com/Futureppo/antigravity_bypass/issues)。

## License

[AGPL-3.0](LICENSE)
