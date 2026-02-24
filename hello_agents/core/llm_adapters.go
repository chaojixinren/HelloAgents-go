package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BaseLLMAdapter is the provider abstraction mirrored from Python.
type BaseLLMAdapter interface {
	Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error)
	StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error)
	InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error)
	LastStats() *StreamStats
}

type adapterBase struct {
	APIKey  string
	BaseURL string
	Timeout int
	Model   string
	stats   *StreamStats
}

func (a *adapterBase) LastStats() *StreamStats { return a.stats }

func (a *adapterBase) httpClient() *http.Client {
	return &http.Client{Timeout: time.Duration(a.Timeout) * time.Second}
}

func (a *adapterBase) isThinkingModel() bool {
	m := strings.ToLower(a.Model)
	for _, kw := range []string{"reasoner", "o1", "o3", "thinking"} {
		if strings.Contains(m, kw) {
			return true
		}
	}
	return false
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeMap(dst map[string]any, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func toJSONMap(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

func openAIEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func anthropicEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	if strings.HasSuffix(base, "/messages") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/messages"
	}
	return base + "/v1/messages"
}

func geminiEndpoint(baseURL string, model string, stream bool) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}
	if strings.Contains(base, "/models/") && strings.Contains(base, ":") {
		return base
	}
	modelEscaped := url.PathEscape(model)
	return fmt.Sprintf("%s/models/%s:%s", base, modelEscaped, action)
}

func addAPIKeyQuery(rawURL string, key string, stream bool) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if q.Get("key") == "" {
		q.Set("key", key)
	}
	if stream {
		q.Set("alt", "sse")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func doJSONRequest(client *http.Client, method string, endpoint string, headers map[string]string, body map[string]any) ([]byte, int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("http %d: %s", resp.StatusCode, string(respData))
	}
	return respData, resp.StatusCode, nil
}

var errSSEDone = errors.New("sse done")

