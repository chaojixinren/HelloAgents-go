package context

// HistoryManager manages append-only message history with round-aware compression.
type HistoryManager[T any] struct {
	history              []T
	MinRetainRounds      int
	CompressionThreshold float64
	summaryFactory       func(summary string) T
	roleExtractor        func(item T) string
}

func NewHistoryManager[T any](
	minRetainRounds int,
	compressionThreshold float64,
	summaryFactory func(summary string) T,
	roleExtractor func(item T) string,
) *HistoryManager[T] {
	return &HistoryManager[T]{
		history:              make([]T, 0, 128),
		MinRetainRounds:      minRetainRounds,
		CompressionThreshold: compressionThreshold,
		summaryFactory:       summaryFactory,
		roleExtractor:        roleExtractor,
	}
}

func (h *HistoryManager[T]) Append(message T) {
	h.history = append(h.history, message)
}

func (h *HistoryManager[T]) GetHistory() []T {
	out := make([]T, len(h.history))
	copy(out, h.history)
	return out
}

func (h *HistoryManager[T]) Clear() {
	h.history = h.history[:0]
}

func (h *HistoryManager[T]) EstimateRounds() int {
	if len(h.history) == 0 {
		return 0
	}

	// Mirror Python behavior when we can identify "user" messages.
	if h.roleExtractor != nil {
		rounds := 0
		i := 0
		for i < len(h.history) {
			role := h.roleExtractor(h.history[i])
			if role == "user" {
				rounds++
				i++
				for i < len(h.history) && h.roleExtractor(h.history[i]) != "user" {
					i++
				}
				continue
			}
			i++
		}
		return rounds
	}

	// Fallback for generic types without role.
	rounds := len(h.history) / 2
	if len(h.history)%2 != 0 {
		rounds++
	}
	return rounds
}

func (h *HistoryManager[T]) FindRoundBoundaries() []int {
	if len(h.history) == 0 {
		return []int{}
	}

	// Mirror Python behavior when we can identify "user" messages.
	if h.roleExtractor != nil {
		boundaries := make([]int, 0)
		for i, item := range h.history {
			if h.roleExtractor(item) == "user" {
				boundaries = append(boundaries, i)
			}
		}
		return boundaries
	}

	// Fallback: every two messages starts a round.
	boundaries := []int{0}
	for i := 2; i < len(h.history); i += 2 {
		boundaries = append(boundaries, i)
	}
	return boundaries
}

func (h *HistoryManager[T]) Compress(summary string) {
	if h.summaryFactory == nil || len(h.history) == 0 {
		return
	}

	rounds := h.EstimateRounds()
	if rounds <= h.MinRetainRounds {
		return
	}

	boundaries := h.FindRoundBoundaries()
	if len(boundaries) <= h.MinRetainRounds {
		return
	}

	boundaryIndex := len(boundaries) - h.MinRetainRounds
	if h.MinRetainRounds == 0 {
		// Python list[-0] resolves to list[0], not list[len].
		boundaryIndex = 0
	}
	keepFromIndex := boundaries[boundaryIndex]
	if keepFromIndex < 0 || keepFromIndex > len(h.history) {
		return
	}

	newHistory := make([]T, 0, len(h.history)-keepFromIndex+1)
	newHistory = append(newHistory, h.summaryFactory(summary))
	newHistory = append(newHistory, h.history[keepFromIndex:]...)
	h.history = newHistory
}

func (h *HistoryManager[T]) ToMap(serializer func(T) map[string]any) map[string]any {
	items := make([]map[string]any, 0, len(h.history))
	for _, item := range h.history {
		items = append(items, serializer(item))
	}

	return map[string]any{
		"history":    items,
		"created_at": nowPythonISOTime(),
		"rounds":     h.EstimateRounds(),
	}
}

// ToDict keeps naming parity with Python HistoryManager.to_dict().
func (h *HistoryManager[T]) ToDict(serializer func(T) map[string]any) map[string]any {
	return h.ToMap(serializer)
}

func (h *HistoryManager[T]) LoadFromMap(data map[string]any, parser func(map[string]any) (T, error)) {
	h.Clear()
	rawItems, ok := data["history"].([]any)
	if !ok {
		return
	}

	for _, raw := range rawItems {
		itemMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		item, err := parser(itemMap)
		if err == nil {
			h.Append(item)
		}
	}
}
