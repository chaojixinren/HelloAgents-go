package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"helloagents-go/HelloAgents-go/memory"
	"helloagents-go/HelloAgents-go/tools"
)

// MemoryTool is an expandable tool that provides memory operations.
// It automatically expands into multiple sub-tools for different memory operations.
type MemoryTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

// NewMemoryTool creates a new MemoryTool with the given memory manager.
func NewMemoryTool(manager *memory.MemoryManager) *MemoryTool {
	return &MemoryTool{
		BaseTool: tools.NewBaseTool(
			"memory",
			"A tool for managing memories across different types (working, episodic, semantic, perceptual). "+
				"Supports adding, searching, updating, and removing memories.",
			[]tools.ToolParameter{},
		),
		manager: manager,
	}
}

// GetExpandedTools returns the expanded sub-tools.
func (mt *MemoryTool) GetExpandedTools() []tools.Tool {
	return []tools.Tool{
		&memoryAddTool{manager: mt.manager},
		&memorySearchTool{manager: mt.manager},
		&memorySummaryTool{manager: mt.manager},
		&memoryStatsTool{manager: mt.manager},
		&memoryUpdateTool{manager: mt.manager},
		&memoryRemoveTool{manager: mt.manager},
		&memoryForgetAllTool{manager: mt.manager},
		&memoryConsolidateTool{manager: mt.manager},
	}
}

// memoryAddTool adds memories.
type memoryAddTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mat *memoryAddTool) Name() string {
	return "memory_add"
}

func (mat *memoryAddTool) Description() string {
	return "Add a new memory. Supports working, episodic, semantic, and perceptual memory types."
}

func (mat *memoryAddTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "type",
			Type:        "string",
			Description: "Memory type: working, episodic, semantic, or perceptual",
			Required:    true,
			Enum:        []string{"working", "episodic", "semantic", "perceptual"},
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "The content of the memory",
			Required:    true,
		},
		{
			Name:        "importance",
			Type:        "number",
			Description: "Importance score from 0.0 to 1.0 (default: 0.5)",
			Required:    false,
		},
		{
			Name:        "metadata",
			Type:        "string",
			Description: "Optional JSON string with additional metadata",
			Required:    false,
		},
		{
			Name:        "tags",
			Type:        "string",
			Description: "Comma-separated list of tags",
			Required:    false,
		},
	}
}

func (mat *memoryAddTool) Run(parameters map[string]interface{}) (string, error) {
	memType := memory.MemoryType(parameters["type"].(string))
	content := parameters["content"].(string)

	importance := 0.5
	if imp, ok := parameters["importance"].(float64); ok {
		importance = imp
	}

	var metadata map[string]interface{}
	if metadataStr, ok := parameters["metadata"].(string); ok && metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return "", fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	var tags []string
	if tagsStr, ok := parameters["tags"].(string); ok && tagsStr != "" {
		// Simple comma-separated parsing
		// In production, use a proper CSV parser
		tags = splitTags(tagsStr)
	}

	id, err := mat.manager.Add(context.Background(), memType, content, importance, metadata, tags)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Memory added successfully with ID: %s", id), nil
}

func (mat *memoryAddTool) Validate(parameters map[string]interface{}) bool {
	if _, ok := parameters["type"]; !ok {
		return false
	}
	if _, ok := parameters["content"]; !ok {
		return false
	}
	return true
}

func (mat *memoryAddTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mat.Name(), ToolDescription: mat.Description(), Parameters: mat.GetParameters()}.ToDict()
}

func (mat *memoryAddTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mat.Name(), ToolDescription: mat.Description(), Parameters: mat.GetParameters()}.ToOpenAISchema()
}

// memorySearchTool searches memories.
type memorySearchTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mst *memorySearchTool) Name() string {
	return "memory_search"
}

func (mst *memorySearchTool) Description() string {
	return "Search for memories by query, type, tags, or importance threshold"
}

func (mst *memorySearchTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "Search query string",
			Required:    false,
		},
		{
			Name:        "type",
			Type:        "string",
			Description: "Memory type filter (optional)",
			Required:    false,
			Enum:        []string{"working", "episodic", "semantic", "perceptual"},
		},
		{
			Name:        "tags",
			Type:        "string",
			Description: "Comma-separated list of tags to filter by",
			Required:    false,
		},
		{
			Name:        "min_importance",
			Type:        "number",
			Description: "Minimum importance threshold (0.0 to 1.0)",
			Required:    false,
		},
		{
			Name:        "limit",
			Type:        "integer",
			Description: "Maximum number of results to return",
			Required:    false,
		},
	}
}

