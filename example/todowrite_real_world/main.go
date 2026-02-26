// TodoWrite 实战案例：复杂项目开发
//
// 对应 Python 版本: examples/todowrite_real_world.py
// 演示在实际项目开发中如何使用 TodoWrite 管理任务进度。
// 场景：开发一个完整的博客系统。
package main

import (
	"fmt"
	"os"

	"helloagents-go/hello_agents/tools/builtin"
)

// BlogProjectManager 博客项目管理器
type BlogProjectManager struct {
	tool *builtin.TodoWriteTool
}

func NewBlogProjectManager(baseDir string) *BlogProjectManager {
	return &BlogProjectManager{
		tool: builtin.NewTodoWriteTool(baseDir, "memory/todos"),
	}
}

func (m *BlogProjectManager) createProjectPlan() {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("📋 创建博客系统开发计划")
	fmt.Println("============================================================")

	resp := m.tool.Run(map[string]any{
		"action":  "create",
		"summary": "开发完整的博客系统",
		"todos": []any{
			map[string]any{"content": "设计数据库模型（User, Post, Comment）", "status": "pending"},
			map[string]any{"content": "实现用户注册和登录", "status": "pending"},
			map[string]any{"content": "实现文章 CRUD 功能", "status": "pending"},
			map[string]any{"content": "实现评论系统", "status": "pending"},
			map[string]any{"content": "实现全文搜索", "status": "pending"},
			map[string]any{"content": "编写单元测试", "status": "pending"},
			map[string]any{"content": "部署到生产环境", "status": "pending"},
		},
	})

	fmt.Printf("\n✅ 项目计划已创建\n")
	fmt.Printf("📊 %s\n", resp.Text)
}

func (m *BlogProjectManager) updateTaskStatus(todos []map[string]any) {
	resp := m.tool.Run(map[string]any{
		"action":  "update",
		"summary": "开发完整的博客系统",
		"todos":   toAnySlice(todos),
	})
	fmt.Printf("📊 %s\n", resp.Text)
}

func (m *BlogProjectManager) simulateDevelopment() {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🎬 模拟开发过程")
	fmt.Println("============================================================")

	m.createProjectPlan()

	tasks := []string{
		"设计数据库模型（User, Post, Comment）",
		"实现用户注册和登录",
		"实现文章 CRUD 功能",
		"实现评论系统",
		"实现全文搜索",
		"编写单元测试",
		"部署到生产环境",
	}

	for day, task := range tasks {
		fmt.Println()
		fmt.Println("------------------------------------------------------------")
		fmt.Printf("第 %d 天：%s\n", day+1, task)
		fmt.Println("------------------------------------------------------------")

		// Build todos list with the current task in_progress
		fmt.Printf("🚀 开始任务: %s\n", task)
		todos := make([]map[string]any, len(tasks))
		for i, t := range tasks {
			status := "pending"
			if i < day {
				status = "completed"
			} else if i == day {
				status = "in_progress"
			}
			todos[i] = map[string]any{"content": t, "status": status}
		}
		m.updateTaskStatus(todos)

		// Complete the task
		fmt.Printf("✅ 完成任务: %s\n", task)
		todos[day]["status"] = "completed"
		m.updateTaskStatus(todos)
	}

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🎉 项目开发完成！")
	fmt.Println("============================================================")
}

func demoMultiPhaseProject(baseDir string) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🎯 演示多阶段项目管理")
	fmt.Println("============================================================")

	tool := builtin.NewTodoWriteTool(baseDir, "memory/todos")

	// 阶段 1: MVP 开发
	fmt.Println("\n📌 阶段 1：MVP 开发")
	resp := tool.Run(map[string]any{
		"action":  "create",
		"summary": "博客系统 MVP 开发",
		"todos": []any{
			map[string]any{"content": "实现基础用户功能", "status": "completed"},
			map[string]any{"content": "实现文章发布", "status": "completed"},
			map[string]any{"content": "实现简单评论", "status": "in_progress"},
		},
	})
	fmt.Printf("   %s\n", resp.Text)

	// 完成 MVP
	fmt.Println("\n✅ MVP 开发完成")
	resp = tool.Run(map[string]any{
		"action":  "update",
		"summary": "博客系统 MVP 开发",
		"todos": []any{
			map[string]any{"content": "实现基础用户功能", "status": "completed"},
			map[string]any{"content": "实现文章发布", "status": "completed"},
			map[string]any{"content": "实现简单评论", "status": "completed"},
		},
	})
	fmt.Printf("   %s\n", resp.Text)

	// 阶段 2: 功能增强
	fmt.Println("\n📌 阶段 2：功能增强")
	resp = tool.Run(map[string]any{
		"action":  "create",
		"summary": "博客系统功能增强",
		"todos": []any{
			map[string]any{"content": "添加富文本编辑器", "status": "in_progress"},
			map[string]any{"content": "实现图片上传", "status": "pending"},
			map[string]any{"content": "添加标签系统", "status": "pending"},
			map[string]any{"content": "实现文章分类", "status": "pending"},
		},
	})
	fmt.Printf("   %s\n", resp.Text)
}

func toAnySlice(items []map[string]any) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

func main() {
	tmpDir, _ := os.MkdirTemp("", "todowrite_real_world")
	defer os.RemoveAll(tmpDir)

	fmt.Println("🚀 TodoWrite 实战案例：复杂项目开发")

	// 案例 1: 完整开发流程
	manager := NewBlogProjectManager(tmpDir)
	manager.simulateDevelopment()

	// 案例 2: 多阶段项目
	demoMultiPhaseProject(tmpDir)

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("✅ 所有案例演示完成")
	fmt.Println("============================================================")
}
