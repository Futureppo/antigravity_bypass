package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const jsResourcePath = "vs/workbench/workbench.desktop.main.js"

var (
	legacyLimitReference   = regexp.MustCompile(`Cannot enable more tools because it would exceed the limit of \$\{([A-Za-z_$][\w$]*)\}`)
	legacyLimitDeclaration = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*=\s*50\s*,\s*([A-Za-z_$][\w$]*)\s*=\s*(100|8192|114514)\s*,`)
	unlimitedToolMethods   = regexp.MustCompile(`async updateToolInConfigFile\([^)]*\)\{(.+?)\}async updateServerInConfigFile\([^)]*\)\{(.+?)\}getPreferredCommandKey\(`)
	unlimitedCountDisplay  = regexp.MustCompile("this\\._totalToolCount\\.textContent=`\\$\\{[A-Za-z_$][\\w$]*\\} tools`")
)

func planJS(data []byte) ([]byte, string, error) {
	content := string(data)
	refs := legacyLimitReference.FindAllStringSubmatch(content, -1)
	if len(refs) > 0 {
		// Match the variable actually interpolated into the MCP error, not nearby
		// numbers or a global '=114514,' marker from an unrelated component.
		variables := make(map[string]bool)
		for _, ref := range refs {
			variables[ref[1]] = true
		}
		var candidates [][]int
		for _, m := range legacyLimitDeclaration.FindAllStringSubmatchIndex(content, -1) {
			if variables[content[m[4]:m[5]]] {
				candidates = append(candidates, m)
			}
		}
		if len(candidates) != 1 {
			return nil, "", fmt.Errorf("前端 MCP 上限定义不唯一（%d 个候选），未修改", len(candidates))
		}
		m := candidates[0]
		if content[m[6]:m[7]] == strconv.Itoa(toolLimit) {
			return data, "前端: 已修补", nil
		}
		updated := content[:m[6]] + strconv.Itoa(toolLimit) + content[m[7]:]
		return []byte(updated), fmt.Sprintf("前端: %s -> %d", content[m[6]:m[7]], toolLimit), nil
	}
	// IDE 2.5.5 removed the frontend cap. Require the recognizable MCP
	// configuration methods AND the count-only display; absence of an old
	// signature alone is not evidence that a future version is unrestricted.
	methods := unlimitedToolMethods.FindStringSubmatch(content)
	if len(methods) == 3 && unlimitedCountDisplay.MatchString(content) &&
		!strings.Contains(content, "Cannot enable more tools") {
		valid := true
		for _, method := range methods[1:] {
			// Arrow functions use =>; strip that syntax before checking for comparisons.
			clean := strings.ReplaceAll(method, "=>", "")
			if len(method) > 8000 || strings.Contains(strings.ToLower(method), "limit") || strings.ContainsAny(clean, "<>") ||
				!strings.Contains(method, "_getServerSpecFromConfigFile") || !strings.Contains(method, "saveMcpServerToConfigFile") {
				valid = false
			}
		}
		if valid {
			return data, "前端: 官方已取消 100 工具限制，无需修改", nil
		}
	}
	return nil, "", fmt.Errorf("未识别前端 MCP 结构，未修改；请附上 IDE 版本号反馈")
}
