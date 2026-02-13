package memory

import (
	"context"
	"fmt"
	"time"
)

// MemoryType represents the type of memory.
type MemoryType string

const (
	// WorkingMemory is for short-term, task-specific information.
	WorkingMemory MemoryType = "working"
	// EpisodicMemory is for specific events and experiences.
	EpisodicMemory MemoryType = "episodic"
	// SemanticMemory is for general knowledge and facts.
	SemanticMemory MemoryType = "semantic"
	// PerceptualMemory is for sensory information.
	PerceptualMemory MemoryType = "perceptual"
)

// Memory represents a single memory entry.
type Memory struct {
	ID          string                 `json:"id"`
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	Importance  float64                `json:"importance"` // 0.0 to 1.0
	Timestamp   time.Time              `json:"timestamp"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	Tags        []string               `json:"tags"`
}

// MemorySearchParams represents parameters for memory search.
type MemorySearchParams struct {
	Query      string
	Type       MemoryType
	Tags       []string
	MinImportance float64
	Limit      int
	Since      time.Time
	Until      time.Time
}

// MemoryUpdate represents parameters for updating a memory.
type MemoryUpdate struct {
	Content     *string
	Metadata    map[string]interface{}
	Importance  *float64
	AddTags     []string
	RemoveTags  []string
}

// Memory is the interface that all memory implementations must implement.
type Memory interface {
	// Add adds a new memory entry.
	Add(ctx context.Context, memory *Memory) error

	// Get retrieves a memory by ID.
	Get(ctx context.Context, id string) (*Memory, error)

	// Search searches for memories matching the given parameters.
	Search(ctx context.Context, params MemorySearchParams) ([]*Memory, error)

	// Update updates an existing memory.
	Update(ctx context.Context, id string, update *MemoryUpdate) error

	// Delete removes a memory by ID.
	Delete(ctx context.Context, id string) error

	// List lists all memories of a given type.
	List(ctx context.Context, memType MemoryType) ([]*Memory, error)

	// Clear removes all memories of a given type.
	Clear(ctx context.Context, memType MemoryType) error

	// Count returns the number of memories of a given type.
	Count(ctx context.Context, memType MemoryType) (int, error)

	// ForgetAll removes all memories.
	ForgetAll(ctx context.Context) error

	// Consolidate merges and organizes memories.
	Consolidate(ctx context.Context) error

	// Prune removes old or unimportant memories.
	Prune(ctx context.Context, maxMemories int, minImportance float64) error
}

// BaseMemory provides a default implementation of the Memory interface.
// Other memory types can embed this struct for common functionality.
type BaseMemory struct {
	memType  MemoryType
	memories map[string]*Memory
}

// NewBaseMemory creates a new BaseMemory.
func NewBaseMemory(memType MemoryType) *BaseMemory {
	return &BaseMemory{
		memType:  memType,
		memories: make(map[string]*Memory),
	}
}

// Add adds a new memory entry.
func (bm *BaseMemory) Add(ctx context.Context, memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateID()
	}
	if memory.Timestamp.IsZero() {
		memory.Timestamp = time.Now()
	}
	if memory.Importance == 0 {
		memory.Importance = 0.5 // Default importance
	}
	if memory.LastAccess.IsZero() {
		memory.LastAccess = time.Now()
	}
	if memory.Tags == nil {
		memory.Tags = []string{}
	}
	if memory.Metadata == nil {
		memory.Metadata = make(map[string]interface{})
	}

	bm.memories[memory.ID] = memory
	return nil
}

// Get retrieves a memory by ID.
func (bm *BaseMemory) Get(ctx context.Context, id string) (*Memory, error) {
	memory, exists := bm.memories[id]
	if !exists {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	// Update access statistics
	memory.AccessCount++
	memory.LastAccess = time.Now()

	return memory, nil
}

// Search searches for memories matching the given parameters.
func (bm *BaseMemory) Search(ctx context.Context, params MemorySearchParams) ([]*Memory, error) {
	results := make([]*Memory, 0)

	for _, memory := range bm.memories {
		// Filter by type if specified
		if params.Type != "" && memory.Type != params.Type {
			continue
		}

		// Filter by query
		if params.Query != "" {
			if !containsSubstring(memory.Content, params.Query) {
				continue
			}
		}

		// Filter by tags
		if len(params.Tags) > 0 {
			if !containsAllTags(memory.Tags, params.Tags) {
				continue
			}
		}

		// Filter by importance
		if params.MinImportance > 0 && memory.Importance < params.MinImportance {
			continue
		}

		// Filter by time range
		if !params.Since.IsZero() && memory.Timestamp.Before(params.Since) {
			continue
		}
		if !params.Until.IsZero() && memory.Timestamp.After(params.Until) {
			continue
		}

		results = append(results, memory)

		// Apply limit
		if params.Limit > 0 && len(results) >= params.Limit {
			break
		}
	}

	return results, nil
}

// Update updates an existing memory.
func (bm *BaseMemory) Update(ctx context.Context, id string, update *MemoryUpdate) error {
	memory, exists := bm.memories[id]
	if !exists {
		return fmt.Errorf("memory not found: %s", id)
	}

	if update.Content != nil {
		memory.Content = *update.Content
	}

	if update.Metadata != nil {
		for k, v := range update.Metadata {
			memory.Metadata[k] = v
		}
	}

	if update.Importance != nil {
		memory.Importance = *update.Importance
	}

	if len(update.AddTags) > 0 {
		memory.Tags = append(memory.Tags, update.AddTags...)
		memory.Tags = uniqueTags(memory.Tags)
	}

	if len(update.RemoveTags) > 0 {
		memory.Tags = removeTags(memory.Tags, update.RemoveTags)
	}

	return nil
}

// Delete removes a memory by ID.
func (bm *BaseMemory) Delete(ctx context.Context, id string) error {
	if _, exists := bm.memories[id]; !exists {
		return fmt.Errorf("memory not found: %s", id)
	}

	delete(bm.memories, id)
	return nil
}

// List lists all memories of a given type.
func (bm *BaseMemory) List(ctx context.Context, memType MemoryType) ([]*Memory, error) {
	results := make([]*Memory, 0)

	for _, memory := range bm.memories {
		if memory.Type == memType {
			results = append(results, memory)
		}
	}

	return results, nil
}

// Clear removes all memories of a given type.
func (bm *BaseMemory) Clear(ctx context.Context, memType MemoryType) error {
	for id, memory := range bm.memories {
		if memory.Type == memType {
			delete(bm.memories, id)
		}
	}

	return nil
}

// Count returns the number of memories of a given type.
func (bm *BaseMemory) Count(ctx context.Context, memType MemoryType) (int, error) {
	count := 0

	for _, memory := range bm.memories {
		if memory.Type == memType {
			count++
		}
	}

	return count, nil
}

// ForgetAll removes all memories.
func (bm *BaseMemory) ForgetAll(ctx context.Context) error {
	bm.memories = make(map[string]*Memory)
	return nil
}

// Consolidate merges and organizes memories.
func (bm *BaseMemory) Consolidate(ctx context.Context) error {
	// Simple implementation: remove duplicate memories
	seen := make(map[string]bool)
	for id, memory := range bm.memories {
		key := memory.Content
		if seen[key] {
			delete(bm.memories, id)
		} else {
			seen[key] = true
		}
	}

	return nil
}

// Prune removes old or unimportant memories.
func (bm *BaseMemory) Prune(ctx context.Context, maxMemories int, minImportance float64) error {
	if len(bm.memories) <= maxMemories {
		return nil
	}

	// Sort memories by importance and recency
	type memScore struct {
		id    string
		score float64
	}

	scores := make([]memScore, 0, len(bm.memories))
	now := time.Now()

	for id, memory := range bm.memories {
		// Calculate score based on importance and recency
		daysSinceAccess := now.Sub(memory.LastAccess).Hours() / 24
		recencyScore := 1.0 / (1.0 + daysSinceAccess/30.0) // Decay over 30 days
		score := memory.Importance*0.7 + recencyScore*0.3

		scores = append(scores, memScore{id: id, score: score})
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Keep only top maxMemories
	for i := maxMemories; i < len(scores); i++ {
		if scores[i].score < minImportance {
			delete(bm.memories, scores[i].id)
		}
	}

	return nil
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		len(substr) == 0 ||
		containsSubstringRecursive(str, substr))
}

func containsSubstringRecursive(str, substr string) bool {
	if len(str) < len(substr) {
		return false
	}
	if str[:len(substr)] == substr {
		return true
	}
	return containsSubstringRecursive(str[1:], substr)
}

func containsAllTags(tags []string, required []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, req := range required {
		if !tagSet[req] {
			return false
		}
	}

	return true
}

func uniqueTags(tags []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	return result
}

func removeTags(tags []string, toRemove []string) []string {
	removeSet := make(map[string]bool)
	for _, tag := range toRemove {
		removeSet[tag] = true
	}

	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if !removeSet[tag] {
			result = append(result, tag)
		}
	}

	return result
}
