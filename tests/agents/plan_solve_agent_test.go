package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func TestPlannerStepsFromArgsKeepsWhitespaceAndEmptyItems(t *testing.T) {
	args := map[string]any{
		"steps": []any{"  step one  ", "", "step three"},
	}

	got := agents.ExportPlannerStepsFromArgs(args)
	want := []string{"  step one  ", "", "step three"}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExecutorKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	executor := agents.NewExecutor(
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		nil,
		false,
		0,
	)
	if executor.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", executor.MaxToolIterations)
	}
}

func TestPlanSolveAgentKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewPlanSolveAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		"",
		"",
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewPlanSolveAgentWithOptions error: %v", err)
	}
	if agent.Executor.MaxToolIterations != 0 {
		t.Fatalf("Executor.MaxToolIterations = %d, want 0", agent.Executor.MaxToolIterations)
	}
}
