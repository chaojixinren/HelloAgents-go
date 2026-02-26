// 简单工具模板
//
// 对应 Python 版本: examples/custom_tools/simple_tool_template.py
// 演示创建自定义工具的最简方式。
package main

import (
	"fmt"
	"strings"

	"helloagents-go/hello_agents/tools"
)

// TextProcessorTool 文本处理工具
type TextProcessorTool struct {
	tools.BaseTool
}

func NewTextProcessorTool() *TextProcessorTool {
	t := &TextProcessorTool{}
	t.Name = "TextProcessor"
	t.Description = "处理文本：大小写转换、反转、统计等"
	t.Parameters = map[string]tools.ToolParameter{
		"text":   {Name: "text", Type: "string", Description: "要处理的文本", Required: true},
		"action": {Name: "action", Type: "string", Description: "操作: upper/lower/reverse/count", Required: true},
	}
	return t
}

func (t *TextProcessorTool) GetName() string                      { return t.Name }
func (t *TextProcessorTool) GetDescription() string               { return t.Description }
func (t *TextProcessorTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *TextProcessorTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *TextProcessorTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *TextProcessorTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *TextProcessorTool) Run(parameters map[string]any) tools.ToolResponse {
	text, _ := parameters["text"].(string)
	action, _ := parameters["action"].(string)

	switch action {
	case "upper":
		return tools.Success(strings.ToUpper(text), map[string]any{"result": strings.ToUpper(text)})
	case "lower":
		return tools.Success(strings.ToLower(text), map[string]any{"result": strings.ToLower(text)})
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result := string(runes)
		return tools.Success(result, map[string]any{"result": result})
	case "count":
		return tools.Success(
			fmt.Sprintf("字符数: %d, 字数: %d", len([]rune(text)), len(strings.Fields(text))),
			map[string]any{"chars": len([]rune(text)), "words": len(strings.Fields(text))},
		)
	default:
		return tools.Error(fmt.Sprintf("未知操作: %s", action), tools.ToolErrorCodeInvalidParam, nil)
	}
}

func main() {
	fmt.Println("=== 简单工具模板示例 ===")
	fmt.Println()

	tool := NewTextProcessorTool()

	tests := []map[string]any{
		{"text": "Hello World", "action": "upper"},
		{"text": "Hello World", "action": "lower"},
		{"text": "Hello World", "action": "reverse"},
		{"text": "Go 语言 是 一门 优秀的 编程 语言", "action": "count"},
	}

	for _, params := range tests {
		resp := tool.Run(params)
		fmt.Printf("  %s(%q) -> %s\n", params["action"], params["text"], resp.Text)
	}
}
