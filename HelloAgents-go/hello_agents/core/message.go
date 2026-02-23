package core

import (
	"fmt"
	"time"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
	MessageRoleSummary   MessageRole = "summary"
)

// Message mirrors hello_agents.core.message.Message.
type Message struct {
	Content   string         `json:"content"`
	Role      MessageRole    `json:"role"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata"`
}

func NewMessage(content string, role MessageRole, metadata map[string]any) Message {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return Message{
		Content:   content,
		Role:      role,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

func (m Message) ToMap() map[string]any {
	return map[string]any{
		"role":      string(m.Role),
		"content":   m.Content,
		"timestamp": m.Timestamp.Format(time.RFC3339Nano),
		"metadata":  m.Metadata,
	}
}

// ToDict keeps naming parity with Python Message.to_dict().
func (m Message) ToDict() map[string]any {
	return m.ToMap()
}

func MessageFromMap(data map[string]any) (Message, error) {
	msg := Message{Metadata: map[string]any{}}

	content, ok := data["content"].(string)
	if !ok {
		return msg, fmt.Errorf("missing content")
	}
	msg.Content = content

	role, ok := data["role"].(string)
	if !ok {
		return msg, fmt.Errorf("missing role")
	}
	msg.Role = MessageRole(role)

	if ts, ok := data["timestamp"].(string); ok && ts != "" {
		parsed, err := time.Parse(time.RFC3339Nano, ts)
		if err == nil {
			msg.Timestamp = parsed
		}
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	if md, ok := data["metadata"].(map[string]any); ok {
		msg.Metadata = md
	}

	return msg, nil
}

// MessageFromDict keeps naming parity with Python Message.from_dict().
func MessageFromDict(data map[string]any) (Message, error) {
	return MessageFromMap(data)
}

func (m Message) ToText() string {
	return fmt.Sprintf("[%s] %s", m.Role, m.Content)
}

func (m Message) String() string {
	return m.ToText()
}
