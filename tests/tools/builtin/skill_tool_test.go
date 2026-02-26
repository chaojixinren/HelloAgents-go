package builtin_test

import (
	"os"
	"path/filepath"
	"testing"

	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func TestSkillToolWhitespaceSkillNameIsNotEmpty(t *testing.T) {
	loader, err := skills.NewSkillLoader(t.TempDir())
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	tool := builtin.NewSkillTool(loader)

	resp := tool.Run(map[string]any{
		"skill": "   ",
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeNotFound {
		t.Fatalf("error = %#v, want NOT_FOUND", resp.ErrorInfo)
	}
}

func TestSkillToolNonStringArgsReturnsInternalErrorWhenSkillExists(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	skillContent := `---
name: demo
description: demo skill
---
hello $ARGUMENTS
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md error: %v", err)
	}

	loader, err := skills.NewSkillLoader(root)
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	tool := builtin.NewSkillTool(loader)

	resp := tool.Run(map[string]any{
		"skill": "demo",
		"args":  123,
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("error = %#v, want INVALID_PARAM", resp.ErrorInfo)
	}
}
