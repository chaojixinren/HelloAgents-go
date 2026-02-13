package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ToolParameter defines a tool parameter with its schema.
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// Tool is the interface that all tools must implement.
type Tool interface {
	// Name returns the unique name of the tool.
	Name() string

	// Description returns a human-readable description of what the tool does.
	Description() string

	// Run executes the tool with the given parameters.
	// Returns the result as a string and any error that occurred.
	Run(parameters map[string]interface{}) (string, error)

	// GetParameters returns the list of parameters this tool accepts.
	GetParameters() []ToolParameter

	// Validate checks if the provided parameters are valid for this tool.
	Validate(parameters map[string]interface{}) bool

	// ToDict returns a dictionary representation of the tool.
	ToDict() map[string]interface{}

	// ToOpenAISchema returns the OpenAI function calling schema for this tool.
	ToOpenAISchema() map[string]interface{}
}

// ExpandableTool is an interface for tools that can expand into multiple sub-tools.
type ExpandableTool interface {
	Tool
	// GetExpandedTools returns a list of sub-tools that this tool expands into.
	GetExpandedTools() []Tool
}

// ToolFunction represents a function that can be called by an agent.
type ToolFunction struct {
	Name        string
	Description string
	Parameters  []ToolParameter
	Func        func(map[string]interface{}) (string, error)
}

// Name returns the function name.
func (tf *ToolFunction) Name() string {
	return tf.Name
}

// Description returns the function description.
func (tf *ToolFunction) Description() string {
	return tf.Description
}

// Run executes the function with the given parameters.
func (tf *ToolFunction) Run(parameters map[string]interface{}) (string, error) {
	return tf.Func(parameters)
}

// GetParameters returns the function parameters.
func (tf *ToolFunction) GetParameters() []ToolParameter {
	return tf.Parameters
}

// Validate checks if all required parameters are present and have valid types.
func (tf *ToolFunction) Validate(parameters map[string]interface{}) bool {
	for _, param := range tf.Parameters {
		if param.Required {
			if _, exists := parameters[param.Name]; !exists {
				return false
			}
		}

		// Type validation
		if value, exists := parameters[param.Name]; exists {
			if !validateType(value, param.Type) {
				return false
			}
		}
	}
	return true
}

// ToDict returns a dictionary representation of the function.
func (tf *ToolFunction) ToDict() map[string]interface{} {
	params := make(map[string]interface{})
	for _, p := range tf.Parameters {
		paramMap := map[string]interface{}{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			paramMap["required"] = true
		}
		if p.Default != nil {
			paramMap["default"] = p.Default
		}
		if len(p.Enum) > 0 {
			paramMap["enum"] = p.Enum
		}
		params[p.Name] = paramMap
	}

	return map[string]interface{}{
		"name":        tf.Name,
		"description": tf.Description,
		"parameters":  params,
	}
}

