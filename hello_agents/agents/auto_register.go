package agents

import (
	"fmt"
	"os"
	"time"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools/builtin"
)

func autoRegisterBuiltinTools(base *core.BaseAgent) {
	if base == nil || base.ToolRegistry == nil {
		return
	}

	if base.Config.SubagentEnabled {
		registerTaskTool(base)
	}
	if base.Config.TodoWriteEnabled {
		registerTodoWriteTool(base)
	}
	if base.Config.DevLogEnabled {
		registerDevLogTool(base)
	}
}

func registerTaskTool(base *core.BaseAgent) {
	agentFactory := func(agentType string) (any, error) {
		llm := base.LLM
		if base.Config.SubagentUseLightLLM {
			llm = createLightLLM(base)
		}
		return DefaultSubagentFactory(agentType, llm, base.ToolRegistry, &base.Config)
	}

	taskTool := builtin.NewTaskTool(agentFactory, base.ToolRegistry)
	base.ToolRegistry.RegisterTool(taskTool, false)
}

func registerTodoWriteTool(base *core.BaseAgent) {
	todoTool := builtin.NewTodoWriteTool(".", base.Config.TodoWritePersistenceDir)
	base.ToolRegistry.RegisterTool(todoTool, false)
}

func registerDevLogTool(base *core.BaseAgent) {
	sessionID := ""
	if base.TraceLogger != nil {
		sessionID = base.TraceLogger.SessionID
	}
	if sessionID == "" {
		sessionID = generateFallbackSessionID()
	}

	devlogTool := builtin.NewDevLogTool(
		sessionID,
		base.Name,
		".",
		base.Config.DevLogPersistenceDir,
	)
	base.ToolRegistry.RegisterTool(devlogTool, false)
}

func generateFallbackSessionID() string {
	return fmt.Sprintf("s-%s-%04x", time.Now().Format("20060102-150405"), time.Now().UnixNano()&0xffff)
}

func createLightLLM(base *core.BaseAgent) *core.HelloAgentsLLM {
	model := base.Config.SubagentLightLLMModel
	if model == "" {
		model = base.LLM.Model
	}

	apiKey := base.LLM.APIKey
	baseURL := base.LLM.BaseURL
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}

	timeout := base.LLM.Timeout
	llm, err := core.NewHelloAgentsLLM(
		model,
		apiKey,
		baseURL,
		base.LLM.Temperature,
		base.LLM.MaxTokens,
		&timeout,
		nil,
	)
	if err != nil {
		return base.LLM
	}
	return llm
}
