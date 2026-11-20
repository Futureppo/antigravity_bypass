# Antigravity IDE 2.8.0 compatibility

Verified on 2026-11-20 against the official **Windows x64** package, IDE version **2.8.0**, commit `8dc5315afe097a21a6cb38dda76edc0835dfabfa` (Electron application version 1.107.0). This release keeps the shared `agy` backend introduced in 2.7.0.

## Layout

`resources/app/extensions/agy` still contains no language server. The extension installs and launches the shared backend at `~/.gemini/bin/agy[.exe]`, so the same patch covers IDE 2.8.0, the Antigravity CLI, and the IDE extensions that use this backend.

The workbench has no frontend MCP cap. No frontend or `product.json` change is made.

| File | Size | SHA-256 |
| --- | --- | --- |
| Official Windows x64 installer | 199,656,240 bytes | `0f63d94c0be212e25157125eb3757272b5aa95ea06538298631c14a3d7709197` |
| `workbench.desktop.main.js` | 27,232,619 bytes | `6278c16e31c9966d17c347de19f6e07608075711c0b38563d6664ef2273ae656` |
| `product.json` | 44,752 bytes | `fd430b1a5e51a476f847894eb729c3fabf686fdab79087761bb6caa34888e7bd` |

## Shared backend

IDE 2.8.0 requests Antigravity CLI **1.5.13**. The shared backend still has two `number of tools exceeds %d` references, each associated with an **800** check. On Windows x64 the comparison immediates are at file offsets `0x02142c16` and `0x02cdb543`; on Windows ARM64 they are at `0x01de186c` and `0x027fc908`. The matcher follows the string references instead of trusting these offsets.

The IDE 2.5.5 and desktop rules remain separate. IDE 2.8.0 uses the shared-backend matcher, which accepts **800** and **8192** and supports x64 and ARM64. See the [CLI 1.5.13 notes](compatibility-cli-1.5.13.md) for the full platform table.

## Verification

`--check --client ide` on the extracted 2.8.0 application with a 1.5.13 shared backend reports both 800 checks and no frontend change. `go test ./...` and `go vet ./...` pass, including the shared-backend, ARM64, patch, repeated-patch, and restore tests.

Package-level verification only; signed-in IDE/model requests and Linux/macOS runtime behavior were not exercised.
