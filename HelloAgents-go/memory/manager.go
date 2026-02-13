package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryManager manages multiple memory types.
type MemoryManager struct {
	working    Memory
	episodic   Memory
	semantic   Memory
	perceptual Memory
	mu         sync.RWMutex
}

// NewMemoryManager creates a new MemoryManager with all memory types initialized.
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		working:    NewBaseMemory(WorkingMemory),
		episodic:   NewBaseMemory(EpisodicMemory),
		semantic:   NewBaseMemory(SemanticMemory),
		perceptual: NewBaseMemory(PerceptualMemory),
	}
}

// Add adds a memory to the specified type.
func (m *MemoryManager) Add(ctx context.Context, memType MemoryType, content string, importance float64, metadata map[string]interface{}, tags []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	memory := &Memory{
		Type:       memType,
		Content:    content,
		Importance: importance,
		Metadata:   metadata,
		Tags:       tags,
		Timestamp:  time.Now(),
		LastAccess: time.Now(),
	}

	var memoryStore Memory
	switch memType {
	case WorkingMemory:
		memoryStore = m.working
	case EpisodicMemory:
		memoryStore = m.episodic
	case SemanticMemory:
		memoryStore = m.semantic
	case PerceptualMemory:
		memoryStore = m.perceptual
	default:
		return "", fmt.Errorf("unknown memory type: %s", memType)
	}

	if err := memoryStore.Add(ctx, memory); err != nil {
		return "", fmt.Errorf("failed to add memory: %w", err)
	}

	return memory.ID, nil
}

// Get retrieves a memory by ID from any memory type.
func (m *MemoryManager) Get(ctx context.Context, id string) (*Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try each memory type
	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		memory, err := store.Get(ctx, id)
		if err == nil {
			return memory, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", id)
}

// Search searches for memories across all types or a specific type.
func (m *MemoryManager) Search(ctx context.Context, params MemorySearchParams) ([]*Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if params.Type != "" {
		// Search in specific type
		var memoryStore Memory
		switch params.Type {
		case WorkingMemory:
			memoryStore = m.working
		case EpisodicMemory:
			memoryStore = m.episodic
		case SemanticMemory:
			memoryStore = m.semantic
		case PerceptualMemory:
			memoryStore = m.perceptual
		default:
			return nil, fmt.Errorf("unknown memory type: %s", params.Type)
		}

		return memoryStore.Search(ctx, params)
	}

	// Search across all types
	allResults := make([]*Memory, 0)

	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}
	for _, store := range memories {
		results, err := store.Search(ctx, params)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// Update updates an existing memory.
func (m *MemoryManager) Update(ctx context.Context, id string, update *MemoryUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the memory in any store
	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		if err := store.Update(ctx, id, update); err == nil {
			return nil
		}
	}

	return fmt.Errorf("memory not found: %s", id)
}

// Delete removes a memory by ID.
func (m *MemoryManager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		if err := store.Delete(ctx, id); err == nil {
			return nil
		}
	}

	return fmt.Errorf("memory not found: %s", id)
}

// List lists all memories of a given type.
func (m *MemoryManager) List(ctx context.Context, memType MemoryType) ([]*Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memoryStore Memory
	switch memType {
	case WorkingMemory:
		memoryStore = m.working
	case EpisodicMemory:
		memoryStore = m.episodic
	case SemanticMemory:
		memoryStore = m.semantic
	case PerceptualMemory:
		memoryStore = m.perceptual
	default:
		return nil, fmt.Errorf("unknown memory type: %s", memType)
	}

	return memoryStore.List(ctx, memType)
}

// Clear removes all memories of a given type.
func (m *MemoryManager) Clear(ctx context.Context, memType MemoryType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var memoryStore Memory
	switch memType {
	case WorkingMemory:
		memoryStore = m.working
	case EpisodicMemory:
		memoryStore = m.episodic
	case SemanticMemory:
		memoryStore = m.semantic
	case PerceptualMemory:
		memoryStore = m.perceptual
	default:
		return fmt.Errorf("unknown memory type: %s", memType)
	}

	return memoryStore.Clear(ctx, memType)
}

// Count returns the number of memories of a given type.
func (m *MemoryManager) Count(ctx context.Context, memType MemoryType) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memoryStore Memory
	switch memType {
	case WorkingMemory:
		memoryStore = m.working
	case EpisodicMemory:
		memoryStore = m.episodic
	case SemanticMemory:
		memoryStore = m.semantic
	case PerceptualMemory:
		memoryStore = m.perceptual
	default:
		return 0, fmt.Errorf("unknown memory type: %s", memType)
	}

	return memoryStore.Count(ctx, memType)
}

// ForgetAll removes all memories from all types.
func (m *MemoryManager) ForgetAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		if err := store.ForgetAll(ctx); err != nil {
			return fmt.Errorf("failed to clear memory store: %w", err)
		}
	}

	return nil
}

// Consolidate consolidates memories across all types.
func (m *MemoryManager) Consolidate(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		if err := store.Consolidate(ctx); err != nil {
			return fmt.Errorf("failed to consolidate memory store: %w", err)
		}
	}

	return nil
}

// Prune prunes memories across all types.
func (m *MemoryManager) Prune(ctx context.Context, maxMemories int, minImportance float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memories := []Memory{m.working, m.episodic, m.semantic, m.perceptual}

	for _, store := range memories {
		if err := store.Prune(ctx, maxMemories, minImportance); err != nil {
			return fmt.Errorf("failed to prune memory store: %w", err)
		}
	}

	return nil
}

// GetStats returns statistics about all memory types.
func (m *MemoryManager) GetStats(ctx context.Context) (map[MemoryType]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[MemoryType]int)

	types := []MemoryType{WorkingMemory, EpisodicMemory, SemanticMemory, PerceptualMemory}
	for _, memType := range types {
		count, err := m.Count(ctx, memType)
		if err != nil {
			return nil, err
		}
		stats[memType] = count
	}

	return stats, nil
}

// GetSummary returns a summary of all memories.
func (m *MemoryManager) GetSummary(ctx context.Context) (string, error) {
	stats, err := m.GetStats(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Memory Summary:\n- Working: %d\n- Episodic: %d\n- Semantic: %d\n- Perceptual: %d\n- Total: %d",
		stats[WorkingMemory],
		stats[EpisodicMemory],
		stats[SemanticMemory],
		stats[PerceptualMemory],
		stats[WorkingMemory]+stats[EpisodicMemory]+stats[SemanticMemory]+stats[PerceptualMemory],
	), nil
}

// SetMemoryStore sets a custom memory implementation for a specific type.
func (m *MemoryManager) SetMemoryStore(memType MemoryType, store Memory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch memType {
	case WorkingMemory:
		m.working = store
	case EpisodicMemory:
		m.episodic = store
	case SemanticMemory:
		m.semantic = store
	case PerceptualMemory:
		m.perceptual = store
	default:
		return fmt.Errorf("unknown memory type: %s", memType)
	}

	return nil
}

// GetMemoryStore retrieves the memory store for a specific type.
func (m *MemoryManager) GetMemoryStore(memType MemoryType) (Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch memType {
	case WorkingMemory:
		return m.working, nil
	case EpisodicMemory:
		return m.episodic, nil
	case SemanticMemory:
		return m.semantic, nil
	case PerceptualMemory:
		return m.perceptual, nil
	default:
		return nil, fmt.Errorf("unknown memory type: %s", memType)
	}
}
