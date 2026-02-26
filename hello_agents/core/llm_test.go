package core

import "testing"

type captureStreamAdapter struct {
	lastKwargs map[string]any
}

func (a *captureStreamAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	return LLMResponse{}, nil
}

func (a *captureStreamAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	a.lastKwargs = copyMap(kwargs)
	out := make(chan string)
	errCh := make(chan error)
	close(out)
	close(errCh)
	return out, errCh
}

func (a *captureStreamAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (a *captureStreamAdapter) LastStats() *StreamStats {
	return nil
}

func TestNewHelloAgentsLLMTimeoutZeroFallsBackToEnvLikePythonTruthy(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	t.Setenv("LLM_TIMEOUT", "77")

	timeout := 0
	llm, err := NewHelloAgentsLLM("model-x", "", "", 0.7, nil, &timeout, nil)
	if err != nil {
		t.Fatalf("NewHelloAgentsLLM() error = %v", err)
	}
	if llm.Timeout != 77 {
		t.Fatalf("Timeout = %d, want 77", llm.Timeout)
	}
}

func TestNewHelloAgentsLLMMaxTokensZeroTreatedAsNoneLikePython(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")

	maxTokens := 0
	llm, err := NewHelloAgentsLLM("model-x", "", "", 0.7, &maxTokens, nil, nil)
	if err != nil {
		t.Fatalf("NewHelloAgentsLLM() error = %v", err)
	}
	if llm.MaxTokens != nil {
		t.Fatalf("MaxTokens should be nil when input is 0, got %v", *llm.MaxTokens)
	}
}

func TestNewHelloAgentsLLMReturnsErrorOnInvalidEnvTimeout(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	t.Setenv("LLM_TIMEOUT", "bad-timeout")

	_, err := NewHelloAgentsLLM("model-x", "", "", 0.7, nil, nil, nil)
	if err == nil {
		t.Fatalf("NewHelloAgentsLLM should return error when LLM_TIMEOUT is invalid")
	}
}

func TestNewHelloAgentsLLMIgnoresProviderKwargLikePython(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")

	llm, err := NewHelloAgentsLLM(
		"model-x",
		"",
		"",
		0.7,
		nil,
		nil,
		map[string]any{"provider": "deepseek"},
	)
	if err != nil {
		t.Fatalf("NewHelloAgentsLLM() error = %v", err)
	}
	if llm.Provider != "" {
		t.Fatalf("Provider = %q, want empty like python (provider kwarg is not instance attribute)", llm.Provider)
	}
}

func TestStreamInvokePassesNilTemperatureByDefaultLikePython(t *testing.T) {
	adapter := &captureStreamAdapter{}
	llm := &HelloAgentsLLM{
		Model:       "model-x",
		Temperature: 0.9,
		adapter:     adapter,
	}

	out, errCh := llm.StreamInvoke([]map[string]any{{"role": "user", "content": "hi"}}, nil)
	for range out {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamInvoke() error = %v", err)
		}
	}

	if _, ok := adapter.lastKwargs["temperature"]; !ok {
		t.Fatalf("temperature key missing, want explicit nil like python stream_invoke")
	}
	if adapter.lastKwargs["temperature"] != nil {
		t.Fatalf("temperature = %#v, want nil", adapter.lastKwargs["temperature"])
	}
	if _, ok := adapter.lastKwargs["max_tokens"]; ok {
		t.Fatalf("max_tokens should be omitted when llm.max_tokens is nil")
	}
}

func TestStreamInvokeUsesConfiguredMaxTokensAndExplicitTemperature(t *testing.T) {
	adapter := &captureStreamAdapter{}
	maxTokens := 88
	temp := 0.25
	llm := &HelloAgentsLLM{
		Model:       "model-x",
		Temperature: 0.9,
		MaxTokens:   &maxTokens,
		adapter:     adapter,
	}

	out, errCh := llm.StreamInvoke(
		[]map[string]any{{"role": "user", "content": "hi"}},
		map[string]any{"temperature": temp, "extra": "x"},
	)
	for range out {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamInvoke() error = %v", err)
		}
	}

	if got := adapter.lastKwargs["temperature"]; got != temp {
		t.Fatalf("temperature = %#v, want %v", got, temp)
	}
	if got := adapter.lastKwargs["max_tokens"]; got != maxTokens {
		t.Fatalf("max_tokens = %#v, want %d", got, maxTokens)
	}
	if got := adapter.lastKwargs["extra"]; got != "x" {
		t.Fatalf("extra = %#v, want %q", got, "x")
	}
}
