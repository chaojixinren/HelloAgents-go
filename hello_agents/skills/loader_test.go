package skills

import (
	"os"
	"path/filepath"
	"testing"
)

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

	loader, err := NewSkillLoader(root)
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	skills := loader.ListSkills()
	if len(skills) != 1 || skills[0] != "" {
		t.Fatalf("ListSkills() = %#v, want [\"\"]", skills)
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

	loader, err := NewSkillLoader(root)
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

	loader, err := NewSkillLoader("")
	if err != nil {
		t.Fatalf("NewSkillLoader error: %v", err)
	}
	if loader.SkillsDir != "." {
		t.Fatalf("SkillsDir = %q, want %q", loader.SkillsDir, ".")
	}
}
