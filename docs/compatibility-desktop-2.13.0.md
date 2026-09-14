# Antigravity desktop 2.13.0 compatibility

Verified on 2026-09-15 against the standalone **Windows x64** desktop app, build `2.13.0-6362815968182272`. This change adds desktop support alongside the existing [IDE 2.5.5 support](compatibility-2.5.5.md). CLI, Linux/macOS desktop, and ARM64 desktop support are outside this change.

## Official package

- [Windows x64 installer](https://storage.googleapis.com/antigravity-public/antigravity-hub/2.13.0-6362815968182272/windows-x64/Antigravity-x64.exe), linked from the [official download page](https://antigravity.google/download/).
- Size: 148,797,976 bytes.
- SHA-256: `417d8965b8e0a4134fe30c8c18e05158ca07503a348f300d3fe4257e34dd3b38`.
- Windows Authenticode status: valid, signer Google LLC.
- NSIS installer, extracted with 7-Zip 26.02; the application payload is `$PLUGINSDIR/app-64.7z`.

## Layout and scope

```text
Antigravity/
  Antigravity.exe
  resources/
    app.asar
    app.asar.unpacked/
    bin/language_server.exe
```

`app.asar/package.json` identifies `antigravity`, product `Antigravity`, version `2.13.0`, description `Antigravity - Agentic Desktop Application`, and entry point `dist/main.js`. The launcher in `dist/languageServer.js` starts `resources/bin/language_server.exe` with `--standalone` and `--subclient_type hub`.

The frontend is an embedded ZIP in the language server, served over a local HTTP connection. Its MCP tool update code changes `disabledTools` and saves the configuration without the old frontend count check. The patch leaves this frontend, `app.asar`, and the Electron executable untouched.

The patcher reads only the ASAR identity and launcher entries, with bounded header/entry sizes, offset checks, and SHA-256 entry verification when present. It requires the recognized product, exact version **2.13.0**, known launcher structure, and a Windows x64 PE backend. Other desktop versions fail before any write.

## Backend changes

The 156,513,792-byte language server contains two references to `number of tools exceeds %d`, each associated with an **800**-tool comparison:

| Comparison file offset | Original instructions | Immediate file offset | Error reference file offset |
| --- | --- | --- | --- |
| `0x0214742a` | `cmp rbx, 0x320; jle ...` | `0x0214742d` | `0x02147456` |
| `0x0225d028` | `cmp rcx, 0x320; jg ...` | `0x0225d02b` | `0x0225d68c` |

The patcher follows the error string references and validates the branch/error-block relationship using the existing x86 decoder. Both checks must be identified uniquely. It changes the two immediate operands from `20 03 00 00` (800) to `00 20 00 00` (8192), changing **four bytes total**. Instruction opcodes, branches, file size, and all other bytes are preserved. These offsets document the verified package; they are not hard-coded matching rules.

The desktop matcher accepts only 800 and the patched value 8192. The IDE's existing accepted limits are unchanged.

## Usage

Fully quit the desktop client, including its background process, before patching or restoring. Replace the example path with the installation root or its `resources` directory:

```powershell
.\antigravity_bypass.exe --client desktop --check --dir "C:\Apps\Antigravity"
.\antigravity_bypass.exe --client desktop --patch --dir "C:\Apps\Antigravity"
.\antigravity_bypass.exe --client desktop --restore --dir "C:\Apps\Antigravity"
```

Omit `--dir` for discovery through standard Windows install paths and the registry. Omit the action flag to show the interactive menu. `--client auto` is the default; discovering both IDE and desktop requires selecting a client or explicit path. `--dir` takes precedence over `ANTIGRAVITY_DIR`. An explicit desktop selection never falls back to an IDE installation.

## Verification

On an isolated extraction of the official package:

- The initial `--check` created no backups or modified files. Selecting `--client ide` for the desktop path was rejected.
- `--patch` changed exactly the four bytes listed above; a full byte comparison confirmed no other changes.
- Repeated `--patch` and `--check` found the completed patch and preserved the original backup.
- `Antigravity.exe`, `app.asar`, and the embedded frontend ZIP retained their original hashes.
- The patched language server started in standalone/headless mode using an isolated profile, disabled telemetry, and loopback service endpoints. Its local HTTP server served `/` (3,846 bytes) and `/main.js` (9,141,756 bytes). The served main bundle matched the original embedded frontend hash.
- `--restore` reproduced the original language server SHA-256 and removed `.backup` and `.backup.json`.

| File/content | SHA-256 |
| --- | --- |
| Original/restored language server | `a027c2931cb4d8440403006dc1cb4c91e14faa169a537517124c54eb74203354` |
| Patched language server | `e8df3e06191222aa78e0e4aaff195ca71236280ca87770192346e44a4b3c7dfc` |
| `Antigravity.exe` | `b34cf787ca8fd11d92720580fc89403cd7a08c8644cebefe35037c097568af1a` |
| `app.asar` | `29e2e4f5b2b1ae796453228da0cc40f08a14bf5541865149dae1f051336a121b` |
| Embedded frontend ZIP | `01301ce12261bdc9d861206439c746bcbe73f4afc55283c7aa2efcde592fae68` |
| Embedded/HTTP-served `main.js` | `af2d07d01fd1a81edb320d2618445d3aaa494b0156407a125edde4872c638f3e` |

Automated tests cover desktop selection, ASAR parsing and integrity, package identity/version/launcher rejection, unsupported architectures, exact patch bytes, repeated patching, restoration, and separation from IDE matching rules.

`go test ./...` and `go vet ./...` pass on Windows x64. The updated executable also passed read-only checks against the original Windows x64 and Linux x64 IDE 2.5.5 packages, still identifying both 512-tool checks. All existing release targets (Windows, Linux, and macOS, each amd64/arm64) compile successfully; compilation does not add runtime or binary-patching support for other desktop platforms.

This verifies package integrity and local backend/frontend delivery. It does **not** establish successful Electron GUI sessions, signed-in model requests, or real tool calls. Remote model constraints, including Google-side tool-count limits, remain unchanged.
