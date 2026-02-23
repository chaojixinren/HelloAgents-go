package core

import "fmt"

// LLMResponse mirrors Python dataclass with usage and reasoning fields.
type LLMResponse struct {
	Content          string         `json:"content"`
	Model            string         `json:"model"`
	Usage            map[string]int `json:"usage"`
	LatencyMS        int            `json:"latency_ms"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
}

func (r LLMResponse) String() string {
	return r.Content
}

func (r LLMResponse) Repr() string {
	tokens := 0
	if r.Usage != nil {
		tokens = r.Usage["total_tokens"]
	}
	return fmt.Sprintf("LLMResponse(model=%s, latency=%dms, tokens=%d, content_length=%d)", r.Model, r.LatencyMS, tokens, len(r.Content))
}

func (r LLMResponse) ToMap() map[string]any {
	out := map[string]any{
		"content":    r.Content,
		"model":      r.Model,
		"usage":      r.Usage,
		"latency_ms": r.LatencyMS,
	}
	if r.ReasoningContent != "" {
		out["reasoning_content"] = r.ReasoningContent
	}
	return out
}

// ToDict keeps naming parity with Python LLMResponse.to_dict().
func (r LLMResponse) ToDict() map[string]any {
	return r.ToMap()
}

// StreamStats mirrors Python StreamStats for stream mode metrics.
type StreamStats struct {
	Model            string         `json:"model"`
	Usage            map[string]int `json:"usage"`
	LatencyMS        int            `json:"latency_ms"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
}

func (s StreamStats) ToMap() map[string]any {
	out := map[string]any{
		"model":      s.Model,
		"usage":      s.Usage,
		"latency_ms": s.LatencyMS,
	}
	if s.ReasoningContent != "" {
		out["reasoning_content"] = s.ReasoningContent
	}
	return out
}

// ToDict keeps naming parity with Python StreamStats.to_dict().
func (s StreamStats) ToDict() map[string]any {
	return s.ToMap()
}
