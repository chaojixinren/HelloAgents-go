package agents

import (
	"testing"

	"helloagents-go/hello_agents/core"
)

func TestSimpleAgentBuildMessagesKeepsWhitespaceSystemPrompt(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := NewSimpleAgentWithOptions(
		"tester",
		&core.HelloAgentsLLM{Model: "test-model"},
		"   ",
		&cfg,
		nil,
		false,
		3,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}

	messages := agent.buildMessages("hello")
	if len(messages) < 2 {
		t.Fatalf("messages length = %d, want at least 2", len(messages))
	}
	if messages[0]["role"] != "system" || messages[0]["content"] != "   " {
		t.Fatalf("first message = %#v, want whitespace system prompt preserved", messages[0])
	}
}

func TestSimpleAgentKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := NewSimpleAgentWithOptions(
		"tester",
		&core.HelloAgentsLLM{Model: "test-model"},
		"",
		&cfg,
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}
	if agent.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", agent.MaxToolIterations)
	}
}