func (mst *memorySearchTool) Run(parameters map[string]interface{}) (string, error) {
	params := memory.MemorySearchParams{}

	if query, ok := parameters["query"].(string); ok {
		params.Query = query
	}

	if typeStr, ok := parameters["type"].(string); ok {
		params.Type = memory.MemoryType(typeStr)
	}

	if tagsStr, ok := parameters["tags"].(string); ok && tagsStr != "" {
		params.Tags = splitTags(tagsStr)
	}

	if minImp, ok := parameters["min_importance"].(float64); ok {
		params.MinImportance = minImp
	}

	if limit, ok := parameters["limit"].(float64); ok {
		params.Limit = int(limit)
	}

	results, err := mst.manager.Search(context.Background(), params)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No memories found matching the criteria", nil
	}

	output := fmt.Sprintf("Found %d memories:\n", len(results))
	for i, mem := range results {
		output += fmt.Sprintf("\n%d. [%s] %s (Importance: %.2f)", i+1, mem.Type, mem.Content, mem.Importance)
		if len(mem.Tags) > 0 {
			output += fmt.Sprintf(" Tags: %v", mem.Tags)
		}
	}

	return output, nil
}

func (mst *memorySearchTool) Validate(parameters map[string]interface{}) bool {
	return true // All parameters optional
}

func (mst *memorySearchTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToDict()
}

func (mst *memorySearchTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToOpenAISchema()
}

// memorySummaryTool provides a summary of memories.
type memorySummaryTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mst *memorySummaryTool) Name() string {
	return "memory_summary"
}

func (mst *memorySummaryTool) Description() string {
	return "Get a summary of all memories by type"
}

func (mst *memorySummaryTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{}
}

func (mst *memorySummaryTool) Run(parameters map[string]interface{}) (string, error) {
	return mst.manager.GetSummary(context.Background())
}

func (mst *memorySummaryTool) Validate(parameters map[string]interface{}) bool {
	return true
}

func (mst *memorySummaryTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToDict()
}

func (mst *memorySummaryTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToOpenAISchema()
}

// memoryStatsTool provides memory statistics.
type memoryStatsTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mst *memoryStatsTool) Name() string {
	return "memory_stats"
}

func (mst *memoryStatsTool) Description() string {
	return "Get detailed statistics about memory usage"
}

func (mst *memoryStatsTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{}
}

func (mst *memoryStatsTool) Run(parameters map[string]interface{}) (string, error) {
	stats, err := mst.manager.GetStats(context.Background())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Memory Statistics:\n- Working: %d\n- Episodic: %d\n- Semantic: %d\n- Perceptual: %d\n- Total: %d",
		stats[memory.WorkingMemory],
		stats[memory.EpisodicMemory],
		stats[memory.SemanticMemory],
		stats[memory.PerceptualMemory],
		stats[memory.WorkingMemory]+stats[memory.EpisodicMemory]+stats[memory.SemanticMemory]+stats[memory.PerceptualMemory],
	), nil
}

func (mst *memoryStatsTool) Validate(parameters map[string]interface{}) bool {
	return true
}

func (mst *memoryStatsTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToDict()
}

func (mst *memoryStatsTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mst.Name(), ToolDescription: mst.Description(), Parameters: mst.GetParameters()}.ToOpenAISchema()
}

// memoryUpdateTool updates existing memories.
type memoryUpdateTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mut *memoryUpdateTool) Name() string {
	return "memory_update"
}

func (mut *memoryUpdateTool) Description() string {
	return "Update an existing memory by ID"
}

func (mut *memoryUpdateTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "id",
			Type:        "string",
			Description: "Memory ID to update",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "New content (optional)",
			Required:    false,
		},
		{
			Name:        "importance",
			Type:        "number",
			Description: "New importance score (optional)",
			Required:    false,
		},
		{
			Name:        "add_tags",
			Type:        "string",
			Description: "Comma-separated tags to add",
			Required:    false,
		},
		{
			Name:        "remove_tags",
			Type:        "string",
			Description: "Comma-separated tags to remove",
			Required:    false,
		},
	}
}

func (mut *memoryUpdateTool) Run(parameters map[string]interface{}) (string, error) {
	id := parameters["id"].(string)
	update := &memory.MemoryUpdate{}

	if content, ok := parameters["content"].(string); ok {
		update.Content = &content
	}

	if importance, ok := parameters["importance"].(float64); ok {
		update.Importance = &importance
	}

	if addTagsStr, ok := parameters["add_tags"].(string); ok && addTagsStr != "" {
		update.AddTags = splitTags(addTagsStr)
	}

	if removeTagsStr, ok := parameters["remove_tags"].(string); ok && removeTagsStr != "" {
		update.RemoveTags = splitTags(removeTagsStr)
	}

	if err := mut.manager.Update(context.Background(), id, update); err != nil {
		return "", err
	}

	return fmt.Sprintf("Memory %s updated successfully", id), nil
}

