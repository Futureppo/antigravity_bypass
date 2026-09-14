package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const unlimitedJS = "async updateToolInConfigFile(a,b,c){const s=await this._getServerSpecFromConfigFile(a);s.disabledTools=s.disabledTools.filter(t=>t!==b);return this.saveMcpServerToConfigFile(a,s)}async updateServerInConfigFile(a,b){const s=await this._getServerSpecFromConfigFile(a);s.disabled=b;return this.saveMcpServerToConfigFile(a,s)}getPreferredCommandKey(a){};this._totalToolCount.textContent=`${n} tools`"

func TestFrontendDetection(t *testing.T) {
	legacy := "var warn=50,$longLimitName=100,Controller=class{enable(){throw Error(`Cannot enable more tools because it would exceed the limit of ${$longLimitName}.`)}}"
	for _, value := range []string{"100", "114514", "8192"} {
		source := strings.Replace(legacy, "=100,", "="+value+",", 1)
		patched, _, err := planJS([]byte("unrelated=114514;" + source))
		if err != nil || !bytes.Contains(patched, []byte("$longLimitName=8192,")) {
			t.Fatalf("%s: %s %v", value, patched, err)
		}
		again, _, err := planJS(patched)
		if err != nil || !bytes.Equal(patched, again) {
			t.Fatalf("not idempotent: %v", err)
		}
	}
	updated, _, err := planJS([]byte(unlimitedJS))
	if err != nil || string(updated) != unlimitedJS {
		t.Fatalf("native unlimited: %v", err)
	}
	for _, source := range []string{
		"var a=50,b=100,c=class{};unrelated=114514,",
		legacy + ";var other=50,$longLimitName=100,Another=class{};",
		strings.Replace(unlimitedJS, "const s=", "if(count>512)throw Error('too many');const s=", 1),
		strings.Replace(unlimitedJS, "${n} tools", "${n} / 512 tools", 1),
	} {
		if _, _, err := planJS([]byte(source)); err == nil {
			t.Fatalf("accepted unknown/ambiguous source: %s", source)
		}
	}
}

// A small PE64 with the two different control-flow shapes found in IDE 2.5.5.
// Addresses deliberately differ from the real installers: detection must follow
// RIP-relative references, not hard-coded file offsets.
func samplePE() []byte {
	b := make([]byte, 0x700)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], 0x80)
	copy(b[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[0x84:], 0x8664)
	binary.LittleEndian.PutUint16(b[0x86:], 2)
	binary.LittleEndian.PutUint16(b[0x94:], 240)
	binary.LittleEndian.PutUint16(b[0x98:], 0x20b)
	binary.LittleEndian.PutUint64(b[0x98+24:], 0x140000000)
	binary.LittleEndian.PutUint32(b[0x98+108:], 16)
	for i, s := range []struct {
		name          string
		off, size, va uint32
	}{{".text", 0x200, 0x400, 0x1000}, {".rdata", 0x600, 0x100, 0x2000}} {
		p := 0x188 + i*40
		copy(b[p:], s.name)
		binary.LittleEndian.PutUint32(b[p+8:], s.size)
		binary.LittleEndian.PutUint32(b[p+12:], s.va)
		binary.LittleEndian.PutUint32(b[p+16:], s.size)
		binary.LittleEndian.PutUint32(b[p+20:], s.off)
	}
	for i := 0x200; i < 0x600; i++ {
		b[i] = 0x90
	}
	copy(b[0x610:], toolCountError)
	c := b[0x200:0x600]
	copy(c[0x20:], []byte{0x48, 0x81, 0xfb, 0, 2, 0, 0, 0x7e, 0x67}) // CMP RBX,512; JLE 0x90
	c[0x8f] = 0xc3
	copy(c[0x120:], []byte{0x48, 0x81, 0xf9, 0, 2, 0, 0, 0x0f, 0x8f, 0x63, 0, 0, 0}) // CMP RCX,512; JG 0x190
	for _, pos := range []int{0x50, 0x1b0} {
		copy(c[pos:], []byte{0x48, 0x8d, 0x1d, 0, 0, 0, 0, 0xb9, 26, 0, 0, 0})
		binary.LittleEndian.PutUint32(c[pos+3:], uint32(0x2010-(0x1000+pos+7)))
	}
	// The old implementation incorrectly matched this unrelated Unicode branch.
	copy(c[0x2a0:], []byte{0x48, 0x83, 0xf9, 0x64, 0x0f, 0x8e, 0xa9, 0, 0, 0})
	copy(c[0x2b0:], []byte{0x48, 0x81, 0xf9, 0, 2, 0, 0, 0x7e, 0x52})
	return b
}

