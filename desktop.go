package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Desktop directories are normalized to resources, whereas IDE directories
// are normalized to resources/app. Windows, Linux, and macOS layouts are
// recognized; the backend's own format/architecture is validated separately.
func findDesktopLanguageServer(base string) (string, error) {
	for _, name := range []string{"language_server.exe", "language_server"} {
		path := filepath.Join(base, "bin", name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到桌面版后端: %s", filepath.Join(base, "bin", "language_server"))
}

func isValidDesktopDir(path string) bool {
	if info, err := os.Stat(filepath.Join(path, "app.asar")); err != nil || !info.Mode().IsRegular() {
		return false
	}
	_, err := findDesktopLanguageServer(path)
	return err == nil
}

func resolveDesktopPath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "\"")
	for _, candidate := range []string{path, filepath.Join(path, "resources"), filepath.Join(path, "Contents", "Resources")} {
		if isValidDesktopDir(candidate) {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("未找到桌面版的 resources/app.asar 和 resources/bin/language_server(.exe): %s", path)
}

func resolveClientPath(path, client string) (string, error) {
	if client == "cli" {
		return resolveCLIPath(path)
	}
	if client == "auto" {
		if base, err := resolveCLIPath(path); err == nil {
			return base, nil
		}
	}
	if client != "desktop" {
		if base, err := resolveIDEPath(path); err == nil {
			return base, nil
		}
	}
	if client != "ide" {
		if base, err := resolveDesktopPath(path); err == nil {
			return base, nil
		}
	}
	return "", fmt.Errorf("路径与所选客户端 (%s) 不匹配: %s", client, path)
}

func detectClientPath(custom, client string) (string, error) {
	if client != "auto" && client != "ide" && client != "desktop" && client != "cli" {
		return "", fmt.Errorf("--client 只接受 auto、ide、desktop 或 cli")
	}
	if custom == "" {
		custom = os.Getenv("ANTIGRAVITY_DIR")
	}
	if client == "cli" {
		return detectCLIPath(custom)
	}
	if custom != "" {
		return resolveClientPath(custom, client)
	}
	var idePath, desktopPath string
	if client != "desktop" {
		idePath, _ = detectIDEPath("")
	}
	if client != "ide" {
		for _, candidate := range getCandidatePaths() {
			if path, err := resolveDesktopPath(filepath.Dir(candidate)); err == nil {
				desktopPath = path
				break
			}
		}
		if desktopPath == "" {
			desktopPath = findDesktopFromRegistry()
		}
	}
	if idePath != "" && desktopPath != "" {
		return "", fmt.Errorf("同时找到 IDE 和桌面版，请用 --client ide / --client desktop / --client cli 或 --dir 选择目标")
	}
	if desktopPath != "" {
		return desktopPath, nil
	}
	if idePath != "" {
		return idePath, nil
	}
	if client == "auto" {
		if path, err := detectCLIPath(""); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到客户端 (%s)，请用 --dir 指定安装目录", client)
}

type asarEntry struct {
	Files     map[string]asarEntry              `json:"files"`
	Size      uint64                            `json:"size"`
	Offset    string                            `json:"offset"`
	Unpacked  bool                              `json:"unpacked"`
	Link      string                            `json:"link"`
	Integrity *struct{ Algorithm, Hash string } `json:"integrity"`
}

// Read just the small identity/launcher files, without extracting or changing
// app.asar (and thus without invalidating Electron's archive integrity data).
func readASARFile(path, name string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var prefix [16]byte
	if _, err := io.ReadFull(f, prefix[:]); err != nil {
		return nil, err
	}
	headerSize := uint64(binary.LittleEndian.Uint32(prefix[4:8]))
	jsonSize := uint64(binary.LittleEndian.Uint32(prefix[12:16]))
	if binary.LittleEndian.Uint32(prefix[:4]) != 4 || headerSize < 8 || headerSize > 8<<20 ||
		binary.LittleEndian.Uint32(prefix[8:12]) != uint32(headerSize-4) ||
		jsonSize == 0 || jsonSize > headerSize-8 || 8+headerSize > uint64(info.Size()) {
		return nil, fmt.Errorf("app.asar 头部无效")
	}
	raw := make([]byte, int(jsonSize))
	if _, err := io.ReadFull(f, raw); err != nil {
		return nil, err
	}
	var entry asarEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, fmt.Errorf("app.asar 索引无效: %w", err)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("无效 ASAR 路径")
		}
		var ok bool
		entry, ok = entry.Files[part]
		if !ok || entry.Link != "" {
			return nil, fmt.Errorf("app.asar 未找到普通文件: %s", name)
		}
	}
	if entry.Unpacked || entry.Files != nil || entry.Size == 0 || entry.Size > 1<<20 {
		return nil, fmt.Errorf("app.asar 文件类型或大小不受支持: %s", name)
	}
	offset, err := strconv.ParseUint(entry.Offset, 10, 64)
	remaining := uint64(info.Size()) - 8 - headerSize
	if err != nil || offset > remaining || entry.Size > remaining-offset {
		return nil, fmt.Errorf("app.asar 文件范围无效: %s", name)
	}
	data := make([]byte, int(entry.Size))
	_, err = f.ReadAt(data, int64(8+headerSize+offset))
	if err != nil {
		return nil, err
	}
	if entry.Integrity != nil && (entry.Integrity.Algorithm != "SHA256" || entry.Integrity.Hash != digest(data)) {
		return nil, fmt.Errorf("app.asar 文件校验失败: %s", name)
	}
	return data, nil
}

func inspectDesktopInstallation(base string) (installationPlan, error) {
	var plan installationPlan
	archive := filepath.Join(base, "app.asar")
	raw, err := readASARFile(archive, "package.json")
	if err != nil {
		return plan, err
	}
	var product struct{ Name, ProductName, Version, Description, Main string }
	if err := json.Unmarshal(raw, &product); err != nil {
		return plan, err
	}
	if product.Name != "antigravity" || product.ProductName != "Antigravity" ||
		product.Description != "Antigravity - Agentic Desktop Application" || product.Main != "dist/main.js" {
		return plan, fmt.Errorf("app.asar 不是已识别的 Antigravity 独立桌面版")
	}
	plan.messages = append(plan.messages, "桌面版版本: "+product.Version)
	launcher, err := readASARFile(archive, "dist/languageServer.js")
	if err != nil {
		return plan, err
	}
	for _, marker := range []string{
		"const binName = isWindows ? 'language_server.exe' : 'language_server';",
		"path_1.default.join(process.resourcesPath, 'bin', binName)",
	} {
		if !bytes.Contains(launcher, []byte(marker)) {
			return plan, fmt.Errorf("桌面版后端启动结构未识别；未修改")
		}
	}
	legacyArgs := bytes.Contains(launcher, []byte("'--standalone'")) &&
		bytes.Contains(launcher, []byte("'--subclient_type'"))
	splitArgs := bytes.Contains(launcher, []byte("baseLsArgs")) &&
		bytes.Contains(launcher, []byte("'--enable_sidecars'"))
	if !legacyArgs && !splitArgs {
		return plan, fmt.Errorf("桌面版后端启动参数未识别；未修改")
	}
	lsPath, err := findDesktopLanguageServer(base)
	if err != nil {
		return plan, err
	}
	ls, err := loadPatch(lsPath)
	if err != nil {
		return plan, err
	}
	var messages []string
	ls.updated, messages, err = planLanguageServer(ls.original, 100, 128, 512, 800, 114514, toolLimit)
	if err != nil {
		return plan, err
	}
	plan.messages = append(plan.messages, "桌面版前端: "+product.Version+" 内置界面无需修改", "桌面版后端: "+lsPath)
	plan.messages = append(plan.messages, messages...)
	if !bytes.Equal(ls.original, ls.updated) {
		plan.files = append(plan.files, ls)
	}
	return plan, nil
}
