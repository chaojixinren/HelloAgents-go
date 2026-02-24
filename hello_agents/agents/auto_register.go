package agents

import (
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools/builtin"
)

func autoRegisterBuiltinTools(base *core.BaseAgent) {
	if base == nil || base.ToolRegistry == nil {
		return
	}

	if base.Config.SubagentEnabled {
		agentFactory := func(agentType string) (any, error) {
			llm := base.LLM
			if base.Config.SubagentUseLightLLM {
				llm = base.CreateLightLLM()
			}
			return DefaultSubagentFactory(agentType, llm, base.ToolRegistry, &base.Config)
		}
		base.RegisterTaskTool(builtin.AgentFactory(agentFactory))
	}
	if base.Config.TodoWriteEnabled {
		base.RegisterTodoWriteTool()
	}
	if base.Config.DevLogEnabled {
		base.RegisterDevLogTool()
	}
}
