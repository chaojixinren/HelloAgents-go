package agents

import (
	"testing"

	"helloagents-go/hello_agents/core"
)

func TestReflectionAgentKeepsExplicitZeroIterationSettings(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := NewReflectionAgentWithOptions(
		"tester",
		&core.HelloAgentsLLM{Model: "test-model"},
		"",
		&cfg,
		0,
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewReflectionAgentWithOptions error: %v", err)
	}
	if agent.MaxIterations != 0 {
		t.Fatalf("MaxIterations = %d, want 0", agent.MaxIterations)
	}
	if agent.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", agent.MaxToolIterations)
	}
}
