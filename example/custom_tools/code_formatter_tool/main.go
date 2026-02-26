// 代码格式化工具 - 复杂逻辑示例
//
// 对应 Python 版本: examples/custom_tools/code_formatter_tool.py
// 展示如何处理复杂的文本逻辑、多种格式化选项、验证和错误处理。
package main

import (
	"fmt"
	"sort"
	"strings"

	"helloagents-go/hello_agents/tools"
)

// CodeFormatterTool 代码格式化工具
type CodeFormatterTool struct {
	tools.BaseTool
}

func NewCodeFormatterTool() *CodeFormatterTool {
	t := &CodeFormatterTool{}
	t.Name = "code_formatter"
	t.Description = "格式化代码，支持自定义缩进和行宽"
	t.Parameters = map[string]tools.ToolParameter{
		"code": {
			Name: "code", Type: "string",
			Description: "要格式化的代码", Required: true,
		},
		"indent": {
			Name: "indent", Type: "integer",
			Description: "缩进空格数（1-8）", Required: false, Default: 4,
		},
		"max_line_length": {
			Name: "max_line_length", Type: "integer",
			Description: "最大行宽（>=40）", Required: false, Default: 80,
		},
		"fix_imports": {
			Name: "fix_imports", Type: "boolean",
			Description: "是否自动整理 import 语句", Required: false, Default: true,
		},
	}
	return t
}

func (t *CodeFormatterTool) GetName() string                      { return t.Name }
func (t *CodeFormatterTool) GetDescription() string               { return t.Description }
func (t *CodeFormatterTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *CodeFormatterTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *CodeFormatterTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *CodeFormatterTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *CodeFormatterTool) Run(parameters map[string]any) tools.ToolResponse {
	code, _ := parameters["code"].(string)
	code = strings.TrimSpace(code)
	if code == "" {
		return tools.Error("参数 'code' 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	indent := 4
	if v, ok := parameters["indent"]; ok {
		if n, isNum := toInt(v); isNum {
			indent = n
		}
	}
	if indent < 1 || indent > 8 {
		return tools.Error("参数 'indent' 必须是 1-8 之间的整数", tools.ToolErrorCodeInvalidParam, nil)
	}

	maxLineLen := 80
	if v, ok := parameters["max_line_length"]; ok {
		if n, isNum := toInt(v); isNum {
			maxLineLen = n
		}
	}
	if maxLineLen < 40 {
		return tools.Error("参数 'max_line_length' 必须大于等于 40", tools.ToolErrorCodeInvalidParam, nil)
	}

	fixImports := true
	if v, ok := parameters["fix_imports"].(bool); ok {
		fixImports = v
	}

	formattedCode, changes := formatCode(code, indent, maxLineLen, fixImports)

	var text string
	if len(changes) > 0 {
		text = fmt.Sprintf("代码格式化完成，应用了以下修改: %s", strings.Join(changes, ", "))
	} else {
		text = "代码已经符合格式规范，无需修改"
	}

	return tools.Success(text, map[string]any{
		"original_code":   code,
		"formatted_code":  formattedCode,
		"changes":         changes,
		"original_lines":  len(strings.Split(code, "\n")),
		"formatted_lines": len(strings.Split(formattedCode, "\n")),
	}, map[string]any{
		"changes_count": len(changes),
	})
}

func formatCode(code string, indent, maxLineLen int, fixImports bool) (string, []string) {
	var changes []string
	lines := strings.Split(code, "\n")
	formatted := make([]string, len(lines))

	// 1. 修复缩进
	indentChanged := false
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" {
			formatted[i] = ""
			continue
		}
		currentIndent := len(line) - len(stripped)
		indentLevel := currentIndent / indent
		newLine := strings.Repeat(" ", indentLevel*indent) + stripped
		formatted[i] = newLine
		if newLine != line {
			indentChanged = true
		}
	}
	if indentChanged {
		changes = append(changes, "修复缩进")
	}

	// 2. 整理 import 语句
	if fixImports {
		importFixed := fixImportStatements(formatted)
		if !sliceEqual(importFixed, formatted) {
			formatted = importFixed
			changes = append(changes, "整理 import 语句")
		}
	}

	// 3. 移除多余空行
	cleaned := removeExtraBlankLines(formatted)
	if !sliceEqual(cleaned, formatted) {
		formatted = cleaned
		changes = append(changes, "移除多余空行")
	}

	// 4. 检查行宽
	longCount := 0
	for _, line := range formatted {
		if len(line) > maxLineLen {
			longCount++
		}
	}
	if longCount > 0 {
		changes = append(changes, fmt.Sprintf("检测到 %d 行超过最大行宽", longCount))
	}

	return strings.Join(formatted, "\n"), changes
}

func fixImportStatements(lines []string) []string {
	var importLines, fromImportLines, otherLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") {
			importLines = append(importLines, line)
		} else if strings.HasPrefix(trimmed, "from ") {
			fromImportLines = append(fromImportLines, line)
		} else {
			otherLines = append(otherLines, line)
		}
	}

	sort.Strings(importLines)
	sort.Strings(fromImportLines)

	var result []string
	if len(importLines) > 0 {
		result = append(result, importLines...)
	}
	if len(fromImportLines) > 0 {
		if len(importLines) > 0 {
			result = append(result, "")
		}
		result = append(result, fromImportLines...)
	}
	if len(otherLines) > 0 {
		if len(importLines) > 0 || len(fromImportLines) > 0 {
			result = append(result, "", "")
		}
		result = append(result, otherLines...)
	}
	return result
}

func removeExtraBlankLines(lines []string) []string {
	var result []string
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}
	return result
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func main() {
	// 1. 创建工具
	fmt.Println("=== 创建代码格式化工具 ===")
	tool := NewCodeFormatterTool()

	// 2. 测试基本格式化
	fmt.Println("\n=== 测试基本格式化 ===")
	messyCode := `import os
from typing import Dict
import sys


def hello(  ):
        print(  'hello'  )


class MyClass:
  def __init__(self):
    self.value=42`

	resp := tool.Run(map[string]any{
		"code":            messyCode,
		"indent":          4,
		"max_line_length": 80,
	})

	fmt.Printf("状态: %s\n", resp.Status)
	fmt.Printf("变更: %v\n", resp.Data["changes"])
	fmt.Printf("\n格式化后的代码:\n%s\n", resp.Data["formatted_code"])
	fmt.Println()

	// 3. 测试错误处理
	fmt.Println("=== 测试错误处理 ===")
	resp2 := tool.Run(map[string]any{"code": ""})
	fmt.Printf("空代码: status=%s, error=%v\n", resp2.Status, resp2.ErrorInfo)
	fmt.Println()

	// 4. 通过 Registry 使用
	fmt.Println("=== 通过 Registry 使用 ===")
	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(tool, false)
	fmt.Printf("已注册工具: %v\n", registry.ListTools())

	schema := tool.ToOpenAISchema()
	schemaJSON, _ := fmt.Printf("OpenAI Schema: %v\n", schema)
	_ = schemaJSON
}
