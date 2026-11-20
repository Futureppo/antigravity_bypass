# Antigravity desktop 2.33.0 compatibility

Verified on 2026-11-20 against the official **Antigravity App 2.33.0** packages for Windows, Linux, and macOS, both x64 and ARM64. This is the current desktop release sequence; the previously verified 2.13.0 layout remains supported by the same structural checks.

## Package identity

`resources/app.asar/package.json` still identifies `antigravity`, product `Antigravity`, description `Antigravity - Agentic Desktop Application`, and entry point `dist/main.js`. The version is now `2.33.0`.

The launcher moved its argument list into `baseLsArgs(...)` and passes `--enable_sidecars`. The patcher accepts either the 2.13.0 inline arguments (`--standalone`, `--subclient_type`) or the 2.33.0 `baseLsArgs` structure. The exact version string is no longer a gate; if the identity, launcher, and backend checks cannot be recognized, the tool fails before writing.

| File | Size | SHA-256 |
| --- | --- | --- |
| Windows x64 installer | 182,549,552 bytes | `44aea1c00f3b347fd60ab7291d267dbb04a5080f6c3212a8a31cf1426ddf86cb` |
| Windows ARM64 installer | 172,414,672 bytes | `0b260f12acb71458a62b7c8c0b172fb05d4c71357b5ff567a0e010c30b5f7bee` |
| Linux x64 archive | 209,975,480 bytes | `a73387b229c6b1f5288860731133c5b26e153340eaed96d865032cd8a8535050` |
| Linux ARM64 archive | 203,199,363 bytes | `fb321bd9832403cf766e3e9719c2063cee1747a320925d3aecd4f0ce67ca6013` |
| macOS x64 DMG | 246,015,244 bytes | `97de4c520d6d2c867926933b5d7f5d0f19fcf080956540532e9a71f5fa352110` |
| macOS ARM64 DMG | 235,309,425 bytes | `01267a5a4c4b615110c08eb14c61a3505ee8708e7a64f85d2880c867c31baf17` |
| `app.asar` (x64 and ARM64) | 6,473,373 bytes | `b33c81600e82e65593aa8430ef01b37d6b28d9d98f0faf45e25957783b71e64b` |

## Layout

Windows keeps `resources/bin/language_server.exe`. Linux uses `resources/bin/language_server`, and macOS uses `Antigravity.app/Contents/Resources/bin/language_server`. `--dir` accepts the installation root, the `resources` directory, or the macOS `.app` bundle.

The frontend is still embedded in the backend and needs no patch.

## Backend checks

Every 2.33.0 backend has two `number of tools exceeds %d` references with **800** comparisons:

| Platform | Size | Comparison file offsets | Original/restored SHA-256 | Patched SHA-256 |
| --- | --- | --- | --- | --- |
| Windows x64 | 203,692,544 | `0x02873e0a`, `0x02d94320` | `eb6d986dfc5a859c6e9b1b36a0601f5c362993cc16eb8962ad554c9141defc9c` | `810926497e86cc7875ec37d7c54f9861b6ab9962010357d42f61c1b19144c421` |
| Windows ARM64 | 193,598,464 | `0x0245e174`, `0x0291baf8` | `96051eeb29017f266ed404a46762f08bdcfd346f99c0988e412a9c4fb358912a` | `a727e2cbe724bd1b56076a8ba9de427fe284a093a042bc6db0d4f5c1e11b518d` |
| Linux x64 | 224,658,120 | `0x07c599aa`, `0x0803c860` | `6bc1f046c4974fb4cfa52733ae7857115cd22af415e1bcfbff8bcadc38bdc97b` | `1a5be95da05ca8dfadd1096b8fc47d8e3f0490756d6a26cf668fd36d2b55ee70` |
| Linux ARM64 | 212,030,128 | `0x0776d104`, `0x07ad03f8` | `9772a2fd0d7654b317d439afec450a051018d0c69c9fb96e642a801846170fec` | `391ec6711814bac62603bfbf9724a6c8bf7c1105db477d2a8e61d7bcfea98847` |
| macOS x64 | 200,907,840 | `0x02651c0a`, `0x02ad04a0` | `3cf90744a0a29a5afa38092f46211ec85ec8bbe867a4ece7195d3e6425138dbe` | `0147aa09efbdc36e853562c989699c54d6f400befca94da90ff44efea70b1d29` |
| macOS ARM64 | 193,844,912 | `0x02213674`, `0x0261c2c8` | `59f20c05e424a13a04d0a93c342e06ec61155b87fa040a5a280bab240ba8348b` | `858e906e4568bd2916247248eefdca6c14e0fa157692d82dee897d6972aed100` |

On x64 the patcher writes 8192 (`0x2000`) into the two 32-bit comparison immediates, changing four bytes per binary. On ARM64 `CMP #imm12` cannot encode 8192 directly, so the instruction is changed to `CMP Xn, #2, LSL #12` (numerically 8192), also four bytes per binary. Instruction opcodes, branches, sizes, and all other bytes are preserved.

## Verification

`--check --client desktop --dir ...` identified both 800 checks on Windows x64/ARM64, Linux x64/ARM64, and macOS x64/ARM64. Automated tests cover identity and launcher validation, extensionless Linux/macOS binaries, ARM64 instruction rewriting, repeated patching, and restore. `go test ./...` and `go vet ./...` pass.

Package-level verification only; signed-in Electron sessions and real model/tool calls were not exercised.

