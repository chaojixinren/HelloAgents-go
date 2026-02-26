// 熔断器机制使用示例
//
// 对应 Python 版本: examples/circuit_breaker_demo.py
// 演示 CircuitBreaker 防止工具连续失败。
package main

import (
	"fmt"
	"time"

	"helloagents-go/hello_agents/tools"
)

type UnstableTool struct {
	tools.BaseTool
	callCount int
	failUntil int
}

func NewUnstableTool(failUntil int) *UnstableTool {
	t := &UnstableTool{failUntil: failUntil}
	t.Name = "UnstableTool"
	t.Description = "不稳定的工具，用于测试熔断器"
	t.Parameters = map[string]tools.ToolParameter{
		"input": {Name: "input", Type: "string", Description: "输入", Required: false},
	}
	return t
}

func (t *UnstableTool) GetName() string                      { return t.Name }
func (t *UnstableTool) GetDescription() string               { return t.Description }
func (t *UnstableTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *UnstableTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *UnstableTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *UnstableTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *UnstableTool) Run(parameters map[string]any) tools.ToolResponse {
	t.callCount++
	if t.callCount <= t.failUntil {
		return tools.Error(
			fmt.Sprintf("调用 #%d 失败", t.callCount),
			tools.ToolErrorCodeExecutionError,
			nil,
		)
	}
	return tools.Success(fmt.Sprintf("调用 #%d 成功", t.callCount), nil)
}

func main() {
	fmt.Println("=== 熔断器机制示例 ===")
	fmt.Println()

	// 创建带熔断器的 ToolRegistry
	cb := tools.NewCircuitBreaker(3, 2, true) // 3次失败后熔断, 2秒恢复
	registry := tools.NewToolRegistry(cb)

	tool := NewUnstableTool(5) // 前5次调用会失败
	registry.RegisterTool(tool, false)

	// 连续调用，观察熔断
	fmt.Println("1. 连续调用不稳定工具:")
	for i := 0; i < 5; i++ {
		result := registry.ExecuteTool("UnstableTool", "test")
		fmt.Printf("   调用 %d: %s\n", i+1, result.Text)

		// 检查熔断器状态
		status := cb.GetStatus("UnstableTool")
		fmt.Printf("   熔断器状态: %s (失败次数: %v)\n", status["state"], status["failure_count"])

		if cb.IsOpen("UnstableTool") {
			fmt.Println("   ⚡ 工具已被熔断!")
			break
		}
	}
	fmt.Println()

	// 等待恢复
	fmt.Println("2. 等待熔断器恢复...")
	time.Sleep(3 * time.Second)
	if !cb.IsOpen("UnstableTool") {
		fmt.Println("   ✅ 熔断器已恢复")
	}
	fmt.Println()

	// 手动控制
	fmt.Println("3. 手动熔断和恢复:")
	cb.Open("UnstableTool")
	fmt.Printf("   手动熔断后: isOpen=%v\n", cb.IsOpen("UnstableTool"))
	cb.Close("UnstableTool")
	fmt.Printf("   手动恢复后: isOpen=%v\n", cb.IsOpen("UnstableTool"))
}
