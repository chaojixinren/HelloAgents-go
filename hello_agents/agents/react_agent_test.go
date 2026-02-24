package agents

import (
	"testing"

	"helloagents-go/hello_agents/core"
)

func TestReActAgentKeepsExplicitZeroMaxSteps(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := NewReActAgent(
		"tester",
		&core.HelloAgentsLLM{Model: "test-model"},
		"",
		&cfg,
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}
	if agent.MaxSteps != 0 {
		t.Fatalf("MaxSteps = %d, want 0", agent.MaxSteps)
	}
}
