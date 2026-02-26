package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core/testutil"
)

func TestPlanSolveAgentCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")
	cfg := noTraceConfig()

	agent, err := agents.NewPlanSolveAgentWithOptions("ps-test", llm, "", cfg, "", "", nil, false, 3)
	if err != nil {
		t.Fatalf("NewPlanSolveAgentWithOptions error: %v", err)
	}

	if agent.AgentType != "PlanSolveAgent" {
		t.Fatalf("AgentType = %q, want PlanSolveAgent", agent.AgentType)
	}
}

func TestPlannerAndExecutorCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")

	planner := agents.NewPlanner(llm, "")
	if planner == nil {
		t.Fatal("NewPlanner returned nil")
	}

	executor := agents.NewExecutor(llm, "", nil, false, 3)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}
}

func TestPlannerStepsFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected int
	}{
		{"nil kwargs", nil, 0},
		{"missing key", map[string]any{}, 0},
		{"with steps", map[string]any{"steps": []any{"step1", "step2"}}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agents.ExportPlannerStepsFromArgs(tt.input)
			if len(got) != tt.expected {
				t.Fatalf("len(plannerStepsFromArgs()) = %d, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestPlanSolveAgentRunBasicFlow(t *testing.T) {
	llm := testutil.NewMockLLM(
		"步骤1: 分析问题\n步骤2: 收集信息\n步骤3: 生成答案",
		"执行步骤1的结果",
		"执行步骤2的结果",
		"执行步骤3的结果：最终答案是42",
	)

	cfg := noTraceConfig()
	agent, err := agents.NewPlanSolveAgentWithOptions("ps-test", llm, "", cfg, "", "", nil, false, 3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	result, err := agent.Run("解决这个复杂问题", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}
