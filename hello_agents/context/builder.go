package context

import (
	"math"
	"sort"
	"strings"
	"time"
)

// ConversationMessage is a lightweight history item for ContextBuilder.
// It mirrors Message role/content fields without creating package cycles.
type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ContextPacket struct {
	Content        string         `json:"content"`
	Timestamp      time.Time      `json:"timestamp"`
	Metadata       map[string]any `json:"metadata"`
	TokenCount     int            `json:"token_count"`
	RelevanceScore float64        `json:"relevance_score"`
}

func NewContextPacket(content string, timestamp *time.Time, metadata map[string]any, tokenCount int) ContextPacket {
	ts := time.Now()
	if timestamp != nil {
		ts = *timestamp
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if tokenCount <= 0 {
		tokenCount = countTokens(content)
	}
	return ContextPacket{
		Content:        content,
		Timestamp:      ts,
		Metadata:       metadata,
		TokenCount:     tokenCount,
		RelevanceScore: 0.0,
	}
}

type ContextConfig struct {
	MaxTokens            int     `json:"max_tokens"`
	ReserveRatio         float64 `json:"reserve_ratio"`
	MinRelevance         float64 `json:"min_relevance"`
	EnableMMR            bool    `json:"enable_mmr"`
	MMRLambda            float64 `json:"mmr_lambda"`
	SystemPromptTemplate string  `json:"system_prompt_template"`
	EnableCompression    bool    `json:"enable_compression"`
}

func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		MaxTokens:            8000,
		ReserveRatio:         0.15,
		MinRelevance:         0.3,
		EnableMMR:            true,
		MMRLambda:            0.7,
		SystemPromptTemplate: "",
		EnableCompression:    true,
	}
}

func (c ContextConfig) GetAvailableTokens() int {
	return int(float64(c.MaxTokens) * (1.0 - c.ReserveRatio))
}

type ContextBuilder struct {
	Config ContextConfig
}

func NewContextBuilder(config ContextConfig) *ContextBuilder {
	if config == (ContextConfig{}) {
		config = DefaultContextConfig()
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 8000
	}
	if config.ReserveRatio <= 0 || config.ReserveRatio >= 1 {
		config.ReserveRatio = 0.15
	}
	if config.MinRelevance < 0 {
		config.MinRelevance = 0.3
	}
	if config.MMRLambda <= 0 || config.MMRLambda > 1 {
		config.MMRLambda = 0.7
	}
	return &ContextBuilder{Config: config}
}

// Build mirrors Python ContextBuilder.build().
func (b *ContextBuilder) Build(
	userQuery string,
	conversationHistory []ConversationMessage,
	systemInstructions string,
	additionalPackets []ContextPacket,
) string {
	packets := b.gather(userQuery, conversationHistory, systemInstructions, additionalPackets)
	selected := b.selectPackets(packets, userQuery)
	structured := b.structure(selected, userQuery, systemInstructions)
	return b.compress(structured)
}

func (b *ContextBuilder) gather(
	_ string,
	conversationHistory []ConversationMessage,
	systemInstructions string,
	additionalPackets []ContextPacket,
) []ContextPacket {
	packets := make([]ContextPacket, 0, len(additionalPackets)+2)

	if systemInstructions != "" {
		packets = append(packets, NewContextPacket(
			systemInstructions,
			nil,
			map[string]any{"type": "instructions"},
			0,
		))
	}

	if len(conversationHistory) > 0 {
		start := 0
		if len(conversationHistory) > 10 {
			start = len(conversationHistory) - 10
		}

		lines := make([]string, 0, len(conversationHistory)-start)
		for _, msg := range conversationHistory[start:] {
			lines = append(lines, "["+msg.Role+"] "+msg.Content)
		}

		packets = append(packets, NewContextPacket(
			strings.Join(lines, "\n"),
			nil,
			map[string]any{
				"type":  "history",
				"count": len(lines),
			},
			0,
		))
	}

	for _, packet := range additionalPackets {
		if packet.Timestamp.IsZero() {
			packet.Timestamp = time.Now()
		}
		if packet.Metadata == nil {
			packet.Metadata = map[string]any{}
		}
		if packet.TokenCount <= 0 {
			packet.TokenCount = countTokens(packet.Content)
		}
		packets = append(packets, packet)
	}

	return packets
}

