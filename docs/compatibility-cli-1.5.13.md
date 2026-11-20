# Antigravity CLI 1.5.13 compatibility

Verified on 2026-11-20 against the official CLI **1.5.13** builds for Windows, Linux, and macOS, x64 and ARM64. The installed CLI 1.5.10 has the same check layout and remains supported.

The CLI binary is also the shared backend used by Antigravity IDE 2.7.0/2.8.0 and the Antigravity IDE extensions. Patching it once covers all of them.

## Locations

- Windows standalone installer: `%LOCALAPPDATA%\agy\bin\agy.exe`
- Windows shared/IDE install: `%USERPROFILE%\.gemini\bin\agy.exe`
- Linux/macOS standalone installer: `~/.local/bin/agy`
- Linux/macOS shared/IDE install: `~/.gemini/bin/agy`

`--client cli` prefers the standalone installer path, then the shared path, then `agy` on `PATH`. `--client ide` continues to use the shared path for IDE 2.7.0/2.8.0. `--dir` accepts the binary itself or its parent directory.

## Packages

| Platform | Package | Size | SHA-256 |
| --- | --- | --- | --- |
| Windows x64 | `cli_windows_x64.exe` | 236,040,344 | `91bd7720100dec46acd5a852e8291e5704eb753f5801c625950a15dc414fdf0a` |
| Windows ARM64 | `cli_windows_arm64.exe` | 224,102,040 | `f7b8d30696ad7df8cc7423a267e738410c452b7d5a9ca42a465df9c400c3cb82` |
| Linux x64 | `cli_linux_x64.tar.gz` | 100,548,668 | `7fa9cc6df0d09562693d637c406a86c1b0f52ab3364714b1fd2d79cc6060d4b2` |
| Linux ARM64 | `cli_linux_arm64.tar.gz` | 91,596,782 | `36596ad8038531fd75f0e97b5ba8967a30f2fd67f8d9c61959d2e4761d7b215f` |
| macOS x64 | `cli_mac_x64.tar.gz` | 91,068,379 | `04974195ddad0315834e053b1c93831ac66dc27acb6c335fff404882bd55fa99` |
| macOS ARM64 | `cli_mac_arm64.tar.gz` | 85,497,754 | `1c83e3e888b5410c44a6d8a5467f7eca0d75c4d5551c80076b7da8dc7738bd9c` |

## Backend checks

All six builds contain two `number of tools exceeds %d` references and two **800** comparisons:

| Platform | Binary size | Comparison file offsets | Original/restored SHA-256 | Patched SHA-256 |
| --- | --- | --- | --- | --- |
| Windows x64 | 236,040,344 | `0x02142c13`, `0x02cdb540` | `91bd7720100dec46acd5a852e8291e5704eb753f5801c625950a15dc414fdf0a` | `82de8588fdf28cfc1b12a9375762fd5dca7e01272b4da6db250d5b3c33ae69c5` |
| Windows ARM64 | 224,102,040 | `0x01de186c`, `0x027fc908` | `f7b8d30696ad7df8cc7423a267e738410c452b7d5a9ca42a465df9c400c3cb82` | `e89c517b43eb8bdcad101d486e7540619ef823df738553c5c585166d81f7f4e3` |
| Linux x64 | 270,103,240 | `0x09200053`, `0x09d22380` | `c3fd2c848abf88a243936f33e25bf5970802a7651840b5199318fcd8eb3fdb65` | `be77792fee829a0d3d09be1d978c13a590298d6f77e2922f2267e728c7604786` |
| Linux ARM64 | 253,711,024 | `0x08dcfbcc`, `0x09761508` | `235147bb0b58673c6c8d130baab9c220921607220b7b0164dda27dd5e1c97cff` | `9f23f4ab96eb72dd7745d04098427da3859aee12a13fcec99ac04cd0d02873a3` |
| macOS x64 | 236,218,048 | `0x01f8ea73`, `0x02aac440` | `5fdf66e4ec7b46898d7cafd166373ba8f40fabf109faa887b26e895a89e9228c` | `bf7374f232b25719d48033a22e0867c32297275e68f4e614476b4cb1d32c7c12` |
| macOS ARM64 | 227,120,336 | `0x01c15a1c`, `0x025a4318` | `3f669d53b6a9807b161c327bf6ce33cdd24ecbf3026edcbb7347c004d1a2d9f2` | `19896d8569b3eb40c0195ae5d48f72d89904a29b59d72aaf78d2a6daa8e302b5` |

On x64 the immediate operand is changed from `20 03 00 00` (800) to `00 20 00 00` (8192). On ARM64 the check is encoded as `CMP Xn, #0x320`; 8192 does not fit in its 12-bit immediate, so it is rewritten as `CMP Xn, #0x2, LSL #12`, which evaluates to 8192. The two instructions change four bytes per binary.

## Verification

`--client cli --check`, `--patch`, a second `--check`, and `--restore` passed against isolated copies of all six builds and against the installed 1.5.10 Windows build. Automated tests cover standalone/shared path selection, directory input, lifecycle, and rejection of invalid binaries. `go test ./...` and `go vet ./...` pass.

Package-level verification only; signed-in CLI sessions and real model/tool calls were not exercised.
