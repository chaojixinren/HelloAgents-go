package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func TestReActAgentKeepsExplicitZeroMaxSteps(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewReActAgent(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
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
