package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type FunctionTool struct {
	Description string
	Handler     func(input string) any
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

// RegisterFunction mirrors Python register_function with both call styles:
// 1) RegisterFunction(handler, name?, description?)
// 2) RegisterFunction(name, description, handler)
func (r *ToolRegistry) RegisterFunction(funcOrName any, args ...any) {
	name, description, handler, hasDescription := parseFunctionRegistration(funcOrName, args...)
	if handler == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !hasDescription {
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

	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		r.toolOrder = removeName(r.toolOrder, name)
		return
	}

	if _, exists := r.functions[name]; exists {
		delete(r.functions, name)
		r.funcOrder = removeName(r.funcOrder, name)
	}
}

// DisableTool mirrors Python subagent filtering behavior:
// only remove Tool objects while keeping function-tools untouched.
func (r *ToolRegistry) DisableTool(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; !exists {
		return false
	}
	delete(r.tools, name)
	r.toolOrder = removeName(r.toolOrder, name)
	return true
}

func (r *ToolRegistry) GetTool(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

func (r *ToolRegistry) GetFunction(name string) func(input string) any {
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

func (r *ToolRegistry) ListFunctions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.funcOrder))
	for _, name := range r.funcOrder {
		if _, ok := r.functions[name]; ok {
			out = append(out, name)
		}
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
		if parsedParams, parsed, parseErr := parseToolParameters(inputText); parseErr != nil {
			response = Error(
				fmt.Sprintf("执行工具 '%s' 时发生异常: %v", name, parseErr),
				ToolErrorCodeExecutionError,
				map[string]any{"tool_name": name, "input": inputText},
			)
			if r.CircuitBreaker != nil {
				r.CircuitBreaker.RecordResult(name, response)
			}
			return response
		} else if parsed {
			parameters = parsedParams
		}

		var recovered any
		func() {
			defer func() {
				if p := recover(); p != nil {
					recovered = p
				}
			}()
			response = tool.RunWithTiming(parameters)
		}()

		if recovered != nil {
			response = Error(
				fmt.Sprintf("执行工具 '%s' 时发生异常: %v", name, recovered),
				ToolErrorCodeExecutionError,
				map[string]any{"tool_name": name, "input": inputText},
			)
		}
	} else if hasFn {
		start := time.Now()
		result := any(nil)
		var recovered any
		func() {
			defer func() {
				if p := recover(); p != nil {
					recovered = p
				}
			}()
			result = fnInfo.Handler(inputText)
		}()

		if recovered != nil {
			response = Error(
				fmt.Sprintf("函数执行失败: %v", recovered),
				ToolErrorCodeExecutionError,
				map[string]any{"tool_name": name, "input": inputText},
			)
		} else {
			response = Success(
				fmt.Sprintf("%v", result),
				map[string]any{"output": result},
				nil,
				nil,
			)
		}

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
	if r.readMetadataCache == nil {
		r.readMetadataCache = map[string]map[string]any{}
	}
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

func parseFunctionRegistration(funcOrName any, args ...any) (string, string, func(input string) any, bool) {
	// Legacy style: register_function(name, description, func)
	if name, ok := funcOrName.(string); ok {
		if len(args) < 2 {
			return "", "", nil, false
		}
		description, _ := args[0].(string)
		handler := coerceFunctionHandler(args[1])
		return name, description, handler, true
	}

	// Modern style: register_function(func, name=None, description=None)
	handler := coerceFunctionHandler(funcOrName)
	if handler == nil {
		return "", "", nil, false
	}

	name := ""
	hasExplicitName := false
	if len(args) > 0 {
		hasExplicitName = true
		name, _ = args[0].(string)
	}
	if !hasExplicitName {
		name = inferFunctionName(funcOrName)
	}

	description := ""
	hasDescription := false
	if len(args) > 1 {
		hasDescription = true
		description, _ = args[1].(string)
	}

	return name, description, handler, hasDescription
}

func inferFunctionName(handler any) string {
	if handler == nil {
		return ""
	}

	value := reflect.ValueOf(handler)
	if value.Kind() != reflect.Func {
		return ""
	}

	fn := runtime.FuncForPC(value.Pointer())
	if fn == nil {
		return ""
	}

	name := fn.Name()
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.TrimSuffix(name, "-fm")
}

func coerceFunctionHandler(handler any) func(input string) any {
	switch fn := handler.(type) {
	case func(string) any:
		return fn
	case func(string) ToolResponse:
		return func(input string) any {
			return fn(input)
		}
	case func(string) string:
		return func(input string) any {
			return fn(input)
		}
	default:
		return nil
	}
}

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

func parseToolParameters(inputText string) (map[string]any, bool, error) {
	var parsed any
	if err := json.Unmarshal([]byte(inputText), &parsed); err != nil {
		return nil, false, nil
	}
	parameters, ok := parsed.(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("工具参数必须是 JSON 对象")
	}
	return parameters, true, nil
}