func TestBinaryPatching(t *testing.T) {
	source := samplePE()
	updated, messages, err := planLanguageServer(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d sites", len(messages))
	}
	want := bytes.Clone(source)
	for _, off := range []int{0x223, 0x323} {
		binary.LittleEndian.PutUint32(want[off:], 8192)
	}
	if !bytes.Equal(updated, want) {
		t.Fatal("changed bytes outside the two tool-count immediates")
	}
	again, _, err := planLanguageServer(updated)
	if err != nil || !bytes.Equal(updated, again) {
		t.Fatalf("not idempotent: %v", err)
	}
	for _, limit := range []uint32{128, 114514} {
		old := bytes.Clone(source)
		for _, off := range []int{0x223, 0x323} {
			binary.LittleEndian.PutUint32(old[off:], limit)
		}
		got, _, err := planLanguageServer(old)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("migration %d: %v", limit, err)
		}
	}
}

func TestBinaryRejectsUnknownOrUnsafeMatches(t *testing.T) {
	cases := map[string]func([]byte){
		"ARM64":               func(b []byte) { binary.LittleEndian.PutUint16(b[0x84:], 0xaa64) },
		"truncated section":   func(b []byte) { binary.LittleEndian.PutUint32(b[0x188+16:], 0xffffffff) },
		"wrong reference":     func(b []byte) { b[0x253]++ },
		"wrong string length": func(b []byte) { b[0x258]++ },
		"unknown limit":       func(b []byte) { binary.LittleEndian.PutUint32(b[0x223:], 1024) },
		"call in error setup": func(b []byte) { copy(b[0x229:], []byte{0xe8, 0, 0, 0, 0}) },
		"branch outside text": func(b []byte) { binary.LittleEndian.PutUint32(b[0x329:], 0x7fffffff) },
		"wrong message":       func(b []byte) { b[0x610] = 'X' },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			b := samplePE()
			change(b)
			if _, _, err := planLanguageServer(b); err == nil {
				t.Fatal("accepted unsafe binary")
			}
		})
	}
	for _, b := range [][]byte{nil, []byte("MZ"), {0x7f, 'E', 'L', 'F'}, samplePE()[:100]} {
		if _, _, err := planLanguageServer(b); err == nil {
			t.Fatal("accepted malformed binary")
		}
	}
}

