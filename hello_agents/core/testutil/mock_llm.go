package testutil

import (
	"encoding/json"
	"fmt"

	"helloagents-go/hello_agents/core"
)

// MockLLMAdapter returns canned responses without HTTP calls.
type MockLLMAdapter struct {
	Responses     []string
	ToolResponses []map[string]any
	InvokedCount  int
	LastMessages  []map[string]any
	StreamChunks  []string
	FailOnInvoke  bool
	FailOnNthCall int
}

func (m *MockLLMAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (core.LLMResponse, error) {
	m.LastMessages = messages
	m.InvokedCount++

	if m.FailOnInvoke || (m.FailOnNthCall > 0 && m.InvokedCount == m.FailOnNthCall) {
		return core.LLMResponse{}, fmt.Errorf("mock LLM error")
	}

	idx := m.InvokedCount - 1
	if idx >= len(m.Responses) {
		idx = len(m.Responses) - 1
	}
	content := ""
	if idx >= 0 && len(m.Responses) > 0 {
		content = m.Responses[idx]
	}

	return core.LLMResponse{
		Content: content,
		Model:   "mock-model",
		Usage: map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		},
		LatencyMS: 1,
	}, nil
}

func (m *MockLLMAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	m.LastMessages = messages
	m.InvokedCount++

	out := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		if m.FailOnInvoke {
			errCh <- fmt.Errorf("mock LLM stream error")
			return
		}

		if len(m.StreamChunks) > 0 {
			for _, chunk := range m.StreamChunks {
				out <- chunk
			}
			return
		}

		idx := m.InvokedCount - 1
		if idx >= len(m.Responses) {
			idx = len(m.Responses) - 1
		}
		if idx >= 0 && len(m.Responses) > 0 {
			out <- m.Responses[idx]
		}
	}()

	return out, errCh
}

func (m *MockLLMAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	m.LastMessages = messages
	m.InvokedCount++

	if m.FailOnInvoke || (m.FailOnNthCall > 0 && m.InvokedCount == m.FailOnNthCall) {
		return nil, fmt.Errorf("mock LLM tool error")
	}

	if m.InvokedCount-1 < len(m.ToolResponses) {
		return m.ToolResponses[m.InvokedCount-1], nil
	}

	idx := m.InvokedCount - 1
	if idx >= len(m.Responses) {
		idx = len(m.Responses) - 1
	}
	content := ""
	if idx >= 0 && len(m.Responses) > 0 {
		content = m.Responses[idx]
	}

	return MockTextResponse(content), nil
}

func (m *MockLLMAdapter) LastStats() *core.StreamStats {
	return nil
}

// NewMockLLM creates a HelloAgentsLLM with a mock adapter for testing.
func NewMockLLM(responses ...string) *core.HelloAgentsLLM {
	adapter := &MockLLMAdapter{Responses: responses}
	return NewMockLLMFromAdapter(adapter)
}

// NewMockLLMFromAdapter creates a HelloAgentsLLM with a given mock adapter.
func NewMockLLMFromAdapter(adapter *MockLLMAdapter) *core.HelloAgentsLLM {
	return core.NewLLMFromAdapter("mock-model", "mock-key", "https://mock.example.com/v1", 60, 0.7, adapter)
}

// MockToolCallResponse creates an OpenAI-compatible tool call response.
func MockToolCallResponse(toolName, callID string, args map[string]any) map[string]any {
	argsJSON, _ := json.Marshal(args)
	return map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []any{
						map[string]any{
							"id":   callID,
							"type": "function",
							"function": map[string]any{
								"name":      toolName,
								"arguments": string(argsJSON),
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}
}

// MockTextResponse creates a simple text response payload.
func MockTextResponse(content string) map[string]any {
	return map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	}
}
