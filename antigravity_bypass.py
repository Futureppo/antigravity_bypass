"""
去你妈的工具限制
Antigravity IDE 工具数量限制绕过工具  By https://github.com/Futureppo
全版本自动适配
"""

import os
import re
import sys
import json
import shutil
import struct
import hashlib
import base64
import platform


def _log(icon, msg, indent=0):
    print("{}{}  {}".format("  " * indent, icon, msg))


_VERIFY_FILE = os.path.join("out", "vs", "workbench", "workbench.desktop.main.js")


def _is_valid_ide_dir(path):
    return os.path.isfile(os.path.join(path, _VERIFY_FILE))


def _find_ide_dir():
    candidates = []
    env_dirs = [
        os.environ.get("LOCALAPPDATA", ""),
        os.environ.get("APPDATA", ""),
        os.environ.get("PROGRAMFILES", ""),
        os.environ.get("PROGRAMFILES(X86)", ""),
        os.path.expanduser("~"),
    ]
    for base in env_dirs:
        if not base:
            continue
        candidates.append(os.path.join(base, "Programs", "Antigravity", "resources", "app"))
        candidates.append(os.path.join(base, "Antigravity", "resources", "app"))

    custom = os.environ.get("ANTIGRAVITY_DIR", "")
    if custom:
        candidates.insert(0, os.path.join(custom, "resources", "app"))

    for path in candidates:
        if _is_valid_ide_dir(path):
            return path

    reg_path = _find_from_registry()
    if reg_path:
        return reg_path

    disk_path = _find_from_disk_scan()
    if disk_path:
        return disk_path

    return None


def _find_from_registry():
    try:
        import winreg
    except ImportError:
        return None

    reg_searches = [
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe"),
        (winreg.HKEY_CURRENT_USER, r"SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe"),
    ]
    for hive, key_path in reg_searches:
        try:
            with winreg.OpenKey(hive, key_path) as key:
                exe_path, _ = winreg.QueryValueEx(key, "")
                if exe_path and os.path.exists(exe_path):
                    app_dir = os.path.join(os.path.dirname(exe_path), "resources", "app")
                    if _is_valid_ide_dir(app_dir):
                        return app_dir
        except (OSError, FileNotFoundError):
            continue

    uninstall_bases = [
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"),
        (winreg.HKEY_CURRENT_USER, r"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"),
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"),
    ]
    for hive, base_key in uninstall_bases:
        try:
            with winreg.OpenKey(hive, base_key) as key:
                for i in range(winreg.QueryInfoKey(key)[0]):
                    try:
                        sub_name = winreg.EnumKey(key, i)
                        if "antigravity" not in sub_name.lower():
                            continue
                        with winreg.OpenKey(key, sub_name) as sub_key:
                            install_loc, _ = winreg.QueryValueEx(sub_key, "InstallLocation")
                            if install_loc:
                                app_dir = os.path.join(install_loc, "resources", "app")
                                if _is_valid_ide_dir(app_dir):
                                    return app_dir
                    except (OSError, FileNotFoundError):
                        continue
        except (OSError, FileNotFoundError):
            continue

    return None


def _find_from_disk_scan():
    if sys.platform != "win32":
        return None
    drives = []
    for letter in "CDEFGHIJKLMNOPQRSTUVWXYZ":
        drive = "{}:\\".format(letter)
        if os.path.isdir(drive):
            drives.append(drive)
    search_subdirs = [
        "Program Files",
        "Program Files (x86)",
        os.path.join("Users", os.environ.get("USERNAME", ""), "AppData", "Local", "Programs"),
    ]
    for drive in drives:
        for sub in search_subdirs:
            app_dir = os.path.join(drive, sub, "Antigravity", "resources", "app")
            if _is_valid_ide_dir(app_dir):
                return app_dir
    return None


def detect_ide_path():
    path = _find_ide_dir()
    if path:
        _log("*", "自动定位 IDE: {}".format(path))
        return path

    _log("!", "未找到 IDE，请手动输入 resources/app 目录路径")
    while True:
        user_input = input("  > ").strip().strip('"')
        if not user_input:
            continue
        if _is_valid_ide_dir(user_input):
            return user_input
        _log("!", "路径无效，请重试")


IDE_BASE_DIR = None
JS_FILE = None
PRODUCT_FILE = None
LS_FILE = None


def _get_ls_filename():
    system = platform.system()
    machine = platform.machine().lower()
    is_arm = "arm" in machine or "aarch64" in machine
    if system == "Windows":
        return "language_server_windows_{}.exe".format("arm64" if is_arm else "x64")
    elif system == "Linux":
        return "language_server_linux_{}".format("arm64" if is_arm else "x64")
    elif system == "Darwin":
        return "language_server_macos_{}".format("arm64" if is_arm else "x64")
    return "language_server_windows_x64.exe"


