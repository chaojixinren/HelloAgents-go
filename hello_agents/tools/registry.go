package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type FunctionTool struct {
	Description string
	Handler     func(input string) ToolResponse
}

type expandableTool interface {
	GetExpandedTools() []Tool
}

// ToolRegistry keeps both Tool objects and function-style tools.
type ToolRegistry struct {
	mu sync.RWMutex

	tools     map[string]Tool
	functions map[string]FunctionTool
	toolOrder []string
	funcOrder []string

	CircuitBreaker    *CircuitBreaker
	readMetadataCache map[string]map[string]any
}

func NewToolRegistry(circuitBreaker *CircuitBreaker) *ToolRegistry {
	if circuitBreaker == nil {
		circuitBreaker = NewCircuitBreaker(3, 300, true)
	}
	return &ToolRegistry{
		tools:             map[string]Tool{},
		functions:         map[string]FunctionTool{},
		toolOrder:         []string{},
		funcOrder:         []string{},
		CircuitBreaker:    circuitBreaker,
		readMetadataCache: map[string]map[string]any{},
	}
}

func (r *ToolRegistry) RegisterTool(tool Tool, autoExpandArgs ...bool) {
	if tool == nil {
		return
	}
	autoExpand := true
	if len(autoExpandArgs) > 0 {
		autoExpand = autoExpandArgs[0]
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if autoExpand {
		if ex, ok := tool.(expandableTool); ok {
			expanded := ex.GetExpandedTools()
			if len(expanded) > 0 {
				for _, sub := range expanded {
					if sub != nil {
						name := sub.GetName()
						if _, exists := r.tools[name]; !exists {
							r.toolOrder = append(r.toolOrder, name)
						}
						r.tools[name] = sub
					}
				}
				return
			}
		}
	}

	name := tool.GetName()
	if _, exists := r.tools[name]; !exists {
		r.toolOrder = append(r.toolOrder, name)
	}
	r.tools[name] = tool
}

func (r *ToolRegistry) RegisterFunction(name string, handler func(input string) ToolResponse, description string) {
	if name == "" || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if description == "" {
		description = "执行 " + name
	}
	if _, exists := r.functions[name]; !exists {
		r.funcOrder = append(r.funcOrder, name)
	}
	r.functions[name] = FunctionTool{Description: description, Handler: handler}
}

func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
	delete(r.functions, name)
	r.toolOrder = removeName(r.toolOrder, name)
	r.funcOrder = removeName(r.funcOrder, name)
}

func (r *ToolRegistry) UnregisterTool(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, toolExists := r.tools[name]
	_, fnExists := r.functions[name]
	delete(r.tools, name)
	delete(r.functions, name)
	r.toolOrder = removeName(r.toolOrder, name)
	r.funcOrder = removeName(r.funcOrder, name)
	return toolExists || fnExists
}

func (r *ToolRegistry) GetTool(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

func (r *ToolRegistry) GetFunction(name string) func(input string) ToolResponse {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if fn, ok := r.functions[name]; ok {
		return fn.Handler
	}
	return nil
}

func (r *ToolRegistry) GetAllFunctions() map[string]FunctionTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]FunctionTool, len(r.functions))
	for k, v := range r.functions {
		out[k] = v
	}
	return out
}

func (r *ToolRegistry) ExecuteTool(name string, inputText string) ToolResponse {
	if r.CircuitBreaker != nil && r.CircuitBreaker.IsOpen(name) {
		status := r.CircuitBreaker.GetStatus(name)
		recoverIn := 0
		if v, ok := status["recover_in_seconds"].(int); ok {
			recoverIn = v
		}
		return Error(
			fmt.Sprintf("工具 '%s' 当前被禁用，由于连续失败。%d 秒后可用。", name, recoverIn),
			ToolErrorCodeCircuitOpen,
			map[string]any{"tool_name": name, "circuit_status": status},
		)
	}

	r.mu.RLock()
	tool := r.tools[name]
	fnInfo, hasFn := r.functions[name]
	r.mu.RUnlock()

	var response ToolResponse

	if tool != nil {
		parameters := map[string]any{"input": inputText}
		trimmed := strings.TrimSpace(inputText)
		if strings.HasPrefix(trimmed, "{") {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				parameters = parsed
			}
		}
		response = tool.RunWithTiming(parameters)
	} else if hasFn {
		start := time.Now()
		func() {
			defer func() {
				if p := recover(); p != nil {
					response = Error(
						fmt.Sprintf("函数执行失败: %v", p),
						ToolErrorCodeExecutionError,
						map[string]any{"tool_name": name, "input": inputText},
					)
				}
			}()
			resp := fnInfo.Handler(inputText)
			response = resp
		}()
		if response.Stats == nil {
			response.Stats = map[string]any{}
		}
		response.Stats["time_ms"] = time.Since(start).Milliseconds()
		if response.Context == nil {
			response.Context = map[string]any{}
		}
		response.Context["tool_name"] = name
		response.Context["input"] = inputText
	} else {
		response = Error(
			fmt.Sprintf("未找到名为 '%s' 的工具", name),
			ToolErrorCodeNotFound,
			map[string]any{"tool_name": name},
		)
	}

	if r.CircuitBreaker != nil {
		r.CircuitBreaker.RecordResult(name, response)
	}
	return response
}

func (r *ToolRegistry) GetToolsDescription() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	descriptions := make([]string, 0, len(r.tools)+len(r.functions))
	for _, name := range r.toolOrder {
		tool := r.tools[name]
		if tool == nil {
			continue
		}
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", tool.GetName(), tool.GetDescription()))
	}
	for _, name := range r.funcOrder {
		fn, ok := r.functions[name]
		if !ok {
			continue
		}
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", name, fn.Description))
	}
	if len(descriptions) == 0 {
		return "暂无可用工具"
	}
	return strings.Join(descriptions, "\n")
}

func (r *ToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools)+len(r.functions))
	for _, name := range r.toolOrder {
		if _, ok := r.tools[name]; ok {
			names = append(names, name)
		}
	}
	for _, name := range r.funcOrder {
		if _, ok := r.functions[name]; ok {
			names = append(names, name)
		}
	}
	return names
}

func (r *ToolRegistry) GetAllTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Tool, 0, len(r.tools))
	for _, name := range r.toolOrder {
		if t, ok := r.tools[name]; ok {
			items = append(items, t)
		}
	}
	return items
}

func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = map[string]Tool{}
	r.functions = map[string]FunctionTool{}
	r.toolOrder = []string{}
	r.funcOrder = []string{}
	r.readMetadataCache = map[string]map[string]any{}
}

func (r *ToolRegistry) CacheReadMetadata(filePath string, metadata map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.readMetadataCache[filePath] = metadata
}

func (r *ToolRegistry) GetReadMetadata(filePath string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.readMetadataCache[filePath]
}

func (r *ToolRegistry) ClearReadCache(filePath *string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if filePath == nil {
		r.readMetadataCache = map[string]map[string]any{}
		return
	}
	delete(r.readMetadataCache, *filePath)
}

func (r *ToolRegistry) ReadMetadataCache() map[string]map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]map[string]any{}
	for k, v := range r.readMetadataCache {
		out[k] = v
	}
	return out
}

var GlobalRegistry = NewToolRegistry(nil)

func removeName(names []string, target string) []string {
	if len(names) == 0 {
		return names
	}
	out := names[:0]
	for _, name := range names {
		if name != target {
			out = append(out, name)
		}
	}
	return out
}
