# Antigravity Bypass

[![Go](https://img.shields.io/badge/Go-%3E%3D1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-4caf50)](README.md#支持平台)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/Futureppo/antigravity_bypass?color=green)](https://github.com/Futureppo/antigravity_bypass/releases)

> Antigravity IDE 工具数量限制绕过工具 — 全版本自动适配，跨平台支持

Antigravity IDE 将 MCP 工具数量硬编码限制为 **100 个**，本工具一键移除该限制。采用自动搜索机制，无需手动更新特征码。

## 免责声明

本项目仅供学习和研究用途。使用本工具产生的任何后果由使用者自行承担。


## 使用方式

### 直接下载

前往 [Releases](https://github.com/Futureppo/antigravity_bypass/releases) 下载对应平台的可执行文件，双击运行即可。

### 从源码构建

```bash
# 克隆仓库
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

> 运行前请**完全退出 Antigravity IDE**。
> IDE 更新后需重新执行。

## 支持平台

| 平台    | 架构        | 状态   |
| ------- | ----------- | ------ |
| Windows | x64 / ARM64 | 已验证 |
| Linux   | x64 / ARM64 | 待测试 |
| macOS   | x64 / ARM64 | 待测试 |

遇到问题请带着日志提 [Issue](https://github.com/Futureppo/antigravity_bypass/issues)。

## 常见问题

**Q: 提示"文件被占用"怎么办？**
> 请确保已完全退出 Antigravity IDE（包括托盘进程），或以管理员权限运行。

**Q: IDE 更新后工具限制又回来了？**
> 正常现象。IDE 更新会覆盖被修补的文件，重新运行本工具即可。

**Q: 提示"未找到工具限制特征"？**
> 可能是 IDE 新版本变更了代码结构。请提 Issue 并附上 IDE 版本号。

**Q: 二进制补丁安全吗？**
> 工具修补前会自动创建 `.backup` 文件。如需恢复，删除修补后的文件并将备份文件去掉 `.backup` 后缀即可。

## License

[AGPL-3.0](LICENSE)
