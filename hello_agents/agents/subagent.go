package agents

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

type subagentRuntime interface {
	Run(inputText string, kwargs map[string]any) (string, error)
	GetHistory() []core.Message
	ClearHistory()
	AddMessage(message core.Message)
}

type subagentToolRegistryProvider interface {
	GetToolRegistry() *tools.ToolRegistry
}

type subagentMaxStepProvider interface {
	GetMaxSteps() int
	SetMaxSteps(v int)
}

func runAsSubagent(runtime subagentRuntime, task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	originalHistory := runtime.GetHistory()
	runtime.ClearHistory()

	var toolRegistry *tools.ToolRegistry
	disabledTools := map[string]tools.Tool{}
	disabledFunctions := map[string]tools.FunctionTool{}
	if provider, ok := runtime.(subagentToolRegistryProvider); ok {
		toolRegistry = provider.GetToolRegistry()
	}

	if toolFilter != nil && toolRegistry != nil {
		originalTools := toolRegistry.ListTools()
		filtered := toolFilter.Filter(originalTools)
		allowed := map[string]struct{}{}
		for _, name := range filtered {
			allowed[name] = struct{}{}
		}

		allFunctions := toolRegistry.GetAllFunctions()
		for _, name := range originalTools {
			if _, ok := allowed[name]; ok {
				continue
			}

			if tool := toolRegistry.GetTool(name); tool != nil {
				disabledTools[name] = tool
			}
			if fn, ok := allFunctions[name]; ok {
				disabledFunctions[name] = fn
			}
			toolRegistry.UnregisterTool(name)
		}
	}

	var originalMaxSteps *int
	if maxStepsOverride != nil {
		if provider, ok := runtime.(subagentMaxStepProvider); ok {
			current := provider.GetMaxSteps()
			originalMaxSteps = &current
			provider.SetMaxSteps(*maxStepsOverride)
		}
	}

	start := time.Now()
	result, err := runtime.Run(task, nil)
	duration := time.Since(start).Seconds()
	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
		result = "执行失败: " + errorMsg
	}

	isolatedHistory := runtime.GetHistory()
	metadata := getSubagentMetadata(isolatedHistory, duration, errorMsg)

	runtime.ClearHistory()
	for _, msg := range originalHistory {
		runtime.AddMessage(msg)
	}

	if toolRegistry != nil {
		for name, tool := range disabledTools {
			toolRegistry.RegisterTool(tool, false)
			delete(disabledTools, name)
		}
		for name, fn := range disabledFunctions {
			toolRegistry.RegisterFunction(name, fn.Handler, fn.Description)
			delete(disabledFunctions, name)
		}
	}

	if originalMaxSteps != nil {
		if provider, ok := runtime.(subagentMaxStepProvider); ok {
			provider.SetMaxSteps(*originalMaxSteps)
		}
	}

	if returnSummary {
		summary := generateSubagentSummary(task, result, metadata)
		return map[string]any{"success": success, "summary": summary, "metadata": metadata}
	}
	return map[string]any{"success": success, "result": result, "metadata": metadata}
}

func getSubagentMetadata(history []core.Message, duration float64, errorMsg string) map[string]any {
	steps := 0
	totalChars := 0
	for _, msg := range history {
		if msg.Role == core.MessageRoleAssistant {
			steps++
		}
		totalChars += len(msg.Content)
	}

	metadata := map[string]any{
		"steps":            steps,
		"tokens":           totalChars / 4,
		"duration_seconds": float64(int(duration*100)) / 100,
		"tools_used":       extractToolsFromHistory(history),
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	return metadata
}

func extractToolsFromHistory(history []core.Message) []string {
	toolSet := map[string]struct{}{}
	re := regexp.MustCompile(`Action:\s*(\w+)\[`)
	for _, msg := range history {
		if msg.Metadata != nil {
			if rawCalls, ok := msg.Metadata["tool_calls"].([]any); ok {
				for _, raw := range rawCalls {
					callMap, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					if function, ok := callMap["function"].(map[string]any); ok {
						if name, ok := function["name"].(string); ok && name != "" {
							toolSet[name] = struct{}{}
						}
					}
				}
			}
		}

		if msg.Role == core.MessageRoleAssistant {
			for _, m := range re.FindAllStringSubmatch(msg.Content, -1) {
				if len(m) > 1 {
					toolSet[m[1]] = struct{}{}
				}
			}
		}
	}
	out := make([]string, 0, len(toolSet))
	for name := range toolSet {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func generateSubagentSummary(task string, result string, metadata map[string]any) string {
	preview := result
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	parts := []string{
		fmt.Sprintf("任务: %s", task),
		fmt.Sprintf("结果: %s", preview),
		fmt.Sprintf("步数: %v", metadata["steps"]),
		fmt.Sprintf("耗时: %v秒", metadata["duration_seconds"]),
	}
	if toolsUsed, ok := metadata["tools_used"].([]string); ok && len(toolsUsed) > 0 {
		parts = append(parts, fmt.Sprintf("工具: %s", strings.Join(toolsUsed, ", ")))
	}
	if errMsg, ok := metadata["error"].(string); ok && errMsg != "" {
		parts = append(parts, fmt.Sprintf("错误: %s", errMsg))
	}
	return strings.Join(parts, "\n")
}
