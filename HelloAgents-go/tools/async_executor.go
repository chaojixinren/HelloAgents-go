package tools

import (
	"fmt"
	"sync"
)

// taskResult 任务执行结果
type taskResult struct {
	taskID    int
	toolName  string
	inputData string
	result    string
	status    string
}

// AsyncToolExecutor 异步工具执行器
type AsyncToolExecutor struct {
	registry   *ToolRegistry
	maxWorkers int
}

// NewAsyncToolExecutor 创建异步工具执行器
func NewAsyncToolExecutor(registry *ToolRegistry, maxWorkers int) *AsyncToolExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	return &AsyncToolExecutor{
		registry:   registry,
		maxWorkers: maxWorkers,
	}
}

// ExecuteToolAsync 异步执行单个工具
func (e *AsyncToolExecutor) ExecuteToolAsync(toolName, inputData string) chan string {
	resultChan := make(chan string, 1)

	go func() {
		defer close(resultChan)
		result := e.registry.ExecuteTool(toolName, inputData)
		resultChan <- result
	}()

	return resultChan
}

// task 定义任务
type task struct {
	toolName  string
	inputData string
}

// ExecuteToolsParallel 并行执行多个工具
// tasks: 任务列表，每个任务包含 tool_name 和 input_data
// 返回: 执行结果列表，包含任务信息和结果
func (e *AsyncToolExecutor) ExecuteToolsParallel(tasks []map[string]string) []map[string]interface{} {
	fmt.Printf("🚀 开始并行执行 %d 个工具任务\n", len(tasks))

	// 创建通道控制并发数
	semaphore := make(chan struct{}, e.maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]map[string]interface{}, 0)

	for i, task := range tasks {
		toolName, ok1 := task["tool_name"]
		inputData, ok2 := task["input_data"]
		if !ok1 || !ok2 {
			inputData = ""
		}

		if toolName == "" {
			continue
		}

		fmt.Printf("📝 创建任务 %d: %s\n", i+1, toolName)

		wg.Add(1)
		go func(taskID int, tName, iData string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行工具
			result := e.registry.ExecuteTool(tName, iData)

			// 保存结果
			mu.Lock()
			results = append(results, map[string]interface{}{
				"task_id":    taskID,
				"tool_name":  tName,
				"input_data": iData,
				"result":     result,
				"status":     "success",
			})
			mu.Unlock()

			fmt.Printf("✅ 任务 %d 完成: %s\n", taskID+1, tName)
		}(i, toolName, inputData)
	}

	wg.Wait()

	// 统计成功数
	successCount := 0
	for _, r := range results {
		if r["status"] == "success" {
			successCount++
		}
	}
	fmt.Printf("🎉 并行执行完成，成功: %d/%d\n", successCount, len(results))

	return results
}

// ExecuteToolsBatch 批量执行同一个工具
// toolName: 工具名称
// inputList: 输入数据列表
// 返回: 执行结果列表
func (e *AsyncToolExecutor) ExecuteToolsBatch(toolName string, inputList []string) []map[string]interface{} {
	tasks := make([]map[string]string, 0, len(inputList))
	for _, inputData := range inputList {
		tasks = append(tasks, map[string]string{
			"tool_name":  toolName,
			"input_data": inputData,
		})
	}
	return e.ExecuteToolsParallel(tasks)
}

// Close 关闭执行器（Go 不需要显式关闭，保留以兼容 Python 接口）
func (e *AsyncToolExecutor) Close() {
	fmt.Println("🔒 异步工具执行器已关闭")
}

// ==================== 便捷函数 ====================

// RunParallelTools 并行执行多个工具
// registry: 工具注册表
// tasks: 任务列表
// maxWorkers: 最大工作线程数
// 返回: 执行结果列表
func RunParallelTools(registry *ToolRegistry, tasks []map[string]string, maxWorkers int) []map[string]interface{} {
	executor := NewAsyncToolExecutor(registry, maxWorkers)
	defer executor.Close()
	return executor.ExecuteToolsParallel(tasks)
}

// RunBatchTool 批量执行同一个工具
// registry: 工具注册表
// toolName: 工具名称
// inputList: 输入数据列表
// maxWorkers: 最大工作线程数
// 返回: 执行结果列表
func RunBatchTool(registry *ToolRegistry, toolName string, inputList []string, maxWorkers int) []map[string]interface{} {
	executor := NewAsyncToolExecutor(registry, maxWorkers)
	defer executor.Close()
	return executor.ExecuteToolsBatch(toolName, inputList)
}

// ==================== 示例函数 ====================

// DemoParallelExecution 演示并行执行的示例
func DemoParallelExecution() []map[string]interface{} {
	// 创建注册表（这里假设已经注册了工具）
	registry := NewToolRegistry()

	// 定义并行任务
	tasks := []map[string]string{
		{"tool_name": "my_calculator", "input_data": "2 + 2"},
		{"tool_name": "my_calculator", "input_data": "3 * 4"},
		{"tool_name": "my_calculator", "input_data": "sqrt(16)"},
		{"tool_name": "my_calculator", "input_data": "10 / 2"},
	}

	// 并行执行
	results := RunParallelTools(registry, tasks, 4)

	// 显示结果
	fmt.Println("\n📊 并行执行结果:")
	for _, result := range results {
		statusIcon := "✅"
		if result["status"] != "success" {
			statusIcon = "❌"
		}
		fmt.Printf("%s %s(%s) = %s\n", statusIcon, result["tool_name"], result["input_data"], result["result"])
	}

	return results
}
