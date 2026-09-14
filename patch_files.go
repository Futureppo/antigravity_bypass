package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type filePatch struct {
	path              string
	original, updated []byte
	mode              os.FileMode
}

type installationPlan struct {
	files    []filePatch
	messages []string
}

func loadPatch(path string) (filePatch, error) {
	info, err := os.Stat(path)
	if err != nil {
		return filePatch{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return filePatch{}, err
	}
	return filePatch{path: path, original: data, mode: info.Mode().Perm()}, nil
}

func inspectInstallation(base string) (installationPlan, error) {
	var plan installationPlan
	product, err := loadPatch(filepath.Join(base, "product.json"))
	if err != nil {
		return plan, err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(product.original, &config); err != nil {
		return plan, fmt.Errorf("product.json 无效: %w", err)
	}
	var version string
	if err := json.Unmarshal(config["ideVersion"], &version); err != nil {
		version = "未知"
	}
	plan.messages = append(plan.messages, "IDE 版本: "+version)
	js, err := loadPatch(filepath.Join(base, "out", filepath.FromSlash(jsResourcePath)))
	if err != nil {
		return plan, err
	}
	var message string
	js.updated, message, err = planJS(js.original)
	if err != nil {
		return plan, err
	}
	plan.messages = append(plan.messages, message)
	if !bytes.Equal(js.original, js.updated) {
		plan.files = append(plan.files, js)
	}
	// Repair a stale checksum even if a previous run already patched the JS.
	if raw, exists := config["checksums"]; exists {
		var checksums map[string]string
		if err := json.Unmarshal(raw, &checksums); err != nil {
			return plan, fmt.Errorf("product.json checksums 无效: %w", err)
		}
		if old, exists := checksums[jsResourcePath]; exists {
			hash := sha256.Sum256(js.updated)
			value := base64.RawStdEncoding.EncodeToString(hash[:])
			if old != value {
				checksums[jsResourcePath] = value
				config["checksums"], err = json.Marshal(checksums)
				if err != nil {
					return plan, err
				}
				product.updated, err = json.MarshalIndent(config, "", "\t")
				if err != nil {
					return plan, err
				}
				product.updated = append(product.updated, '\n')
				plan.files = append(plan.files, product)
				plan.messages = append(plan.messages, "前端: 同步 product.json 校验值")
			}
		}
	}
	lsPath, err := findLanguageServer(base)
	if err != nil {
		return plan, err
	}
	ls, err := loadPatch(lsPath)
	if err != nil {
		return plan, err
	}
	var messages []string
	ls.updated, messages, err = planLanguageServer(ls.original)
	if err != nil {
		return plan, err
	}
	plan.messages = append(plan.messages, messages...)
	if !bytes.Equal(ls.original, ls.updated) {
		plan.files = append(plan.files, ls)
	}
	return plan, nil
}

type backupRecord struct{ Original, Patched, Previous string }

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func writeExclusive(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".antigravity-bypass-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func validateCurrent(p filePatch) error {
	current, err := os.ReadFile(p.path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, p.original) {
		return fmt.Errorf("检查后文件发生变化: %s；请关闭所选客户端后重试", p.path)
	}
	return nil
}

func applyPlan(files []filePatch) error {
	// Validate every target and pre-existing backup before creating or changing files.
	originalHashes := make(map[string]string)
	for _, p := range files {
		if err := validateCurrent(p); err != nil {
			return err
		}
		backup, err := os.ReadFile(p.path + ".backup")
		originalHashes[p.path] = digest(p.original)
		if err == nil {
			originalHashes[p.path] = digest(backup)
			if !bytes.Equal(backup, p.original) {
				var record backupRecord
				raw, recordErr := os.ReadFile(p.path + ".backup.json")
				if recordErr != nil || json.Unmarshal(raw, &record) != nil || record.Original != digest(backup) ||
					(record.Patched != digest(p.original) && record.Previous != digest(p.original)) {
					return fmt.Errorf("备份属于其他版本或状态: %s.backup；请先将旧备份移走后重试", p.path)
				}
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, p := range files {
		if err := writeExclusive(p.path+".backup", p.original, p.mode); err != nil && !os.IsExist(err) {
			return fmt.Errorf("备份失败: %w", err)
		}
		record, err := json.Marshal(backupRecord{Original: originalHashes[p.path], Patched: digest(p.updated), Previous: digest(p.original)})
		if err != nil {
			return err
		}
		if err := replaceFile(p.path+".backup.json", record, 0644); err != nil {
			return fmt.Errorf("写入备份校验失败: %w", err)
		}
	}
	for i, p := range files {
		err := validateCurrent(p)
		if err == nil {
			err = replaceFile(p.path, p.updated, p.mode)
		}
		if err != nil {
			// Targets already changed by this invocation are rolled back. Keep all
			// backups so interrupted/failed runs remain recoverable.
			for j := i - 1; j >= 0; j-- {
				err = errors.Join(err, replaceFile(files[j].path, files[j].original, files[j].mode))
			}
			return fmt.Errorf("写入失败，已尝试回滚: %w", err)
		}
		logMsg("+", "已修补: "+filepath.Base(p.path), 0)
	}
	return nil
}

func restoreInstallation(base string) error {
	paths := []string{filepath.Join(base, "out", filepath.FromSlash(jsResourcePath)), filepath.Join(base, "product.json")}
	backups, err := filepath.Glob(filepath.Join(base, "extensions", "antigravity", "bin", "language_server_*.backup"))
	if err != nil {
		return err
	}
	for _, path := range backups {
		paths = append(paths, path[:len(path)-len(".backup")])
	}
	return restoreFiles(paths)
}

func restoreFiles(paths []string) error {
	var files []filePatch
	for _, path := range paths {
		backup, err := os.ReadFile(path + ".backup")
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		p, err := loadPatch(path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path + ".backup.json")
		if err != nil {
			return fmt.Errorf("备份缺少校验记录，请核对版本后手动恢复: %s.backup", path)
		}
		var record backupRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return err
		}
		if digest(backup) != record.Original || (digest(p.original) != record.Patched && digest(p.original) != record.Original && digest(p.original) != record.Previous) {
			return fmt.Errorf("文件或备份已变化（可能客户端已更新），拒绝恢复旧版本: %s", path)
		}
		p.updated = backup
		files = append(files, p)
	}
	if len(files) == 0 {
		logMsg("-", "未找到备份文件", 0)
		return nil
	}
	for _, p := range files {
		if err := validateCurrent(p); err != nil {
			return err
		}
		if err := replaceFile(p.path, p.updated, p.mode); err != nil {
			return err
		}
	}
	for _, p := range files {
		if err := os.Remove(p.path + ".backup"); err != nil {
			return err
		}
		if err := os.Remove(p.path + ".backup.json"); err != nil {
			return err
		}
	}
	logMsg("+", fmt.Sprintf("已恢复 %d 个文件", len(files)), 0)
	return nil
}
