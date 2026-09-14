<h1 align="center">Antigravity Bypass</h1>

<p align="center">
  <img src="docs/banner.png" alt="Antigravity Bypass Banner" width="800">
</p>

---

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-%3E%3D1.25-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="README.md#supported-platforms"><img src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-4caf50" alt="Platform"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="License"></a>
  <a href="https://github.com/Futureppo/antigravity_bypass/releases"><img src="https://img.shields.io/github/v/release/Futureppo/antigravity_bypass?color=green" alt="GitHub Release"></a>
</p>

<p align="center">
  <b>English</b> | <a href="README_zh.md">简体中文</a>
</p>

> Local tool count limit patch for Antigravity IDE — updated for IDE 2.5.5

Raises recognized local tool count limits to **8192**. IDE **2.5.5** removed its frontend 100-tool cap, but the language server still contains two **512**-tool checks. This tool locates them through the tool-count error string references and control flow. If both checks cannot be identified, it exits without modifying files.

This project targets **Antigravity IDE**, not the separate Antigravity app on the download page. It changes local checks only; model service limits on tool counts, context, or request sizes still apply.

**For the full reverse engineering write-up, see the blog post: [From an Error Message to Two Patches: Reversing the MCP Tool Limit in Antigravity IDE](https://blog.futureppo.top/posts/antigravity/)**

## Disclaimer

This project is for educational and research purposes only. Any consequences of using this tool are the sole responsibility of the user.

## Usage

### Download

Head to [Releases](https://github.com/Futureppo/antigravity_bypass/releases) to download the executable for your platform. Run it and follow the menu to patch or restore.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Futureppo/antigravity_bypass.git
cd antigravity_bypass

# Build for current platform
go build -ldflags="-s -w" -o antigravity_bypass .

# Cross-compile for other platforms
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o antigravity_bypass_linux_x64 .
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o antigravity_bypass_mac_arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o antigravity_bypass_win_x64.exe .
```

### Custom IDE Path

If automatic detection fails, specify the IDE installation directory via an environment variable:

```bash
# Accepts an installation root, resources/app directory, or macOS .app path
ANTIGRAVITY_DIR="/path/to/Antigravity IDE" ./antigravity_bypass
```

> **Fully quit Antigravity IDE** before running.
> Re-run the tool after IDE updates.

### Command-line mode

```powershell
.\antigravity_bypass.exe --check --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --patch --dir "C:\Apps\Antigravity IDE"
.\antigravity_bypass.exe --restore --dir "C:\Apps\Antigravity IDE"
```

Without an action flag, the interactive menu is displayed. `--dir` overrides `ANTIGRAVITY_DIR`; invalid explicit paths fail without falling back to another installation. Both the new `Antigravity IDE` and legacy `Antigravity` installation names are detected.

## Supported Platforms

| Platform | Architecture | IDE 2.5.5 verification |
| --- | --- | --- |
| Windows | x64 | Official package: inspection, patch, repeated execution, and restore verified |
| Linux | x64 | Official package: inspection, patch, repeated execution, and restore verified |
| macOS | x64 | Mach-O parsing implemented; package and runtime not verified |
| Windows / Linux / macOS | ARM64 | Binary patching unsupported; fails explicitly without applying x64 instructions |

Verification uses extracted package files and byte comparisons; signed-in IDE/model calls have not been tested. See the [2.5.5 compatibility notes](docs/compatibility-2.5.5.md) for sources, hashes, and patch sites. Legacy frontend constants of 100 / 114514 are recognized, but older backends must meet the new complete matching requirements. Compatibility with every historical release is not guaranteed.

If you encounter issues, please open an [Issue](https://github.com/Futureppo/antigravity_bypass/issues) with the relevant logs.

## FAQ

**Q: "File is in use" error?**
> Make sure Antigravity IDE is fully closed (including tray processes), or run the tool with administrator privileges.

**Q: The tool limit came back after an IDE update?**
> Updates overwrite patched files. Run `--check` first. If old backups do not match the new files, move the old `.backup` and `.backup.json` files elsewhere before patching again. Do not restore an older binary over an updated IDE.

**Q: "Tool limit signature not found"?**
> The code structure may differ. Include the `--check` output, IDE version, OS, and architecture in an Issue. IDE 2.5.5 needing no frontend patch is expected.

**Q: How do I restore the original files?**
> Patching creates `.backup` files and `.backup.json` checksum records. Select "Restore backups" or use `--restore`. Files are restored only when their checksums match the recorded state; both backup files are then removed. Older backups without checksum records require manual restoration after confirming the IDE version.

**Q: Already used the old patcher on 2.5.5?**
> The old signature can match unrelated string-processing instructions. Restore matching original files or reinstall the official 2.5.5 package before using this version.

## License

[AGPL-3.0](LICENSE)
