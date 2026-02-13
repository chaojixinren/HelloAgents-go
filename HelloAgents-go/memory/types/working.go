package types

import (
	"context"
	"fmt"
	"sync"
	"time"

	"helloagents-go/HelloAgents-go/memory"
)

// WorkingMemory implements short-term, task-specific memory.
// It has a limited capacity and automatically prunes old entries.
type WorkingMemory struct {
	*memory.BaseMemory
	maxSize   int
	mu        sync.RWMutex
	ttl       time.Duration // Time-to-live for memories
}

// NewWorkingMemory creates a new WorkingMemory with default settings.
func NewWorkingMemory() *WorkingMemory {
	return &WorkingMemory{
		BaseMemory: memory.NewBaseMemory(memory.WorkingMemory),
		maxSize:    50,  // Default max 50 items
		ttl:        24 * time.Hour, // Default 24 hour TTL
	}
}

// NewWorkingMemoryWithConfig creates a new WorkingMemory with custom settings.
func NewWorkingMemoryWithConfig(maxSize int, ttl time.Duration) *WorkingMemory {
	return &WorkingMemory{
		BaseMemory: memory.NewBaseMemory(memory.WorkingMemory),
		maxSize:    maxSize,
		ttl:        ttl,
	}
}

// Add adds a memory to working memory, enforcing capacity limits.
func (wm *WorkingMemory) Add(ctx context.Context, mem *memory.Memory) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Set memory type
	mem.Type = memory.WorkingMemory

	// Check capacity and prune if necessary
	count, _ := wm.BaseMemory.Count(ctx, memory.WorkingMemory)
	if count >= wm.maxSize {
		wm.pruneOldest(ctx)
	}

	// Add the memory
	return wm.BaseMemory.Add(ctx, mem)
}

// Get retrieves a memory by ID, checking TTL.
func (wm *WorkingMemory) Get(ctx context.Context, id string) (*memory.Memory, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	mem, err := wm.BaseMemory.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if memory has expired
	if wm.ttl > 0 && time.Since(mem.Timestamp) > wm.ttl {
		return nil, fmt.Errorf("memory expired: %s", id)
	}

	return mem, nil
}

// Search searches for memories, excluding expired ones.
func (wm *WorkingMemory) Search(ctx context.Context, params memory.MemorySearchParams) ([]*memory.Memory, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	// Set type to working memory if not specified
	if params.Type == "" {
		params.Type = memory.WorkingMemory
	}

	results, err := wm.BaseMemory.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Filter out expired memories
	now := time.Now()
	filtered := make([]*memory.Memory, 0)

	for _, mem := range results {
		if wm.ttl == 0 || now.Sub(mem.Timestamp) <= wm.ttl {
			filtered = append(filtered, mem)
		}
	}

	return filtered, nil
}

// Prune removes memories older than the TTL or exceeding capacity.
func (wm *WorkingMemory) Prune(ctx context.Context) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Prune by TTL
	if wm.ttl > 0 {
		list, _ := wm.BaseMemory.List(ctx, memory.WorkingMemory)
		now := time.Now()

		for _, mem := range list {
			if now.Sub(mem.Timestamp) > wm.ttl {
				wm.BaseMemory.Delete(ctx, mem.ID)
			}
		}
	}

	// Prune by capacity
	count, _ := wm.BaseMemory.Count(ctx, memory.WorkingMemory)
	if count > wm.maxSize {
		wm.pruneOldest(ctx)
	}

	return nil
}

// pruneOldest removes the oldest memories to make room.
func (wm *WorkingMemory) pruneOldest(ctx context.Context) {
	list, _ := wm.BaseMemory.List(ctx, memory.WorkingMemory)

	// Sort by timestamp (oldest first)
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Timestamp.Before(list[i].Timestamp) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	// Remove oldest entries
	toRemove := len(list) - wm.maxSize + 1
	for i := 0; i < toRemove && i < len(list); i++ {
		wm.BaseMemory.Delete(ctx, list[i].ID)
	}
}

// SetMaxSize sets the maximum size of the working memory.
func (wm *WorkingMemory) SetMaxSize(size int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.maxSize = size
}

// GetMaxSize returns the maximum size of the working memory.
func (wm *WorkingMemory) GetMaxSize() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.maxSize
}

// SetTTL sets the time-to-live for memories.
func (wm *WorkingMemory) SetTTL(ttl time.Duration) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.ttl = ttl
}

// GetTTL returns the time-to-live for memories.
func (wm *WorkingMemory) GetTTL() time.Duration {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.ttl
}
