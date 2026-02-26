package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func TestReflectionAgentKeepsExplicitZeroIterationSettings(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewReflectionAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
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
