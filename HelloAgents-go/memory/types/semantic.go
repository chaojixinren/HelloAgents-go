package types

import (
	"context"

	"helloagents-go/HelloAgents-go/memory"
)

// SemanticMemory implements memory for general knowledge and facts.
// It stores information that is not tied to specific events.
type SemanticMemory struct {
	*memory.BaseMemory
}

// NewSemanticMemory creates a new SemanticMemory.
func NewSemanticMemory() *SemanticMemory {
	return &SemanticMemory{
		BaseMemory: memory.NewBaseMemory(memory.SemanticMemory),
	}
}

// AddFact adds a factual memory with categorization.
func (sm *SemanticMemory) AddFact(ctx context.Context, fact string, category string, importance float64) (string, error) {
	mem := &memory.Memory{
		Type:       memory.SemanticMemory,
		Content:    fact,
		Importance: importance,
		Metadata: map[string]interface{}{
			"category": category,
		},
		Tags: []string{"fact", category},
	}

	if err := sm.Add(ctx, mem); err != nil {
		return "", err
	}

	return mem.ID, nil
}

// GetFactsByCategory retrieves facts from a specific category.
func (sm *SemanticMemory) GetFactsByCategory(ctx context.Context, category string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type: memory.SemanticMemory,
		Tags: []string{"fact", category},
	}

	return sm.Search(ctx, params)
}

// GetCategories returns all unique fact categories.
func (sm *SemanticMemory) GetCategories(ctx context.Context) ([]string, error) {
	list, err := sm.List(ctx, memory.SemanticMemory)
	if err != nil {
		return nil, err
	}

	categories := make(map[string]bool)

	for _, mem := range list {
		if category, ok := mem.Metadata["category"].(string); ok {
			categories[category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}

	return result, nil
}

// SearchFacts searches for facts matching a query.
func (sm *SemanticMemory) SearchFacts(ctx context.Context, query string, category string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type:  memory.SemanticMemory,
		Query: query,
		Tags:  []string{"fact"},
	}

	if category != "" {
		params.Tags = append(params.Tags, category)
	}

	return sm.Search(ctx, params)
}

// UpdateFact updates an existing fact.
func (sm *SemanticMemory) UpdateFact(ctx context.Context, id string, newFact string, newCategory string) error {
	update := &memory.MemoryUpdate{
		Content: &newFact,
	}

	if newCategory != "" {
		update.Metadata = map[string]interface{}{
			"category": newCategory,
		}
		update.AddTags = []string{newCategory}
	}

	return sm.Update(ctx, id, update)
}

// Consolidate merges similar facts to avoid redundancy.
func (sm *SemanticMemory) Consolidate(ctx context.Context) error {
	// Get all facts
	list, err := sm.List(ctx, memory.SemanticMemory)
	if err != nil {
		return err
	}

	// Group by category
	categoryGroups := make(map[string][]*memory.Memory)

	for _, mem := range list {
		category := ""
		if c, ok := mem.Metadata["category"].(string); ok {
			category = c
		}

		categoryGroups[category] = append(categoryGroups[category], mem)
	}

	// For each category, merge similar facts
	for category, memories := range categoryGroups {
		if len(memories) <= 1 {
			continue
		}

		// Keep only the most important/clear facts
		// This is a simplified implementation
		seen := make(map[string]bool)

		for _, mem := range memories {
			if seen[mem.Content] {
				// Duplicate fact, remove it
				sm.Delete(ctx, mem.ID)
			} else {
				seen[mem.Content] = true
			}
		}
	}

	return nil
}
