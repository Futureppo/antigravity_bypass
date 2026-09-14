package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const toolLimit = 8192

func logMsg(icon, msg string, indent int) {
	fmt.Printf("%s%s  %s\n", strings.Repeat("  ", indent), icon, msg)
}

func isValidIDEDir(path string) bool {
	info, err := os.Stat(filepath.Join(path, "out", filepath.FromSlash(jsResourcePath)))
	return err == nil && !info.IsDir()
}

func resolveIDEPath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "\"")
	for _, candidate := range []string{path, filepath.Join(path, "resources", "app"), filepath.Join(path, "Contents", "Resources", "app")} {
		if isValidIDEDir(candidate) {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("无效的 IDE 路径: %s", path)
}

func detectIDEPath(custom string) (string, error) {
	if custom == "" {
		custom = os.Getenv("ANTIGRAVITY_DIR")
	}
	// Never fall back to another installation when an explicit path is wrong.
	if custom != "" {
		return resolveIDEPath(custom)
	}
	for _, path := range getCandidatePaths() {
		if isValidIDEDir(path) {
			return path, nil
		}
	}
	if path := findFromRegistry(); path != "" {
		return path, nil
	}
	if path := findFromDiskScan(); path != "" {
		return path, nil
	}
	return "", fmt.Errorf("未找到 IDE，请通过 --dir 指定安装目录或 resources/app 目录")
}

func getLSFilename() string {
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	name := fmt.Sprintf("language_server_%s_%s", osName, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func findLanguageServer(base string) (string, error) {
	binDir := filepath.Join(base, "extensions", "antigravity", "bin")
	preferred := filepath.Join(binDir, getLSFilename())
	if info, err := os.Stat(preferred); err == nil && !info.IsDir() {
		return preferred, nil
	}
	// Allow inspecting an extracted package for another OS/architecture.
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, "language_server_") &&
			(strings.HasSuffix(name, "_x64") || strings.HasSuffix(name, "_arm64") ||
				strings.HasSuffix(name, "_x64.exe") || strings.HasSuffix(name, "_arm64.exe")) {
			candidates = append(candidates, filepath.Join(binDir, name))
		}
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("无法唯一确定语言服务器，找到 %d 个候选", len(candidates))
	}
	return candidates[0], nil
}

func runOperation(base, action string) error {
	logMsg("*", "IDE: "+base, 0)
	if action == "restore" {
		return restoreInstallation(base)
	}
	plan, err := inspectInstallation(base)
	for _, message := range plan.messages {
		logMsg("*", message, 0)
	}
	if err != nil {
		return err
	}
	if action == "check" {
		logMsg("+", fmt.Sprintf("检查完成，%d 个文件需要修改（未写入文件）", len(plan.files)), 0)
		return nil
	}
	if err := applyPlan(plan.files); err != nil {
		return err
	}
	logMsg("+", "全部完成，重启 IDE 即可生效", 0)
	return nil
}

func main() {
	dir := flag.String("dir", "", "IDE 安装目录或 resources/app 目录")
	check := flag.Bool("check", false, "只检查兼容性，不修改任何文件")
	patch := flag.Bool("patch", false, "修补本地工具数量限制")
	restore := flag.Bool("restore", false, "还原备份")
	flag.Parse()
	count, action := 0, ""
	for name, enabled := range map[string]bool{"check": *check, "patch": *patch, "restore": *restore} {
		if enabled {
			count++
			action = name
		}
	}
	if count > 1 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "请只指定 --check、--patch 或 --restore 中的一项")
		os.Exit(1)
	}
	fmt.Println("Antigravity IDE 工具数量限制修补工具  By https://github.com/Futureppo")
	if action != "check" {
		fmt.Println("注意: 请先完全退出 Antigravity IDE 的所有进程")
	}
	scanner := bufio.NewScanner(os.Stdin)
	base, err := detectIDEPath(*dir)
	if err != nil && action == "" && *dir == "" && os.Getenv("ANTIGRAVITY_DIR") == "" {
		fmt.Println("未找到 IDE，请输入安装目录或 resources/app 路径:")
		if scanner.Scan() {
			base, err = resolveIDEPath(scanner.Text())
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	interactive := action == ""
	if interactive {
		fmt.Print("\n1. 修补工具数量限制\n2. 还原备份\n3. 只检查兼容性\n> ")
		if !scanner.Scan() {
			return
		}
		action = map[string]string{"1": "patch", "2": "restore", "3": "check"}[strings.TrimSpace(scanner.Text())]
		if action == "" {
			fmt.Fprintln(os.Stderr, "无效选项")
			os.Exit(1)
		}
	}
	err = runOperation(base, action)
	if err != nil {
		logMsg("x", err.Error(), 0)
	}
	if interactive {
		fmt.Print("\n按回车键退出...")
		scanner.Scan()
	}
	if err != nil {
		os.Exit(1)
	}
}
