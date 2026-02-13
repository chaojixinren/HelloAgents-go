package tools

import (
	"testing"
)

func TestToolParameter(t *testing.T) {
	param := ToolParameter{
		Name:        "test_param",
		Type:        "string",
		Description: "A test parameter",
		Required:    true,
	}

	if param.Name != "test_param" {
		t.Errorf("Expected Name to be 'test_param', got '%s'", param.Name)
	}
	if param.Type != "string" {
		t.Errorf("Expected Type to be 'string', got '%s'", param.Type)
	}
	if !param.Required {
		t.Error("Expected Required to be true")
	}
}

func TestConvertParameters(t *testing.T) {
	// Test map input
	paramsMap := map[string]interface{}{"key": "value"}
	result, err := ConvertParameters(paramsMap)
	if err != nil {
		t.Fatalf("ConvertParameters failed: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Expected key to be 'value', got '%v'", result["key"])
	}

	// Test JSON string input
	jsonStr := `{"key": "value"}`
	result, err = ConvertParameters(jsonStr)
	if err != nil {
		t.Fatalf("ConvertParameters failed: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Expected key to be 'value', got '%v'", result["key"])
	}

	// Test nil input
	result, err = ConvertParameters(nil)
	if err != nil {
		t.Fatalf("ConvertParameters failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil input, got %v", result)
	}
}

func TestParseToolCall(t *testing.T) {
	// Valid tool call
	input := "[TOOL_CALL:calculator:2 + 2]"
	toolName, params, found := ParseToolCall(input)
	if !found {
		t.Error("Expected to find tool call")
	}
	if toolName != "calculator" {
		t.Errorf("Expected tool name 'calculator', got '%s'", toolName)
	}
	if params != "2 + 2" {
		t.Errorf("Expected params '2 + 2', got '%s'", params)
	}

	// No tool call
	input = "Just regular text"
	_, _, found = ParseToolCall(input)
	if found {
		t.Error("Expected not to find tool call")
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		typeStr  string
		expected bool
	}{
		{"string valid", "hello", "string", true},
		{"string invalid", 123, "string", false},
		{"integer valid", 42, "integer", true},
		{"integer valid float", 42.0, "integer", true},
		{"integer invalid float", 42.5, "integer", false},
		{"number valid", 42.5, "number", true},
		{"number valid int", 42, "number", true},
		{"boolean valid", true, "boolean", true},
		{"boolean invalid string", "not_bool", "boolean", false},
		{"array valid", []interface{}{1, 2, 3}, "array", true},
		{"array invalid", "not_array", "array", false},
		{"object valid", map[string]interface{}{"key": "value"}, "object", true},
		{"object invalid", "not_object", "object", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateType(tt.value, tt.typeStr)
			if result != tt.expected {
				t.Errorf("validateType(%v, %s) = %v; want %v",
					tt.value, tt.typeStr, result, tt.expected)
			}
		})
	}
}

func TestConvertTypeToOpenAI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"integer", "integer"},
		{"float", "number"},
		{"number", "number"},
		{"bool", "boolean"},
		{"boolean", "boolean"},
		{"list", "array"},
		{"array", "array"},
		{"dict", "object"},
		{"map", "object"},
		{"object", "object"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertTypeToOpenAI(tt.input)
			if result != tt.expected {
				t.Errorf("convertTypeToOpenAI(%s) = %s; want %s",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestBaseTool(t *testing.T) {
	params := []ToolParameter{
		{Name: "param1", Type: "string", Description: "First param", Required: true},
		{Name: "param2", Type: "integer", Description: "Second param", Required: false},
	}

	tool := NewBaseTool("test_tool", "A test tool", params)

	// Test basic properties
	if tool.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", tool.Description())
	}

	// Test GetParameters
	retrievedParams := tool.GetParameters()
	if len(retrievedParams) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(retrievedParams))
	}

	// Test Validate with valid parameters
	validParams := map[string]interface{}{"param1": "value"}
	if !tool.Validate(validParams) {
		t.Error("Expected validation to pass with required param")
	}

	// Test Validate with missing required parameter
	invalidParams := map[string]interface{}{"param2": 42}
	if tool.Validate(invalidParams) {
		t.Error("Expected validation to fail without required param")
	}

	// Test ToDict
	dict := tool.ToDict()
	if dict["name"] != "test_tool" {
		t.Errorf("Expected dict name 'test_tool', got '%v'", dict["name"])
	}

	// Test ToOpenAISchema
	schema := tool.ToOpenAISchema()
	if schema["type"] != "function" {
		t.Errorf("Expected schema type 'function', got '%v'", schema["type"])
	}
}

func TestToolFunction(t *testing.T) {
	called := false
	fn := func(params map[string]interface{}) (string, error) {
		called = true
		return "result", nil
	}

	tf := &ToolFunction{
		Name:        "test_func",
		Description: "A test function",
		Parameters: []ToolParameter{
			{Name: "input", Type: "string", Description: "Input", Required: true},
		},
		Func: fn,
	}

	// Test Run
	result, err := tf.Run(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result != "result" {
		t.Errorf("Expected result 'result', got '%s'", result)
	}
	if !called {
		t.Error("Function was not called")
	}

	// Test Validate
	if !tf.Validate(map[string]interface{}{"input": "test"}) {
		t.Error("Expected validation to pass")
	}
	if tf.Validate(map[string]interface{}{}) {
		t.Error("Expected validation to fail without required param")
	}
}