func parseSSE(reader io.Reader, onEvent func(event string, data string) error) error {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	eventName := "message"
	dataLines := make([]string, 0)
	flush := func() error {
		if len(dataLines) == 0 {
			eventName = "message"
			return nil
		}
		data := strings.Join(dataLines, "\n")
		err := onEvent(eventName, data)
		eventName = "message"
		dataLines = dataLines[:0]
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := flush(); err != nil {
		return err
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func extractTextFromOpenAIMessage(message map[string]any) string {
	if message == nil {
		return ""
	}
	if content, ok := message["content"].(string); ok {
		return content
	}
	parts := asSlice(message["content"])
	if len(parts) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, part := range parts {
		pm := asMap(part)
		if pm == nil {
			continue
		}
		if text, ok := pm["text"].(string); ok {
			builder.WriteString(text)
		}
	}
	return builder.String()
}

func usageFromOpenAIMap(resp map[string]any) map[string]int {
	usage := asMap(resp["usage"])
	if usage == nil {
		return map[string]int{}
	}
	return map[string]int{
		"prompt_tokens":     intFromAny(usage["prompt_tokens"]),
		"completion_tokens": intFromAny(usage["completion_tokens"]),
		"total_tokens":      intFromAny(usage["total_tokens"]),
	}
}

func usageFromAnthropicMap(resp map[string]any) map[string]int {
	usage := asMap(resp["usage"])
	if usage == nil {
		return map[string]int{}
	}
	input := intFromAny(usage["input_tokens"])
	output := intFromAny(usage["output_tokens"])
	return map[string]int{
		"prompt_tokens":     input,
		"completion_tokens": output,
		"total_tokens":      input + output,
	}
}

func usageFromGeminiMap(resp map[string]any) map[string]int {
	usage := asMap(resp["usageMetadata"])
	if usage == nil {
		return map[string]int{}
	}
	prompt := intFromAny(usage["promptTokenCount"])
	completion := intFromAny(usage["candidatesTokenCount"])
	total := intFromAny(usage["totalTokenCount"])
	if total == 0 {
		total = prompt + completion
	}
	return map[string]int{
		"prompt_tokens":     prompt,
		"completion_tokens": completion,
		"total_tokens":      total,
	}
}

func extractAnthropicText(resp map[string]any) string {
	content := asSlice(resp["content"])
	if len(content) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, block := range content {
		bm := asMap(block)
		if bm == nil {
			continue
		}
		if t, ok := bm["text"].(string); ok {
			builder.WriteString(t)
		}
	}
	return builder.String()
}

func extractGeminiText(resp map[string]any) string {
	candidates := asSlice(resp["candidates"])
	if len(candidates) == 0 {
		return ""
	}
	first := asMap(candidates[0])
	if first == nil {
		return ""
	}
	content := asMap(first["content"])
	if content == nil {
		return ""
	}
	parts := asSlice(content["parts"])
	builder := strings.Builder{}
	for _, p := range parts {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		if txt, ok := pm["text"].(string); ok {
			builder.WriteString(txt)
		}
	}
	return builder.String()
}

type OpenAIAdapter struct{ adapterBase }

type AnthropicAdapter struct{ adapterBase }

type GeminiAdapter struct{ adapterBase }

func (a *OpenAIAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	client := a.httpClient()
	endpoint := openAIEndpoint(a.BaseURL)
	body := mergeMap(map[string]any{
		"model":    a.Model,
		"messages": messages,
	}, copyMap(kwargs))

	start := time.Now()
	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, map[string]string{
		"Authorization": "Bearer " + a.APIKey,
	}, body)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("OpenAI API调用失败: %v", err))
	}

	respMap, err := toJSONMap(respBody)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("OpenAI 响应解析失败: %v", err))
	}

	choices := asSlice(respMap["choices"])
	content := ""
	reasoning := ""
	if len(choices) > 0 {
		choice := asMap(choices[0])
		message := asMap(choice["message"])
		content = extractTextFromOpenAIMessage(message)
		if a.isThinkingModel() {
			reasoning = asString(message["reasoning_content"])
			if reasoning == "" {
				reasoning = asString(choice["reasoning_content"])
			}
		}
	}

	return LLMResponse{
		Content:          content,
		Model:            a.Model,
		Usage:            usageFromOpenAIMap(respMap),
		LatencyMS:        int(time.Since(start).Milliseconds()),
		ReasoningContent: reasoning,
	}, nil
}

func (a *OpenAIAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		client := a.httpClient()
		endpoint := openAIEndpoint(a.BaseURL)
		body := mergeMap(map[string]any{
			"model":    a.Model,
			"messages": messages,
			"stream":   true,
		}, copyMap(kwargs))

		payload, err := json.Marshal(body)
		if err != nil {
			errCh <- err
			return
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+a.APIKey)

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			errCh <- NewHelloAgentsException(fmt.Sprintf("OpenAI API流式调用失败: %v", err))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			errCh <- NewHelloAgentsException(fmt.Sprintf("OpenAI API流式调用失败: http %d: %s", resp.StatusCode, string(data)))
			return
		}

		reasoningBuilder := strings.Builder{}
		usage := map[string]int{}

		err = parseSSE(resp.Body, func(_ string, data string) error {
			if data == "[DONE]" {
				return errSSEDone
			}
			chunk, err := toJSONMap([]byte(data))
			if err != nil {
				return nil
			}
			choices := asSlice(chunk["choices"])
			if len(choices) > 0 {
				choice := asMap(choices[0])
				delta := asMap(choice["delta"])
				if delta != nil {
					if text := asString(delta["content"]); text != "" {
						out <- text
					}
					if a.isThinkingModel() {
						if rc := asString(delta["reasoning_content"]); rc != "" {
							reasoningBuilder.WriteString(rc)
						}
					}
				}
			}
			if usageMap := asMap(chunk["usage"]); usageMap != nil {
				usage = map[string]int{
					"prompt_tokens":     intFromAny(usageMap["prompt_tokens"]),
					"completion_tokens": intFromAny(usageMap["completion_tokens"]),
					"total_tokens":      intFromAny(usageMap["total_tokens"]),
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errSSEDone) {
			errCh <- NewHelloAgentsException(fmt.Sprintf("OpenAI API流式调用失败: %v", err))
			return
		}

		a.stats = &StreamStats{
			Model:            a.Model,
			Usage:            usage,
			LatencyMS:        int(time.Since(start).Milliseconds()),
			ReasoningContent: reasoningBuilder.String(),
		}
	}()

	return out, errCh
}

func (a *OpenAIAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	client := a.httpClient()
	endpoint := openAIEndpoint(a.BaseURL)
	body := mergeMap(map[string]any{
		"model":    a.Model,
		"messages": messages,
		"tools":    tools,
	}, copyMap(kwargs))

	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, map[string]string{
		"Authorization": "Bearer " + a.APIKey,
	}, body)
	if err != nil {
		return nil, NewHelloAgentsException(fmt.Sprintf("OpenAI Function Calling调用失败: %v", err))
	}
	return toJSONMap(respBody)
}

