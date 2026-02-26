// Skills 知识外化使用示例
//
// 对应 Python 版本: examples/skills_demo.py
// 演示如何使用 Skills 系统实现知识外化。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	fmt.Println("=== Skills 知识外化示例 ===")
	fmt.Println()

	// 示例 1: 使用项目自带的 skills 目录
	demoWithProjectSkills()

	// 示例 2: 自定义技能创建
	demoCustomSkill()
}

func demoWithProjectSkills() {
	fmt.Println("示例 1: 加载项目 skills 目录")
	fmt.Println(string(repeatRune('=', 50)))

	loader, err := skills.NewSkillLoader("skills")
	if err != nil {
		fmt.Printf("❌ 加载失败: %v\n", err)
		return
	}

	skillList := loader.ListSkills()
	fmt.Printf("发现 %d 个技能:\n", len(skillList))
	for _, name := range skillList {
		meta := loader.MetadataCache[name]
		desc := meta["description"]
		if len(desc) > 60 {
			desc = desc[:60] + "..."
		}
		fmt.Printf("  - %-20s %s\n", name, desc)
	}
	fmt.Println()

	// 按需加载一个技能
	if len(skillList) > 0 {
		name := skillList[0]
		skill, _ := loader.GetSkill(name)
		if skill != nil {
			bodyPreview := skill.Body
			if len(bodyPreview) > 200 {
				bodyPreview = bodyPreview[:200] + "..."
			}
			fmt.Printf("加载技能 '%s':\n", skill.Name)
			fmt.Printf("  描述: %s\n", skill.Description)
			fmt.Printf("  内容预览: %s\n", bodyPreview)
		}
	}
	fmt.Println()

	// 使用 SkillTool
	skillTool := builtin.NewSkillTool(loader)
	if len(skillList) > 0 {
		fmt.Println("通过 SkillTool 加载技能:")
		resp := skillTool.Run(map[string]any{
			"skill": skillList[0],
		})
		fmt.Printf("  Status: %s\n", resp.Status)
		preview := resp.Text
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Printf("  内容预览: %s\n", preview)
	}
	fmt.Println()
}

func demoCustomSkill() {
	fmt.Println("示例 2: 自定义技能")
	fmt.Println(string(repeatRune('=', 50)))

	tmpDir, _ := os.MkdirTemp("", "skills_demo")
	defer os.RemoveAll(tmpDir)

	// 创建自定义技能
	skillDir := filepath.Join(tmpDir, "greeting")
	os.MkdirAll(skillDir, 0o755)
	skillContent := `---
name: greeting
description: 生成个性化问候语的技能
---

# 问候语生成技能

## 使用方法
根据用户提供的 $ARGUMENTS 生成合适的问候语。

## 规则
1. 使用友好的语气
2. 根据时间段选择合适的问候
3. 可以加入个性化元素
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644)

	// 加载自定义技能
	loader, _ := skills.NewSkillLoader(tmpDir)
	fmt.Printf("自定义技能目录: %s\n", tmpDir)
	fmt.Printf("发现技能: %v\n", loader.ListSkills())

	skill, _ := loader.GetSkill("greeting")
	if skill != nil {
		fmt.Printf("技能名称: %s\n", skill.Name)
		fmt.Printf("技能描述: %s\n", skill.Description)
		fmt.Printf("技能内容:\n%s\n", skill.Body)
	}

	// 使用 SkillTool 加载并替换参数
	skillTool := builtin.NewSkillTool(loader)
	resp := skillTool.Run(map[string]any{
		"skill": "greeting",
		"args":  "张三",
	})
	fmt.Printf("\nSkillTool 输出 (status=%s):\n%s\n", resp.Status, resp.Text)
}

func repeatRune(r rune, n int) []rune {
	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}
	return result
}