def _init_paths(base_dir):
    global IDE_BASE_DIR, JS_FILE, PRODUCT_FILE, LS_FILE
    IDE_BASE_DIR = base_dir
    JS_FILE = os.path.join(IDE_BASE_DIR, "out", "vs", "workbench", "workbench.desktop.main.js")
    PRODUCT_FILE = os.path.join(IDE_BASE_DIR, "product.json")
    LS_FILE = os.path.join(IDE_BASE_DIR, "extensions", "antigravity", "bin", _get_ls_filename())


JS_PATCHED_MARKER = "=114514,"
JS_RESOURCE_PATH = "vs/workbench/workbench.desktop.main.js"

# 匹配 minified 代码中的 <var>=50,<var>=100,<var>=class 模式
JS_AUTO_PATTERN = re.compile(
    r'([A-Za-z_$][\w$]{0,4})=50,([A-Za-z_$][\w$]{0,4})=100,([A-Za-z_$][\w$]{0,4})=class'
)

JS_CONTEXT_KEYWORDS = [
    "tool", "Tool", "cannot enable", "Cannot enable",
    "exceed", "limit", "Limit", "maximum", "Maximum",
]

# cmp rcx, 0x64 + jle near → cmp rcx, 0x64 + nop + jmp
BIN_SEARCH = b"\x48\x83\xf9\x64\x0f\x8e"
BIN_PATCH = b"\x48\x83\xf9\x64\x90\xe9"


def _find_text_section(data):
    """定位 PE/ELF .text 段范围。"""
    if data[:2] == b'MZ':
        try:
            pe_off = struct.unpack_from('<I', data, 0x3c)[0]
            num_sec = struct.unpack_from('<H', data, pe_off + 6)[0]
            opt_size = struct.unpack_from('<H', data, pe_off + 20)[0]
            sec_off = pe_off + 24 + opt_size
            for s in range(num_sec):
                so = sec_off + s * 40
                name = data[so:so + 8].rstrip(b'\x00')
                if name == b'.text':
                    ptr = struct.unpack_from('<I', data, so + 20)[0]
                    size = struct.unpack_from('<I', data, so + 16)[0]
                    return (ptr, ptr + size)
        except Exception:
            pass
    elif data[:4] == b'\x7fELF' and data[4] == 2:
        try:
            e_shoff = struct.unpack_from('<Q', data, 0x28)[0]
            e_shentsize = struct.unpack_from('<H', data, 0x3a)[0]
            e_shnum = struct.unpack_from('<H', data, 0x3c)[0]
            e_shstrndx = struct.unpack_from('<H', data, 0x3e)[0]
            str_sh_off = e_shoff + e_shstrndx * e_shentsize
            str_tab_off = struct.unpack_from('<Q', data, str_sh_off + 0x18)[0]
            for i in range(e_shnum):
                sh_off = e_shoff + i * e_shentsize
                sh_name_idx = struct.unpack_from('<I', data, sh_off)[0]
                sh_type = struct.unpack_from('<I', data, sh_off + 4)[0]
                name_end = data.find(b'\x00', str_tab_off + sh_name_idx)
                sec_name = data[str_tab_off + sh_name_idx:name_end]
                if sec_name == b'.text' and sh_type == 1:
                    offset = struct.unpack_from('<Q', data, sh_off + 0x18)[0]
                    size = struct.unpack_from('<Q', data, sh_off + 0x20)[0]
                    return (offset, offset + size)
        except Exception:
            pass
    return None


def _verify_patch_context(data, idx):
    """验证 jle 跳转偏移是否在合理范围内 (0~0x2000)。"""
    if idx + 10 > len(data):
        return False
    rel_offset = struct.unpack_from('<i', data, idx + 6)[0]
    return 0 < rel_offset < 0x2000


