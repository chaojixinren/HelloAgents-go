package context

import (
	"strings"

	"helloagents-go/HelloAgents-go/core"
)

// CompressionStrategy defines how to compress conversation history.
type CompressionStrategy string

const (
	// StrategyNone does not compress history.
	StrategyNone CompressionStrategy = "none"
	// StrategyRecent keeps only recent messages.
	StrategyRecent CompressionStrategy = "recent"
	// StrategySummarized summarizes older messages.
	StrategySummarized CompressionStrategy = "summarized"
	// StrategySlidingWindow uses a sliding window approach.
	StrategySlidingWindow CompressionStrategy = "sliding_window"
)

// ContextBuilder builds context for LLM calls from history and current input.
type ContextBuilder struct {
	maxContextLength     int
	compressionStrategy  CompressionStrategy
	compressionRatio     float64 // For summarized strategy
	slidingWindowSize    int     // For sliding window strategy
}

// NewContextBuilder creates a new ContextBuilder with default settings.
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		maxContextLength:    4000, // Default max tokens
		compressionStrategy: StrategyNone,
		compressionRatio:    0.3, // Keep 30% of original content
		slidingWindowSize:   10,  // Keep last 10 messages
	}
}

// NewContextBuilderWithConfig creates a new ContextBuilder with custom settings.
func NewContextBuilderWithConfig(maxLength int, strategy CompressionStrategy) *ContextBuilder {
	return &ContextBuilder{
		maxContextLength:    maxLength,
		compressionStrategy: strategy,
		compressionRatio:    0.3,
		slidingWindowSize:   10,
	}
}

// BuildContext builds the complete context for an LLM call.
func (b *ContextBuilder) BuildContext(
	systemPrompt string,
	history []core.Message,
	currentInput string,
) []core.ChatMessage {
	messages := make([]core.ChatMessage, 0)

	// Add system prompt
	if systemPrompt != "" {
		messages = append(messages, core.ChatMessage{
			Role:    core.RoleSystem,
			Content: systemPrompt,
		})
	}

	// Compress history if needed
	compressedHistory := b.CompressHistory(history)

	// Add history messages
	for _, msg := range compressedHistory {
		messages = append(messages, msg.ToChatMessage())
	}

	// Add current input
	if currentInput != "" {
		messages = append(messages, core.ChatMessage{
			Role:    core.RoleUser,
			Content: currentInput,
		})
	}

	return messages
}

// CompressHistory compresses the conversation history based on the strategy.
func (b *ContextBuilder) CompressHistory(history []core.Message) []core.Message {
	if b.compressionStrategy == StrategyNone {
		return history
	}

	switch b.compressionStrategy {
	case StrategyRecent:
		return b.compressByRecency(history)
	case StrategySummarized:
		return b.compressBySummarization(history)
	case StrategySlidingWindow:
		return b.compressBySlidingWindow(history)
	default:
		return history
	}
}

// compressByRecency keeps only the most recent messages that fit within max length.
func (b *ContextBuilder) compressByRecency(history []core.Message) []core.Message {
	if b.maxContextLength <= 0 {
		return history
	}

	// Estimate token count (rough approximation: 4 chars per token)
	totalLength := 0
	result := make([]core.Message, 0)

	// Process from newest to oldest
	for i := len(history) - 1; i >= 0; i-- {
		msgLength := len(history[i].Content) / 4
		if totalLength+msgLength > b.maxContextLength {
			break
		}

		result = append([]core.Message{history[i]}, result...)
		totalLength += msgLength
	}

	return result
}

// compressBySummarization summarizes older messages.
func (b *ContextBuilder) compressBySummarization(history []core.Message) []core.Message {
	if len(history) == 0 {
		return history
	}

	// For simplicity, we'll truncate older messages
	// In a real implementation, you would use an LLM to summarize
	result := make([]core.Message, 0)

	for i, msg := range history {
		if i == len(history)-1 {
			// Keep the last message intact
			result = append(result, msg)
		} else {
			// Truncate older messages
			truncated := b.truncateMessage(msg, b.compressionRatio)
			result = append(result, truncated)
		}
	}

	return result
}

