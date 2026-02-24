package context

import "sync"

import tiktoken "github.com/pkoukk/tiktoken-go"

// TokenCounter is a lightweight estimator with caching.
type TokenCounter[T any] struct {
	Model           string
	messageToTextFn func(T) string
	messageToKeyFn  func(T) string
	encoding        *tiktoken.Tiktoken

	mu        sync.RWMutex
	textCache map[string]int
}

func NewTokenCounter[T any](model string, messageToTextFn func(T) string, messageToKeyFn func(T) string) *TokenCounter[T] {
	if messageToTextFn == nil {
		messageToTextFn = func(_ T) string { return "" }
	}
	if messageToKeyFn == nil {
		messageToKeyFn = func(item T) string { return messageToTextFn(item) }
	}

	encoding, err := tiktoken.EncodingForModel(model)
	if err != nil {
		encoding, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			encoding = nil
		}
	}

	return &TokenCounter[T]{
		Model:           model,
		messageToTextFn: messageToTextFn,
		messageToKeyFn:  messageToKeyFn,
		encoding:        encoding,
		textCache:       map[string]int{},
	}
}

func (c *TokenCounter[T]) CountMessages(messages []T) int {
	total := 0
	for _, m := range messages {
		total += c.CountMessage(m)
	}
	return total
}

func (c *TokenCounter[T]) CountMessage(message T) int {
	cacheKey := c.messageToKeyFn(message)

	c.mu.RLock()
	if n, ok := c.textCache[cacheKey]; ok {
		c.mu.RUnlock()
		return n
	}
	c.mu.RUnlock()

	tokens := c._countText(c.messageToTextFn(message))
	tokens += 4 // Role marker overhead, aligned with Python.

	c.mu.Lock()
	c.textCache[cacheKey] = tokens
	c.mu.Unlock()
	return tokens
}

func (c *TokenCounter[T]) CountText(text string) int {
	return c._countText(text)
}

func (c *TokenCounter[T]) _countText(text string) int {
	if c.encoding != nil {
		return len(c.encoding.Encode(text, nil, nil))
	}
	// Fallback estimate: one token ~= 4 chars.
	return len([]rune(text)) / 4
}

func (c *TokenCounter[T]) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.textCache = map[string]int{}
}

func (c *TokenCounter[T]) GetCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.textCache)
}

func (c *TokenCounter[T]) GetCacheStats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalTokens := 0
	for _, v := range c.textCache {
		totalTokens += v
	}
	return map[string]int{
		"cached_messages":     len(c.textCache),
		"total_cached_tokens": totalTokens,
	}
}
