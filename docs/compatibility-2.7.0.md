# Antigravity IDE 2.7.0 compatibility

Verified on 2026-11-20 against an installed **Windows x64** build of Antigravity IDE **2.7.0**, commit `71bedb45923debaf49871bdcb7c7d3a47918e247` (Electron application version 1.107.0). This change adds IDE 2.7.0 support alongside the existing [IDE 2.5.5](compatibility-2.5.5.md) and [desktop 2.13.0](compatibility-desktop-2.13.0.md) support.

## Layout change

Antigravity IDE 2.7.0 no longer ships a language server inside `resources/app/extensions/antigravity/bin`. The extension is renamed to `extensions/agy` and installs a shared backend on first launch:

```text
resources/app/
  extensions/agy/
    dist/extension.js
    vscode/extension.js
%USERPROFILE%/
  .gemini/
    bin/agy.exe
    bin/agy.version.json
```

The extension log records the launch:

```text
[LAUNCH] Spawning C:\Users\<user>\.gemini\bin\agy.exe (v1.5.10)
  --hub --hub-port=<port> --app_data_dir=antigravity --csrf_token=<token>
```

The backend path is `~/.gemini/bin/agy` on Linux and macOS. It is the same binary used by the standalone Antigravity CLI, so patching it modifies the shared backend rather than an IDE-private copy. The patcher looks for this path only when the `extensions/agy` layout is present; legacy IDE releases keep using `extensions/antigravity/bin/language_server_*`.

`agy.version.json` stores `schemaVersion`, `version`, `size`, `mtimeMs`, `ino`, `sha512`, and `releaseBaseUrl`. The extension validates the sidecar by size, mtime, and inode, then checks the binary version by running it. Patching in place preserves the file size but changes mtime, so the extension re-verifies the version on the next launch and rewrites the sidecar. A real backend update replaces the file and discards the patch.

## Frontend

The workbench still has no numeric MCP tool cap. Both MCP update methods save the configuration directly, and the total count display is `${n} tools`. No frontend file is modified.

| File | Size | SHA-256 |
| --- | --- | --- |
| `resources/app/out/vs/workbench/workbench.desktop.main.js` | 27,232,619 bytes | `6a76dbfe6c815b7978e9f8f67fed42bfd96f40d30d004edffdf600abbdeb553d` |
| `resources/app/product.json` | 44,752 bytes | `5e8a6ef4809e3963da6ac11fb3355ce1dbe6e9e58ea748332f0adedca8b5dd23` |

## Backend changes

The 233,313,944-byte shared backend `agy.exe` (version 1.5.10) contains two references to `number of tools exceeds %d`, each associated with an **800**-tool comparison:

| Comparison file offset | Original instructions | Immediate file offset | Error reference file offset |
| --- | --- | --- | --- |
| `0x0213b2aa` | `cmp rbx, 0x320; jle ...` | `0x0213b2ad` | `0x0213b2d6` |
| `0x02d730a0` | `cmp rbx, 0x320; jg ...` | `0x02d730a3` | `0x02d73865` |

The second comparison's error block starts at the `jg` target and reaches the format-string load through `mov`/`lea` instructions, exactly like the desktop 2.13.0 layout. The existing RIP-relative reference scan and x86 decoder validate both blocks without hard-coded file offsets. Both checks must be found exactly once.

The shared backend matcher accepts **800** and the patched value **8192**. The legacy IDE matcher is unchanged. Writing 8192 changes each immediate from `20 03 00 00` to `00 20 00 00`, for **four changed bytes total**. Instruction opcodes, branches, file size, and all other bytes are preserved.

## Hashes

| File/content | SHA-256 |
| --- | --- |
| Original/restored `agy.exe` | `5806364c0d9dde9d72cab903372d9d1d89a5b6f09c29ccaa42e375efbecc69c4` |
| Patched `agy.exe` | `08a444b933960749cf81d293c9dbe4ec536ebb522dcba1eb7254cb5a8fe69b9a` |

## Verification

Read-only `--check` against the installed layout reports both 800 checks and no frontend change. A byte comparison on a copy of the backend confirmed that only the two immediate operands change. Automated tests cover the new `extensions/agy` layout, shared-backend discovery, 800-tool matching, repeated patching, and restoration. `go test ./...` and `go vet ./...` pass.

This verifies discovery and binary patch sites. It does not establish successful signed-in IDE/model calls or changed runtime behavior of the CLI. Remote model constraints, including Google-side tool-count limits, remain unchanged.
