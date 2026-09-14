# Antigravity IDE 2.5.5 compatibility

Verified on 2026-09-15 against build `2.5.5-4923483625488384`, linked from the [official download page](https://antigravity.google/download/). These notes cover the IDE. The separate Antigravity app listed on that page has a different release sequence; its Windows x64 support is documented in the [desktop 2.13.0 notes](compatibility-desktop-2.13.0.md).

## Official packages

- [Windows x64 installer](https://edgedl.me.gvt1.com/edgedl/release2/j0qc3/antigravity/stable/2.5.5-4923483625488384/windows-x64/Antigravity%20IDE.exe)
  - SHA-256: `a8c25631fd5e43bf217a6cf510ca796441817f4ad9c50b1d892282c7366edc4a`
  - Windows Authenticode status: valid, signer Google LLC.
  - Extracted with InnoUnp 2.71.1; the installer uses Inno Setup 6.4.0.1.
- [Linux x64 archive](https://edgedl.me.gvt1.com/edgedl/release2/j0qc3/antigravity/stable/2.5.5-4923483625488384/linux-x64/Antigravity%20IDE.tar.gz)
  - SHA-256: `0c5233b297d2b3aebb61af49f8944012c2953d361a5ebb16978490636917f831`
  - Extracted with tar.

## Changes in the application

The installation/product name is now `Antigravity IDE` (`antigravity-ide` on Linux). The legacy `Antigravity` paths alone no longer suffice. Windows uninstall entries can also use an identifier rather than the product name, so detection reads `DisplayName`.

The workbench no longer contains the old `Cannot enable more tools because it would exceed the limit of ...` error or the adjacent `50,100` declarations. Its MCP tool/server update methods save configuration without a numeric cap, and its total count display is `${n} tools`. Both extracted packages contain identical workbench JS:

`f4bd347d94be4634d2adeec8b3ef65e4d65d7dd0d72281b0e1982fe9c71fe6a7`

The language server contains two references to `number of tools exceeds %d`. Each has an associated comparison with **512**:

| Package | Comparison file offset | Original instructions | Immediate file offset |
| --- | --- | --- | --- |
| Windows x64 | `0x01dd8baa` | `cmp rbx, 0x200; jle ...` | `0x01dd8bad` |
| Windows x64 | `0x01eccee8` | `cmp rcx, 0x200; jg ...` | `0x01ecceeb` |
| Linux x64 | `0x05ad954a` | `cmp rbx, 0x200; jle ...` | `0x05ad954d` |
| Linux x64 | `0x05bacb08` | `cmp rcx, 0x200; jg ...` | `0x05bacb0b` |

The new patcher follows RIP-relative string references inside the code section, validates the error argument setup using an x86 decoder, and requires both checks to be identified uniquely. It writes **8192** (`0x2000`) into the two immediate operands, preserving instructions, branches, and file sizes. The offsets above are evidence, not hard-coded matching rules.

The old pattern `48 83 f9 64 0f 8e` also matches an unrelated string-processing branch in 2.5.5. It is no longer used as a patch target. ARM64 is rejected before scanning for x64 instructions. Unknown or incomplete structures fail before writing any target file.

## Package verification

Both packages passed `--check`, `--patch`, a second `--patch`, a second `--check`, and `--restore` on isolated extracted files. Checks included:

- The initial read-only check creates no backups or modified files.
- Only the two expected immediate operands change. For 512 → 8192 this changes exactly two bytes in each language server, from `02` to `20`.
- Workbench JS and `product.json` remain byte-for-byte unchanged.
- Repeated patching does not change the files or replace original backups; the second check reports zero files requiring modification.
- Restoration reproduces all original SHA-256 values and removes generated backups and checksum records.

| Language server | Original SHA-256 | Patched SHA-256 |
| --- | --- | --- |
| Windows x64 | `2f44b0a25eb4630afc1a2817363ef453d51a72c94bbdaa6aad8ba00e8d7aa0e1` | `6fd73fcd9d8b6815cdb66a2af4c92fcd4a1a8edbc35ef7ffa388d0e6cbbb9cf4` |
| Linux x64 | `251e555ee87178f47a3a765242b1d34dc75fd55add0cba939ae7f4b280815710` | `d5b00f8388a4bb4d264a6f0a69352a4da96a305090d9c30cac81b9a201e823e9` |

`go test ./...` and `go vet ./...` pass. Regression tests cover incorrect references, unsupported architectures, damaged section headers, unknown limits, unrelated instructions, legacy frontend constants including 114514, checksum repair, repeated patching, backup preservation, and rejection of stale backups after an IDE update. Release builds run tests before compilation; CI runs tests on Windows and Linux.

This is package-level verification performed on Windows. It does not establish successful signed-in IDE/model calls, Linux runtime behavior, or macOS/ARM64 compatibility. Remote model constraints are unaffected.
