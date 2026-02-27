package tests_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"helloagents-go/hello_agents/core"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type captureStreamAdapter struct {
	lastKwargs map[string]any
}

func (a *captureStreamAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (core.LLMResponse, error) {
	return core.LLMResponse{}, nil
}

func (a *captureStreamAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	a.lastKwargs = core.ExportCopyMap(kwargs)
	out := make(chan string)
	errCh := make(chan error)
	close(out)
	close(errCh)
	return out, errCh
}

func (a *captureStreamAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (a *captureStreamAdapter) LastStats() *core.StreamStats {
	return nil
}

// ---------------------------------------------------------------------------
// LLM tests (from core/llm_test.go)
// ---------------------------------------------------------------------------

func TestNewHelloAgentsLLMTimeoutZeroFallsBackToEnvLikePythonTruthy(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	t.Setenv("LLM_TIMEOUT", "77")

	timeout := 0
	llm, err := core.NewHelloAgentsLLM("model-x", "", "", 0.7, nil, &timeout, nil)
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
	llm, err := core.NewHelloAgentsLLM("model-x", "", "", 0.7, &maxTokens, nil, nil)
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

	_, err := core.NewHelloAgentsLLM("model-x", "", "", 0.7, nil, nil, nil)
	if err == nil {
		t.Fatalf("NewHelloAgentsLLM should return error when LLM_TIMEOUT is invalid")
	}
}

func TestNewHelloAgentsLLMIgnoresProviderKwargLikePython(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")

	llm, err := core.NewHelloAgentsLLM(
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

func TestStreamInvokeOmitsTemperatureWhenNotProvided(t *testing.T) {
	adapter := &captureStreamAdapter{}
	llm := core.NewLLMFromAdapter("model-x", "", "", 0, 0.9, adapter)

	out, errCh := llm.StreamInvoke([]map[string]any{{"role": "user", "content": "hi"}}, nil)
	for range out {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamInvoke() error = %v", err)
		}
	}

	if _, ok := adapter.lastKwargs["temperature"]; ok {
		t.Fatalf("temperature should be omitted when not explicitly provided, got %#v", adapter.lastKwargs["temperature"])
	}
	if _, ok := adapter.lastKwargs["max_tokens"]; ok {
		t.Fatalf("max_tokens should be omitted when llm.max_tokens is nil")
	}
}

func TestStreamInvokeUsesConfiguredMaxTokensAndExplicitTemperature(t *testing.T) {
	adapter := &captureStreamAdapter{}
	maxTokens := 88
	temp := 0.25
	llm := core.NewLLMFromAdapter("model-x", "", "", 0, 0.9, adapter)
	llm.MaxTokens = &maxTokens

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

// ---------------------------------------------------------------------------
// LLM adapter tests (from core/llm_adapters_test.go)
// ---------------------------------------------------------------------------

func TestConvertAnthropicMessagesPreservesNonSystemMessageFields(t *testing.T) {
	system, converted := core.ExportConvertAnthropicMessages([]map[string]any{
		{"role": "system", "content": "sys"},
		{"role": "user", "content": "hello", "extra": 123},
		{"role": "assistant", "content": map[string]any{"type": "text", "text": "ok"}},
	})

	if system != "sys" {
		t.Fatalf("system = %q, want %q", system, "sys")
	}
	if len(converted) != 2 {
		t.Fatalf("len(converted) = %d, want 2", len(converted))
	}
	if converted[0]["extra"] != 123 {
		t.Fatalf("converted[0].extra = %v, want 123", converted[0]["extra"])
	}
	if _, ok := converted[1]["content"].(map[string]any); !ok {
		t.Fatalf("converted[1].content type = %T, want map[string]any", converted[1]["content"])
	}
}

func TestOpenAIInvokeUsesConfiguredModelLikePython(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"server-side-model",
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
		}`))
	}))
	defer server.Close()

	adapter := core.NewOpenAIAdapterForTest("k", server.URL+"/v1", 5, "configured-model")

	resp, err := adapter.Invoke([]map[string]any{
		{"role": "user", "content": "hi"},
	}, nil)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.Model != "configured-model" {
		t.Fatalf("resp.Model = %q, want %q", resp.Model, "configured-model")
	}
	if resp.Content != "hello" {
		t.Fatalf("resp.Content = %q, want %q", resp.Content, "hello")
	}
}

func TestAnthropicStreamAvoidsDuplicateDeltaAndContentBlockText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		_, _ = fmt.Fprint(w, "data: {\"delta\":{\"text\":\"A\"},\"content_block\":{\"text\":\"A\"}}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	adapter := core.NewAnthropicAdapterForTest("k", server.URL+"/v1", 5, "claude-test")

	out, errCh := adapter.StreamInvoke([]map[string]any{
		{"role": "user", "content": "hi"},
	}, nil)

	chunks := make([]string, 0)
	for chunk := range out {
		chunks = append(chunks, chunk)
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamInvoke() error = %v", err)
		}
	}

	if len(chunks) != 1 || chunks[0] != "A" {
		t.Fatalf("chunks = %#v, want [\"A\"]", chunks)
	}
}

func TestGeminiStreamReturnsErrorWithoutInvokeFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := core.NewGeminiAdapterForTest("k", server.URL+"/v1beta", 5, "gemini-test")

	out, errCh := adapter.StreamInvoke([]map[string]any{
		{"role": "user", "content": "hi"},
	}, nil)

	gotChunks := 0
	for range out {
		gotChunks++
	}

	var gotErr error
	for err := range errCh {
		if err != nil {
			gotErr = err
		}
	}

	if gotChunks != 0 {
		t.Fatalf("gotChunks = %d, want 0 when stream endpoint fails", gotChunks)
	}
	if gotErr == nil {
		t.Fatalf("expected stream error, got nil")
	}
}

func TestConvertGeminiMessagesPreservesNonStringContentLikePython(t *testing.T) {
	systemInstruction, converted := core.ExportConvertGeminiMessages([]map[string]any{
		{"role": "system", "content": map[string]any{"lang": "zh"}},
		{"role": "user", "content": map[string]any{"text": "hi"}},
	})

	systemMap, ok := systemInstruction.(map[string]any)
	if !ok {
		t.Fatalf("system_instruction type = %T, want map[string]any", systemInstruction)
	}
	if systemMap["lang"] != "zh" {
		t.Fatalf("system_instruction.lang = %v, want zh", systemMap["lang"])
	}

	if len(converted) != 1 {
		t.Fatalf("len(converted) = %d, want 1", len(converted))
	}
	parts, ok := converted[0]["parts"].([]map[string]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("parts type/len = %T/%d, want []map[string]any/1", converted[0]["parts"], len(parts))
	}
	contentMap, ok := parts[0]["text"].(map[string]any)
	if !ok {
		t.Fatalf("parts[0].text type = %T, want map[string]any", parts[0]["text"])
	}
	if contentMap["text"] != "hi" {
		t.Fatalf("parts[0].text.text = %v, want hi", contentMap["text"])
	}
}

// ---------------------------------------------------------------------------
// Config tests (from core/config_test.go)
// ---------------------------------------------------------------------------

func TestFromEnvDebugMatchesPythonTrueOnlyRule(t *testing.T) {
	t.Setenv("DEBUG", "1")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "")

	cfg := core.FromEnv()
	if cfg.Debug {
		t.Fatalf("FromEnv().Debug = true, want false when DEBUG=1")
	}
}

func TestFromEnvFallsBackOnInvalidTemperature(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "bad-number")
	t.Setenv("MAX_TOKENS", "")

	cfg := core.FromEnv()
	if cfg.Temperature != 0.7 {
		t.Fatalf("FromEnv().Temperature = %v, want 0.7 fallback on invalid input", cfg.Temperature)
	}
}

func TestFromEnvIgnoresInvalidMaxTokens(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "not-int")

	cfg := core.FromEnv()
	if cfg.MaxTokens != nil {
		t.Fatalf("FromEnv().MaxTokens = %v, want nil on invalid input", *cfg.MaxTokens)
	}
}

func TestFromEnvKeepsExplicitEmptyLogLevelLikeOsGetenv(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "")

	cfg := core.FromEnv()
	if cfg.LogLevel != "" {
		t.Fatalf("FromEnv().LogLevel = %q, want explicit empty string", cfg.LogLevel)
	}
}

func TestFromEnvFallsBackWhenTemperatureIsExplicitEmpty(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "")
	t.Setenv("MAX_TOKENS", "")

	cfg := core.FromEnv()
	if cfg.Temperature != 0.7 {
		t.Fatalf("FromEnv().Temperature = %v, want 0.7 fallback on empty string", cfg.Temperature)
	}
}
