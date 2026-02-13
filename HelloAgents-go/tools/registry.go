package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ToolRegistry manages tool registration and execution.
type ToolRegistry struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	functions map[string]*ToolFunction
}

// NewToolRegistry creates a new ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:     make(map[string]Tool),
		functions: make(map[string]*ToolFunction),
	}
}

// RegisterTool registers a tool. If autoExpand is true and the tool implements
// ExpandableTool, it will register all expanded sub-tools.
func (r *ToolRegistry) RegisterTool(tool Tool, autoExpand bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check if tool is expandable
	if expandable, ok := tool.(ExpandableTool); ok && autoExpand {
		// Register all expanded tools
		expandedTools := expandable.GetExpandedTools()
		if len(expandedTools) == 0 {
			return fmt.Errorf("expandable tool %s returned no sub-tools", name)
		}

		// Register the expandable tool itself
		r.tools[name] = tool

		// Register each sub-tool
		for _, subTool := range expandedTools {
			subName := subTool.Name()
			if existingTool, exists := r.tools[subName]; exists {
				return fmt.Errorf("tool %s already registered (from %s)", subName, existingTool.Name())
			}
			r.tools[subName] = subTool
		}

		return nil
	}

	// Register as a regular tool
	if existingTool, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered (from %s)", name, existingTool.Name())
	}

	r.tools[name] = tool
	return nil
}

// RegisterFunction registers a ToolFunction directly.
func (r *ToolRegistry) RegisterFunction(fn *ToolFunction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("function cannot be nil")
	}

	name := fn.Name()
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	if _, exists := r.functions[name]; exists {
		return fmt.Errorf("function %s already registered", name)
	}

	r.functions[name] = fn
	return nil
}

// UnregisterTool removes a tool from the registry.
func (r *ToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	return nil
}

// UnregisterFunction removes a function from the registry.
func (r *ToolRegistry) UnregisterFunction(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.functions[name]; !exists {
		return fmt.Errorf("function %s not found", name)
	}

	delete(r.functions, name)
	return nil
}

// ExecuteTool executes a tool by name with the given input.
// The input can be a JSON string or a map[string]interface{}.
func (r *ToolRegistry) ExecuteTool(name string, input interface{}) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try tools
	tool, exists := r.tools[name]
	if !exists {
		// Try functions
		fn, exists := r.functions[name]
		if !exists {
			return "", fmt.Errorf("tool %s not found", name)
		}
		tool = fn
	}

	// Convert input to parameters
	var parameters map[string]interface{}
	var err error

	switch v := input.(type) {
	case map[string]interface{}:
		parameters = v
	case string:
		parameters, err = ConvertParameters(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse parameters: %w", err)
		}
	case nil:
		parameters = make(map[string]interface{})
	default:
		return "", fmt.Errorf("unsupported input type: %T", input)
	}

	// Validate parameters
	if !tool.Validate(parameters) {
		return "", fmt.Errorf("parameter validation failed for tool %s", name)
	}

	// Execute the tool
	result, err := tool.Run(parameters)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// GetTool retrieves a tool by name.
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetFunction retrieves a function by name.
func (r *ToolRegistry) GetFunction(name string) (*ToolFunction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, exists := r.functions[name]
	return fn, exists
}

// ListTools returns a list of all registered tool names.
func (r *ToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools)+len(r.functions))
	for name := range r.tools {
		names = append(names, name)
	}
	for name := range r.functions {
		names = append(names, name)
	}

	return names
}

// ListToolsWithDetails returns a list of all tools with their descriptions.
func (r *ToolRegistry) ListToolsWithDetails() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	details := make([]map[string]interface{}, 0, len(r.tools)+len(r.functions))

	for _, tool := range r.tools {
		details = append(details, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.GetParameters(),
		})
	}

	for _, fn := range r.functions {
		details = append(details, map[string]interface{}{
			"name":        fn.Name(),
			"description": fn.Description(),
			"parameters":  fn.GetParameters(),
		})
	}

	return details
}

// GetToolsDescription returns a formatted string describing all available tools.
func (r *ToolRegistry) GetToolsDescription() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.tools) == 0 && len(r.functions) == 0 {
		return "No tools available."
	}

	var builder strings.Builder
	builder.WriteString("Available tools:\n")

	// List tools
	for _, tool := range r.tools {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
		params := tool.GetParameters()
		if len(params) > 0 {
			builder.WriteString("  Parameters:\n")
			for _, p := range params {
				required := ""
				if p.Required {
					required = " (required)"
				}
				builder.WriteString(fmt.Sprintf("    - %s (%s)%s: %s\n", p.Name, p.Type, required, p.Description))
			}
		}
	}

	// List functions
	for _, fn := range r.functions {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", fn.Name(), fn.Description()))
		params := fn.GetParameters()
		if len(params) > 0 {
			builder.WriteString("  Parameters:\n")
			for _, p := range params {
				required := ""
				if p.Required {
					required = " (required)"
				}
				builder.WriteString(fmt.Sprintf("    - %s (%s)%s: %s\n", p.Name, p.Type, required, p.Description))
			}
		}
	}

	return builder.String()
}

// ToOpenAITools returns all tools in OpenAI function calling format.
func (r *ToolRegistry) ToOpenAITools() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(r.tools)+len(r.functions))

	for _, tool := range r.tools {
		// Skip expandable tools - only include leaf tools
		if _, isExpandable := tool.(ExpandableTool); !isExpandable {
			tools = append(tools, tool.ToOpenAISchema())
		}
	}

	for _, fn := range r.functions {
		tools = append(tools, fn.ToOpenAISchema())
	}

	return tools
}

// ToOpenAIToolsJSON returns all tools in OpenAI function calling format as JSON.
func (r *ToolRegistry) ToOpenAIToolsJSON() (string, error) {
	tools := r.ToOpenAITools()
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tools to JSON: %w", err)
	}
	return string(data), nil
}

// HasTool checks if a tool with the given name exists.
func (r *ToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	if !exists {
		_, exists = r.functions[name]
	}
	return exists
}

// Clear removes all tools and functions from the registry.
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
	r.functions = make(map[string]*ToolFunction)
}

// Count returns the total number of registered tools and functions.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools) + len(r.functions)
}

// Merge merges another registry into this one.
// If there are conflicts, the tools from the other registry take precedence.
func (r *ToolRegistry) Merge(other *ToolRegistry) {
	r.mu.Lock()
	other.mu.RLock()
	defer r.mu.Unlock()
	defer other.mu.RUnlock()

	for name, tool := range other.tools {
		r.tools[name] = tool
	}

	for name, fn := range other.functions {
		r.functions[name] = fn
	}
}

// Clone creates a deep copy of the registry.
func (r *ToolRegistry) Clone() *ToolRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	newRegistry := NewToolRegistry()

	// Copy tools
	for name, tool := range r.tools {
		newRegistry.tools[name] = tool
	}

	// Copy functions
	for name, fn := range r.functions {
		newRegistry.functions[name] = fn
	}

	return newRegistry
}