// compressBySlidingWindow keeps a fixed number of recent messages.
func (b *ContextBuilder) compressBySlidingWindow(history []core.Message) []core.Message {
	if len(history) <= b.slidingWindowSize {
		return history
	}

	start := len(history) - b.slidingWindowSize
	return history[start:]
}

// truncateMessage truncates a message's content to a ratio of its original length.
func (b *ContextBuilder) truncateMessage(msg core.Message, ratio float64) core.Message {
	targetLength := int(float64(len(msg.Content)) * ratio)

	if targetLength >= len(msg.Content) {
		return msg
	}

	truncated := core.Message{
		Content:   msg.Content[:targetLength] + "...",
		Role:      msg.Role,
		Timestamp: msg.Timestamp,
		Metadata:  msg.Metadata,
	}

	return truncated
}

// EstimateTokenCount estimates the token count of a string.
// This is a rough approximation (4 characters per token).
func EstimateTokenCount(text string) int {
	return len(text) / 4
}

// BuildContextWithInstructions builds context with additional instructions.
func (b *ContextBuilder) BuildContextWithInstructions(
	systemPrompt string,
	history []core.Message,
	currentInput string,
	additionalInstructions string,
) []core.ChatMessage {
	messages := b.BuildContext(systemPrompt, history, currentInput)

	// Prepend additional instructions to the user message
	if additionalInstructions != "" {
		for i := range messages {
			if messages[i].Role == core.RoleUser && messages[i].Content == currentInput {
				messages[i].Content = additionalInstructions + "\n\n" + currentInput
				break
			}
		}
	}

	return messages
}

// BuildContextForReflection builds context optimized for reflection tasks.
func (b *ContextBuilder) BuildContextForReflection(
	systemPrompt string,
	history []core.Message,
	currentInput string,
	previousAttempts []string,
) []core.ChatMessage {
	messages := make([]core.ChatMessage, 0)

	// Add system prompt with reflection context
	reflectionPrompt := systemPrompt
	if len(previousAttempts) > 0 {
		reflectionPrompt += "\n\nPrevious attempts to improve upon:\n" +
			strings.Join(previousAttempts, "\n\n---\n\n")
	}

	messages = append(messages, core.ChatMessage{
		Role:    core.RoleSystem,
		Content: reflectionPrompt,
	})

	// Add compressed history
	compressedHistory := b.CompressHistory(history)
	for _, msg := range compressedHistory {
		messages = append(messages, msg.ToChatMessage())
	}

	// Add current input
	messages = append(messages, core.ChatMessage{
		Role:    core.RoleUser,
		Content: currentInput,
	})

	return messages
}

// SetMaxContextLength sets the maximum context length in tokens.
func (b *ContextBuilder) SetMaxContextLength(maxLength int) {
	b.maxContextLength = maxLength
}

// GetMaxContextLength returns the maximum context length.
func (b *ContextBuilder) GetMaxContextLength() int {
	return b.maxContextLength
}

// SetCompressionStrategy sets the compression strategy.
func (b *ContextBuilder) SetCompressionStrategy(strategy CompressionStrategy) {
	b.compressionStrategy = strategy
}

// GetCompressionStrategy returns the compression strategy.
func (b *ContextBuilder) GetCompressionStrategy() CompressionStrategy {
	return b.compressionStrategy
}

// SetCompressionRatio sets the compression ratio for summarization.
func (b *ContextBuilder) SetCompressionRatio(ratio float64) {
	b.compressionRatio = ratio
}

// GetCompressionRatio returns the compression ratio.
func (b *ContextBuilder) GetCompressionRatio() float64 {
	return b.compressionRatio
}

// SetSlidingWindowSize sets the sliding window size.
func (b *ContextBuilder) SetSlidingWindowSize(size int) {
	b.slidingWindowSize = size
}

// GetSlidingWindowSize returns the sliding window size.
func (b *ContextBuilder) GetSlidingWindowSize() int {
	return b.slidingWindowSize
}
