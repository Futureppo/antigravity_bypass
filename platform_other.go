//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func getCandidatePaths() []string {
	var candidates []string
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates,
			"/Applications/Antigravity IDE.app/Contents/Resources/app",
			"/Applications/Antigravity.app/Contents/Resources/app",
		)
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, "Applications", "Antigravity IDE.app", "Contents", "Resources", "app"),
				filepath.Join(home, "Applications", "Antigravity.app", "Contents", "Resources", "app"),
			)
		}
	case "linux":
		candidates = append(candidates,
			"/usr/share/antigravity-ide/resources/app",
			"/usr/lib/antigravity-ide/resources/app",
			"/opt/antigravity-ide/resources/app",
			"/opt/Antigravity IDE/resources/app",
			"/usr/share/antigravity/resources/app",
			"/usr/lib/antigravity/resources/app",
			"/opt/antigravity/resources/app",
			"/opt/Antigravity/resources/app",
		)
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, ".local", "share", "antigravity-ide", "resources", "app"),
				filepath.Join(home, ".local", "share", "Antigravity IDE", "resources", "app"),
				filepath.Join(home, ".local", "share", "antigravity", "resources", "app"),
				filepath.Join(home, ".local", "share", "Antigravity", "resources", "app"),
			)
		}
	}
	return candidates
}

func findFromRegistry() string {
	return ""
}

func findFromDiskScan() string {
	return ""
}
