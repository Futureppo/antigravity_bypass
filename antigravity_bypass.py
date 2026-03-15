"""
去你妈的工具限制
Antigravity IDE 工具数量限制绕过工具  By https://github.com/Futureppo
"""

import os
import sys
import json
import shutil


def _log(icon, msg, indent=0):
    print("{}{} {}".format("  " * indent, icon, msg))


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
    _log(" ", "例: C:\\Users\\用户名\\AppData\\Local\\Programs\\Antigravity\\resources\\app")
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


def _init_paths(base_dir):
    global IDE_BASE_DIR, JS_FILE, PRODUCT_FILE, LS_FILE
    IDE_BASE_DIR = base_dir
    JS_FILE = os.path.join(IDE_BASE_DIR, "out", "vs", "workbench", "workbench.desktop.main.js")
    PRODUCT_FILE = os.path.join(IDE_BASE_DIR, "product.json")
    LS_FILE = os.path.join(
        IDE_BASE_DIR, "extensions", "antigravity", "bin", "language_server_windows_x64.exe",
    )


JS_ORIGINAL = "wfa=50,pGe=100,rjn=class"
JS_PATCHED = "wfa=50,pGe=114514,rjn=class"
JS_PATCHED_MARKER = "pGe=114514"
JS_RESOURCE_PATH = "out/vs/workbench/workbench.desktop.main.js"

# cmp rcx, 0x64 + jle -> cmp rcx, 0x64 + nop + jmp
BIN_SEARCH_PATTERN = b"\x48\x83\xf9\x64\x0f\x8e"
BIN_PATCH_PATTERN = b"\x48\x83\xf9\x64\x90\xe9"
BIN_OFFSET_RANGE = (0x1600000, 0x1700000)


def patch_js():
    _log("#", "[1/2] 前端 JS 工具数量限制")

    if not os.path.exists(JS_FILE):
        _log("x", "找不到目标文件", indent=1)
        return False

    with open(JS_FILE, "r", encoding="utf-8") as f:
        content = f.read()

    if JS_ORIGINAL in content:
        new_content = content.replace(JS_ORIGINAL, JS_PATCHED)
        with open(JS_FILE, "w", encoding="utf-8") as f:
            f.write(new_content)
        _log("+", "工具上限 100 -> 114514", indent=1)
        _remove_checksum()
        return True
    elif JS_PATCHED_MARKER in content:
        _log("-", "已修改过，跳过", indent=1)
        return True
    else:
        _log("x", "未找到特征码，可能版本不兼容", indent=1)
        return False


def _remove_checksum():
    if not os.path.exists(PRODUCT_FILE):
        return

    with open(PRODUCT_FILE, "r", encoding="utf-8") as f:
        product_data = json.load(f)

    if "checksums" in product_data and JS_RESOURCE_PATH in product_data["checksums"]:
        del product_data["checksums"][JS_RESOURCE_PATH]
        with open(PRODUCT_FILE, "w", encoding="utf-8") as f:
            json.dump(product_data, f, separators=(",", ":"))
        _log("+", "已清除 product.json 校验值", indent=1)


def patch_language_server():
    _log("#", "[2/2] 语言服务器二进制文件")

    if not os.path.exists(LS_FILE):
        _log("x", "找不到目标文件", indent=1)
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
            already_patched = data.count(BIN_PATCH_PATTERN)
            patch_count = 0

            idx = -1
            while True:
                idx = data.find(BIN_SEARCH_PATTERN, idx + 1)
                if idx == -1:
                    break
                if BIN_OFFSET_RANGE[0] < idx < BIN_OFFSET_RANGE[1]:
                    patch_count += 1
                    f.seek(idx + 4)  # 跳过 cmp 指令，覆写 jle -> nop+jmp
                    f.write(b"\x90\xe9")

            if patch_count > 0:
                _log("+", "修补了 {} 处检查指令".format(patch_count), indent=1)
                return True
            elif already_patched >= 2:
                _log("-", "已修补过，跳过", indent=1)
                return True
            else:
                _log("x", "未找到特征位点，可能版本不兼容", indent=1)
                return False

    except PermissionError:
        _log("x", "文件被占用，请先关闭 Antigravity 再重试", indent=1)
        return False
    except Exception as e:
        _log("x", "异常: {}".format(e), indent=1)
        return False


def main():
    print("Antigravity IDE 工具数量限制绕过工具")
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
