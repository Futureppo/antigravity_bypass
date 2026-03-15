"""
去你妈的工具限制
Antigravity IDE 工具数量限制绕过工具  By https://github.com/Futureppo
"""

import os
import sys
import json
import shutil
import glob

# ============================================================
# 自动检测 IDE 安装路径
# ============================================================

# 用于验证目录是否为有效 Antigravity IDE 安装位置的标志文件
_VERIFY_FILE = os.path.join("out", "vs", "workbench", "workbench.desktop.main.js")


def _is_valid_ide_dir(path):
    """检查路径是否为有效的 Antigravity IDE resources/app 目录。"""
    return os.path.isfile(os.path.join(path, _VERIFY_FILE))


def _find_ide_dir():
    """
    自动检测 Antigravity IDE 安装路径，按以下优先级搜索:
    1. 常见安装目录
    2. Windows 注册表 (App Paths / Uninstall)
    3. 磁盘根目录快速扫描
    """
    candidates = []

    # --- 策略 1: 常见安装目录 ---
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
        # 覆盖典型 Electron 应用安装约定
        candidates.append(os.path.join(base, "Programs", "Antigravity", "resources", "app"))
        candidates.append(os.path.join(base, "Antigravity", "resources", "app"))

    # 检查常见路径
    for path in candidates:
        if _is_valid_ide_dir(path):
            return path

    # --- 策略 2: 从注册表查找 ---
    reg_path = _find_from_registry()
    if reg_path:
        return reg_path

    # --- 策略 3: 磁盘根目录快速扫描 ---
    disk_path = _find_from_disk_scan()
    if disk_path:
        return disk_path

    return None


def _find_from_registry():
    """从 Windows 注册表查找 Antigravity 安装位置。"""
    try:
        import winreg
    except ImportError:
        return None

    # 搜索的注册表位置
    reg_searches = [
        # App Paths
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe"),
        (winreg.HKEY_CURRENT_USER, r"SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe"),
    ]

    for hive, key_path in reg_searches:
        try:
            with winreg.OpenKey(hive, key_path) as key:
                exe_path, _ = winreg.QueryValueEx(key, "")
                if exe_path and os.path.exists(exe_path):
                    # 从 exe 路径反推 resources/app 目录
                    app_dir = os.path.join(os.path.dirname(exe_path), "resources", "app")
                    if _is_valid_ide_dir(app_dir):
                        return app_dir
        except (OSError, FileNotFoundError):
            continue

    # 搜索 Uninstall 注册表项
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
    """在各磁盘根目录快速扫描 Antigravity 安装目录（限制深度，避免太慢）。"""
    # 仅在 Windows 上执行磁盘扫描
    if sys.platform != "win32":
        return None

    # 获取所有盘符
    drives = []
    for letter in "CDEFGHIJKLMNOPQRSTUVWXYZ":
        drive = "{}:\\".format(letter)
        if os.path.isdir(drive):
            drives.append(drive)

    # 在每个盘符下搜索常见位置（避免全盘遍历）
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
    """检测并返回 IDE 安装路径。找不到时提示用户手动输入。"""
    path = _find_ide_dir()
    if path:
        print("[检测] 已自动定位 IDE 安装路径:")
        print("       {}".format(path))
        return path

    print("[检测] 未能自动定位 Antigravity IDE 安装路径。")
    print("       请手动输入 IDE 的 resources/app 目录路径")
    print("       (例如: C:\\Users\\你的用户名\\AppData\\Local\\Programs\\Antigravity\\resources\\app)")
    print()
    while True:
        user_input = input("路径> ").strip().strip('"')
        if not user_input:
            print("  输入不能为空，请重试。")
            continue
        if _is_valid_ide_dir(user_input):
            return user_input
        print("  该路径下未找到 IDE 核心文件，请确认路径是否正确后重试。")


IDE_BASE_DIR = None
JS_FILE = None
PRODUCT_FILE = None
LS_FILE = None


