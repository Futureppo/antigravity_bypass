/*
去你妈的工具数量限制
Antigravity IDE 工具数量限制绕过工具  By https://github.com/Futureppo
*/
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const (
	jsPatchedMarker = "=114514,"
	jsResourcePath  = "vs/workbench/workbench.desktop.main.js"
)

var (
	jsAutoPattern = regexp.MustCompile(
		`([A-Za-z_$][\w$]{0,4})=50,([A-Za-z_$][\w$]{0,4})=100,([A-Za-z_$][\w$]{0,4})=class`,
	)

	jsContextKeywords = []string{
		"tool", "Tool", "cannot enable", "Cannot enable",
		"exceed", "limit", "Limit", "maximum", "Maximum",
	}

	binSearch = []byte{0x48, 0x83, 0xf9, 0x64, 0x0f, 0x8e} // cmp rcx,0x64 + jle near
	binPatch  = []byte{0x48, 0x83, 0xf9, 0x64, 0x90, 0xe9} // cmp rcx,0x64 + nop + jmp
)

func logMsg(icon, msg string, indent int) {
	fmt.Printf("%s%s  %s\n", strings.Repeat("  ", indent), icon, msg)
}

func isValidIDEDir(path string) bool {
	jsPath := filepath.Join(path, "out", "vs", "workbench", "workbench.desktop.main.js")
	info, err := os.Stat(jsPath)
	return err == nil && !info.IsDir()
}

func getLSFilename() string {
	archStr := "x64"
	if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
		archStr = "arm64"
	}
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("language_server_windows_%s.exe", archStr)
	case "linux":
		return fmt.Sprintf("language_server_linux_%s", archStr)
	case "darwin":
		return fmt.Sprintf("language_server_macos_%s", archStr)
	default:
		return "language_server_windows_x64.exe"
	}
}

func detectIDEPath() string {
	candidates := getCandidatePaths()
	if custom := os.Getenv("ANTIGRAVITY_DIR"); custom != "" {
		candidates = append([]string{filepath.Join(custom, "resources", "app")}, candidates...)
	}

	for _, path := range candidates {
		if isValidIDEDir(path) {
			logMsg("*", fmt.Sprintf("自动定位 IDE: %s", path), 0)
			return path
		}
	}

	if regPath := findFromRegistry(); regPath != "" {
		logMsg("*", fmt.Sprintf("自动定位 IDE: %s", regPath), 0)
		return regPath
	}

	if diskPath := findFromDiskScan(); diskPath != "" {
		logMsg("*", fmt.Sprintf("自动定位 IDE: %s", diskPath), 0)
		return diskPath
	}

	logMsg("!", "未找到 IDE，请手动输入 resources/app 目录路径", 0)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("  > ")
		if !scanner.Scan() {
			os.Exit(1)
		}
		userInput := strings.TrimSpace(scanner.Text())
		userInput = strings.Trim(userInput, "\"")
		if userInput == "" {
			continue
		}
		if isValidIDEDir(userInput) {
			return userInput
		}
		logMsg("!", "路径无效，请重试", 0)
	}
}

func patchJS(jsFile, productFile string) bool {
	logMsg("#", "[1/2] 前端 JS 工具数量限制", 0)

	if _, err := os.Stat(jsFile); os.IsNotExist(err) {
		logMsg("x", "找不到目标文件", 1)
		return false
	}

	contentBytes, err := os.ReadFile(jsFile)
	if err != nil {
		logMsg("x", fmt.Sprintf("读取文件失败: %v", err), 1)
		return false
	}
	content := string(contentBytes)

	if strings.Contains(content, jsPatchedMarker) {
		logMsg("-", "已修改过，跳过", 1)
		return true
	}

	matches := jsAutoPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		logMsg("x", "未找到工具限制特征，可能版本不兼容", 1)
		return false
	}

	targetIdx := 0
	if len(matches) > 1 {
		type scored struct {
			score int
			idx   int
		}
		scoredList := make([]scored, len(matches))
		for i, m := range matches {
			pos := m[0]
			ctxStart := max(0, pos-500)
			ctxEnd := min(len(content), pos+500)
			ctx := content[ctxStart:ctxEnd]
			score := 0
			for _, kw := range jsContextKeywords {
				if strings.Contains(ctx, kw) {
					score++
				}
			}
			scoredList[i] = scored{score: score, idx: i}
		}
		sort.Slice(scoredList, func(a, b int) bool {
			return scoredList[a].score > scoredList[b].score
		})

		if scoredList[0].score > 0 && scoredList[0].score > scoredList[1].score {
			targetIdx = scoredList[0].idx
		} else {
			logMsg("x", "无法自动确定目标，找到以下候选:", 1)
			for i, s := range scoredList {
				m := matches[s.idx]
				matchStr := content[m[0]:m[1]]
				logMsg(" ", fmt.Sprintf("候选 %d (关联度 %d): %s", i+1, s.score, matchStr), 2)
			}
			return false
		}
	}

	m := matches[targetIdx]
	originalStr := content[m[0]:m[1]]
	limitVar := content[m[4]:m[5]]
	patchedStr := strings.Replace(originalStr,
		fmt.Sprintf("%s=100", limitVar),
		fmt.Sprintf("%s=114514", limitVar), 1)
	newContent := strings.Replace(content, originalStr, patchedStr, 1)

	if err := os.WriteFile(jsFile, []byte(newContent), 0644); err != nil {
		logMsg("x", fmt.Sprintf("写入文件失败: %v", err), 1)
		return false
	}

	logMsg("+", "工具上限 100 -> 114514", 1)
	logMsg("*", fmt.Sprintf("特征: %s", originalStr), 1)
	updateChecksum(jsFile, productFile)
	return true
}