// ToOpenAISchema returns the OpenAI function calling schema.
func (tf *ToolFunction) ToOpenAISchema() map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, p := range tf.Parameters {
		prop := map[string]interface{}{
			"type":        convertTypeToOpenAI(p.Type),
			"description": p.Description,
		}
		if p.Default != nil {
			prop["default"] = p.Default
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	return map[string]interface{}{
		"type":        "function",
		"function": map[string]interface{}{
			"name":        tf.Name,
			"description": tf.Description,
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

// validateType checks if a value matches the expected type.
func validateType(value interface{}, expectedType string) bool {
	if value == nil {
		return true
	}

	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer", "int":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		case float64:
			// JSON numbers are float64 by default
			f := value.(float64)
			return f == float64(int(f))
		case string:
			// Try to parse string as int
			_, err := strconv.ParseInt(value.(string), 10, 64)
			return err == nil
		}
		return false
	case "number", "float":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		case string:
			_, err := strconv.ParseFloat(value.(string), 64)
			return err == nil
		}
		return false
	case "boolean", "bool":
		switch v := value.(type) {
		case bool:
			return true
		case string:
			lowered := strings.ToLower(v)
			return lowered == "true" || lowered == "false"
		}
		return false
	case "array", "list":
		_, ok := value.([]interface{})
		if !ok {
			// Check for slices via reflection
			rv := reflect.ValueOf(value)
			return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
		}
		return true
	case "object", "dict", "map":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// convertTypeToOpenAI converts tool parameter types to OpenAI schema types.
func convertTypeToOpenAI(toolType string) string {
	switch strings.ToLower(toolType) {
	case "string":
		return "string"
	case "integer", "int":
		return "integer"
	case "number", "float":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "array", "list":
		return "array"
	case "object", "dict", "map":
		return "object"
	default:
		return "string"
	}
}

// BaseTool provides a default implementation of the Tool interface.
// Other tools can embed this struct to get default behavior.
type BaseTool struct {
	toolName        string
	toolDescription string
	parameters      []ToolParameter
}

// NewBaseTool creates a new BaseTool with the given name, description, and parameters.
func NewBaseTool(name, description string, parameters []ToolParameter) *BaseTool {
	return &BaseTool{
		toolName:        name,
		toolDescription: description,
		parameters:      parameters,
	}
}

// Name returns the tool name.
func (bt *BaseTool) Name() string {
	return bt.toolName
}

// Description returns the tool description.
func (bt *BaseTool) Description() string {
	return bt.toolDescription
}

// GetParameters returns the tool parameters.
func (bt *BaseTool) GetParameters() []ToolParameter {
	return bt.parameters
}

// Validate checks if all required parameters are present.
func (bt *BaseTool) Validate(parameters map[string]interface{}) bool {
	for _, param := range bt.parameters {
		if param.Required {
			if _, exists := parameters[param.Name]; !exists {
				return false
			}
		}
		if value, exists := parameters[param.Name]; exists {
			if !validateType(value, param.Type) {
				return false
			}
		}
	}
	return true
}

// ToDict returns a dictionary representation of the tool.
func (bt *BaseTool) ToDict() map[string]interface{} {
	params := make(map[string]interface{})
	for _, p := range bt.parameters {
		paramMap := map[string]interface{}{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			paramMap["required"] = true
		}
		if p.Default != nil {
			paramMap["default"] = p.Default
		}
		if len(p.Enum) > 0 {
			paramMap["enum"] = p.Enum
		}
		params[p.Name] = paramMap
	}

	return map[string]interface{}{
		"name":        bt.toolName,
		"description": bt.toolDescription,
		"parameters":  params,
	}
}

// ToOpenAISchema returns the OpenAI function calling schema.
func (bt *BaseTool) ToOpenAISchema() map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, p := range bt.parameters {
		prop := map[string]interface{}{
			"type":        convertTypeToOpenAI(p.Type),
			"description": p.Description,
		}
		if p.Default != nil {
			prop["default"] = p.Default
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	return map[string]interface{}{
		"type":        "function",
		"function": map[string]interface{}{
			"name":        bt.toolName,
			"description": bt.toolDescription,
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

// Run must be implemented by concrete tools.
func (bt *BaseTool) Run(parameters map[string]interface{}) (string, error) {
	return "", fmt.Errorf("Run method must be implemented by concrete tool")
}

// ConvertParameters converts parameters from various types to map[string]interface{}.
func ConvertParameters(params interface{}) (map[string]interface{}, error) {
	switch v := params.(type) {
	case map[string]interface{}:
		return v, nil
	case string:
		// Try to parse as JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return nil, fmt.Errorf("failed to parse parameters as JSON: %w", err)
		}
		return result, nil
	case nil:
		return make(map[string]interface{}), nil
	default:
		// Try to convert via reflection
		rv := reflect.ValueOf(params)
		if rv.Kind() == reflect.Map {
			result := make(map[string]interface{})
			for _, key := range rv.MapKeys() {
				val := rv.MapIndex(key)
				result[key.String()] = val.Interface()
			}
			return result, nil
		}
		return nil, fmt.Errorf("unsupported parameter type: %T", params)
	}
}

// ParseToolCall parses a tool call string in the format [TOOL_CALL:tool_name:parameters].
func ParseToolCall(input string) (toolName, params string, found bool) {
	re := regexp.MustCompile(`\[TOOL_CALL:([^:]+):(.+)\]`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 3 {
		return matches[1], matches[2], true
	}
	return "", "", false
}
