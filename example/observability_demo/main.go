// 可观测性使用示例
//
// 对应 Python 版本: examples/observability_demo.py
// 演示如何使用 TraceLogger 记录 Agent 执行过程。
package main

import (
	"fmt"
	"os"

	"helloagents-go/hello_agents/observability"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "observability_demo")
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== 可观测性示例 ===")
	fmt.Println()

	logger, err := observability.NewTraceLogger(tmpDir, true, false)
	if err != nil {
		fmt.Printf("❌ 创建 TraceLogger 失败: %v\n", err)
		return
	}

	// 1. 记录会话开始
	logger.LogEvent("session_start", map[string]any{
		"agent_name": "DemoAgent",
		"agent_type": "SimpleAgent",
	}, nil)

	// 2. 记录用户输入
	logger.LogEvent("message_written", map[string]any{
		"role":    "user",
		"content": "你好，帮我计算 2+2",
	}, nil)

	// 3. 记录模型输出
	step := 1
	logger.LogEvent("model_output", map[string]any{
		"content":    "我来帮你计算。2+2=4。",
		"tool_calls": 0,
	}, &step)

	// 4. 记录工具调用
	logger.LogEvent("tool_call", map[string]any{
		"tool_name": "calculator",
		"args":      map[string]any{"expression": "2+2"},
	}, &step)

	logger.LogEvent("tool_result", map[string]any{
		"tool_name": "calculator",
		"result":    "4",
	}, &step)

	// 5. 记录会话结束
	logger.LogEvent("session_end", map[string]any{
		"status":       "success",
		"final_answer": "2+2=4",
		"total_steps":  1,
	}, nil)

	// 6. 保存 trace
	err = logger.Finalize()
	if err != nil {
		fmt.Printf("❌ 保存 Trace 失败: %v\n", err)
		return
	}

	fmt.Println("✅ Trace 已保存")
	fmt.Printf("   Session ID: %s\n", logger.SessionID)
	fmt.Printf("   JSONL: %s\n", logger.JSONLPath)
	fmt.Printf("   HTML:  %s\n", logger.HTMLPath)
}
