package builtin

import (
	"fmt"
	"strconv"
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
	base := tools.NewBaseTool(
		"Task",
		"启动子代理处理特定的子任务，使用隔离的上下文。适用于：探索代码库、规划任务、实现功能等需要独立上下文的场景。",
		false,
	)
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
			Description: "子代理类型：react（推理行动）、reflection（反思）、plan（规划）、simple（简单对话）",
			Required:    false,
			Default:     "react",
		},
		"tool_filter": {
			Name:        "tool_filter",
			Type:        "string",
			Description: "工具过滤策略：readonly（只读工具）、full（完全访问）、none（无过滤）",
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
	t := &TaskTool{BaseTool: base, agentFactory: agentFactory, toolRegistry: toolRegistry}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *TaskTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *TaskTool) Run(parameters map[string]any) (response tools.ToolResponse) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response = tools.Error(
				fmt.Sprintf("子代理执行失败: %v", recovered),
				tools.ToolErrorCodeExecutionError,
				nil,
			)
		}
	}()

	start := time.Now()
	task, _ := parameters["task"].(string)
	agentType := "react"
	if raw, exists := parameters["agent_type"]; exists {
		agentType, _ = raw.(string)
	}
	toolFilterType := "none"
	if raw, exists := parameters["tool_filter"]; exists {
		toolFilterType, _ = raw.(string)
	}

	if task == "" {
		return tools.Error("参数 'task' 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}
	agentType = strings.ToLower(agentType)
	toolFilterType = strings.ToLower(toolFilterType)

	var maxSteps *int
	if raw, ok := parameters["max_steps"]; ok && raw != nil {
		if v, parsed := parseOptionalInt(raw); parsed {
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

func parseOptionalInt(value any) (int, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
