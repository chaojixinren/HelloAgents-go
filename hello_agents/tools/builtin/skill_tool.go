package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools"
)

// SkillTool allows agents to load domain skills on demand.
type SkillTool struct {
	tools.BaseTool
	skillLoader *skills.SkillLoader
}

func NewSkillTool(skillLoader *skills.SkillLoader) *SkillTool {
	descriptions := "（暂无可用技能）"
	if skillLoader != nil {
		descriptions = skillLoader.GetDescriptions()
	}

	base := tools.NewBaseTool(
		"Skill",
		fmt.Sprintf(`加载技能获取专业知识。

可用技能：
%s

何时使用：
- 任务明确匹配某个技能描述时，立即使用
- 开始领域特定工作之前
- 需要模型不具备的专业知识时

注意：加载技能后，请严格遵循技能说明来完成用户任务。`, descriptions),
		false,
	)
	base.Parameters = map[string]tools.ToolParameter{
		"skill": {
			Name:        "skill",
			Type:        "string",
			Description: "要加载的技能名称",
			Required:    true,
		},
		"args": {
			Name:        "args",
			Type:        "string",
			Description: "可选参数，将替换 SKILL.md 中的 $ARGUMENTS 占位符",
			Required:    false,
			Default:     "",
		},
	}

	return &SkillTool{
		BaseTool:    base,
		skillLoader: skillLoader,
	}
}

func (t *SkillTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *SkillTool) Run(parameters map[string]any) (resp tools.ToolResponse) {
	defer func() {
		if recovered := recover(); recovered != nil {
			errText := fmt.Sprintf("%v", recovered)
			resp = tools.Error(
				fmt.Sprintf("加载技能失败：%s", errText),
				tools.ToolErrorCodeInternalError,
				map[string]any{"params_input": parameters, "error": errText},
			)
		}
	}()

	skillName, _ := parameters["skill"].(string)
	if skillName == "" {
		return tools.Error(
			"必须指定技能名称",
			tools.ToolErrorCodeInvalidParam,
			map[string]any{"params_input": parameters},
		)
	}

	skill, err := t.skillLoader.GetSkill(skillName)
	if err != nil {
		return tools.Error(
			fmt.Sprintf("加载技能失败：%v", err),
			tools.ToolErrorCodeInternalError,
			map[string]any{"params_input": parameters, "error": err.Error()},
		)
	}
	if skill == nil {
		available := t.skillLoader.ListSkills()
		return tools.Error(
			fmt.Sprintf("技能 '%s' 不存在。可用技能：%s", skillName, strings.Join(available, ", ")),
			tools.ToolErrorCodeNotFound,
			map[string]any{"params_input": parameters, "available_skills": available},
		)
	}

	args := ""
	if raw, ok := parameters["args"]; ok && raw != nil {
		var castOK bool
		args, castOK = raw.(string)
		if !castOK {
			return tools.Error(
				fmt.Sprintf("args 参数必须是字符串类型，实际为 %T", raw),
				tools.ToolErrorCodeInvalidParam,
				map[string]any{"params_input": parameters},
			)
		}
	}

	content := strings.ReplaceAll(skill.Body, "$ARGUMENTS", args)
	resourcesHint := t.getResourcesHint(skill)

	fullContent := fmt.Sprintf(`<skill-loaded name="%s">
%s
%s
</skill-loaded>

✅ 技能已加载：%s
📝 描述：%s

请严格遵循上述技能说明来完成用户任务。`, skillName, content, resourcesHint, skill.Name, skill.Description)

	return tools.Success(
		fullContent,
		map[string]any{
			"name":           skill.Name,
			"description":    skill.Description,
			"loaded":         true,
			"token_estimate": len(fullContent),
			"has_resources":  resourcesHint != "",
		},
	)
}

func (t *SkillTool) getResourcesHint(skill *skills.Skill) string {
	if skill == nil {
		return ""
	}

	type folderInfo struct {
		name  string
		label string
	}
	folders := []folderInfo{
		{name: "scripts", label: "脚本"},
		{name: "references", label: "参考文档"},
		{name: "assets", label: "资源"},
		{name: "examples", label: "示例"},
	}

	lines := make([]string, 0)
	for _, folder := range folders {
		folderPath := filepath.Join(skill.Dir, folder.name)
		stat, err := os.Stat(folderPath)
		if err != nil || !stat.IsDir() {
			continue
		}

		entries, err := os.ReadDir(folderPath)
		if err != nil || len(entries) == 0 {
			continue
		}

		visible := make([]string, 0, len(entries))
		for _, entry := range entries {
			visible = append(visible, entry.Name())
		}

		display := visible
		if len(display) > 5 {
			display = display[:5]
		}

		fileList := strings.Join(display, ", ")
		if len(visible) > 5 {
			fileList += fmt.Sprintf(" 等 %d 个文件", len(visible))
		}
		lines = append(lines, fmt.Sprintf("  - %s：%s", folder.label, fileList))
	}

	if len(lines) == 0 {
		return ""
	}

	return "\n\n**可用资源**：\n" + strings.Join(lines, "\n")
}
