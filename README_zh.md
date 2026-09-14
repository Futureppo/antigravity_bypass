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

> Antigravity 本地工具数量限制修补工具 — 支持 IDE 2.5.5 和 Windows x64 桌面版 2.13.0

将可识别的本地工具数量上限提高至 **8192**。IDE **2.5.5** 有两处 **512** 工具检查，独立桌面版 **2.13.0** 有两处 **800** 工具检查；本工具通过工具数量报错的字符串引用和跳转关系定位它们。未能完整识别两处检查时会退出，不修改文件。这两个版本均无需修改前端。

支持 **Antigravity IDE** 和 **Windows x64 独立桌面版 Antigravity 2.13.0**。CLI 及 Linux、macOS、ARM64 桌面版暂不支持。只修改本地检查，不改变模型服务端的工具数量、上下文或请求大小限制。

**逆向分析与探索过程请参考博客：[从一条报错到两处修补：逆向 Antigravity IDE 的 MCP 工具数量限制](https://blog.futureppo.top/posts/antigravity/)**

## 免责声明

本项目仅供学习和研究用途。使用本工具产生的任何后果由使用者自行承担。


## 使用方式

### 直接下载

前往 [Releases](https://github.com/Futureppo/antigravity_bypass/releases) 下载对应平台的可执行文件，双击运行后根据菜单选择修补或恢复。

已发布的 **v2.1.0** 仅支持 IDE。桌面版功能需使用下方更新后的源码构建，可执行文件应能识别 `--client desktop`。

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

### 自定义安装路径

如果自动定位失败，可通过 `--dir` 或 `ANTIGRAVITY_DIR` 指定安装目录：

```bash
# IDE：安装根目录、resources/app 目录，以及 macOS .app 路径
# Windows 桌面版：安装根目录或 resources 目录
ANTIGRAVITY_DIR="/path/to/Antigravity IDE" ./antigravity_bypass
```

> 修补或恢复前请**完全退出所选 Antigravity 客户端**。
> 客户端更新后，请先重新运行 `--check` 确认兼容性。

### 命令行模式

```powershell
.\antigravity_bypass.exe --client ide --check --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --client ide --patch --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --client ide --restore --dir "C:\Apps\Antigravity IDE"
```

Windows x64 桌面版 **2.13.0**：

```powershell
.\antigravity_bypass.exe --client desktop --check --dir "C:\Apps\Antigravity"
.\antigravity_bypass.exe --client desktop --patch --dir "C:\Apps\Antigravity"
.\antigravity_bypass.exe --client desktop --restore --dir "C:\Apps\Antigravity"
```

省略 `--dir` 可自动查找安装。不带操作参数时仍显示交互菜单。`--client` 默认为 `auto`；同时找到 IDE 和桌面版时，需用 `--client ide`、`--client desktop` 或显式路径选择目标。`--dir` 优先于 `ANTIGRAVITY_DIR`；显式路径无效或与所选客户端不匹配时直接报错，不会改动其他安装。支持自动识别新版 `Antigravity IDE` 和旧版 `Antigravity` 安装目录。

桌面版只修改 `resources/bin/language_server.exe`，读取 `app.asar` 验证产品、版本和启动结构，不修改该归档或 Electron 主程序。桌面版目前仅接受已验证的 **2.13.0**，其他版本会拒绝修补。

## 支持平台

| 平台 | 架构 | IDE 2.5.5 验证情况 |
| --- | --- | --- |
| Windows | x64 | 官方安装包：检查、修补、重复执行、恢复验证通过 |
| Linux | x64 | 官方安装包：检查、修补、重复执行、恢复验证通过 |
| macOS | x64 | 已实现 Mach-O 识别，未验证安装包或运行效果 |
| Windows / Linux / macOS | ARM64 | 暂不支持二进制修补，明确报错；不会套用 x64 指令 |

上述验证基于解包文件和字节差异，尚未完成登录 IDE 后的真实模型调用测试。具体来源、文件哈希和检查点见 [2.5.5 适配记录](docs/compatibility-2.5.5.md)。旧版前端的 100 / 114514 常量仍可识别；旧版后端需要满足新的完整检查条件，不保证所有历史版本兼容。

| 独立桌面版 | 架构 | 2.13.0 验证情况 |
| --- | --- | --- |
| Windows | x64 | 官方安装包：检查、修补、重复执行、完整恢复、后台启动及本地前端 HTTP 验证通过 |
| Linux / macOS | 全部 | 暂不支持 |
| Windows | ARM64 | 暂不支持 |

桌面版启动验证使用隔离配置，未登录账号，尚未验证真实模型请求和工具调用。来源、哈希及检查点见 [桌面版 2.13.0 适配记录](docs/compatibility-desktop-2.13.0.md)。Antigravity CLI 暂不在支持范围内。

遇到问题请带着日志提 [Issue](https://github.com/Futureppo/antigravity_bypass/issues)。

## 常见问题

**Q: 提示"文件被占用"怎么办？**
> 请确保已完全退出所选 Antigravity 客户端（包括托盘进程），或以管理员权限运行。

**Q: 客户端更新后工具限制又回来了？**
> 客户端更新会覆盖修补文件。先运行 `--check` 确认兼容性。如果旧备份与新文件不匹配，请将旧 `.backup` 和 `.backup.json` 文件移到其他位置后重新修补；不要把旧文件恢复到新版客户端。桌面版升级到 2.13.0 以外的版本后，需要等待适配。

**Q: 提示"未找到工具限制特征"？**
> 可能是其他版本变更了代码结构。请提 Issue 并附上 `--check` 输出、客户端类型、版本号、系统和架构。IDE 2.5.5 和桌面版 2.13.0 前端不需要修补是正常状态。

**Q: 如何恢复原始文件？**
> 修补前会创建 `.backup` 和 `.backup.json` 校验记录。选择「还原备份」或使用 `--restore` 并指定相同的客户端/路径，只有文件与记录匹配时才恢复。恢复完成后删除这两个备份文件。历史版本生成的无校验备份需要核对客户端版本后手动恢复。

**Q: 已经用旧工具修补过 2.5.5？**
> 旧规则可能命中无关的字符串处理指令。请先恢复同版本原文件或重新安装官方 2.5.5，再使用本版工具。

## License

[AGPL-3.0](LICENSE)
