package agents

import (
	"testing"

	"helloagents-go/hello_agents/core"
)

func TestDefaultSubagentFactoryAssignsZeroSubagentMaxStepsLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	cfg.SubagentMaxSteps = 0

	subagent, err := DefaultSubagentFactory(
		"react",
		&core.HelloAgentsLLM{Model: "test-model"},
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*ReActAgent)
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

	subagent, err := DefaultSubagentFactory(
		"ReAct",
		&core.HelloAgentsLLM{Model: "test-model"},
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*ReActAgent)
	if !ok {
		t.Fatalf("subagent type = %T, want *ReActAgent", subagent)
	}
	if reactAgent.Name != "subagent-ReAct" {
		t.Fatalf("Name = %q, want %q", reactAgent.Name, "subagent-ReAct")
	}
}