func convertAnthropicMessages(messages []map[string]any) (string, []map[string]any) {
	system := ""
	converted := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		role := asString(msg["role"])
		if role == "system" {
			system = asString(msg["content"])
			continue
		}
		converted = append(converted, copyMap(msg))
	}
	return system, converted
}

func (a *AnthropicAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	client := a.httpClient()
	endpoint := anthropicEndpoint(a.BaseURL)
	system, converted := convertAnthropicMessages(messages)

	body := mergeMap(map[string]any{
		"model":      a.Model,
		"messages":   converted,
		"max_tokens": 4096,
	}, copyMap(kwargs))
	if system != "" {
		body["system"] = system
	}

	start := time.Now()
	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, map[string]string{
		"x-api-key":         a.APIKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("Anthropic API调用失败: %v", err))
	}
	respMap, err := toJSONMap(respBody)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("Anthropic 响应解析失败: %v", err))
	}

	return LLMResponse{
		Content:   extractAnthropicText(respMap),
		Model:     a.Model,
		Usage:     usageFromAnthropicMap(respMap),
		LatencyMS: int(time.Since(start).Milliseconds()),
	}, nil
}

func (a *AnthropicAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)

		client := a.httpClient()
		endpoint := anthropicEndpoint(a.BaseURL)
		system, converted := convertAnthropicMessages(messages)
		body := mergeMap(map[string]any{
			"model":      a.Model,
			"messages":   converted,
			"max_tokens": 4096,
			"stream":     true,
		}, copyMap(kwargs))
		if system != "" {
			body["system"] = system
		}

		payload, err := json.Marshal(body)
		if err != nil {
			errCh <- err
			return
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", a.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			errCh <- NewHelloAgentsException(fmt.Sprintf("Anthropic API流式调用失败: %v", err))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			errCh <- NewHelloAgentsException(fmt.Sprintf("Anthropic API流式调用失败: http %d: %s", resp.StatusCode, string(data)))
			return
		}

		usage := map[string]int{}
		err = parseSSE(resp.Body, func(_ string, data string) error {
			if data == "[DONE]" {
				return errSSEDone
			}
			eventMap, err := toJSONMap([]byte(data))
			if err != nil {
				return nil
			}
			emitted := false
			if delta := asMap(eventMap["delta"]); delta != nil {
				if text := asString(delta["text"]); text != "" {
					out <- text
					emitted = true
				}
			}
			if !emitted {
				if contentBlock := asMap(eventMap["content_block"]); contentBlock != nil {
					if text := asString(contentBlock["text"]); text != "" {
						out <- text
					}
				}
			}
			if usageMap := asMap(eventMap["usage"]); usageMap != nil {
				input := intFromAny(usageMap["input_tokens"])
				output := intFromAny(usageMap["output_tokens"])
				usage = map[string]int{
					"prompt_tokens":     input,
					"completion_tokens": output,
					"total_tokens":      input + output,
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errSSEDone) {
			errCh <- NewHelloAgentsException(fmt.Sprintf("Anthropic API流式调用失败: %v", err))
			return
		}
		a.stats = &StreamStats{Model: a.Model, Usage: usage, LatencyMS: int(time.Since(start).Milliseconds())}
	}()
	return out, errCh
}

func (a *AnthropicAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	client := a.httpClient()
	endpoint := anthropicEndpoint(a.BaseURL)
	system, converted := convertAnthropicMessages(messages)
	body := mergeMap(map[string]any{
		"model":      a.Model,
		"messages":   converted,
		"tools":      tools,
		"max_tokens": 4096,
	}, copyMap(kwargs))
	if system != "" {
		body["system"] = system
	}

	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, map[string]string{
		"x-api-key":         a.APIKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return nil, NewHelloAgentsException(fmt.Sprintf("Anthropic工具调用失败: %v", err))
	}
	return toJSONMap(respBody)
}

func convertGeminiMessages(messages []map[string]any) (any, []map[string]any) {
	var systemInstruction any
	converted := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		role := asString(msg["role"])
		content := msg["content"]
		if role == "system" {
			systemInstruction = content
			continue
		}
		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}
		converted = append(converted, map[string]any{
			"role": geminiRole,
			"parts": []map[string]any{
				{"text": content},
			},
		})
	}
	return systemInstruction, converted
}

