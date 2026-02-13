package types

import (
	"context"
	"time"

	"helloagents-go/HelloAgents-go/memory"
)

// EpisodicMemory implements memory for specific events and experiences.
// It stores contextual information about what happened, when, and where.
type EpisodicMemory struct {
	*memory.BaseMemory
}

// NewEpisodicMemory creates a new EpisodicMemory.
func NewEpisodicMemory() *EpisodicMemory {
	return &EpisodicMemory{
		BaseMemory: memory.NewBaseMemory(memory.EpisodicMemory),
	}
}

// AddEpisode adds an episodic memory with contextual information.
func (em *EpisodicMemory) AddEpisode(ctx context.Context, event string, when time.Time, where string, who []string, importance float64) (string, error) {
	mem := &memory.Memory{
		Type:       memory.EpisodicMemory,
		Content:    event,
		Importance: importance,
		Metadata: map[string]interface{}{
			"when": when,
			"where": where,
			"who":   who,
		},
		Timestamp:  time.Now(),
		LastAccess: time.Now(),
		Tags:       []string{"episode"},
	}

	if err := em.Add(ctx, mem); err != nil {
		return "", err
	}

	return mem.ID, nil
}

// GetRecentEpisodes retrieves recent episodes within a time window.
func (em *EpisodicMemory) GetRecentEpisodes(ctx context.Context, duration time.Duration) ([]*memory.Memory, error) {
	since := time.Now().Add(-duration)

	params := memory.MemorySearchParams{
		Type:  memory.EpisodicMemory,
		Since: since,
	}

	return em.Search(ctx, params)
}

// GetEpisodesByLocation retrieves episodes that occurred at a specific location.
func (em *EpisodicMemory) GetEpisodesByLocation(ctx context.Context, location string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type: memory.EpisodicMemory,
		Tags: []string{"episode"},
	}

	results, err := em.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Filter by location
	filtered := make([]*memory.Memory, 0)
	for _, mem := range results {
		if where, ok := mem.Metadata["where"].(string); ok && where == location {
			filtered = append(filtered, mem)
		}
	}

	return filtered, nil
}

// GetEpisodesByWho retrieves episodes involving specific people.
func (em *EpisodicMemory) GetEpisodesByWho(ctx context.Context, people []string) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type: memory.EpisodicMemory,
		Tags: []string{"episode"},
	}

	results, err := em.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Filter by people
	filtered := make([]*memory.Memory, 0)
	for _, mem := range results {
		if who, ok := mem.Metadata["who"].([]string); ok {
			for _, person := range people {
				for _, w := range who {
					if w == person {
						filtered = append(filtered, mem)
						break
					}
				}
			}
		}
	}

	return filtered, nil
}

// GetTimeline retrieves episodes in chronological order.
func (em *EpisodicMemory) GetTimeline(ctx context.Context, limit int) ([]*memory.Memory, error) {
	params := memory.MemorySearchParams{
		Type:  memory.EpisodicMemory,
		Limit: limit,
	}

	results, err := em.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Sort by timestamp
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Timestamp.Before(results[i].Timestamp) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// Consolidate merges similar episodes to avoid redundancy.
func (em *EpisodicMemory) Consolidate(ctx context.Context) error {
	// Get all episodes
	list, err := em.List(ctx, memory.EpisodicMemory)
	if err != nil {
		return err
	}

	// Group by similarity (simplified - groups by location)
	locationGroups := make(map[string][]*memory.Memory)

	for _, mem := range list {
		where := ""
		if w, ok := mem.Metadata["where"].(string); ok {
			where = w
		}

		locationGroups[where] = append(locationGroups[where], mem)
	}

	// For each location, keep only the most important episodes
	for location, memories := range locationGroups {
		if len(memories) <= 1 {
			continue
		}

		// Sort by importance
		for i := 0; i < len(memories); i++ {
			for j := i + 1; j < len(memories); j++ {
				if memories[j].Importance > memories[i].Importance {
					memories[i], memories[j] = memories[j], memories[i]
				}
			}
		}

		// Remove less important memories
		for i := 1; i < len(memories); i++ {
			if memories[i].Importance < 0.5 {
				em.Delete(ctx, memories[i].ID)
			}
		}
	}

	return nil
}
