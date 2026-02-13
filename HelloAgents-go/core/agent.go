package core

import (
	"context"
	"fmt"
)

// Agent 是 Agent 抽象接口，与 Python 的 Agent 基类对应。
// 具体实现需实现 Run 方法。
type Agent interface {
	Run(ctx context.Context, inputText string) (string, error)
}

// BaseAgent 提供 Agent 的公共字段与历史方法，供具体 Agent 嵌入使用。
// 对应 Python Agent 基类中除 run 以外的部分。
type BaseAgent struct {
	Name         string
	LLM          *HelloAgentsLLM
	SystemPrompt string
	Config       *Config
	history      []Message
}

// NewBaseAgent 构造 BaseAgent。若 config 为 nil 则使用 DefaultConfig()。
func NewBaseAgent(name string, llm *HelloAgentsLLM, systemPrompt string, config *Config) *BaseAgent {
	if config == nil {
		config = DefaultConfig()
	}
	return &BaseAgent{
		Name:         name,
		LLM:          llm,
		SystemPrompt: systemPrompt,
		Config:       config,
		history:      make([]Message, 0),
	}
}

// AddMessage 添加消息到历史记录，与 Python 的 add_message 对应。
func (a *BaseAgent) AddMessage(m Message) {
	a.history = append(a.history, m)
}

// ClearHistory 清空历史记录，与 Python 的 clear_history 对应。
func (a *BaseAgent) ClearHistory() {
	a.history = a.history[:0]
}

// GetHistory 返回历史记录的副本，与 Python 的 get_history 对应。
func (a *BaseAgent) GetHistory() []Message {
	out := make([]Message, len(a.history))
	copy(out, a.history)
	return out
}

// String 实现 fmt.Stringer，与 Python 的 __str__ 对应。
func (a *BaseAgent) String() string {
	if a.LLM != nil {
		return fmt.Sprintf("Agent(name=%s, provider=%s)", a.Name, a.LLM.Provider)
	}
	return fmt.Sprintf("Agent(name=%s, provider=)", a.Name)
}

// GoString 便于调试时与 Python 的 __repr__ 行为一致。
func (a *BaseAgent) GoString() string {
	return a.String()
}
