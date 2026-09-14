package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const desktopLauncherFixture = "const binName = isWindows ? 'language_server.exe' : 'language_server';\nexports.LS_BINARY = path_1.default.join(process.resourcesPath, 'bin', binName);\nconst args = ['--standalone', '--subclient_type', 'hub'];"

func makeASAR(t *testing.T, files map[string]string) []byte {
	t.Helper()
	root := asarEntry{Files: make(map[string]asarEntry)}
	var payload []byte
	for name, content := range files {
		entry := asarEntry{Size: uint64(len(content)), Offset: strconv.Itoa(len(payload))}
		entry.Integrity = &struct{ Algorithm, Hash string }{"SHA256", digest([]byte(content))}
		parts := strings.Split(name, "/")
		tree := root.Files
		for _, part := range parts[:len(parts)-1] {
			dir := tree[part]
			if dir.Files == nil {
				dir.Files = make(map[string]asarEntry)
				tree[part] = dir
			}
			tree = dir.Files
		}
		tree[parts[len(parts)-1]] = entry
		payload = append(payload, content...)
	}
	raw, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	headerSize := 8 + (len(raw)+3)/4*4
	result := make([]byte, 8+headerSize)
	binary.LittleEndian.PutUint32(result, 4)
	binary.LittleEndian.PutUint32(result[4:], uint32(headerSize))
	binary.LittleEndian.PutUint32(result[8:], uint32(headerSize-4))
	binary.LittleEndian.PutUint32(result[12:], uint32(len(raw)))
	copy(result[16:], raw)
	return append(result, payload...)
}

