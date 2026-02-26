// 工具并行执行性能对比示例
//
// 对应 Python 版本: examples/parallel_tools_demo.py
// 对比同步执行 vs 并行执行（goroutine）的性能差异。
package main

import (
	"fmt"
	"sync"
	"time"

	"helloagents-go/hello_agents/tools"
)

// SlowTool 模拟耗时工具
type SlowTool struct {
	tools.BaseTool
	delay time.Duration
}

func NewSlowTool(name string, delay time.Duration) *SlowTool {
	t := &SlowTool{delay: delay}
	t.Name = name
	t.Description = fmt.Sprintf("耗时 %v 的工具", delay)
	t.Parameters = map[string]tools.ToolParameter{
		"data": {Name: "data", Type: "string", Description: "数据", Required: false},
	}
	return t
}

func (t *SlowTool) GetName() string                      { return t.Name }
func (t *SlowTool) GetDescription() string               { return t.Description }
func (t *SlowTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *SlowTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *SlowTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *SlowTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *SlowTool) Run(parameters map[string]any) tools.ToolResponse {
	time.Sleep(t.delay)
	return tools.Success(
		fmt.Sprintf("%s 完成（耗时 %v）", t.Name, t.delay),
		map[string]any{"delay_ms": t.delay.Milliseconds()},
	)
}

// runSequential 顺序执行所有工具
func runSequential(toolList []*SlowTool) []tools.ToolResponse {
	results := make([]tools.ToolResponse, len(toolList))
	for i, tool := range toolList {
		results[i] = tool.Run(map[string]any{"data": "test"})
	}
	return results
}

// runParallel 使用 goroutine 并行执行所有工具
func runParallel(toolList []*SlowTool) []tools.ToolResponse {
	results := make([]tools.ToolResponse, len(toolList))
	var wg sync.WaitGroup
	for i, tool := range toolList {
		wg.Add(1)
		go func(idx int, t *SlowTool) {
			defer wg.Done()
			results[idx] = t.Run(map[string]any{"data": "test"})
		}(i, tool)
	}
	wg.Wait()
	return results
}

// runParallelWithLimit 使用带并发数限制的 goroutine 并行执行
func runParallelWithLimit(toolList []*SlowTool, maxConcurrent int) []tools.ToolResponse {
	results := make([]tools.ToolResponse, len(toolList))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for i, tool := range toolList {
		wg.Add(1)
		go func(idx int, t *SlowTool) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = t.Run(map[string]any{"data": "test"})
		}(i, tool)
	}
	wg.Wait()
	return results
}

func main() {
	fmt.Println("============================================================")
	fmt.Println("工具并行执行性能测试")
	fmt.Println("============================================================")

	// 测试 1: 3 个工具各耗时 500ms，对比顺序 vs 并行
	fmt.Println()
	fmt.Println("🚀 测试 1: 顺序执行 vs 并行执行（3 个工具，各 500ms）")
	fmt.Println("------------------------------------------------------------")

	toolList3 := []*SlowTool{
		NewSlowTool("Tool1", 500*time.Millisecond),
		NewSlowTool("Tool2", 500*time.Millisecond),
		NewSlowTool("Tool3", 500*time.Millisecond),
	}

	// 顺序执行
	start := time.Now()
	seqResults := runSequential(toolList3)
	seqElapsed := time.Since(start)
	fmt.Printf("  顺序执行耗时: %v\n", seqElapsed.Round(time.Millisecond))
	for _, r := range seqResults {
		fmt.Printf("    - %s\n", r.Text)
	}

	// 并行执行
	start = time.Now()
	parResults := runParallel(toolList3)
	parElapsed := time.Since(start)
	fmt.Printf("  并行执行耗时: %v\n", parElapsed.Round(time.Millisecond))
	for _, r := range parResults {
		fmt.Printf("    - %s\n", r.Text)
	}

	speedup := float64(seqElapsed) / float64(parElapsed)
	fmt.Printf("  性能提升: %.2fx\n", speedup)

	// 测试 2: 5 个工具，限制并发数为 2
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🚀 测试 2: 并发数限制（5 个工具，最多 2 个并行，各 500ms）")
	fmt.Println("------------------------------------------------------------")

	toolList5 := make([]*SlowTool, 5)
	for i := range toolList5 {
		toolList5[i] = NewSlowTool(fmt.Sprintf("Tool%d", i+1), 500*time.Millisecond)
	}

	start = time.Now()
	limitResults := runParallelWithLimit(toolList5, 2)
	limitElapsed := time.Since(start)

	fmt.Printf("  限制并发（max=2）执行耗时: %v\n", limitElapsed.Round(time.Millisecond))
	for _, r := range limitResults {
		fmt.Printf("    - %s\n", r.Text)
	}
	fmt.Printf("  理论耗时: ~1500ms（5个工具，每次2个并行: 2+2+1 批次）\n")
	fmt.Printf("  无限制并行: ~500ms\n")
	fmt.Printf("  串行执行: ~2500ms\n")

	// 测试 3: 注册到 ToolRegistry
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🚀 测试 3: ToolRegistry 工具管理")
	fmt.Println("------------------------------------------------------------")

	registry := tools.NewToolRegistry(nil)
	for _, t := range toolList3 {
		registry.RegisterTool(t, false)
	}

	fmt.Printf("  已注册工具: %v\n", registry.ListTools())
	fmt.Printf("  工具描述:\n%s\n", registry.GetToolsDescription())
}
