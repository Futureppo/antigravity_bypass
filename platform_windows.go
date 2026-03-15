//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func getCandidatePaths() []string {
	var candidates []string
	envDirs := []string{
		os.Getenv("LOCALAPPDATA"),
		os.Getenv("APPDATA"),
		os.Getenv("PROGRAMFILES"),
		os.Getenv("PROGRAMFILES(X86)"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		envDirs = append(envDirs, home)
	}

	for _, base := range envDirs {
		if base == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(base, "Programs", "Antigravity", "resources", "app"),
			filepath.Join(base, "Antigravity", "resources", "app"),
		)
	}
	return candidates
}

func findFromRegistry() string {
	regSearches := []struct {
		root    registry.Key
		keyPath string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Antigravity.exe`},
	}

	for _, rs := range regSearches {
		key, err := registry.OpenKey(rs.root, rs.keyPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		exePath, _, err := key.GetStringValue("")
		key.Close()
		if err != nil || exePath == "" {
			continue
		}
		if _, statErr := os.Stat(exePath); statErr != nil {
			continue
		}
		appDir := filepath.Join(filepath.Dir(exePath), "resources", "app")
		if isValidIDEDir(appDir) {
			return appDir
		}
	}

	uninstallBases := []struct {
		root    registry.Key
		keyPath string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, ub := range uninstallBases {
		baseKey, err := registry.OpenKey(ub.root, ub.keyPath, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		subKeys, err := baseKey.ReadSubKeyNames(-1)
		baseKey.Close()
		if err != nil {
			continue
		}
		for _, subName := range subKeys {
			if !strings.Contains(strings.ToLower(subName), "antigravity") {
				continue
			}
			subKey, err := registry.OpenKey(ub.root, ub.keyPath+`\`+subName, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			installLoc, _, err := subKey.GetStringValue("InstallLocation")
			subKey.Close()
			if err != nil || installLoc == "" {
				continue
			}
			appDir := filepath.Join(installLoc, "resources", "app")
			if isValidIDEDir(appDir) {
				return appDir
			}
		}
	}

	return ""
}

func findFromDiskScan() string {
	username := os.Getenv("USERNAME")
	searchSubdirs := []string{
		"Program Files",
		"Program Files (x86)",
	}
	if username != "" {
		searchSubdirs = append(searchSubdirs,
			filepath.Join("Users", username, "AppData", "Local", "Programs"),
		)
	}

	for letter := 'C'; letter <= 'Z'; letter++ {
		drive := string(letter) + `:\`
		if _, err := os.Stat(drive); err != nil {
			continue
		}
		for _, sub := range searchSubdirs {
			appDir := filepath.Join(drive, sub, "Antigravity", "resources", "app")
			if isValidIDEDir(appDir) {
				return appDir
			}
		}
	}
	return ""
}
