package agents_test

import (
	"os"
	"path/filepath"
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

func TestSimpleAgentBuildMessagesKeepsWhitespaceSystemPrompt(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"   ",
		&cfg,
		nil,
		false,
		3,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}

	messages := agent.ExportBuildMessages("hello")
	if len(messages) < 2 {
		t.Fatalf("messages length = %d, want at least 2", len(messages))
	}
	if messages[0]["role"] != "system" || messages[0]["content"] != "   " {
		t.Fatalf("first message = %#v, want whitespace system prompt preserved", messages[0])
	}
}

func TestSimpleAgentKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}
	if agent.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", agent.MaxToolIterations)
	}
}

func TestNewSimpleAgentReturnsErrorWhenTraceLoggerInitFails(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = true
	cfg.TraceDir = filepath.Join(blocker, "traces")
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	_, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		3,
	)
	if err == nil {
		t.Fatalf("NewSimpleAgentWithOptions should return error when trace logger init fails")
	}
}

type expandableTestTool struct {
	tools.BaseTool
	sub tools.Tool
}

func newExpandableTestTool() *expandableTestTool {
	base := tools.NewBaseTool("parent", "parent", true)
	subBase := tools.NewBaseTool("child", "child", false)
	return &expandableTestTool{
		BaseTool: base,
		sub:      &subBase,
	}
}

func (t *expandableTestTool) GetExpandedTools() []tools.Tool {
	return []tools.Tool{t.sub}
}

func TestSimpleAgentAddToolDefaultAutoExpandTrueLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		3,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}

	agent.AddTool(newExpandableTestTool())

	if agent.ToolRegistry.GetTool("child") == nil {
		t.Fatalf("child tool should be registered when auto_expand defaults to true")
	}
	if agent.ToolRegistry.GetTool("parent") != nil {
		t.Fatalf("parent tool should not remain when auto expansion is applied")
	}
}
