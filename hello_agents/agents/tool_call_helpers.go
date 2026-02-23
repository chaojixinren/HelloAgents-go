package agents

import (
	"encoding/json"
	"fmt"
	"strings"
)

type toolCallEnvelope struct {
	ID           string
	Name         string
	Arguments    map[string]any
	RawArguments string
	ParseError   string
}

func extractToolCallsAndContent(response map[string]any) (string, []toolCallEnvelope) {
	if response == nil {
		return "", nil
	}
	if content, calls, ok := parseOpenAIResponse(response); ok {
		return content, calls
	}
	if content, calls, ok := parseAnthropicResponse(response); ok {
		return content, calls
	}
	if content, calls, ok := parseGeminiResponse(response); ok {
		return content, calls
	}
	return "", nil
}

func parseOpenAIResponse(response map[string]any) (string, []toolCallEnvelope, bool) {
	choices, ok := response["choices"].([]any)
	if !ok || len(choices) == 0 {
		return "", nil, false
	}
	firstChoice, _ := choices[0].(map[string]any)
	message, _ := firstChoice["message"].(map[string]any)
	if message == nil {
		return "", nil, false
	}

	content := ""
	if c, ok := message["content"].(string); ok {
		content = c
	}

	calls := make([]toolCallEnvelope, 0)
	rawCalls, _ := message["tool_calls"].([]any)
	for idx, raw := range rawCalls {
		tc, _ := raw.(map[string]any)
		if tc == nil {
			continue
		}
		id, _ := tc["id"].(string)
		if id == "" {
			id = fmt.Sprintf("call-%d", idx+1)
		}
		function, _ := tc["function"].(map[string]any)
		if function == nil {
			continue
		}
		name, _ := function["name"].(string)
		rawArgs, _ := function["arguments"].(string)
		args := map[string]any{}
		parseErr := ""
		if rawArgs != "" {
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				parseErr = err.Error()
			}
		}
		if args == nil {
			args = map[string]any{}
		}
		calls = append(calls, toolCallEnvelope{ID: id, Name: name, Arguments: args, RawArguments: rawArgs, ParseError: parseErr})
	}
	return content, calls, true
}

func parseAnthropicResponse(response map[string]any) (string, []toolCallEnvelope, bool) {
	blocks, ok := response["content"].([]any)
	if !ok || len(blocks) == 0 {
		return "", nil, false
	}
	contentBuilder := strings.Builder{}
	calls := make([]toolCallEnvelope, 0)
	for idx, raw := range blocks {
		block, _ := raw.(map[string]any)
		if block == nil {
			continue
		}
		typ, _ := block["type"].(string)
		switch typ {
		case "text":
			if text, ok := block["text"].(string); ok {
				contentBuilder.WriteString(text)
			}
		case "tool_use":
			id, _ := block["id"].(string)
			if id == "" {
				id = fmt.Sprintf("call-%d", idx+1)
			}
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			if input == nil {
				input = map[string]any{}
			}
			rawArgs, _ := json.Marshal(input)
			calls = append(calls, toolCallEnvelope{ID: id, Name: name, Arguments: input, RawArguments: string(rawArgs)})
		}
	}
	return contentBuilder.String(), calls, true
}

func parseGeminiResponse(response map[string]any) (string, []toolCallEnvelope, bool) {
	candidates, ok := response["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return "", nil, false
	}
	firstCandidate, _ := candidates[0].(map[string]any)
	if firstCandidate == nil {
		return "", nil, false
	}
	content, _ := firstCandidate["content"].(map[string]any)
	if content == nil {
		return "", nil, false
	}
	parts, _ := content["parts"].([]any)
	textBuilder := strings.Builder{}
	calls := make([]toolCallEnvelope, 0)
	for idx, raw := range parts {
		part, _ := raw.(map[string]any)
		if part == nil {
			continue
		}
		if text, ok := part["text"].(string); ok {
			textBuilder.WriteString(text)
		}
		if fnCall, ok := part["functionCall"].(map[string]any); ok {
			name, _ := fnCall["name"].(string)
			args, _ := fnCall["args"].(map[string]any)
			if args == nil {
				args = map[string]any{}
			}
			rawArgs, _ := json.Marshal(args)
			calls = append(calls, toolCallEnvelope{
				ID:           fmt.Sprintf("call-%d", idx+1),
				Name:         name,
				Arguments:    args,
				RawArguments: string(rawArgs),
			})
		}
	}
	return textBuilder.String(), calls, true
}

func toOpenAIToolCallsPayload(toolCalls []toolCallEnvelope) []map[string]any {
	out := make([]map[string]any, 0, len(toolCalls))
	for _, tc := range toolCalls {
		args := tc.RawArguments
		if args == "" {
			payload, _ := json.Marshal(tc.Arguments)
			args = string(payload)
		}
		out = append(out, map[string]any{
			"id":   tc.ID,
			"type": "function",
			"function": map[string]any{
				"name":      tc.Name,
				"arguments": args,
			},
		})
	}
	return out
}

func usageFromLLMRawResponse(response map[string]any) map[string]any {
	usage, _ := response["usage"].(map[string]any)
	if usage == nil {
		usage = map[string]any{}
	}
	return map[string]any{
		"prompt_tokens":     intFromAny(usage["prompt_tokens"]),
		"completion_tokens": intFromAny(usage["completion_tokens"]),
		"total_tokens":      intFromAny(usage["total_tokens"]),
	}
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
