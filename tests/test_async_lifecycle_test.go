package tests_test

import (
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/core/testutil"
)

// ---------------------------------------------------------------------------
// SimpleAgent Arun / ArunStream (from agents/simple_agent_run_test.go)
// ---------------------------------------------------------------------------

func TestSimpleAgentArun(t *testing.T) {
	llm := testutil.NewMockLLM("arun-result")
	agent, err := agents.NewSimpleAgent("test", llm, "system", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var startCalled, finishCalled bool
	hooks := core.Hooks{
		OnStart:  func(e core.AgentEvent) error { startCalled = true; return nil },
		OnFinish: func(e core.AgentEvent) error { finishCalled = true; return nil },
	}

	result, err := agent.Arun("hi", hooks, nil)
	if err != nil {
		t.Fatalf("Arun error: %v", err)
	}
	if result != "arun-result" {
		t.Fatalf("result = %q, want 'arun-result'", result)
	}
	if !startCalled {
		t.Fatal("OnStart hook was not called")
	}
	if !finishCalled {
		t.Fatal("OnFinish hook was not called")
	}
}

func TestSimpleAgentArunStream(t *testing.T) {
	llm := testutil.NewMockLLM("stream-result")
	agent, err := agents.NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	events := agent.ArunStream("hi", nil)
	var types []core.EventType
	for event := range events {
		types = append(types, event.Type)
	}

	if len(types) < 2 {
		t.Fatalf("expected at least 2 events (start + finish), got %d", len(types))
	}
	if types[0] != core.AgentStart {
		t.Fatalf("first event = %q, want agent_start", types[0])
	}
}

// ---------------------------------------------------------------------------
// BaseAgent Arun lifecycle (from core/agent_test.go)
// ---------------------------------------------------------------------------

func TestBaseAgentArunUsesRunDelegate(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	called := false
	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		called = true
		return "delegated:" + inputText, nil
	})

	result, err := agent.Arun("hello", core.Hooks{}, nil)
	if err != nil {
		t.Fatalf("Arun() error = %v", err)
	}
	if !called {
		t.Fatalf("Arun() did not invoke run delegate")
	}
	if result != "delegated:hello" {
		t.Fatalf("Arun() result = %q, want %q", result, "delegated:hello")
	}
}

func TestBaseAgentArunStreamUsesRunDelegate(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		return "stream:" + inputText, nil
	})

	events := agent.ArunStream("hello", nil)
	finishSeen := false
	for event := range events {
		if event.Type != core.AgentFinish {
			continue
		}
		finishSeen = true
		if got := event.Data["result"]; got != "stream:hello" {
			t.Fatalf("AgentFinish result = %v, want %q", got, "stream:hello")
		}
	}
	if !finishSeen {
		t.Fatalf("AgentFinish event not emitted")
	}
}
