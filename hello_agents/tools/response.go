package tools

import (
	"bytes"
	"encoding/json"
	"strings"
)

type ToolStatus string

const (
	ToolStatusSuccess ToolStatus = "success"
	ToolStatusPartial ToolStatus = "partial"
	ToolStatusError   ToolStatus = "error"
)

// ToolResponse mirrors Python tool response protocol.
type ToolResponse struct {
	Status    ToolStatus        `json:"status"`
	Text      string            `json:"text"`
	Data      map[string]any    `json:"data,omitempty"`
	ErrorInfo map[string]string `json:"error,omitempty"`
	Stats     map[string]any    `json:"stats,omitempty"`
	Context   map[string]any    `json:"context,omitempty"`
}

func (r ToolResponse) ToMap() map[string]any {
	result := map[string]any{
		"status": string(r.Status),
		"text":   r.Text,
		"data":   r.Data,
	}
	if len(r.ErrorInfo) > 0 {
		result["error"] = r.ErrorInfo
	}
	if len(r.Stats) > 0 {
		result["stats"] = r.Stats
	}
	if len(r.Context) > 0 {
		result["context"] = r.Context
	}
	return result
}

// ToDict keeps naming parity with Python ToolResponse.to_dict().
func (r ToolResponse) ToDict() map[string]any {
	return r.ToMap()
}

func (r ToolResponse) ToJSON() string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r.ToMap()); err != nil {
		payload, _ := json.Marshal(r.ToMap())
		return string(payload)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func ToolResponseFromMap(data map[string]any) ToolResponse {
	resp := ToolResponse{
		Status: ToolStatusSuccess,
		Data:   map[string]any{},
	}
	if s, ok := data["status"].(string); ok {
		resp.Status = ToolStatus(s)
	}
	if text, ok := data["text"].(string); ok {
		resp.Text = text
	}
	if d, ok := data["data"].(map[string]any); ok {
		resp.Data = d
	}
	if errInfo, ok := data["error"].(map[string]any); ok {
		out := map[string]string{}
		for k, v := range errInfo {
			out[k] = toString(v)
		}
		resp.ErrorInfo = out
	} else if errInfo, ok := data["error"].(map[string]string); ok {
		out := map[string]string{}
		for k, v := range errInfo {
			out[k] = v
		}
		resp.ErrorInfo = out
	}
	if stats, ok := data["stats"].(map[string]any); ok {
		resp.Stats = stats
	}
	if ctx, ok := data["context"].(map[string]any); ok {
		resp.Context = ctx
	}
	return resp
}

// ToolResponseFromDict keeps naming parity with Python ToolResponse.from_dict().
func ToolResponseFromDict(data map[string]any) ToolResponse {
	return ToolResponseFromMap(data)
}

func ToolResponseFromJSON(jsonStr string) ToolResponse {
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		panic(err)
	}
	return ToolResponseFromMap(payload)
}

func Success(text string, data map[string]any, extras ...map[string]any) ToolResponse {
	if data == nil {
		data = map[string]any{}
	}
	var stats map[string]any
	var context map[string]any
	if len(extras) > 0 {
		stats = extras[0]
	}
	if len(extras) > 1 {
		context = extras[1]
	}
	return ToolResponse{
		Status:  ToolStatusSuccess,
		Text:    text,
		Data:    data,
		Stats:   stats,
		Context: context,
	}
}

func Partial(text string, data map[string]any, extras ...map[string]any) ToolResponse {
	if data == nil {
		data = map[string]any{}
	}
	var stats map[string]any
	var context map[string]any
	if len(extras) > 0 {
		stats = extras[0]
	}
	if len(extras) > 1 {
		context = extras[1]
	}
	return ToolResponse{
		Status:  ToolStatusPartial,
		Text:    text,
		Data:    data,
		Stats:   stats,
		Context: context,
	}
}

func Error(text string, code string, details map[string]any) ToolResponse {
	context := details
	if context == nil {
		context = map[string]any{}
	}
	return ToolResponse{
		Status: ToolStatusError,
		Text:   text,
		Data:   map[string]any{},
		ErrorInfo: map[string]string{
			"code":    code,
			"message": text,
		},
		Context: context,
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