func makeInstallation(t *testing.T) string {
	t.Helper()
	base := filepath.Join(t.TempDir(), "Antigravity IDE", "resources", "app")
	js := []byte(unlimitedJS)
	h := sha256.Sum256(js)
	product, _ := json.Marshal(map[string]any{"ideVersion": "2.5.5", "checksums": map[string]string{jsResourcePath: base64.RawStdEncoding.EncodeToString(h[:])}})
	for path, data := range map[string][]byte{
		filepath.Join("out", filepath.FromSlash(jsResourcePath)): js, "product.json": product,
		filepath.Join("extensions", "antigravity", "bin", "language_server_windows_x64.exe"): samplePE(),
	} {
		full := filepath.Join(base, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

func TestInstallationLifecycle(t *testing.T) {
	base := makeInstallation(t)
	for _, path := range []string{base, filepath.Dir(filepath.Dir(base))} {
		got, err := resolveIDEPath(path)
		if err != nil || got != base {
			t.Fatalf("path: %s %v", got, err)
		}
	}
	t.Setenv("ANTIGRAVITY_DIR", base)
	if _, err := detectIDEPath(filepath.Join(base, "missing")); err == nil {
		t.Fatal("invalid explicit path fell back to installed IDE")
	}
	plan, err := inspectInstallation(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.files) != 1 {
		t.Fatalf("expected only LS patch, got %d", len(plan.files))
	}
	p := plan.files[0]
	if _, err := os.Stat(p.path + ".backup"); !os.IsNotExist(err) {
		t.Fatal("inspection created backup")
	}
	if err := applyPlan(plan.files); err != nil {
		t.Fatal(err)
	}
	again, err := inspectInstallation(base)
	if err != nil || len(again.files) != 0 {
		t.Fatalf("second run: %v", err)
	}
	if err := restoreInstallation(base); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p.path)
	if !bytes.Equal(data, p.original) {
		t.Fatal("restore differed from original")
	}
	if _, err := os.Stat(p.path + ".backup"); !os.IsNotExist(err) {
		t.Fatal("backup retained after successful restore")
	}
}

func TestBackupAndUpdateProtection(t *testing.T) {
	base := makeInstallation(t)
	plan, err := inspectInstallation(base)
	if err != nil {
		t.Fatal(err)
	}
	p := plan.files[0]
	if err := os.WriteFile(p.path+".backup", []byte("old version"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(plan.files); err == nil {
		t.Fatal("overwrote mismatched backup")
	}
	if err := os.Remove(p.path + ".backup"); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(plan.files); err != nil {
		t.Fatal(err)
	}
	newVersion := append(bytes.Clone(p.updated), 0)
	if err := os.WriteFile(p.path, newVersion, 0755); err != nil {
		t.Fatal(err)
	}
	if err := restoreInstallation(base); err == nil {
		t.Fatal("restored old binary over newer version")
	}
	current, _ := os.ReadFile(p.path)
	if !bytes.Equal(current, newVersion) {
		t.Fatal("modified updated binary")
	}
}

func TestNoPartialFrontendPatchOnBackendFailure(t *testing.T) {
	base := makeInstallation(t)
	jsPath := filepath.Join(base, "out", filepath.FromSlash(jsResourcePath))
	source := []byte("a=50,b=100,c=class{f(){throw Error(`Cannot enable more tools because it would exceed the limit of ${b}.`)}}")
	if err := os.WriteFile(jsPath, source, 0644); err != nil {
		t.Fatal(err)
	}
	ls, err := findLanguageServer(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ls, []byte("unsupported"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := runOperation(base, "patch"); err == nil {
		t.Fatal("accepted unsupported backend")
	}
	got, _ := os.ReadFile(jsPath)
	if !bytes.Equal(got, source) {
		t.Fatal("changed JS despite failed backend inspection")
	}
	if _, err := os.Stat(jsPath + ".backup"); !os.IsNotExist(err) {
		t.Fatal("created backup despite failed inspection")
	}
}

func TestMigratePatchedLimitPreservesOriginalBackup(t *testing.T) {
	base := makeInstallation(t)
	plan, err := inspectInstallation(base)
	if err != nil {
		t.Fatal(err)
	}
	p := plan.files[0]
	for _, off := range []int{0x223, 0x323} {
		binary.LittleEndian.PutUint32(p.updated[off:], 114514)
	}
	if err := applyPlan([]filePatch{p}); err != nil {
		t.Fatal(err)
	}
	next, err := inspectInstallation(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(next.files); err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(p.path)
	if binary.LittleEndian.Uint32(current[0x223:]) != 8192 {
		t.Fatal("old limit was not migrated")
	}
	if err := restoreInstallation(base); err != nil {
		t.Fatal(err)
	}
	current, _ = os.ReadFile(p.path)
	if !bytes.Equal(current, p.original) {
		t.Fatal("migration lost original backup")
	}
}

func TestFrontendChecksumRepair(t *testing.T) {
	base := makeInstallation(t)
	jsPath := filepath.Join(base, "out", filepath.FromSlash(jsResourcePath))
	source := []byte("a=50,b=8192,c=class{f(){throw Error(`Cannot enable more tools because it would exceed the limit of ${b}.`)}}")
	if err := os.WriteFile(jsPath, source, 0644); err != nil {
		t.Fatal(err)
	}
	plan, err := inspectInstallation(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.files) != 2 {
		t.Fatalf("expected checksum and LS repairs, got %d", len(plan.files))
	}
	if err := applyPlan(plan.files); err != nil {
		t.Fatal(err)
	}
	var product struct {
		Checksums map[string]string `json:"checksums"`
	}
	raw, _ := os.ReadFile(filepath.Join(base, "product.json"))
	if err := json.Unmarshal(raw, &product); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(source)
	if product.Checksums[jsResourcePath] != base64.RawStdEncoding.EncodeToString(hash[:]) {
		t.Fatal("wrong checksum")
	}
	if err := restoreInstallation(base); err != nil {
		t.Fatal(err)
	}
}
