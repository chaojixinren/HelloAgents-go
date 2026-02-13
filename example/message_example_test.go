package example

import (
	"testing"
	"time"

	"helloagents-go/core"
)

func TestNewMessage(t *testing.T) {
	// 零值时间应被替换为当前时间
	m := core.NewMessage("hello", core.RoleUser, time.Time{}, nil)
	if m.Content != "hello" || m.Role != core.RoleUser {
		t.Fatalf("Content/Role mismatch: %+v", m)
	}
	if m.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
	if m.Metadata == nil {
		t.Fatal("expected non-nil Metadata")
	}
}

func TestNewMessage_WithTimestampAndMetadata(t *testing.T) {
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	meta := map[string]interface{}{"k": "v"}
	m := core.NewMessage("hi", core.RoleAssistant, ts, meta)
	if !m.Timestamp.Equal(ts) || m.Metadata["k"] != "v" {
		t.Fatalf("timestamp or metadata not set: %+v", m)
	}
}

func TestMessage_ToChatMessage(t *testing.T) {
	m := core.Message{Content: "test", Role: core.RoleSystem}
	cm := m.ToChatMessage()
	if cm.Role != core.RoleSystem || cm.Content != "test" {
		t.Fatalf("ToChatMessage: got %+v", cm)
	}
}

func TestMessage_String(t *testing.T) {
	m := core.Message{Content: "hello", Role: core.RoleUser}
	s := m.String()
	if s != "[user] hello" {
		t.Fatalf("String(): got %q", s)
	}
}
