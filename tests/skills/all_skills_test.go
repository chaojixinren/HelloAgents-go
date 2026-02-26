package skills_test

import (
	"testing"

	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools/builtin"
)

// TestAllProjectSkillsLoadable verifies that every skill under skills/ can be
// discovered, parsed, and loaded by the framework — including full body content.
func TestAllProjectSkillsLoadable(t *testing.T) {
	loader, err := skills.NewSkillLoader("../../skills")
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

// TestAllSkillsViaSkillTool verifies each skill can be loaded via SkillTool.
func TestAllSkillsViaSkillTool(t *testing.T) {
	loader, err := skills.NewSkillLoader("../../skills")
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

// TestSkillDescriptionsNotEmpty verifies GetDescriptions returns all skills.
func TestSkillDescriptionsNotEmpty(t *testing.T) {
	loader, err := skills.NewSkillLoader("../../skills")
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
