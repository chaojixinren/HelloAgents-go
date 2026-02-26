// TodoWrite 进度管理工具示例
//
// 对应 Python 版本: examples/todowrite_demo.py
// 演示如何使用 TodoWrite 工具管理复杂任务的进度。
package main

import (
	"fmt"
	"os"

	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "todowrite_demo")
	defer os.RemoveAll(tmpDir)

	tool := builtin.NewTodoWriteTool(tmpDir, "memory/todos")

	fmt.Println("=== TodoWrite 进度管理示例 ===")
	fmt.Println()

	// 1. 创建任务列表
	fmt.Println("1. 创建任务列表:")
	resp := tool.Run(map[string]any{
		"action":  "create",
		"summary": "开发一个博客系统",
		"todos": []any{
			map[string]any{"content": "设计数据库模型", "status": "completed"},
			map[string]any{"content": "实现用户注册API", "status": "in_progress"},
			map[string]any{"content": "实现文章CRUD", "status": "pending"},
			map[string]any{"content": "添加评论功能", "status": "pending"},
			map[string]any{"content": "部署上线", "status": "pending"},
		},
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Printf("   Stats: %v\n", resp.Data["stats"])
	fmt.Println()

	// 2. 更新进度
	fmt.Println("2. 更新进度（完成注册API，开始文章CRUD）:")
	resp = tool.Run(map[string]any{
		"action":  "update",
		"summary": "开发一个博客系统",
		"todos": []any{
			map[string]any{"content": "设计数据库模型", "status": "completed"},
			map[string]any{"content": "实现用户注册API", "status": "completed"},
			map[string]any{"content": "实现文章CRUD", "status": "in_progress"},
			map[string]any{"content": "添加评论功能", "status": "pending"},
			map[string]any{"content": "部署上线", "status": "pending"},
		},
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 3. 清空
	fmt.Println("3. 清空任务列表:")
	resp = tool.Run(map[string]any{"action": "clear"})
	fmt.Printf("   %s\n", resp.Text)
}