func convertGeminiTools(toolsInput []map[string]any) []map[string]any {
	functionDecls := make([]map[string]any, 0)
	for _, t := range toolsInput {
		if asString(t["type"]) != "function" {
			continue
		}
		fn := asMap(t["function"])
		if fn == nil {
			continue
		}
		functionDecls = append(functionDecls, map[string]any{
			"name":        asString(fn["name"]),
			"description": asString(fn["description"]),
			"parameters":  fn["parameters"],
		})
	}
	if len(functionDecls) == 0 {
		return nil
	}
	return []map[string]any{{"functionDeclarations": functionDecls}}
}

func (a *GeminiAdapter) Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	client := a.httpClient()
	endpoint := addAPIKeyQuery(geminiEndpoint(a.BaseURL, a.Model, false), a.APIKey, false)
	systemInstruction, converted := convertGeminiMessages(messages)

	generationConfig := map[string]any{}
	kw := copyMap(kwargs)
	if v, ok := kw["temperature"]; ok {
		generationConfig["temperature"] = v
		delete(kw, "temperature")
	}
	if v, ok := kw["max_tokens"]; ok {
		generationConfig["maxOutputTokens"] = v
		delete(kw, "max_tokens")
	}

	body := map[string]any{"contents": converted}
	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}
	if pythonTruthy(systemInstruction) {
		body["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": systemInstruction}}}
	}
	body = mergeMap(body, kw)

	start := time.Now()
	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, nil, body)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("Gemini API调用失败: %v", err))
	}
	respMap, err := toJSONMap(respBody)
	if err != nil {
		return LLMResponse{}, NewHelloAgentsException(fmt.Sprintf("Gemini 响应解析失败: %v", err))
	}

	return LLMResponse{
		Content:   extractGeminiText(respMap),
		Model:     a.Model,
		Usage:     usageFromGeminiMap(respMap),
		LatencyMS: int(time.Since(start).Milliseconds()),
	}, nil
}

