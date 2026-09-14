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

const desktopVersion = "2.13.0"

// Desktop directories are normalized to resources, whereas IDE directories
// are normalized to resources/app. Only the Windows desktop layout is supported.
func isValidDesktopDir(path string) bool {
	for _, name := range []string{"app.asar", filepath.Join("bin", "language_server.exe")} {
		info, err := os.Stat(filepath.Join(path, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func resolveDesktopPath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "\"")
	for _, candidate := range []string{path, filepath.Join(path, "resources")} {
		if isValidDesktopDir(candidate) {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("未找到 Windows 桌面版的 resources/app.asar 和 resources/bin/language_server.exe: %s", path)
}

func resolveClientPath(path, client string) (string, error) {
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
	if client != "auto" && client != "ide" && client != "desktop" {
		return "", fmt.Errorf("--client 只接受 auto、ide 或 desktop")
	}
	if custom == "" {
		custom = os.Getenv("ANTIGRAVITY_DIR")
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
		return "", fmt.Errorf("同时找到 IDE 和桌面版，请用 --client ide / --client desktop 或 --dir 选择目标")
	}
	if desktopPath != "" {
		return desktopPath, nil
	}
	if idePath != "" {
		return idePath, nil
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
	if product.Version != desktopVersion {
		return plan, fmt.Errorf("桌面版目前仅验证 %s，当前为 %s；未修改", desktopVersion, product.Version)
	}
	launcher, err := readASARFile(archive, "dist/languageServer.js")
	if err != nil {
		return plan, err
	}
	for _, marker := range []string{
		"const binName = isWindows ? 'language_server.exe' : 'language_server';",
		"path_1.default.join(process.resourcesPath, 'bin', binName)",
		"'--standalone'", "'--subclient_type'", "'hub'",
	} {
		if !bytes.Contains(launcher, []byte(marker)) {
			return plan, fmt.Errorf("桌面版后端启动结构未识别；未修改")
		}
	}
	ls, err := loadPatch(filepath.Join(base, "bin", "language_server.exe"))
	if err != nil {
		return plan, err
	}
	// Desktop 2.13.0 is a Windows PE64 package. Do not extend IDE or other
	// client/platform support by applying its 800-tool rule globally.
	if !bytes.HasPrefix(ls.original, []byte("MZ")) {
		return plan, fmt.Errorf("桌面版目前只支持 Windows x64 安装包")
	}
	var messages []string
	ls.updated, messages, err = planLanguageServer(ls.original, 800, toolLimit)
	if err != nil {
		return plan, err
	}
	plan.messages = append(plan.messages, "桌面版前端: 2.13.0 内置界面无需修改", "桌面版后端: resources/bin/language_server.exe")
	plan.messages = append(plan.messages, messages...)
	if !bytes.Equal(ls.original, ls.updated) {
		plan.files = append(plan.files, ls)
	}
	return plan, nil
}
