// 工具响应协议使用示例
//
// 对应 Python 版本: examples/tool_response_demo.py
// 演示如何使用标准化的 ToolResponse 协议。
package main

import (
	"fmt"

	"helloagents-go/hello_agents/tools"
)

type DemoCalculatorTool struct {
	tools.BaseTool
}

func NewDemoCalculatorTool() *DemoCalculatorTool {
	t := &DemoCalculatorTool{}
	t.Name = "DemoCalculator"
	t.Description = "演示工具响应协议的计算器"
	t.Parameters = map[string]tools.ToolParameter{
		"expression": {Name: "expression", Type: "string", Description: "数学表达式", Required: true},
	}
	return t
}

func (t *DemoCalculatorTool) GetName() string                            { return t.Name }
func (t *DemoCalculatorTool) GetDescription() string                     { return t.Description }
func (t *DemoCalculatorTool) GetParameters() []tools.ToolParameter       { return t.BaseTool.GetParameters() }
func (t *DemoCalculatorTool) RunWithTiming(p map[string]any) tools.ToolResponse { return t.BaseTool.RunWithTiming(p) }
func (t *DemoCalculatorTool) ARun(p map[string]any) tools.ToolResponse   { return t.Run(p) }
func (t *DemoCalculatorTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *DemoCalculatorTool) Run(parameters map[string]any) tools.ToolResponse {
	expr, _ := parameters["expression"].(string)
	if expr == "" {
		return tools.Error("表达式不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}
	if expr == "1/0" {
		return tools.Error("除零错误", tools.ToolErrorCodeExecutionError, map[string]any{"expression": expr})
	}
	if expr == "partial" {
		return tools.Partial("部分计算完成：中间结果=42", map[string]any{"intermediate": 42})
	}
	return tools.Success("计算结果: 42", map[string]any{"result": 42, "expression": expr})
}

func main() {
	fmt.Println("=== 工具响应协议示例 ===")
	fmt.Println()

	tool := NewDemoCalculatorTool()

	// 成功响应
	fmt.Println("1. 成功响应:")
	resp := tool.Run(map[string]any{"expression": "6*7"})
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Printf("   Text:   %s\n", resp.Text)
	fmt.Printf("   Data:   %v\n", resp.Data)
	fmt.Println()

	// 部分成功
	fmt.Println("2. 部分成功响应:")
	resp = tool.Run(map[string]any{"expression": "partial"})
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Printf("   Text:   %s\n", resp.Text)
	fmt.Println()

	// 错误响应
	fmt.Println("3. 错误响应:")
	resp = tool.Run(map[string]any{"expression": "1/0"})
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Printf("   Text:   %s\n", resp.Text)
	fmt.Printf("   Error:  %v\n", resp.ErrorInfo)
	fmt.Println()

	// JSON 序列化
	fmt.Println("4. JSON 序列化:")
	resp = tool.Run(map[string]any{"expression": "2+2"})
	fmt.Println(resp.ToJSON())
}