func updateChecksum(jsFile, productFile string) {
	if _, err := os.Stat(productFile); os.IsNotExist(err) {
		return
	}
	jsData, err := os.ReadFile(jsFile)
	if err != nil {
		return
	}
	hash := sha256.Sum256(jsData)
	newHash := strings.TrimRight(base64.StdEncoding.EncodeToString(hash[:]), "=")

	productRaw, err := os.ReadFile(productFile)
	if err != nil {
		return
	}
	var product map[string]interface{}
	if err := json.Unmarshal(productRaw, &product); err != nil {
		return
	}

	checksums, ok := product["checksums"]
	if !ok {
		return
	}
	checksumsMap, ok := checksums.(map[string]interface{})
	if !ok {
		return
	}
	if _, exists := checksumsMap[jsResourcePath]; !exists {
		return
	}

	checksumsMap[jsResourcePath] = newHash
	newProductData, err := json.MarshalIndent(product, "", "\t")
	if err != nil {
		return
	}
	newProductData = append(newProductData, '\n')
	if err := os.WriteFile(productFile, newProductData, 0644); err != nil {
		return
	}
	logMsg("+", "已更新 product.json 校验值", 1)
}

func findTextSection(data []byte) (start, end int, found bool) {
	if len(data) >= 0x40 && data[0] == 'M' && data[1] == 'Z' { // PE
		peOff := int(binary.LittleEndian.Uint32(data[0x3c:]))
		if peOff+24 <= len(data) {
			numSec := int(binary.LittleEndian.Uint16(data[peOff+6:]))
			optSize := int(binary.LittleEndian.Uint16(data[peOff+20:]))
			secOff := peOff + 24 + optSize
			for s := 0; s < numSec; s++ {
				so := secOff + s*40
				if so+40 > len(data) {
					break
				}
				name := string(bytes.TrimRight(data[so:so+8], "\x00"))
				if name == ".text" {
					ptr := int(binary.LittleEndian.Uint32(data[so+20:]))
					size := int(binary.LittleEndian.Uint32(data[so+16:]))
					return ptr, ptr + size, true
				}
			}
		}
	}

	if len(data) >= 0x40 && bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) && data[4] == 2 { // ELF64
		eShoff := int(binary.LittleEndian.Uint64(data[0x28:]))
		eShentsize := int(binary.LittleEndian.Uint16(data[0x3a:]))
		eShnum := int(binary.LittleEndian.Uint16(data[0x3c:]))
		eShstrndx := int(binary.LittleEndian.Uint16(data[0x3e:]))

		strShOff := eShoff + eShstrndx*eShentsize
		if strShOff+0x20 > len(data) {
			return 0, 0, false
		}
		strTabOff := int(binary.LittleEndian.Uint64(data[strShOff+0x18:]))

		for i := 0; i < eShnum; i++ {
			shOff := eShoff + i*eShentsize
			if shOff+0x28 > len(data) {
				break
			}
			shNameIdx := int(binary.LittleEndian.Uint32(data[shOff:]))
			shType := binary.LittleEndian.Uint32(data[shOff+4:])
			nameStart := strTabOff + shNameIdx
			if nameStart >= len(data) {
				continue
			}
			nameEnd := bytes.IndexByte(data[nameStart:], 0)
			if nameEnd == -1 {
				continue
			}
			secName := string(data[nameStart : nameStart+nameEnd])
			if secName == ".text" && shType == 1 {
				offset := int(binary.LittleEndian.Uint64(data[shOff+0x18:]))
				size := int(binary.LittleEndian.Uint64(data[shOff+0x20:]))
				return offset, offset + size, true
			}
		}
	}

	return 0, 0, false
}