def _init_paths(base_dir):
    """根据检测到的安装路径初始化所有文件路径。"""
    global IDE_BASE_DIR, JS_FILE, PRODUCT_FILE, LS_FILE
    IDE_BASE_DIR = base_dir
    # 前端 JS 文件路径
    JS_FILE = os.path.join(IDE_BASE_DIR, "out", "vs", "workbench", "workbench.desktop.main.js")
    # product.json 路径
    PRODUCT_FILE = os.path.join(IDE_BASE_DIR, "product.json")
    # 语言服务器可执行文件路径
    LS_FILE = os.path.join(
        IDE_BASE_DIR,
        "extensions", "antigravity", "bin", "language_server_windows_x64.exe",
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
    """修改前端 JS 文件中的工具数量上限，并清除 product.json 中的校验值。"""
    print("[1/2] 正在处理前端 JS 工具数量限制...")

    if not os.path.exists(JS_FILE):
        print("  [失败] 找不到 JS 文件: {}".format(JS_FILE))
        return False

    with open(JS_FILE, "r", encoding="utf-8") as f:
        content = f.read()

    if JS_ORIGINAL in content:
        new_content = content.replace(JS_ORIGINAL, JS_PATCHED)
        with open(JS_FILE, "w", encoding="utf-8") as f:
            f.write(new_content)
        print("  [完成] 已将前端工具数量上限从 100 修改为 114514。")
        _remove_checksum()
        return True

    elif JS_PATCHED_MARKER in content:
        print("  [跳过] JS 文件已经修改过，无需重复操作。")
        return True

    else:
        print("  [失败] 未在 JS 文件中找到目标特征字符串，可能 IDE 版本不兼容。")
        return False


def _remove_checksum():
    """从 product.json 中删除被修改 JS 文件的 checksum，避免完整性校验警告。"""
    if not os.path.exists(PRODUCT_FILE):
        print("  [提示] 找不到 product.json，跳过校验值清理。")
        return

    with open(PRODUCT_FILE, "r", encoding="utf-8") as f:
        product_data = json.load(f)

    if "checksums" in product_data and JS_RESOURCE_PATH in product_data["checksums"]:
        old_val = product_data["checksums"][JS_RESOURCE_PATH]
        print("  [信息] 旧的 checksum: {}".format(old_val))

        del product_data["checksums"][JS_RESOURCE_PATH]

        with open(PRODUCT_FILE, "w", encoding="utf-8") as f:
            json.dump(product_data, f, separators=(",", ":"))

        print("  [完成] 已从 product.json 中移除对应的 checksum。")
    else:
        print("  [跳过] product.json 中未找到对应的 checksum 条目。")


def patch_language_server():
    """修补语言服务器二进制文件中的工具数量检查逻辑。"""
    print("[2/2] 正在处理语言服务器二进制工具数量限制...")

    if not os.path.exists(LS_FILE):
        print("  [失败] 找不到语言服务器文件: {}".format(LS_FILE))
        print("  请确认 IDE 安装路径是否正确。")
        return False

    # 创建备份
    backup_path = LS_FILE + ".backup"
    try:
        if not os.path.exists(backup_path):
            shutil.copyfile(LS_FILE, backup_path)
            print("  [信息] 已创建原始文件备份: {}".format(backup_path))
        else:
            print("  [跳过] 备份文件已存在，跳过备份操作。")
    except PermissionError:
        print("  [失败] 权限不足或文件被占用!")
        print("  请完全关闭所有 Antigravity 相关进程后再执行此脚本。")
        print("  提示: 检查任务管理器是否有 language_server_windows_x64.exe 残留。")
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

                # 安全校验: 仅修补位于预期偏移范围内的匹配项
                if BIN_OFFSET_RANGE[0] < idx < BIN_OFFSET_RANGE[1]:
                    patch_count += 1
                    # 定位到 cmp 指令之后，覆写 JLE (0F 8E) 为 NOP JMP (90 E9)
                    f.seek(idx + 4)
                    f.write(b"\x90\xe9")

            if patch_count > 0:
                print("  [完成] 发现并修补了 {} 处工具数量检查指令。".format(patch_count))
                return True
            elif already_patched >= 2:
                print("  [跳过] 二进制文件已修补过，双重校验点均已移除。")
                return True
            else:
                print("  [失败] 未定位到需要修补的特征位点。")
                print("  可能是 IDE 版本不同导致偏移变化，或文件已被其他方式修改。")
                return False

    except PermissionError:
        print("  [失败] 无法修改文件，它正被其它应用占用!")
        print("  请关闭任务管理器中的 Antigravity 客户端及语言服务器后重试。")
        return False
    except Exception as e:
        print("  [失败] 操作发生异常: {}".format(e))
        return False

def main():
    print("=" * 60)
    print("Antigravity IDE 工具数量限制绕过工具")
    print("=" * 60)
    print()
    print("注意: 执行前请确保已完全退出 Antigravity IDE 的所有进程。")
    print()

    # 自动检测 IDE 安装路径
    base_dir = detect_ide_path()
    _init_paths(base_dir)
    print()

    js_ok = patch_js()
    print()
    ls_ok = patch_language_server()

    print()
    print("-" * 60)
    if js_ok and ls_ok:
        print("[结果] 所有补丁均已成功应用，请重启 IDE 即可生效。")
    elif js_ok or ls_ok:
        print("[结果] 部分补丁已应用，请检查上方的失败信息。")
    else:
        print("[结果] 所有补丁均未能应用，请检查上方的错误信息。")
    print("-" * 60)


if __name__ == "__main__":
    main()
