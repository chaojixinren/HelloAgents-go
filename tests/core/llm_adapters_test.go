package core_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"helloagents-go/hello_agents/core"
)

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
