package tools

import (
	"testing"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Fatal("NewToolRegistry returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected empty registry, got count %d", registry.Count())
	}
}

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})

	err := registry.RegisterTool(tool, false)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}

	// Test duplicate registration
	err = registry.RegisterTool(tool, false)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestToolRegistry_UnregisterTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})
	registry.RegisterTool(tool, false)

	err := registry.UnregisterTool("test_tool")
	if err != nil {
		t.Fatalf("UnregisterTool failed: %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after unregister, got %d", registry.Count())
	}

	// Test unregistering non-existent tool
	err = registry.UnregisterTool("non_existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent tool")
	}
}

func TestToolRegistry_GetTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})
	registry.RegisterTool(tool, false)

	retrieved, exists := registry.GetTool("test_tool")
	if !exists {
		t.Error("Tool not found")
	}
	if retrieved.Name() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", retrieved.Name())
	}

	// Test non-existent tool
	_, exists = registry.GetTool("non_existent")
	if exists {
		t.Error("Expected non-existent tool to not be found")
	}
}

func TestToolRegistry_ListTools(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewBaseTool("tool1", "Tool 1", []ToolParameter{})
	tool2 := NewBaseTool("tool2", "Tool 2", []ToolParameter{})

	registry.RegisterTool(tool1, false)
	registry.RegisterTool(tool2, false)

	tools := registry.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestToolRegistry_HasTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})
	registry.RegisterTool(tool, false)

	if !registry.HasTool("test_tool") {
		t.Error("Expected HasTool to return true for registered tool")
	}

	if registry.HasTool("non_existent") {
		t.Error("Expected HasTool to return false for non-existent tool")
	}
}

func TestToolRegistry_Clear(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewBaseTool("tool1", "Tool 1", []ToolParameter{})
	tool2 := NewBaseTool("tool2", "Tool 2", []ToolParameter{})

	registry.RegisterTool(tool1, false)
	registry.RegisterTool(tool2, false)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after Clear, got %d", registry.Count())
	}
}

func TestToolRegistry_Merge(t *testing.T) {
	registry1 := NewToolRegistry()
	registry2 := NewToolRegistry()

	tool1 := NewBaseTool("tool1", "Tool 1", []ToolParameter{})
	tool2 := NewBaseTool("tool2", "Tool 2", []ToolParameter{})

	registry1.RegisterTool(tool1, false)
	registry2.RegisterTool(tool2, false)

	registry1.Merge(registry2)

	if registry1.Count() != 2 {
		t.Errorf("Expected count 2 after merge, got %d", registry1.Count())
	}

	if !registry1.HasTool("tool2") {
		t.Error("Expected tool2 to be in registry1 after merge")
	}
}

func TestToolRegistry_Clone(t *testing.T) {
	registry1 := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})
	registry1.RegisterTool(tool, false)

	registry2 := registry1.Clone()

	if registry2.Count() != registry1.Count() {
		t.Errorf("Cloned registry has different count: %d vs %d",
			registry2.Count(), registry1.Count())
	}

	if !registry2.HasTool("test_tool") {
		t.Error("Expected cloned registry to have test_tool")
	}
}

func TestToolRegistry_ExecuteTool(t *testing.T) {
	registry := NewToolRegistry()

	// Create a simple test tool
	fn := func(params map[string]interface{}) (string, error) {
		return "executed", nil
	}

	tf := &ToolFunction{
		Name:       "test_func",
		Parameters: []ToolParameter{},
		Func:       fn,
	}

	registry.RegisterFunction(tf)

	// Execute the function
	result, err := registry.ExecuteTool("test_func", nil)
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if result != "executed" {
		t.Errorf("Expected result 'executed', got '%s'", result)
	}
}

func TestToolRegistry_RegisterFunction(t *testing.T) {
	registry := NewToolRegistry()

	fn := func(params map[string]interface{}) (string, error) {
		return "result", nil
	}

	tf := &ToolFunction{
		Name:       "test_func",
		Parameters: []ToolParameter{},
		Func:       fn,
	}

	err := registry.RegisterFunction(tf)
	if err != nil {
		t.Fatalf("RegisterFunction failed: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}
}

func TestToolRegistry_GetToolsDescription(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewBaseTool("tool1", "Description 1", []ToolParameter{})
	tool2 := NewBaseTool("tool2", "Description 2", []ToolParameter{})

	registry.RegisterTool(tool1, false)
	registry.RegisterTool(tool2, false)

	desc := registry.GetToolsDescription()

	if desc == "" {
		t.Error("GetToolsDescription returned empty string")
	}

	// Check that tool names are in the description
	// Note: We can't test the exact format as it may change
}

func TestToolRegistry_ToOpenAITools(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewBaseTool("test_tool", "A test tool", []ToolParameter{})
	registry.RegisterTool(tool, false)

	schemas := registry.ToOpenAITools()

	if len(schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(schemas))
	}

	if schemas[0]["type"] != "function" {
		t.Errorf("Expected schema type 'function', got '%v'", schemas[0]["type"])
	}
}
