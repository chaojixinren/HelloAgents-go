package core_test

import (
	"testing"
	"time"

	"helloagents-go/hello_agents/core"
)

func TestMessageFromMapParsesPythonISOFormatTimestamp(t *testing.T) {
	msg, err := core.MessageFromMap(map[string]any{
		"content":   "hello",
		"role":      "user",
		"timestamp": "2026-02-24T12:34:56.123456",
	})
	if err != nil {
		t.Fatalf("MessageFromMap() error = %v", err)
	}
	if got := msg.Timestamp.Year(); got != 2026 {
		t.Fatalf("timestamp year = %d, want 2026", got)
	}
	if got := int(msg.Timestamp.Month()); got != 2 {
		t.Fatalf("timestamp month = %d, want 2", got)
	}
	if got := msg.Timestamp.Day(); got != 24 {
		t.Fatalf("timestamp day = %d, want 24", got)
	}
}

func TestMessageToMapUsesPythonISOFormat(t *testing.T) {
	msg := core.Message{
		Content:   "hello",
		Role:      core.MessageRoleUser,
		Timestamp: time.Date(2026, 2, 24, 12, 34, 56, 123456000, time.UTC),
		Metadata:  map[string]any{},
	}

	data := msg.ToMap()
	got, _ := data["timestamp"].(string)
	want := "2026-02-24T12:34:56.123456"
	if got != want {
		t.Fatalf("timestamp = %q, want %q", got, want)
	}
}
