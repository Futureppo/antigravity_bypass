# Antigravity Bypass

> Antigravity IDE 工具数量限制绕过工具

Antigravity IDE 默认将 MCP 工具数量硬编码限制为 **100 个**。当你的 MCP Server 提供的工具超过这个数量时，IDE 会拒绝加载多余的工具。

本工具通过修补前端 JS 文件和语言服务器二进制文件，**一键移除该限制**。

---

## 免责声明

本项目仅供学习和研究用途。使用本工具产生的任何后果由使用者自行承担。

## 原理

本工具对 Antigravity IDE 的两处限制点进行修补：

### 1. 前端 JS 限制

修改 `workbench.desktop.main.js` 中硬编码的工具数量上限（`100` → `114514`），并自动清除 `product.json` 中对应的文件校验值（checksum），避免 IDE 弹出完整性校验警告。

### 2. 语言服务器限制

对 `language_server_windows_x64.exe` 进行二进制补丁。定位到 `cmp rcx, 0x64` + `jle` 指令序列（即数量 ≤ 100 的条件跳转），将 `jle`（条件跳转）替换为 `nop` + `jmp`（无条件跳转），从而跳过工具数量检查。

修补前会自动创建 `.backup` 备份文件。

---

## 运行

```bash
python antigravity_bypass.py
```

## 注意事项

- 本工具当前硬编码了默认安装路径，如果你的 IDE 安装在非默认位置，请修改脚本顶部的 `IDE_BASE_DIR` 变量。
- IDE 更新后补丁可能失效，需要重新执行脚本。如果特征字符串或偏移发生变化，脚本会给出提示。

---


---

## License

[AGPL-3.0](LICENSE)
