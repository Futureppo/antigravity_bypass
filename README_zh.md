<h1 align="center">Antigravity Bypass</h1>

<p align="center">
  <img src="docs/banner.png" alt="Antigravity Bypass Banner" width="800">
</p>

---

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-%3E%3D1.25-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="README_zh.md#支持平台"><img src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-4caf50" alt="Platform"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="License"></a>
  <a href="https://github.com/Futureppo/antigravity_bypass/releases"><img src="https://img.shields.io/github/v/release/Futureppo/antigravity_bypass?color=green" alt="GitHub Release"></a>
</p>

<p align="center">
  <a href="README.md">English</a> | <b>简体中文</b>
</p>

> Antigravity IDE 工具数量限制绕过工具 — 全版本自动适配，跨平台支持

Antigravity IDE 将 MCP 工具数量硬编码限制为 **100 个**，本工具一键移除该限制。采用自动搜索机制，无需手动更新特征码。

**逆向分析与探索过程请参考博客：[从一条报错到两处修补：逆向 Antigravity IDE 的 MCP 工具数量限制](https://blog.futureppo.top/posts/antigravity/)**

## 免责声明

本项目仅供学习和研究用途。使用本工具产生的任何后果由使用者自行承担。


## 使用方式

### 直接下载

前往 [Releases](https://github.com/Futureppo/antigravity_bypass/releases) 下载对应平台的可执行文件，双击运行后根据菜单选择修补或恢复。

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

**Q: 如何恢复原始文件？**
> 工具修补前会自动创建 `.backup` 备份。再次运行工具，选择「还原所有备份文件」即可一键恢复。

## License

[AGPL-3.0](LICENSE)
