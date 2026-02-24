package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
	Dir         string
}

func (s Skill) Scripts() []string {
	return collectFiles(filepath.Join(s.Dir, "scripts"))
}

func (s Skill) Examples() []string {
	return collectFiles(filepath.Join(s.Dir, "examples"))
}

func (s Skill) References() []string {
	return collectFiles(filepath.Join(s.Dir, "references"))
}

type SkillLoader struct {
	SkillsDir     string
	SkillsCache   map[string]Skill
	MetadataCache map[string]map[string]string
	metadataOrder []string
}

func NewSkillLoader(skillsDir string) (*SkillLoader, error) {
	if skillsDir == "" {
		skillsDir = "."
	}
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, err
	}
	loader := &SkillLoader{
		SkillsDir:     skillsDir,
		SkillsCache:   map[string]Skill{},
		MetadataCache: map[string]map[string]string{},
		metadataOrder: []string{},
	}
	loader.scanSkills()
	return loader, nil
}

func (l *SkillLoader) scanSkills() {
	entries, err := os.ReadDir(l.SkillsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(l.SkillsDir, entry.Name())
		skillMD := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			continue
		}
		meta := l.parseFrontmatterOnly(skillMD)
		if len(meta) == 0 {
			continue
		}
		name := entry.Name()
		if rawName, exists := meta["name"]; exists {
			name = rawName
		}
		if _, exists := l.MetadataCache[name]; !exists {
			l.metadataOrder = append(l.metadataOrder, name)
		}
		l.MetadataCache[name] = map[string]string{
			"name":        name,
			"description": meta["description"],
			"path":        skillMD,
			"dir":         skillDir,
		}
	}
}

func (l *SkillLoader) parseFrontmatterOnly(path string) map[string]string {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(contentBytes)
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil
	}
	metadata := parseYAMLFrontmatter(match[1])
	if len(metadata) == 0 {
		return nil
	}
	// Python behavior: frontmatter-only scan requires both keys to exist.
	if _, ok := metadata["name"]; !ok {
		return nil
	}
	if _, ok := metadata["description"]; !ok {
		return nil
	}
	return map[string]string{
		"name":        anyToString(metadata["name"]),
		"description": anyToString(metadata["description"]),
	}
}

func (l *SkillLoader) GetDescriptions() string {
	if len(l.MetadataCache) == 0 {
		return "（暂无可用技能）"
	}
	lines := make([]string, 0, len(l.MetadataCache))
	for _, name := range l.metadataOrder {
		meta, ok := l.MetadataCache[name]
		if !ok {
			continue
		}
		lines = append(lines, "- "+name+": "+meta["description"])
	}
	return strings.Join(lines, "\n")
}

func (l *SkillLoader) GetSkill(name string) (*Skill, error) {
	if cached, ok := l.SkillsCache[name]; ok {
		return &cached, nil
	}
	meta, ok := l.MetadataCache[name]
	if !ok {
		return nil, nil
	}
	contentBytes, err := os.ReadFile(meta["path"])
	if err != nil {
		return nil, nil
	}
	content := string(contentBytes)
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 3 {
		return nil, nil
	}
	parsedMeta := parseYAMLFrontmatter(match[1])
	if len(parsedMeta) == 0 {
		return nil, nil
	}
	skillName := name
	if parsedName, exists := parsedMeta["name"]; exists {
		skillName = anyToString(parsedName)
	}
	description := ""
	if parsedDescription, exists := parsedMeta["description"]; exists {
		description = anyToString(parsedDescription)
	}
	skill := Skill{
		Name:        skillName,
		Description: description,
		Body:        strings.TrimSpace(match[2]),
		Path:        meta["path"],
		Dir:         meta["dir"],
	}
	l.SkillsCache[name] = skill
	return &skill, nil
}

func (l *SkillLoader) ListSkills() []string {
	out := make([]string, 0, len(l.metadataOrder))
	for _, name := range l.metadataOrder {
		if _, ok := l.MetadataCache[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

func (l *SkillLoader) Reload() {
	l.SkillsCache = map[string]Skill{}
	l.MetadataCache = map[string]map[string]string{}
	l.metadataOrder = []string{}
	l.scanSkills()
}

func collectFiles(dir string) []string {
	if _, err := os.Stat(dir); err != nil {
		return []string{}
	}
	out := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return []string{}
	}
	return out
}

func parseYAMLFrontmatter(frontmatter string) map[string]any {
	var metadata map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil
	}
	return metadata
}

func anyToString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
