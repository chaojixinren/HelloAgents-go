package agents

import (
	"fmt"
	"strings"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

func CreateAgent(agentType string, name string, llm *core.HelloAgentsLLM, toolRegistry *tools.ToolRegistry, config *core.Config, systemPrompt string) (core.Agent, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm is required")
	}
	if name == "" {
		name = strings.ToLower(agentType) + "_agent"
	}

	switch strings.ToLower(agentType) {
	case "react":
		return NewReActAgent(name, llm, systemPrompt, config, toolRegistry, 5)
	case "reflection":
		return NewReflectionAgent(name, llm, systemPrompt, config, toolRegistry)
	case "plan":
		return NewPlanSolveAgent(name, llm, systemPrompt, config, toolRegistry)
	case "simple":
		return NewSimpleAgent(name, llm, systemPrompt, config, toolRegistry)
	default:
		return nil, fmt.Errorf("不支持的 agent_type: %s。支持的类型: react, reflection, plan, simple", agentType)
	}
}

func DefaultSubagentFactory(agentType string, llm *core.HelloAgentsLLM, toolRegistry *tools.ToolRegistry, config *core.Config) (core.Agent, error) {
	cfg := config
	if cfg == nil {
		defaultCfg := core.DefaultConfig()
		cfg = &defaultCfg
	}

	systemPrompt := getSystemPromptForType(agentType)
	subagent, err := CreateAgent(agentType, "subagent-"+strings.ToLower(agentType), llm, toolRegistry, cfg, systemPrompt)
	if err != nil {
		return nil, err
	}

	if r, ok := subagent.(*ReActAgent); ok && cfg.SubagentMaxSteps > 0 {
		r.MaxSteps = cfg.SubagentMaxSteps
	}
	return subagent, nil
}

func getSystemPromptForType(agentType string) string {
	switch strings.ToLower(agentType) {
	case "react":
		return `你是一个高效的任务执行专家。

目标：快速完成指定的子任务。

规则：
- 使用可用工具高效完成任务
- 保持输出简洁明了
- 在规定步数内完成`
	case "reflection":
		return `你是一个反思型专家。

目标：深入分析问题并提供高质量的解决方案。

规则：
- 先给出初步方案
- 反思并改进方案
- 输出最终优化结果`
	case "plan", "plan_solve", "plan-and-solve", "planandsolve", "plan_and_solve":
		return `你是一个任务规划专家。

目标：将复杂任务分解为可执行的步骤。

规则：
- 分析任务需求
- 制定详细的执行计划
- 标注步骤依赖关系`
	default:
		return `你是一个简洁高效的助手。

目标：直接回答问题或完成任务。

规则：
- 保持回答简洁
- 直接给出结果
- 避免冗余信息`
	}
}
