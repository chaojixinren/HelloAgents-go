package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func TestDefaultSubagentFactoryAssignsZeroSubagentMaxStepsLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	cfg.SubagentMaxSteps = 0

	subagent, err := agents.DefaultSubagentFactory(
		"react",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*agents.ReActAgent)
	if !ok {
		t.Fatalf("subagent type = %T, want *ReActAgent", subagent)
	}
	if reactAgent.MaxSteps != 0 {
		t.Fatalf("MaxSteps = %d, want 0", reactAgent.MaxSteps)
	}
}

func TestDefaultSubagentFactoryKeepsOriginalAgentTypeCaseInNameLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	subagent, err := agents.DefaultSubagentFactory(
		"ReAct",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*agents.ReActAgent)
	if !ok {
		t.Fatalf("subagent type = %T, want *ReActAgent", subagent)
	}
	if reactAgent.Name != "subagent-ReAct" {
		t.Fatalf("Name = %q, want %q", reactAgent.Name, "subagent-ReAct")
	}
}