func verifyPatchContext(data []byte, idx int) bool {
	if idx+10 > len(data) {
		return false
	}
	relOffset := int32(binary.LittleEndian.Uint32(data[idx+6:]))
	return relOffset > 0 && relOffset < 0x2000
}

func patchLanguageServer(lsFile string) bool {
	logMsg("#", "[2/2] 语言服务器二进制文件", 0)

	if _, err := os.Stat(lsFile); os.IsNotExist(err) {
		logMsg("x", fmt.Sprintf("找不到目标文件: %s", filepath.Base(lsFile)), 1)
		return false
	}

	backupPath := lsFile + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		src, err := os.Open(lsFile)
		if err != nil {
			logMsg("x", permOrErr("文件被占用，请先关闭所有 Antigravity 进程", err), 1)
			return false
		}
		dst, err := os.Create(backupPath)
		if err != nil {
			src.Close()
			logMsg("x", fmt.Sprintf("创建备份失败: %v", err), 1)
			return false
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			logMsg("x", fmt.Sprintf("备份失败: %v", copyErr), 1)
			return false
		}
		logMsg("+", "已备份原始文件", 1)
	}

	f, err := os.OpenFile(lsFile, os.O_RDWR, 0)
	if err != nil {
		logMsg("x", permOrErr("文件被占用，请先关闭 Antigravity 再重试", err), 1)
		return false
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		logMsg("x", fmt.Sprintf("读取文件失败: %v", err), 1)
		return false
	}

	searchStart, searchEnd := 0, len(data)
	if s, e, ok := findTextSection(data); ok {
		searchStart, searchEnd = s, e
	}

	alreadyPatched := bytes.Count(data, binPatch)
	patchCount := 0
	skipped := 0

	idx := searchStart - 1
	for {
		pos := bytes.Index(data[idx+1:], binSearch)
		if pos == -1 {
			break
		}
		idx = idx + 1 + pos
		if idx >= searchEnd {
			break
		}
		if !verifyPatchContext(data, idx) {
			skipped++
			continue
		}
		patchCount++
		if _, err := f.Seek(int64(idx+4), io.SeekStart); err != nil {
			logMsg("x", fmt.Sprintf("Seek 失败: %v", err), 1)
			return false
		}
		if _, err := f.Write([]byte{0x90, 0xe9}); err != nil {
			logMsg("x", fmt.Sprintf("写入失败: %v", err), 1)
			return false
		}
		logMsg("+", fmt.Sprintf("0x%08x: jle -> nop+jmp", idx), 1)
	}

	if patchCount > 0 {
		logMsg("+", fmt.Sprintf("修补了 %d 处检查指令", patchCount), 1)
		return true
	} else if alreadyPatched >= 1 {
		logMsg("-", "已修补过，跳过", 1)
		return true
	}
	logMsg("x", "未找到特征位点", 1)
	if skipped > 0 {
		logMsg("*", fmt.Sprintf("%d 处匹配未通过上下文验证", skipped), 1)
	}
	return false
}

func permOrErr(permMsg string, err error) string {
	if os.IsPermission(err) {
		return permMsg
	}
	return fmt.Sprintf("操作失败: %v", err)
}

func main() {
	fmt.Println("Antigravity IDE 工具数量限制绕过工具  By https://github.com/Futureppo")
	fmt.Println("注意: 请先完全退出 Antigravity IDE 的所有进程")
	fmt.Println()

	baseDir := detectIDEPath()

	jsFile := filepath.Join(baseDir, "out", "vs", "workbench", "workbench.desktop.main.js")
	productFile := filepath.Join(baseDir, "product.json")
	lsFile := filepath.Join(baseDir, "extensions", "antigravity", "bin", getLSFilename())

	jsOK := patchJS(jsFile, productFile)
	lsOK := patchLanguageServer(lsFile)

	fmt.Println()
	if jsOK && lsOK {
		logMsg("+", "全部完成，重启 IDE 即可生效", 0)
	} else if jsOK || lsOK {
		logMsg("!", "部分完成，请检查上方失败信息", 0)
	} else {
		logMsg("x", "全部失败，请检查上方错误信息", 0)
	}
}