func (b *ContextBuilder) selectPackets(packets []ContextPacket, userQuery string) []ContextPacket {
	queryTokens := splitLowerTokens(userQuery)
	now := time.Now()

	type scoredPacket struct {
		score  float64
		packet ContextPacket
	}

	scored := make([]scoredPacket, 0, len(packets))
	for _, packet := range packets {
		contentTokens := splitLowerTokens(packet.Content)
		relevance := 0.0
		if len(queryTokens) > 0 {
			overlap := 0
			for token := range queryTokens {
				if _, ok := contentTokens[token]; ok {
					overlap++
				}
			}
			relevance = float64(overlap) / float64(len(queryTokens))
		}
		packet.RelevanceScore = relevance

		delta := now.Sub(packet.Timestamp).Seconds()
		if delta < 0 {
			delta = 0
		}
		recency := math.Exp(-delta / 3600.0)
		score := 0.7*packet.RelevanceScore + 0.3*recency
		scored = append(scored, scoredPacket{score: score, packet: packet})
	}

	systemPackets := make([]ContextPacket, 0)
	remaining := make([]scoredPacket, 0)
	for _, item := range scored {
		if item.packet.Metadata["type"] == "instructions" {
			systemPackets = append(systemPackets, item.packet)
		} else {
			remaining = append(remaining, item)
		}
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].score > remaining[j].score
	})

	filtered := make([]ContextPacket, 0, len(remaining))
	for _, item := range remaining {
		if item.packet.RelevanceScore >= b.Config.MinRelevance {
			filtered = append(filtered, item.packet)
		}
	}

	availableTokens := b.Config.GetAvailableTokens()
	selected := make([]ContextPacket, 0, len(systemPackets)+len(filtered))
	used := 0

	for _, packet := range systemPackets {
		if used+packet.TokenCount > availableTokens {
			continue
		}
		selected = append(selected, packet)
		used += packet.TokenCount
	}

	for _, packet := range filtered {
		if used+packet.TokenCount > availableTokens {
			continue
		}
		selected = append(selected, packet)
		used += packet.TokenCount
	}

	return selected
}

func (b *ContextBuilder) structure(
	selectedPackets []ContextPacket,
	userQuery string,
	_ string,
) string {
	sections := make([]string, 0, 6)

	rolePackets := packetsByType(selectedPackets, "instructions")
	if len(rolePackets) > 0 {
		roleLines := make([]string, 0, len(rolePackets))
		for _, packet := range rolePackets {
			roleLines = append(roleLines, packet.Content)
		}
		sections = append(sections, "[Role & Policies]\n"+strings.Join(roleLines, "\n"))
	}

	sections = append(sections, "[Task]\n用户问题："+userQuery)

	statePackets := packetsByType(selectedPackets, "task_state")
	if len(statePackets) > 0 {
		stateLines := make([]string, 0, len(statePackets))
		for _, packet := range statePackets {
			stateLines = append(stateLines, packet.Content)
		}
		sections = append(sections, "[State]\n关键进展与未决问题：\n"+strings.Join(stateLines, "\n"))
	}

	evidencePackets := make([]ContextPacket, 0)
	for _, packet := range selectedPackets {
		packetType, _ := packet.Metadata["type"].(string)
		switch packetType {
		case "related_memory", "knowledge_base", "retrieval", "tool_result":
			evidencePackets = append(evidencePackets, packet)
		}
	}
	if len(evidencePackets) > 0 {
		var builder strings.Builder
		builder.WriteString("[Evidence]\n事实与引用：\n")
		for _, packet := range evidencePackets {
			builder.WriteString("\n")
			builder.WriteString(packet.Content)
			builder.WriteString("\n")
		}
		sections = append(sections, builder.String())
	}

	contextPackets := packetsByType(selectedPackets, "history")
	if len(contextPackets) > 0 {
		lines := make([]string, 0, len(contextPackets))
		for _, packet := range contextPackets {
			lines = append(lines, packet.Content)
		}
		sections = append(sections, "[Context]\n对话历史与背景：\n"+strings.Join(lines, "\n"))
	}

	outputSection := `[Output]
                            请按以下格式回答：
                            1. 结论（简洁明确）
                            2. 依据（列出支撑证据及来源）
                            3. 风险与假设（如有）
                            4. 下一步行动建议（如适用）`
	sections = append(sections, outputSection)

	return strings.Join(sections, "\n\n")
}

func (b *ContextBuilder) compress(context string) string {
	if !b.Config.EnableCompression {
		return context
	}
	currentTokens := countTokens(context)
	availableTokens := b.Config.GetAvailableTokens()
	if currentTokens <= availableTokens {
		return context
	}

	lines := strings.Split(context, "\n")
	compressed := make([]string, 0, len(lines))
	used := 0
	for _, line := range lines {
		lineTokens := countTokens(line)
		if used+lineTokens > availableTokens {
			break
		}
		compressed = append(compressed, line)
		used += lineTokens
	}
	return strings.Join(compressed, "\n")
}

func packetsByType(packets []ContextPacket, packetType string) []ContextPacket {
	out := make([]ContextPacket, 0)
	for _, packet := range packets {
		t, _ := packet.Metadata["type"].(string)
		if t == packetType {
			out = append(out, packet)
		}
	}
	return out
}

func splitLowerTokens(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range strings.Fields(strings.ToLower(text)) {
		out[token] = struct{}{}
	}
	return out
}

func countTokens(text string) int {
	return len([]rune(text)) / 4
}
