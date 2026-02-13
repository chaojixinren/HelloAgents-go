package types

import (
	"context"
	"time"

	"helloagents-go/HelloAgents-go/memory"
)

// PerceptualMemory implements memory for sensory information.
// It stores data from various sensory modalities (visual, auditory, etc.).
type PerceptualMemory struct {
	*memory.BaseMemory
}

// NewPerceptualMemory creates a new PerceptualMemory.
func NewPerceptualMemory() *PerceptualMemory {
	return &PerceptualMemory{
		BaseMemory: memory.NewBaseMemory(memory.PerceptualMemory),
	}
}

// AddPerception adds a perceptual memory with modality information.
func (pm *PerceptualMemory) AddPerception(ctx context.Context, data string, modality string, confidence float64) (string, error) {
	mem := &memory.Memory{
		Type:       memory.PerceptualMemory,
		Content:    data,
		Importance: confidence, // Use confidence as importance
		Metadata: map[string]interface{}{
			"modality":  modality,
			"confidence": confidence,
		},
		Tags: []string{"perception", modality},
	}

	if err := pm.Add(ctx, mem); err != nil {
		return "", err
	}

	return mem.ID, nil
}

// GetPerceptionsByModality retrieves perceptions from a specific modality.
func (pm *PerceptualMemory) GetPerceptionsByModality(ctx context.Context, modality string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type: memory.PerceptualMemory,
		Tags: []string{"perception", modality},
	}

	return pm.Search(ctx, params)
}

// GetRecentPerceptions retrieves recent perceptions within a time window.
func (pm *PerceptualMemory) GetRecentPerceptions(ctx context.Context, duration time.Duration) ([]*memory.Memory, error) {
	since := time.Now().Add(-duration)

	params := memory.MemorySearchParams{
		Type:  memory.PerceptualMemory,
		Since: since,
		Tags:  []string{"perception"},
	}

	return pm.Search(ctx, params)
}

// GetHighConfidencePerceptions retrieves perceptions with high confidence scores.
func (pm *PerceptualMemory) GetHighConfidencePerceptions(ctx context.Context, minConfidence float64) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type:          memory.PerceptualMemory,
		MinImportance: minConfidence,
		Tags:          []string{"perception"},
	}

	return pm.Search(ctx, params)
}

// GetModalities returns all unique perception modalities.
func (pm *PerceptualMemory) GetModalities(ctx context.Context) ([]string, error) {
	list, err := pm.List(ctx, memory.PerceptualMemory)
	if err != nil {
		return nil, err
	}

	modalities := make(map[string]bool)

	for _, mem := range list {
		if modality, ok := mem.Metadata["modality"].(string); ok {
			modalities[modality] = true
		}
	}

	result := make([]string, 0, len(modalities))
	for modality := range modalities {
		result = append(result, modality)
	}

	return result, nil
}

// SearchPerceptions searches for perceptions matching a query.
func (pm *PerceptualMemory) SearchPerceptions(ctx context.Context, query string, modality string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type:  memory.PerceptualMemory,
		Query: query,
		Tags:  []string{"perception"},
	}

	if modality != "" {
		params.Tags = append(params.Tags, modality)
	}

	return pm.Search(ctx, params)
}

// Consolidate merges similar perceptions and removes low-confidence ones.
func (pm *PerceptualMemory) Consolidate(ctx context.Context) error {
	// Get all perceptions
	list, err := pm.List(ctx, memory.PerceptualMemory)
	if err != nil {
		return err
	}

	// Remove low-confidence perceptions
	for _, mem := range list {
		if confidence, ok := mem.Metadata["confidence"].(float64); ok {
			if confidence < 0.3 {
				// Remove low-confidence perceptions
				pm.Delete(ctx, mem.ID)
			}
		}
	}

	// Group by modality and deduplicate
	modalityGroups := make(map[string]map[string]*memory.Memory)

	for _, mem := range list {
		modality := ""
		if m, ok := mem.Metadata["modality"].(string); ok {
			modality = m
		}

		if _, exists := modalityGroups[modality]; !exists {
			modalityGroups[modality] = make(map[string]*memory.Memory)
		}

		// Check for duplicates
		if existing, exists := modalityGroups[modality][mem.Content]; exists {
			// Keep the one with higher confidence
			existingConfidence, _ := existing.Metadata["confidence"].(float64)
			memConfidence, _ := mem.Metadata["confidence"].(float64)

			if memConfidence > existingConfidence {
				pm.Delete(ctx, existing.ID)
				modalityGroups[modality][mem.Content] = mem
			} else {
				pm.Delete(ctx, mem.ID)
			}
		} else {
			modalityGroups[modality][mem.Content] = mem
		}
	}

	return nil
}