func (a *GeminiAdapter) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)

		client := a.httpClient()
		endpoint := addAPIKeyQuery(geminiEndpoint(a.BaseURL, a.Model, true), a.APIKey, true)
		systemInstruction, converted := convertGeminiMessages(messages)
		generationConfig := map[string]any{}
		kw := copyMap(kwargs)
		if v, ok := kw["temperature"]; ok {
			generationConfig["temperature"] = v
			delete(kw, "temperature")
		}
		if v, ok := kw["max_tokens"]; ok {
			generationConfig["maxOutputTokens"] = v
			delete(kw, "max_tokens")
		}
		body := map[string]any{"contents": converted}
		if len(generationConfig) > 0 {
			body["generationConfig"] = generationConfig
		}
		if pythonTruthy(systemInstruction) {
			body["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": systemInstruction}}}
		}
		body = mergeMap(body, kw)

		payload, err := json.Marshal(body)
		if err != nil {
			errCh <- err
			return
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			errCh <- NewHelloAgentsException(fmt.Sprintf("Gemini API流式调用失败: %v", err))
			return
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			errCh <- NewHelloAgentsException(fmt.Sprintf("Gemini API流式调用失败: http %d: %s", resp.StatusCode, string(data)))
			return
		}
		defer resp.Body.Close()

		usage := map[string]int{}
		err = parseSSE(resp.Body, func(_ string, data string) error {
			if data == "[DONE]" {
				return errSSEDone
			}
			chunkMap, err := toJSONMap([]byte(data))
			if err != nil {
				return nil
			}
			text := extractGeminiText(chunkMap)
			if text != "" {
				out <- text
			}
			if u := usageFromGeminiMap(chunkMap); len(u) > 0 {
				usage = u
			}
			return nil
		})
		if err != nil && !errors.Is(err, errSSEDone) {
			errCh <- NewHelloAgentsException(fmt.Sprintf("Gemini API流式调用失败: %v", err))
			return
		}
		a.stats = &StreamStats{Model: a.Model, Usage: usage, LatencyMS: int(time.Since(start).Milliseconds())}
	}()
	return out, errCh
}

func (a *GeminiAdapter) InvokeWithTools(messages []map[string]any, tools []map[string]any, kwargs map[string]any) (map[string]any, error) {
	client := a.httpClient()
	endpoint := addAPIKeyQuery(geminiEndpoint(a.BaseURL, a.Model, false), a.APIKey, false)
	systemInstruction, converted := convertGeminiMessages(messages)

	generationConfig := map[string]any{}
	kw := copyMap(kwargs)
	if v, ok := kw["temperature"]; ok {
		generationConfig["temperature"] = v
		delete(kw, "temperature")
	}
	if v, ok := kw["max_tokens"]; ok {
		generationConfig["maxOutputTokens"] = v
		delete(kw, "max_tokens")
	}

	body := map[string]any{"contents": converted}
	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}
	if pythonTruthy(systemInstruction) {
		body["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": systemInstruction}}}
	}
	if geminiTools := convertGeminiTools(tools); len(geminiTools) > 0 {
		body["tools"] = geminiTools
	}
	body = mergeMap(body, kw)

	respBody, _, err := doJSONRequest(client, http.MethodPost, endpoint, nil, body)
	if err != nil {
		return nil, NewHelloAgentsException(fmt.Sprintf("Gemini工具调用失败: %v", err))
	}
	return toJSONMap(respBody)
}

// CreateAdapter mirrors Python auto-detection from base_url.
func CreateAdapter(apiKey, baseURL string, timeout int, model string) BaseLLMAdapter {
	base := adapterBase{APIKey: apiKey, BaseURL: baseURL, Timeout: timeout, Model: model}
	urlLower := strings.ToLower(baseURL)
	if strings.Contains(urlLower, "anthropic.com") {
		return &AnthropicAdapter{adapterBase: base}
	}
	if strings.Contains(urlLower, "googleapis.com") || strings.Contains(urlLower, "generativelanguage") {
		return &GeminiAdapter{adapterBase: base}
	}
	// OpenAI-compatible adapters are the default.
	return &OpenAIAdapter{adapterBase: base}
}
