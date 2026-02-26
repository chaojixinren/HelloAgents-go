// DevLogTool 使用示例
//
// 对应 Python 版本: examples/devlog_demo.py
// 演示如何使用 DevLogTool 记录开发决策和问题。
package main

import (
	"fmt"
	"os"

	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "devlog_demo")
	defer os.RemoveAll(tmpDir)

	tool := builtin.NewDevLogTool("demo-session", "DemoAgent", tmpDir, "memory/devlogs")

	fmt.Println("=== DevLog 开发日志示例 ===")
	fmt.Println()

	// 1. 追加日志
	fmt.Println("1. 追加日志:")
	entries := []struct{ category, content string }{
		{"decision", "选择 Go 作为主要开发语言"},
		{"progress", "完成核心模块的基本架构设计"},
		{"issue", "发现并发场景下的数据竞争问题"},
		{"solution", "使用 sync.Mutex 保护共享状态"},
	}
	for _, e := range entries {
		resp := tool.Run(map[string]any{
			"action":   "append",
			"category": e.category,
			"content":  e.content,
			"metadata": map[string]any{"tags": []any{e.category}},
		})
		fmt.Printf("   [%s] %s\n", e.category, resp.Text)
	}
	fmt.Println()

	// 2. 读取日志
	fmt.Println("2. 读取所有日志:")
	resp := tool.Run(map[string]any{
		"action": "read",
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 3. 按类别过滤
	fmt.Println("3. 按类别过滤 (decision):")
	resp = tool.Run(map[string]any{
		"action": "read",
		"filter": map[string]any{"category": "decision"},
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 4. 生成摘要
	fmt.Println("4. 生成摘要:")
	resp = tool.Run(map[string]any{
		"action": "summary",
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 5. 清空
	fmt.Println("5. 清空日志:")
	resp = tool.Run(map[string]any{
		"action": "clear",
	})
	fmt.Printf("   %s\n", resp.Text)
}