func makeDesktopInstallation(t *testing.T, version string) string {
	t.Helper()
	base := filepath.Join(t.TempDir(), "Antigravity", "resources")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	product := `{"name":"antigravity","productName":"Antigravity","description":"Antigravity - Agentic Desktop Application","main":"dist/main.js","version":"` + version + `"}`
	archive := makeASAR(t, map[string]string{"package.json": product, "dist/languageServer.js": desktopLauncherFixture})
	if err := os.WriteFile(filepath.Join(base, "app.asar"), archive, 0644); err != nil {
		t.Fatal(err)
	}
	ls := samplePE()
	for _, offset := range []int{0x223, 0x323} {
		binary.LittleEndian.PutUint32(ls[offset:], 800)
	}
	if err := os.WriteFile(filepath.Join(base, "bin", "language_server.exe"), ls, 0755); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestDesktopLifecycleAndSelection(t *testing.T) {
	base := makeDesktopInstallation(t, desktopVersion)
	archivePath := filepath.Join(base, "app.asar")
	archive, _ := os.ReadFile(archivePath)
	for _, path := range []string{base, filepath.Dir(base)} {
		for _, client := range []string{"auto", "desktop"} {
			got, err := detectClientPath(path, client)
			if err != nil || got != base {
				t.Fatalf("resolve %s %s: %s %v", path, client, got, err)
			}
		}
		if _, err := detectClientPath(path, "ide"); err == nil {
			t.Fatal("desktop selected as IDE")
		}
	}
	ide := makeInstallation(t)
	if _, err := detectClientPath(ide, "desktop"); err == nil {
		t.Fatal("IDE selected as desktop")
	}
	t.Setenv("ANTIGRAVITY_DIR", base)
	if got, err := detectClientPath("", "desktop"); err != nil || got != base {
		t.Fatalf("environment: %s %v", got, err)
	}
	if got, err := detectClientPath(ide, "ide"); err != nil || got != ide {
		t.Fatalf("explicit path must override environment: %s %v", got, err)
	}
	if _, err := detectClientPath(filepath.Join(base, "missing"), "desktop"); err == nil {
		t.Fatal("invalid explicit path fell back")
	}
	if _, err := detectClientPath(base, "cli"); err == nil {
		t.Fatal("accepted unsupported client")
	}
	if err := runOperation(base, "check"); err != nil {
		t.Fatal(err)
	}
	lsPath := filepath.Join(base, "bin", "language_server.exe")
	original, _ := os.ReadFile(lsPath)
	if _, err := os.Stat(lsPath + ".backup"); !os.IsNotExist(err) {
		t.Fatal("check created backup")
	}
	if err := runOperation(base, "patch"); err != nil {
		t.Fatal(err)
	}
	patched, _ := os.ReadFile(lsPath)
	expected := bytes.Clone(original)
	for _, offset := range []int{0x223, 0x323} {
		binary.LittleEndian.PutUint32(expected[offset:], 8192)
	}
	if !bytes.Equal(patched, expected) {
		t.Fatal("changed bytes outside desktop tool-count constants")
	}
	if err := runOperation(base, "patch"); err != nil {
		t.Fatal(err)
	}
	again, err := inspectDesktopInstallation(base)
	if err != nil || len(again.files) != 0 {
		t.Fatalf("not idempotent: %v", err)
	}
	if err := runOperation(base, "restore"); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(lsPath)
	if !bytes.Equal(restored, original) {
		t.Fatal("desktop restore differed")
	}
	currentArchive, _ := os.ReadFile(archivePath)
	if !bytes.Equal(currentArchive, archive) {
		t.Fatal("modified Electron archive")
	}
	if _, err := os.Stat(lsPath + ".backup"); !os.IsNotExist(err) {
		t.Fatal("backup remains after restore")
	}
}

func TestDesktopRuleDoesNotChangeIDESupport(t *testing.T) {
	base := makeDesktopInstallation(t, desktopVersion)
	lsPath := filepath.Join(base, "bin", "language_server.exe")
	data, _ := os.ReadFile(lsPath)
	if _, _, err := planLanguageServer(data); err == nil {
		t.Fatal("desktop 800 rule leaked into IDE matching")
	}
	for _, offset := range []int{0x223, 0x323} {
		binary.LittleEndian.PutUint32(data[offset:], 512)
	}
	if err := os.WriteFile(lsPath, data, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectDesktopInstallation(base); err == nil {
		t.Fatal("desktop accepted an unverified 512-limit backend")
	}
}

func TestDesktopAndIDEAutoSelection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows installation discovery")
	}
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("ANTIGRAVITY_DIR", "")
	ide := filepath.Join(local, "Programs", "Antigravity IDE", "resources", "app")
	desktop := filepath.Join(local, "Programs", "Antigravity", "resources")
	for _, pair := range []struct {
		source, destination string
		files               []string
	}{
		{makeInstallation(t), ide, []string{"product.json", filepath.Join("out", filepath.FromSlash(jsResourcePath)), filepath.Join("extensions", "antigravity", "bin", "language_server_windows_x64.exe")}},
		{makeDesktopInstallation(t, desktopVersion), desktop, []string{"app.asar", filepath.Join("bin", "language_server.exe")}},
	} {
		for _, file := range pair.files {
			data, err := os.ReadFile(filepath.Join(pair.source, file))
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(pair.destination, file)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := detectClientPath("", "auto"); err == nil {
		t.Fatal("silently chose one of two different clients")
	}
	for client, expected := range map[string]string{"ide": ide, "desktop": desktop} {
		got, err := detectClientPath("", client)
		if err != nil || got != expected {
			t.Fatalf("%s: got %s, %v", client, got, err)
		}
	}
}

func TestDesktopRejectsUnverifiedPackages(t *testing.T) {
	// Rebuild integrity records so these cases exercise the package guards,
	// rather than being rejected earlier as corrupted archive entries.
	rewriteEntry := func(base, name, old, replacement string) {
		t.Helper()
		p := filepath.Join(base, "app.asar")
		files := make(map[string]string)
		for _, entry := range []string{"package.json", "dist/languageServer.js"} {
			data, err := readASARFile(p, entry)
			if err != nil {
				t.Fatal(err)
			}
			files[entry] = string(data)
		}
		files[name] = strings.ReplaceAll(files[name], old, replacement)
		if err := os.WriteFile(p, makeASAR(t, files), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for name, modify := range map[string]func(string){
		"version": func(base string) {
			rewriteEntry(base, "package.json", desktopVersion, "2.14.0")
		},
		"identity": func(base string) {
			rewriteEntry(base, "package.json", "Agentic Desktop", "Another Desktop")
		},
		"launcher": func(base string) {
			rewriteEntry(base, "dist/languageServer.js", "--standalone", "--other-mode")
		},
		"ARM64": func(base string) {
			p := filepath.Join(base, "bin", "language_server.exe")
			b, _ := os.ReadFile(p)
			binary.LittleEndian.PutUint16(b[0x84:], 0xaa64)
			os.WriteFile(p, b, 0755)
		},
		"unknown binary": func(base string) {
			os.WriteFile(filepath.Join(base, "bin", "language_server.exe"), []byte("invalid"), 0755)
		},
	} {
		t.Run(name, func(t *testing.T) {
			base := makeDesktopInstallation(t, desktopVersion)
			modify(base)
			p := filepath.Join(base, "bin", "language_server.exe")
			before, _ := os.ReadFile(p)
			if err := runOperation(base, "patch"); err == nil {
				t.Fatal("patched unsupported desktop package")
			}
			after, _ := os.ReadFile(p)
			if !bytes.Equal(before, after) {
				t.Fatal("failed inspection modified binary")
			}
			if _, err := os.Stat(p + ".backup"); !os.IsNotExist(err) {
				t.Fatal("failed inspection created backup")
			}
		})
	}
	base := makeDesktopInstallation(t, "2.14.0")
	if _, err := inspectDesktopInstallation(base); err == nil || !strings.Contains(err.Error(), "2.14.0") {
		t.Fatalf("unknown valid package version: %v", err)
	}
}

func TestASARReader(t *testing.T) {
	valid := makeASAR(t, map[string]string{"package.json": "package metadata", "dist/languageServer.js": desktopLauncherFixture})
	path := filepath.Join(t.TempDir(), "app.asar")
	if err := os.WriteFile(path, valid, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readASARFile(path, "dist/languageServer.js")
	if err != nil || string(got) != desktopLauncherFixture {
		t.Fatalf("nested entry: %v", err)
	}
	for _, name := range []string{"missing", "../package.json", "dist", "/package.json"} {
		if _, err := readASARFile(path, name); err == nil {
			t.Fatalf("accepted invalid entry %s", name)
		}
	}
	for name, modify := range map[string]func([]byte) []byte{
		"truncated":           func(b []byte) []byte { return b[:12] },
		"oversized header":    func(b []byte) []byte { binary.LittleEndian.PutUint32(b[4:], 0xffffffff); return b },
		"oversized JSON":      func(b []byte) []byte { binary.LittleEndian.PutUint32(b[12:], 0xffffffff); return b },
		"invalid header JSON": func(b []byte) []byte { b[16] = '!'; return b },
		"changed payload": func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("package metadata"), []byte("altered metadata"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, modify(bytes.Clone(valid)), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := readASARFile(path, "package.json"); err == nil {
				t.Fatal("accepted malformed ASAR")
			}
		})
	}
}
