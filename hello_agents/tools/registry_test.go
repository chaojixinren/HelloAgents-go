package tools

import (
	"strings"
	"testing"
)

func TestExecuteToolFunctionWrapsReturnAsSuccess(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterFunction("wrapped", "test", func(input string) ToolResponse {
		return Partial(
			"partial-result",
			map[string]any{"echo": input},
			nil,
			nil,
		)
	})

	response := registry.ExecuteTool("wrapped", "payload")

	if response.Status != ToolStatusSuccess {
		t.Fatalf("status = %q, want %q", response.Status, ToolStatusSuccess)
	}
	if response.Data == nil {
		t.Fatalf("data should not be nil")
	}
	output, ok := response.Data["output"].(ToolResponse)
	if !ok {
		t.Fatalf("data.output type = %T, want ToolResponse", response.Data["output"])
	}
	if output.Status != ToolStatusPartial {
		t.Fatalf("wrapped output status = %q, want %q", output.Status, ToolStatusPartial)
	}
	if response.Stats == nil || response.Stats["time_ms"] == nil {
		t.Fatalf("stats.time_ms missing from wrapped response")
	}
	if response.Context == nil {
		t.Fatalf("context should not be nil")
	}
	if response.Context["tool_name"] != "wrapped" {
		t.Fatalf("context.tool_name = %v, want %q", response.Context["tool_name"], "wrapped")
	}
}

func TestClearKeepsReadMetadataCache(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.CacheReadMetadata("a.txt", map[string]any{"file_mtime_ms": 123})

	registry.Clear()

	meta := registry.GetReadMetadata("a.txt")
	if meta == nil {
		t.Fatalf("GetReadMetadata() = nil after Clear(), want preserved metadata")
	}
	if meta["file_mtime_ms"] != 123 {
		t.Fatalf("file_mtime_ms = %v, want 123", meta["file_mtime_ms"])
	}
}

type panicRunWithTimingTool struct{}

func (t *panicRunWithTimingTool) GetName() string {
	return "panic_tool"
}

func (t *panicRunWithTimingTool) GetDescription() string {
	return "panic tool"
}

func (t *panicRunWithTimingTool) GetParameters() []ToolParameter {
	return []ToolParameter{}
}

func (t *panicRunWithTimingTool) Run(parameters map[string]any) ToolResponse {
	return Success("ok", nil, nil, nil)
}

func (t *panicRunWithTimingTool) RunWithTiming(parameters map[string]any) ToolResponse {
	panic("boom")
}

func (t *panicRunWithTimingTool) ARun(parameters map[string]any) ToolResponse {
	return t.Run(parameters)
}

func (t *panicRunWithTimingTool) ARunWithTiming(parameters map[string]any) ToolResponse {
	return t.RunWithTiming(parameters)
}

func TestExecuteToolReturnsExecutionErrorWhenToolPanics(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterTool(&panicRunWithTimingTool{}, false)

	response := registry.ExecuteTool("panic_tool", "payload")
	if response.Status != ToolStatusError {
		t.Fatalf("status = %q, want %q", response.Status, ToolStatusError)
	}
	if response.ErrorInfo == nil || response.ErrorInfo["code"] != ToolErrorCodeExecutionError {
		t.Fatalf("error code = %v, want %q", response.ErrorInfo, ToolErrorCodeExecutionError)
	}
	if !strings.Contains(response.Text, "执行工具 'panic_tool' 时发生异常: boom") {
		t.Fatalf("text = %q, want panic execution error", response.Text)
	}
}

type namedTool struct {
	BaseTool
}

func newNamedTool(name string) *namedTool {
	base := NewBaseTool(name, "named tool", false)
	return &namedTool{BaseTool: base}
}