def patch_js():
    _log("#", "[1/2] 前端 JS 工具数量限制")

    if not os.path.exists(JS_FILE):
        _log("x", "找不到目标文件", indent=1)
        return False

    with open(JS_FILE, "r", encoding="utf-8") as f:
        content = f.read()

    if JS_PATCHED_MARKER in content:
        _log("-", "已修改过，跳过", indent=1)
        return True

    matches = list(JS_AUTO_PATTERN.finditer(content))
    if not matches:
        _log("x", "未找到工具限制特征，可能版本不兼容", indent=1)
        return False

    target = None
    if len(matches) == 1:
        target = matches[0]
    else:
        # 多候选时按上下文关键词评分
        scored = []
        for m in matches:
            pos = m.start()
            ctx = content[max(0, pos - 500):pos + 500]
            score = sum(1 for kw in JS_CONTEXT_KEYWORDS if kw in ctx)
            scored.append((score, m))
        scored.sort(key=lambda x: x[0], reverse=True)
        if scored[0][0] > 0 and scored[0][0] > scored[1][0]:
            target = scored[0][1]
        else:
            _log("x", "无法自动确定目标，找到以下候选:", indent=1)
            for i, (score, m) in enumerate(scored):
                _log(" ", "候选 {} (关联度 {}): {}".format(i + 1, score, m.group(0)), indent=2)
            return False

    original_str = target.group(0)
    limit_var = target.group(2)
    patched_str = original_str.replace("{}=100".format(limit_var), "{}=114514".format(limit_var))
    new_content = content.replace(original_str, patched_str)
    with open(JS_FILE, "w", encoding="utf-8") as f:
        f.write(new_content)

    _log("+", "工具上限 100 -> 114514", indent=1)
    _log("*", "特征: {}".format(original_str), indent=1)
    _update_checksum()
    return True


def _update_checksum():
    """计算修补后 JS 文件的 SHA256 校验值并写回 product.json。"""
    if not os.path.exists(PRODUCT_FILE) or not os.path.exists(JS_FILE):
        return
    with open(JS_FILE, "rb") as f:
        new_hash = base64.b64encode(hashlib.sha256(f.read()).digest()).decode()
    with open(PRODUCT_FILE, "r", encoding="utf-8") as f:
        product_data = json.load(f)
    if "checksums" in product_data and JS_RESOURCE_PATH in product_data["checksums"]:
        product_data["checksums"][JS_RESOURCE_PATH] = new_hash
        with open(PRODUCT_FILE, "w", encoding="utf-8") as f:
            json.dump(product_data, f, separators=(",", ":"))
        _log("+", "已更新 product.json 校验值", indent=1)


def patch_language_server():
    _log("#", "[2/2] 语言服务器二进制文件")

    if not os.path.exists(LS_FILE):
        _log("x", "找不到目标文件: {}".format(os.path.basename(LS_FILE)), indent=1)
        return False

    backup_path = LS_FILE + ".backup"
    try:
        if not os.path.exists(backup_path):
            shutil.copyfile(LS_FILE, backup_path)
            _log("+", "已备份原始文件", indent=1)
    except PermissionError:
        _log("x", "文件被占用，请先关闭所有 Antigravity 进程", indent=1)
        return False

    try:
        with open(LS_FILE, "r+b") as f:
            data = f.read()
            text_range = _find_text_section(data)
            if text_range:
                search_start, search_end = text_range
            else:
                search_start, search_end = 0, len(data)

            already_patched = data.count(BIN_PATCH)
            patch_count = 0
            skipped = 0

            idx = search_start - 1
            while True:
                idx = data.find(BIN_SEARCH, idx + 1)
                if idx == -1 or idx >= search_end:
                    break
                if not _verify_patch_context(data, idx):
                    skipped += 1
                    continue
                patch_count += 1
                f.seek(idx + 4)
                f.write(b"\x90\xe9")
                _log("+", "0x{:08x}: jle -> nop+jmp".format(idx), indent=1)

            if patch_count > 0:
                _log("+", "修补了 {} 处检查指令".format(patch_count), indent=1)
                return True
            elif already_patched >= 1:
                _log("-", "已修补过，跳过", indent=1)
                return True
            else:
                _log("x", "未找到特征位点", indent=1)
                if skipped > 0:
                    _log("*", "{} 处匹配未通过上下文验证".format(skipped), indent=1)
                return False

    except PermissionError:
        _log("x", "文件被占用，请先关闭 Antigravity 再重试", indent=1)
        return False
    except Exception as e:
        _log("x", "异常: {}".format(e), indent=1)
        return False


def main():
    print("Antigravity IDE 工具数量限制绕过工具")
    print("全版本自动适配 - 无需手动更新特征码")
    print("注意: 请先完全退出 Antigravity IDE 的所有进程")
    print()

    base_dir = detect_ide_path()
    _init_paths(base_dir)

    js_ok = patch_js()
    ls_ok = patch_language_server()

    print()
    if js_ok and ls_ok:
        _log("+", "全部完成，重启 IDE 即可生效")
    elif js_ok or ls_ok:
        _log("!", "部分完成，请检查上方失败信息")
    else:
        _log("x", "全部失败，请检查上方错误信息")


if __name__ == "__main__":
    main()
