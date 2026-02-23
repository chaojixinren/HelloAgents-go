package builtin

import (
	"fmt"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

type AgentFactory func(agentType string) (any, error)

type subagentRunnerWithFilter interface {
	RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any
}

type TaskTool struct {
	tools.BaseTool
	agentFactory AgentFactory
	toolRegistry *tools.ToolRegistry
}

func NewTaskTool(agentFactory AgentFactory, toolRegistry *tools.ToolRegistry) *TaskTool {
	base := tools.NewBaseTool("Task", "启动子代理处理特定的子任务，使用隔离的上下文。", false)
	base.Parameters = map[string]tools.ToolParameter{
		"task": {
			Name:        "task",
			Type:        "string",
			Description: "子任务的详细描述，告诉子代理具体要做什么",
			Required:    true,
		},
		"agent_type": {
			Name:        "agent_type",
			Type:        "string",
			Description: "子代理类型：react/reflection/plan/simple",
			Required:    false,
			Default:     "react",
		},
		"tool_filter": {
			Name:        "tool_filter",
			Type:        "string",
			Description: "工具过滤策略：readonly/full/none",
			Required:    false,
			Default:     "none",
		},
		"max_steps": {
			Name:        "max_steps",
			Type:        "integer",
			Description: "最大步数限制（覆盖默认配置）",
			Required:    false,
		},
	}
	return &TaskTool{BaseTool: base, agentFactory: agentFactory, toolRegistry: toolRegistry}
}

func (t *TaskTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *TaskTool) Run(parameters map[string]any) tools.ToolResponse {
	start := time.Now()
	task := strings.TrimSpace(fmt.Sprintf("%v", parameters["task"]))
	agentType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", parameters["agent_type"])))
	toolFilterType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", parameters["tool_filter"])))

	if task == "" || task == "<nil>" {
		return tools.Error("参数 'task' 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}
	if agentType == "" || agentType == "<nil>" {
		agentType = "react"
	}
	if toolFilterType == "" || toolFilterType == "<nil>" {
		toolFilterType = "none"
	}

	var maxSteps *int
	if raw, ok := parameters["max_steps"]; ok {
		v := intFromAny(raw)
		if v > 0 {
			maxSteps = &v
		}
	}

	if t.agentFactory == nil {
		return tools.Error("agent_factory 未配置", tools.ToolErrorCodeInternalError, nil)
	}

	subagent, err := t.agentFactory(agentType)
	if err != nil {
		return tools.Error(fmt.Sprintf("不支持的 agent_type: %s。%v", agentType, err), tools.ToolErrorCodeInvalidParam, nil)
	}

	toolFilter := t.createToolFilter(toolFilterType)

	runner, ok := subagent.(subagentRunnerWithFilter)
	if !ok {
		return tools.Error("子代理不支持 run_as_subagent 接口", tools.ToolErrorCodeExecutionError, map[string]any{"agent_type": agentType})
	}
	result := runner.RunAsSubagent(task, toolFilter, true, maxSteps)

	elapsedMS := time.Since(start).Milliseconds()
	summary := fmt.Sprintf("%v", result["summary"])
	metadata, _ := result["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	payload := map[string]any{"agent_type": agentType, "task": task}
	for k, v := range metadata {
		payload[k] = v
	}

	success, _ := result["success"].(bool)
	if success {
		return tools.Success(
			fmt.Sprintf("[SubAgent-%s] 任务完成\n\n%s", agentType, summary),
			payload,
			map[string]any{"time_ms": elapsedMS},
		)
	}

	return tools.Partial(
		fmt.Sprintf("[SubAgent-%s] 任务未完全完成\n\n%s", agentType, summary),
		payload,
		map[string]any{"time_ms": elapsedMS},
	)
}

func (t *TaskTool) createToolFilter(filterType string) tools.ToolFilter {
	switch filterType {
	case "readonly":
		return tools.NewReadOnlyFilter(nil)
	case "full":
		return tools.NewFullAccessFilter(nil)
	default:
		return nil
	}
}
