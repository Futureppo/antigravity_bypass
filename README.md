# Antigravity Bypass

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

> 运行前请**完全退出 Antigravity IDE**。
> IDE 更新后需重新执行。

## 支持平台

| 平台    | 架构        | 状态   |
| ------- | ----------- | ------ |
| Windows | x64 / ARM64 | 已验证 |
| Linux   | x64 / ARM64 | 待测试 |
| macOS   | x64 / ARM64 | 待测试 |

遇到问题请提 [Issue](https://github.com/Futureppo/antigravity_bypass/issues)。

## License

[AGPL-3.0](LICENSE)