func TestUnregisterPrefersToolBeforeFunctionLikePython(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterTool(newNamedTool("dup"), false)
	registry.RegisterFunction("dup", "fn", func(input string) string {
		return "ok"
	})

	registry.Unregister("dup")

	if tool := registry.GetTool("dup"); tool != nil {
		t.Fatalf("tool should be removed after Unregister")
	}
	if fn := registry.GetFunction("dup"); fn == nil {
		t.Fatalf("function should remain when same name existed in both tool/function registries")
	}
}

type mapExpectTool struct {
	BaseTool
}

func newMapExpectTool() *mapExpectTool {
	base := NewBaseTool("map_expect", "expects object-like input", false)
	return &mapExpectTool{BaseTool: base}
}

func (t *mapExpectTool) Run(parameters map[string]any) ToolResponse {
	return Success("ok", map[string]any{"params": parameters}, nil, nil)
}

func TestExecuteToolReturnsExecutionErrorForNonObjectJSONInput(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterTool(newMapExpectTool(), false)

	response := registry.ExecuteTool("map_expect", `["a","b"]`)
	if response.Status != ToolStatusError {
		t.Fatalf("status = %q, want %q", response.Status, ToolStatusError)
	}
	if response.ErrorInfo == nil || response.ErrorInfo["code"] != ToolErrorCodeExecutionError {
		t.Fatalf("error code = %v, want %q", response.ErrorInfo, ToolErrorCodeExecutionError)
	}
	if !strings.Contains(response.Text, "工具参数必须是 JSON 对象") {
		t.Fatalf("text = %q, want non-object JSON parse error", response.Text)
	}
}

func TestRegisterFunctionAllowsExplicitEmptyNameLikePython(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterFunction(func(input string) any {
		return input
	}, "", "empty-name")

	if fn := registry.GetFunction(""); fn == nil {
		t.Fatalf("function with explicit empty name should be registered")
	}
	if desc, ok := registry.GetAllFunctions()[""]; !ok || desc.Description != "empty-name" {
		t.Fatalf("description for empty-name function not preserved, got %#v", registry.GetAllFunctions()[""])
	}
}

func TestRegisterFunctionKeepsExplicitEmptyDescriptionLikePython(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterFunction(func(input string) any {
		return input
	}, "fn", "")

	fn, ok := registry.GetAllFunctions()["fn"]
	if !ok {
		t.Fatalf("function fn not registered")
	}
	if fn.Description != "" {
		t.Fatalf("description = %q, want explicit empty string", fn.Description)
	}
}

func TestRegisterFunctionLegacyNilDescriptionFallsBackToDefault(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterFunction("legacy", nil, func(input string) string {
		return input
	})

	fn, ok := registry.GetAllFunctions()["legacy"]
	if !ok {
		t.Fatalf("legacy function not registered")
	}
	if fn.Description != "执行 legacy" {
		t.Fatalf("description = %q, want %q", fn.Description, "执行 legacy")
	}
}

func TestRegisterFunctionModernNilNameAndDescriptionUseDefaults(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.RegisterFunction(func(input string) any {
		return input
	}, nil, nil)

	names := registry.ListFunctions()
	if len(names) != 1 {
		t.Fatalf("len(names) = %d, want 1", len(names))
	}
	name := names[0]
	if name == "" {
		t.Fatalf("inferred function name should not be empty")
	}
	fn := registry.GetAllFunctions()[name]
	if fn.Description != "执行 "+name {
		t.Fatalf("description = %q, want %q", fn.Description, "执行 "+name)
	}
}

func TestClearReadCacheEmptyStringClearsAllLikePythonTruthy(t *testing.T) {
	registry := NewToolRegistry(nil)
	registry.CacheReadMetadata("a.txt", map[string]any{"file_mtime_ms": 1})
	registry.CacheReadMetadata("b.txt", map[string]any{"file_mtime_ms": 2})

	empty := ""
	registry.ClearReadCache(&empty)

	if got := registry.ReadMetadataCache(); len(got) != 0 {
		t.Fatalf("ReadMetadataCache len = %d, want 0", len(got))
	}
}
