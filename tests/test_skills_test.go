package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools/builtin"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// SkillLoader tests (from skills/loader_test.go)
// ---------------------------------------------------------------------------

func TestSkillLoaderFrontmatterAllowsEmptyNameAndDescription(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	content := `---
name: ""
description: ""
---
body
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md error: %v", err)
	}

	loader, err := skills.NewSkillLoader(root)
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	skls := loader.ListSkills()
	if len(skls) != 1 || skls[0] != "" {
		t.Fatalf("ListSkills() = %#v, want [\"\"]", skls)
	}
}

func TestSkillLoaderGetSkillFallsBackToRequestedName(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	initial := `---
name: demo
description: demo
---
body
`
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial SKILL.md error: %v", err)
	}

	loader, err := skills.NewSkillLoader(root)
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}

	updated := `---
description: demo
---
body
`
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write updated SKILL.md error: %v", err)
	}

	skill, err := loader.GetSkill("demo")
	if err != nil {
		t.Fatalf("GetSkill error: %v", err)
	}
	if skill == nil {
		t.Fatalf("GetSkill returned nil")
	}
	if skill.Name != "demo" {
		t.Fatalf("skill.Name = %q, want %q", skill.Name, "demo")
	}
}

func TestNewSkillLoaderEmptyDirUsesCurrentDirectoryLikePath(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
	})

	loader, err := skills.NewSkillLoader("")
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	if loader.SkillsDir != "." {
		t.Fatalf("SkillsDir = %q, want %q", loader.SkillsDir, ".")
	}
}

// ---------------------------------------------------------------------------
// All skills validation (from skills/all_skills_test.go)
// ---------------------------------------------------------------------------

func TestAllProjectSkillsLoadable(t *testing.T) {
	loader, err := skills.NewSkillLoader("../skills")
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}

	skillList := loader.ListSkills()
	if len(skillList) == 0 {
		t.Fatal("no skills found in skills/ directory")
	}

	expectedSkills := []string{
		"ASR", "LLM", "TTS", "VLM",
		"docx", "finance", "frontend-design", "gift-evaluator",
		"image-generation", "pdf", "Podcast Generate", "pptx",
		"Video Generation", "video-understand",
		"web-reader", "web-search", "xlsx",
	}

	if len(skillList) != len(expectedSkills) {
		t.Errorf("found %d skills, expected %d. got: %v", len(skillList), len(expectedSkills), skillList)
	}

	for _, name := range expectedSkills {
		t.Run("skill_"+name, func(t *testing.T) {
			meta, exists := loader.MetadataCache[name]
			if !exists {
				t.Fatalf("skill %q not found in MetadataCache", name)
			}
			if meta["name"] == "" {
				t.Errorf("skill %q has empty name in metadata", name)
			}
			if meta["description"] == "" {
				t.Errorf("skill %q has empty description in metadata", name)
			}

			skill, err := loader.GetSkill(name)
			if err != nil {
				t.Fatalf("GetSkill(%q) error: %v", name, err)
			}
			if skill == nil {
				t.Fatalf("GetSkill(%q) returned nil", name)
			}
			if skill.Body == "" {
				t.Errorf("skill %q has empty Body", name)
			}
			if skill.Name == "" {
				t.Errorf("skill %q has empty Name", name)
			}
			if skill.Description == "" {
				t.Errorf("skill %q has empty Description", name)
			}
		})
	}
}

func TestAllSkillsViaSkillTool(t *testing.T) {
	loader, err := skills.NewSkillLoader("../skills")
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}

	tool := builtin.NewSkillTool(loader)

	for _, name := range loader.ListSkills() {
		t.Run("skill_tool_"+name, func(t *testing.T) {
			resp := tool.Run(map[string]any{
				"skill": name,
			})
			if resp.Status != "success" {
				t.Fatalf("SkillTool.Run(%q) status=%q, text=%q", name, resp.Status, resp.Text)
			}
			if resp.Data == nil || resp.Data["loaded"] != true {
				t.Errorf("SkillTool.Run(%q) data.loaded != true", name)
			}
		})
	}
}

func TestSkillDescriptionsNotEmpty(t *testing.T) {
	loader, err := skills.NewSkillLoader("../skills")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	desc := loader.GetDescriptions()
	if desc == "（暂无可用技能）" {
		t.Fatal("GetDescriptions returned empty placeholder")
	}

	for _, name := range loader.ListSkills() {
		if !containsSubstring(desc, name) {
			t.Errorf("GetDescriptions() does not contain skill %q", name)
		}
	}
}

// ---------------------------------------------------------------------------
// SkillTool edge cases (from tools/builtin/skill_tool_test.go)
// ---------------------------------------------------------------------------

func TestSkillToolWhitespaceSkillNameIsNotEmpty(t *testing.T) {
	loader, err := skills.NewSkillLoader(t.TempDir())
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	tool := builtin.NewSkillTool(loader)

	resp := tool.Run(map[string]any{
		"skill": "   ",
	})

	if resp.Status != "error" {
		t.Fatalf("status = %q, want %q", resp.Status, "error")
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != "NOT_FOUND" {
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

	if resp.Status != "error" {
		t.Fatalf("status = %q, want %q", resp.Status, "error")
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != "INVALID_PARAM" {
		t.Fatalf("error = %#v, want INVALID_PARAM", resp.ErrorInfo)
	}
}
