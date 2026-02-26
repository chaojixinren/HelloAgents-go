// 文件操作工具使用示例
//
// 对应 Python 版本: examples/file_tools_demo.py
// 演示 ReadTool、WriteTool、EditTool 和 MultiEditTool。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "file_tools_demo")
	defer os.RemoveAll(tmpDir)

	registry := tools.NewToolRegistry(nil)
	readTool := builtin.NewReadTool(tmpDir, registry)
	writeTool := builtin.NewWriteTool(tmpDir)
	editTool := builtin.NewEditTool(tmpDir)
	multiEditTool := builtin.NewMultiEditTool(tmpDir)

	registry.RegisterTool(readTool, false)
	registry.RegisterTool(writeTool, false)
	registry.RegisterTool(editTool, false)
	registry.RegisterTool(multiEditTool, false)

	fmt.Println("=== 文件操作工具示例 ===")
	fmt.Println()

	// 1. 写入文件
	fmt.Println("1. 写入文件:")
	resp := writeTool.Run(map[string]any{
		"file_path": "hello.txt",
		"content":   "Hello, World!\n这是第二行\n这是第三行\n",
	})
	fmt.Printf("   %s (status: %s)\n", resp.Text, resp.Status)
	fmt.Println()

	// 2. 读取文件
	fmt.Println("2. 读取文件:")
	resp = readTool.Run(map[string]any{
		"file_path": "hello.txt",
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 3. 编辑文件（替换）
	fmt.Println("3. 编辑文件:")
	resp = editTool.Run(map[string]any{
		"file_path":  "hello.txt",
		"old_string": "Hello, World!",
		"new_string": "Hello, Go!",
	})
	fmt.Printf("   %s (status: %s)\n", resp.Text, resp.Status)
	fmt.Println()

	// 4. 验证编辑结果
	fmt.Println("4. 验证编辑结果:")
	resp = readTool.Run(map[string]any{
		"file_path": "hello.txt",
	})
	fmt.Printf("   %s\n", resp.Text)
	fmt.Println()

	// 5. 多处编辑
	fmt.Println("5. 多处编辑:")
	resp = multiEditTool.Run(map[string]any{
		"file_path": "hello.txt",
		"edits": []any{
			map[string]any{"old_string": "这是第二行", "new_string": "第二行已更新"},
			map[string]any{"old_string": "这是第三行", "new_string": "第三行已更新"},
		},
	})
	fmt.Printf("   %s (status: %s)\n", resp.Text, resp.Status)
	fmt.Println()

	// 6. 最终内容
	fmt.Println("6. 最终文件内容:")
	content, _ := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
	fmt.Printf("   %s\n", string(content))
}
