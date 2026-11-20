package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func cliBinaryName() string {
	if runtime.GOOS == "windows" {
		return "agy.exe"
	}
	return "agy"
}

func cliSharedPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "bin", cliBinaryName()), nil
}

func cliStandalonePath() (string, error) {
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			return "", fmt.Errorf("未找到 LOCALAPPDATA")
		}
		return filepath.Join(local, "agy", "bin", cliBinaryName()), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin", cliBinaryName()), nil
}

func isCLIPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	name := strings.ToLower(filepath.Base(path))
	return name == "agy" || name == "agy.exe"
}

func resolveCLIPath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "\"")
	candidates := []string{path}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		candidates = append(candidates,
			filepath.Join(path, cliBinaryName()),
			filepath.Join(path, "bin", cliBinaryName()))
	}
	for _, candidate := range candidates {
		if isCLIPath(candidate) {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("无效的 Antigravity CLI 路径: %s", path)
}

func detectCLIPath(custom string) (string, error) {
	if custom == "" {
		custom = os.Getenv("ANTIGRAVITY_DIR")
	}
	if custom != "" {
		return resolveCLIPath(custom)
	}
	var candidates []string
	if path, err := cliStandalonePath(); err == nil {
		candidates = append(candidates, path)
	}
	if path, err := cliSharedPath(); err == nil {
		candidates = append(candidates, path)
	}
	if path, err := exec.LookPath(cliBinaryName()); err == nil {
		candidates = append(candidates, path)
	}
	for _, path := range candidates {
		if isCLIPath(path) {
			return filepath.Abs(path)
		}
	}
	return "", fmt.Errorf("未找到 Antigravity CLI (agy)，请用 --dir 指定 agy 二进制或所在目录")
}

func cliVersion(path string) string {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "agy.version.json"))
	if err != nil {
		return "未知"
	}
	var sidecar struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &sidecar) != nil || sidecar.Version == "" {
		return "未知"
	}
	return sidecar.Version
}

func inspectCLIInstallation(path string) (installationPlan, error) {
	var plan installationPlan
	plan.messages = append(plan.messages, "CLI 版本: "+cliVersion(path))
	ls, err := loadPatch(path)
	if err != nil {
		return plan, err
	}
	var messages []string
	ls.updated, messages, err = planLanguageServer(ls.original, 100, 128, 512, 800, 114514, toolLimit)
	if err != nil {
		return plan, err
	}
	plan.messages = append(plan.messages, messages...)
	if !bytes.Equal(ls.original, ls.updated) {
		plan.files = append(plan.files, ls)
	}
	return plan, nil
}
