package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// 默认 ReAct 提示词模板
var defaultReActPrompt = `你是一个具备推理和行动能力的AI助手。你可以通过思考分析问题，然后调用合适的工具来获取信息，最终给出准确的答案。

## 可用工具
{tools}

## 工作流程
请严格按照以下格式进行回应，每次只能执行一个步骤：

Thought: 分析问题，确定需要什么信息，制定研究策略。
Action: 选择合适的工具获取信息，格式为：
- ` + "`{tool_name}[{tool_input}]`" + `：调用工具获取信息。
- ` + "`Finish[研究结论]`" + `：当你有足够信息得出结论时。

## 重要提醒
1. 每次回应必须包含Thought和Action两部分
2. 工具调用的格式必须严格遵循：工具名[参数]
3. 只有当你确信有足够信息回答问题时，才使用Finish
4. 如果工具返回的信息不够，继续使用其他工具或相同工具的不同参数

## 当前任务
**Question:** {question}

## 执行历史
{history}

现在开始你的推理和行动：`

// ReActAgent ReAct (Reasoning and Acting) Agent。
// 结合推理和行动的智能体，适合需要外部信息的任务。
type ReActAgent struct {
	*core.BaseAgent
	toolRegistry    *tools.ToolRegistry
	maxSteps        int
	currentHistory  []string
	promptTemplate  string
}

// NewReActAgent 创建 ReActAgent。
func NewReActAgent(
	name string,
	llm *core.HelloAgentsLLM,
	toolRegistry *tools.ToolRegistry,
	systemPrompt string,
	config *core.Config,
	maxSteps int,
	customPrompt string,
) *ReActAgent {
	// 如果没有提供 toolRegistry，创建一个空的
	if toolRegistry == nil {
		toolRegistry = tools.NewToolRegistry()
	}

	if maxSteps <= 0 {
		maxSteps = 5
	}

	promptTemplate := customPrompt
	if promptTemplate == "" {
		promptTemplate = defaultReActPrompt
	}

	return &ReActAgent{
		BaseAgent:      core.NewBaseAgent(name, llm, systemPrompt, config),
		toolRegistry:   toolRegistry,
		maxSteps:       maxSteps,
		currentHistory: make([]string, 0),
		promptTemplate: promptTemplate,
	}
}

// Run 运行 ReAct Agent。
func (a *ReActAgent) Run(ctx context.Context, inputText string) (string, error) {
	a.currentHistory = make([]string, 0)

	fmt.Printf("\n🤖 %s 开始处理问题: %s\n", a.Name, inputText)

	for step := 1; step <= a.maxSteps; step++ {
		fmt.Printf("\n--- 第 %d 步 ---\n", step)

		// 构建提示词
		toolsDesc := a.toolRegistry.GetToolsDescription()
		historyStr := strings.Join(a.currentHistory, "\n")
		prompt := strings.ReplaceAll(a.promptTemplate, "{tools}", toolsDesc)
		prompt = strings.ReplaceAll(prompt, "{question}", inputText)
		prompt = strings.ReplaceAll(prompt, "{history}", historyStr)

		// 调用 LLM
		messages := []core.ChatMessage{
			{Role: core.RoleUser, Content: prompt},
		}
		response, err := a.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		if response == "" {
			fmt.Println("❌ 错误：LLM未能返回有效响应。")
			break
		}

		// 解析输出
		thought, action := a.parseOutput(response)

		if thought != "" {
			fmt.Printf("🤔 思考: %s\n", thought)
		}

		if action == "" {
			fmt.Println("⚠️ 警告：未能解析出有效的Action，流程终止。")
			break
		}

		// 检查是否完成
		if strings.HasPrefix(action, "Finish") {
			finalAnswer := a.parseActionInput(action)
			fmt.Printf("🎉 最终答案: %s\n", finalAnswer)

			a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
			a.AddMessage(core.NewMessage(finalAnswer, core.RoleAssistant, core.Time{}, nil))

			return finalAnswer, nil
		}

		// 执行工具调用
		toolName, toolInput := a.parseAction(action)
		if toolName == "" || toolInput == "" {
			a.currentHistory = append(a.currentHistory, "Observation: 无效的Action格式，请检查。")
			continue
		}

		fmt.Printf("🎬 行动: %s[%s]\n", toolName, toolInput)

		// 调用工具
		observation := a.toolRegistry.ExecuteTool(toolName, toolInput)
		fmt.Printf("👀 观察: %s\n", observation)

		// 更新历史
		a.currentHistory = append(a.currentHistory, fmt.Sprintf("Action: %s", action))
		a.currentHistory = append(a.currentHistory, fmt.Sprintf("Observation: %s", observation))
	}

	fmt.Println("⏰ 已达到最大步数，流程终止。")
	finalAnswer := "抱歉，我无法在限定步数内完成这个任务。"

	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.RoleAssistant, core.Time{}, nil))

	return finalAnswer, nil
}

// parseOutput 解析 LLM 输出，提取思考和行动。
func (a *ReActAgent) parseOutput(text string) (thought, action string) {
	thoughtRe := regexp.MustCompile(`Thought: (.*)`)
	actionRe := regexp.MustCompile(`Action: (.*)`)

	thoughtMatch := thoughtRe.FindStringSubmatch(text)
	actionMatch := actionRe.FindStringSubmatch(text)

	if len(thoughtMatch) > 1 {
		thought = strings.TrimSpace(thoughtMatch[1])
	}
	if len(actionMatch) > 1 {
		action = strings.TrimSpace(actionMatch[1])
	}

	return thought, action
}

// parseAction 解析行动文本，提取工具名称和输入。
func (a *ReActAgent) parseAction(actionText string) (toolName, toolInput string) {
	re := regexp.MustCompile(`(\w+)\[(.*)\]`)
	match := re.FindStringSubmatch(actionText)
	if len(match) >= 3 {
		return match[1], match[2]
	}
	return "", ""
}

// parseActionInput 解析行动输入。
func (a *ReActAgent) parseActionInput(actionText string) string {
	re := regexp.MustCompile(`\w+\[(.*)\]`)
	match := re.FindStringSubmatch(actionText)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// AddTool 添加工具到工具注册表。
func (a *ReActAgent) AddTool(tool tools.Tool, autoExpand bool) {
	a.toolRegistry.RegisterTool(tool, autoExpand)
}

// ListTools 列出所有可用工具。
func (a *ReActAgent) ListTools() []string {
	return a.toolRegistry.ListTools()
}
