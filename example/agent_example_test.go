package example

import (
	"context"
	"testing"
	"time"

	"helloagents-go/core"
)

func TestNewBaseAgent(t *testing.T) {
	// 不传 LLM 和 config，仅测 BaseAgent 构造与历史方法
	base := core.NewBaseAgent("test-agent", nil, "You are helpful.", nil)
	if base.Name != "test-agent" || base.SystemPrompt != "You are helpful." {
		t.Fatalf("Name/SystemPrompt: %+v", base)
	}
	if base.Config == nil {
		t.Fatal("config should be default when nil")
	}
	if base.Config.DefaultModel != "gpt-3.5-turbo" {
		t.Fatalf("default config: %+v", base.Config)
	}
}

func TestBaseAgent_AddMessage_GetHistory_ClearHistory(t *testing.T) {
	base := core.NewBaseAgent("a", nil, "", nil)

	m1 := core.NewMessage("hi", core.RoleUser, time.Time{}, nil)
	m2 := core.NewMessage("hello", core.RoleAssistant, time.Time{}, nil)
	base.AddMessage(m1)
	base.AddMessage(m2)

	hist := base.GetHistory()
	if len(hist) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(hist))
	}
	if hist[0].Content != "hi" || hist[1].Content != "hello" {
		t.Fatalf("history content: %+v", hist)
	}
	// 返回的是副本，修改不应影响内部
	hist[0].Content = "mutated"
	if base.GetHistory()[0].Content != "hi" {
		t.Fatal("GetHistory should return copy")
	}

	base.ClearHistory()
	if len(base.GetHistory()) != 0 {
		t.Fatalf("after ClearHistory expected 0, got %d", len(base.GetHistory()))
	}
}

func TestBaseAgent_String(t *testing.T) {
	base := core.NewBaseAgent("my-agent", nil, "", nil)
	s := base.String()
	if s != "Agent(name=my-agent, provider=)" {
		t.Fatalf("String() with nil LLM: got %q", s)
	}

	// 带 LLM 时需有 Provider（用未初始化的 struct 仅作测试）
	base.LLM = &core.HelloAgentsLLM{Provider: "openai"}
	s = base.String()
	if s != "Agent(name=my-agent, provider=openai)" {
		t.Fatalf("String() with LLM: got %q", s)
	}
}

// mockAgent 实现 Agent 接口，用于测试
type mockAgent struct {
	*core.BaseAgent
	echo string
}

func (a *mockAgent) Run(ctx context.Context, inputText string) (string, error) {
	return a.echo + inputText, nil
}

func TestAgent_Interface(t *testing.T) {
	base := core.NewBaseAgent("mock", nil, "", nil)
	agent := &mockAgent{BaseAgent: base, echo: "echo: "}

	out, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "echo: hello" {
		t.Fatalf("Run: got %q", out)
	}
	// 可当作 Agent 接口使用
	var _ core.Agent = (*mockAgent)(nil)
}
