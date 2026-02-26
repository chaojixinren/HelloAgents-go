package agents

import (
	"testing"

	"helloagents-go/hello_agents/core/testutil"
)

func TestReflectionAgentRunBasicFlow(t *testing.T) {
	llm := testutil.NewMockLLM(
		"初次执行结果",
		"反思：结果看起来不错，满意度 9/10，我对结果满意。",
		"优化后的最终结果",
	)

	cfg := noTraceConfig()
	agent, err := NewReflectionAgentWithOptions("reflect-test", llm, "", cfg, 3, nil, false, 3)
	if err != nil {
		t.Fatalf("NewReflectionAgent error: %v", err)
	}

	result, err := agent.Run("写一首诗", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}

func TestReflectionAgentCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")
	cfg := noTraceConfig()

	agent, err := NewReflectionAgentWithOptions("test", llm, "", cfg, 2, nil, false, 3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if agent.MaxIterations != 2 {
		t.Fatalf("MaxIterations = %d, want 2", agent.MaxIterations)
	}
}