func (mut *memoryUpdateTool) Validate(parameters map[string]interface{}) bool {
	_, ok := parameters["id"]
	return ok
}

func (mut *memoryUpdateTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mut.Name(), ToolDescription: mut.Description(), Parameters: mut.GetParameters()}.ToDict()
}

func (mut *memoryUpdateTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mut.Name(), ToolDescription: mut.Description(), Parameters: mut.GetParameters()}.ToOpenAISchema()
}

// memoryRemoveTool removes memories.
type memoryRemoveTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mrt *memoryRemoveTool) Name() string {
	return "memory_remove"
}

func (mrt *memoryRemoveTool) Description() string {
	return "Remove a memory by ID"
}

func (mrt *memoryRemoveTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "id",
			Type:        "string",
			Description: "Memory ID to remove",
			Required:    true,
		},
	}
}

func (mrt *memoryRemoveTool) Run(parameters map[string]interface{}) (string, error) {
	id := parameters["id"].(string)

	if err := mrt.manager.Delete(context.Background(), id); err != nil {
		return "", err
	}

	return fmt.Sprintf("Memory %s removed successfully", id), nil
}

func (mrt *memoryRemoveTool) Validate(parameters map[string]interface{}) bool {
	_, ok := parameters["id"]
	return ok
}

func (mrt *memoryRemoveTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mrt.Name(), ToolDescription: mrt.Description(), Parameters: mrt.GetParameters()}.ToDict()
}

func (mrt *memoryRemoveTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mrt.Name(), ToolDescription: mrt.Description(), Parameters: mrt.GetParameters()}.ToOpenAISchema()
}

// memoryForgetAllTool clears all memories.
type memoryForgetAllTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mfat *memoryForgetAllTool) Name() string {
	return "memory_forget_all"
}

func (mfat *memoryForgetAllTool) Description() string {
	return "Clear all memories from all memory types (use with caution)"
}

func (mfat *memoryForgetAllTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "confirm",
			Type:        "string",
			Description: "Type 'yes' to confirm clearing all memories",
			Required:    true,
		},
	}
}

func (mfat *memoryForgetAllTool) Run(parameters map[string]interface{}) (string, error) {
	confirm := parameters["confirm"].(string)

	if confirm != "yes" {
		return "", fmt.Errorf("operation cancelled: confirmation required")
	}

	if err := mfat.manager.ForgetAll(context.Background()); err != nil {
		return "", err
	}

	return "All memories cleared successfully", nil
}

func (mfat *memoryForgetAllTool) Validate(parameters map[string]interface{}) bool {
	confirm, ok := parameters["confirm"]
	return ok && confirm == "yes"
}

func (mfat *memoryForgetAllTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mfat.Name(), ToolDescription: mfat.Description(), Parameters: mfat.GetParameters()}.ToDict()
}

func (mfat *memoryForgetAllTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mfat.Name(), ToolDescription: mfat.Description(), Parameters: mfat.GetParameters()}.ToOpenAISchema()
}

// memoryConsolidateTool consolidates memories.
type memoryConsolidateTool struct {
	*tools.BaseTool
	manager *memory.MemoryManager
}

func (mct *memoryConsolidateTool) Name() string {
	return "memory_consolidate"
}

func (mct *memoryConsolidateTool) Description() string {
	return "Consolidate memories to remove duplicates and improve organization"
}

func (mct *memoryConsolidateTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{}
}

func (mct *memoryConsolidateTool) Run(parameters map[string]interface{}) (string, error) {
	if err := mct.manager.Consolidate(context.Background()); err != nil {
		return "", err
	}

	return "Memories consolidated successfully", nil
}

func (mct *memoryConsolidateTool) Validate(parameters map[string]interface{}) bool {
	return true
}

func (mct *memoryConsolidateTool) ToDict() map[string]interface{} {
	return tools.BaseTool{ToolName: mct.Name(), ToolDescription: mct.Description(), Parameters: mct.GetParameters()}.ToDict()
}

func (mct *memoryConsolidateTool) ToOpenAISchema() map[string]interface{} {
	return tools.BaseTool{ToolName: mct.Name(), ToolDescription: mct.Description(), Parameters: mct.GetParameters()}.ToOpenAISchema()
}

// Helper function to split tags
func splitTags(tagsStr string) []string {
	// Simple implementation - split by comma
	// In production, handle quoted strings properly
	tags := make([]string, 0)
	current := ""
	inQuotes := false

	for _, ch := range tagsStr {
		switch ch {
		case '"', '\'':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				if current != "" {
					tags = append(tags, current)
					current = ""
				}
				continue
			}
			fallthrough
		default:
			current += string(ch)
		}
	}

	if current != "" {
		tags = append(tags, current)
	}

	return tags
}
