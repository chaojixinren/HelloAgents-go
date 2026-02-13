package core

import (
	"fmt"
	"time"
)

// MessageRole 表示消息角色，与 Python 的 MessageRole 对应。
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// Message 表示单条对话消息，与 Python 的 Message 对应。
type Message struct {
	Content   string
	Role      string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// NewMessage 创建一条消息。若 timestamp 为零值则使用当前时间。
func NewMessage(content, role string, timestamp time.Time, metadata map[string]interface{}) Message {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	return Message{Content: content, Role: role, Timestamp: timestamp, Metadata: metadata}
}

// ToChatMessage 转为 LLM 使用的 ChatMessage。
func (m Message) ToChatMessage() ChatMessage {
	return ChatMessage{Role: m.Role, Content: m.Content}
}

// String 实现 fmt.Stringer，与 Python 的 __str__ 对应。
func (m Message) String() string {
	return fmt.Sprintf("[%s] %s", m.Role, m.Content)
}
