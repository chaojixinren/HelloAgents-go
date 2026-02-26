package agents_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/core/testutil"
	"helloagents-go/hello_agents/tools"
)

func TestReActAgentRunBasicFlow(t *testing.T) {
	thoughtCall := testutil.MockToolCallResponse("Thought", "call-1", map[string]any{
		"reasoning": "我需要思考这个问题",
	})
	finishCall := testutil.MockToolCallResponse("Finish", "call-2", map[string]any{
		"answer": "最终答案是42",
	})

	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{thoughtCall, finishCall},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	registry := tools.NewToolRegistry(nil)

	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, registry, 10)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	result, err := agent.Run("什么是42？", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}

func TestReActAgentRunWithMaxStepsExceeded(t *testing.T) {
	thoughtCall := testutil.MockToolCallResponse("Thought", "call-1", map[string]any{
		"reasoning": "让我继续思考",
	})

	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{
			thoughtCall, thoughtCall, thoughtCall, thoughtCall, thoughtCall,
		},
		Responses: []string{"超过最大步数的回退响应"},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	registry := tools.NewToolRegistry(nil)

	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, registry, 2)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	result, err := agent.Run("思考这个问题", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty even when max steps exceeded")
	}
}

func TestReActAgentArun(t *testing.T) {
	finishCall := testutil.MockToolCallResponse("Finish", "call-1", map[string]any{
		"answer": "直接回答",
	})
	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{finishCall},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, nil, 5)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	var started bool
	hooks := core.Hooks{
		OnStart: func(e core.AgentEvent) error { started = true; return nil },
	}

	result, err := agent.Arun("hi", hooks, nil)
	if err != nil {
		t.Fatalf("Arun error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
	if !started {
		t.Fatal("OnStart hook was not called")
	}
}
