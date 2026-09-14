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

> Antigravity IDE 本地工具数量限制修补工具 — 已适配 IDE 2.5.5

将可识别的本地工具数量上限提高至 **8192**。IDE **2.5.5** 已取消前端的 100 工具限制，但语言服务器仍有两处 **512** 工具检查；本工具通过工具数量报错的字符串引用和跳转关系定位它们。未能完整识别两处检查时会退出，不修改文件。

本项目针对 **Antigravity IDE**，不是官网单独提供的 Antigravity 应用。只修改本地检查，不改变模型服务端的工具数量、上下文或请求大小限制。

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
# 支持安装根目录、resources/app 目录，以及 macOS .app 路径
ANTIGRAVITY_DIR="/path/to/Antigravity IDE" ./antigravity_bypass
```

> 运行前请**完全退出 Antigravity IDE**。
> IDE 更新后需重新执行。

### 命令行模式

```powershell
.\antigravity_bypass.exe --check --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --patch --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --restore --dir "C:\Apps\Antigravity IDE"
```

不带操作参数时仍显示交互菜单。`--dir` 优先于 `ANTIGRAVITY_DIR`；显式路径无效时直接报错，不会改动其他安装。支持自动识别新版 `Antigravity IDE` 和旧版 `Antigravity` 安装目录。

## 支持平台

| 平台 | 架构 | IDE 2.5.5 验证情况 |
| --- | --- | --- |
| Windows | x64 | 官方安装包：检查、修补、重复执行、恢复验证通过 |
| Linux | x64 | 官方安装包：检查、修补、重复执行、恢复验证通过 |
| macOS | x64 | 已实现 Mach-O 识别，未验证安装包或运行效果 |
| Windows / Linux / macOS | ARM64 | 暂不支持二进制修补，明确报错；不会套用 x64 指令 |

上述验证基于解包文件和字节差异，尚未完成登录 IDE 后的真实模型调用测试。具体来源、文件哈希和检查点见 [2.5.5 适配记录](docs/compatibility-2.5.5.md)。旧版前端的 100 / 114514 常量仍可识别；旧版后端需要满足新的完整检查条件，不保证所有历史版本兼容。

遇到问题请带着日志提 [Issue](https://github.com/Futureppo/antigravity_bypass/issues)。

## 常见问题

**Q: 提示"文件被占用"怎么办？**
> 请确保已完全退出 Antigravity IDE（包括托盘进程），或以管理员权限运行。

**Q: IDE 更新后工具限制又回来了？**
> IDE 更新会覆盖修补文件。先运行 `--check` 确认兼容性。如果旧备份与新文件不匹配，请将旧 `.backup` 和 `.backup.json` 文件移到其他位置后重新修补；不要把旧文件恢复到新版 IDE。

**Q: 提示"未找到工具限制特征"？**
> 可能是其他版本变更了代码结构。请提 Issue 并附上 `--check` 输出、IDE 版本号、系统和架构。2.5.5 前端不需要修补是正常状态。

**Q: 如何恢复原始文件？**
> 修补前会创建 `.backup` 和 `.backup.json` 校验记录。选择「还原备份」或使用 `--restore`，只有文件与记录匹配时才恢复。恢复完成后删除这两个备份文件。历史版本生成的无校验备份需要核对 IDE 版本后手动恢复。

**Q: 已经用旧工具修补过 2.5.5？**
> 旧规则可能命中无关的字符串处理指令。请先恢复同版本原文件或重新安装官方 2.5.5，再使用本版工具。

## License

[AGPL-3.0](LICENSE)
