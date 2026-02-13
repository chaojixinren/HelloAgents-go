package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"helloagents-go/HelloAgents-go/core"
)

// 默认规划器提示词模板
var defaultPlannerPrompt = `
你是一个顶级的AI规划专家。你的任务是将用户提出的复杂问题分解成一个由多个简单步骤组成的行动计划。
请确保计划中的每个步骤都是一个独立的、可执行的子任务，并且严格按照逻辑顺序排列。
你的输出必须是一个Python列表，其中每个元素都是一个描述子任务的字符串。

问题: {question}

请严格按照以下格式输出你的计划:
` + "```python" + `
["步骤1", "步骤2", "步骤3", ...]
` + "```" + `
`

// 默认执行器提示词模板
var defaultExecutorPrompt = `
你是一位顶级的AI执行专家。你的任务是严格按照给定的计划，一步步地解决问题。
你将收到原始问题、完整的计划、以及到目前为止已经完成的步骤和结果。
请你专注于解决"当前步骤"，并仅输出该步骤的最终答案，不要输出任何额外的解释或对话。

# 原始问题:
{question}

# 完整计划:
{plan}

# 历史步骤与结果:
{history}

# 当前步骤:
{current_step}

请仅输出针对"当前步骤"的回答:
`

// Planner 规划器 - 负责将复杂问题分解为简单步骤
type Planner struct {
	llmClient     *core.HelloAgentsLLM
	promptTemplate string
}

// NewPlanner 创建规划器
func NewPlanner(llmClient *core.HelloAgentsLLM, promptTemplate string) *Planner {
	if promptTemplate == "" {
		promptTemplate = defaultPlannerPrompt
	}
	return &Planner{
		llmClient:      llmClient,
		promptTemplate: promptTemplate,
	}
}

// Plan 生成执行计划
func (p *Planner) Plan(ctx context.Context, question string) []string {
	prompt := strings.ReplaceAll(p.promptTemplate, "{question}", question)
	messages := []core.ChatMessage{
		{Role: core.RoleUser, Content: prompt},
	}

	fmt.Println("--- 正在生成计划 ---")
	responseText, err := p.llmClient.Invoke(ctx, messages, nil)
	if err != nil {
		fmt.Printf("❌ 调用 LLM 失败: %s\n", err)
		return nil
	}
	responseText = strings.TrimSpace(responseText)
	fmt.Printf("✅ 计划已生成:\n%s\n", responseText)

	// 解析计划
	plan := parsePythonList(responseText)
	if len(plan) == 0 {
		fmt.Println("❌ 解析计划失败")
		return nil
	}

	return plan
}

// parsePythonList 解析类似 Python 列表格式的字符串
// 支持: ["步骤1", "步骤2", "步骤3"] 或 ['步骤1', '步骤2', '步骤3']
func parsePythonList(text string) []string {
	// 首先尝试提取代码块中的内容
	codeBlockPattern := regexp.MustCompile("```(?:python)?\\s*\\n?([\\s\\S]*?)```")
	if match := codeBlockPattern.FindStringSubmatch(text); match != nil {
		text = strings.TrimSpace(match[1])
	}

	// 检查是否是列表格式
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") {
		return nil
	}

	// 移除最外层的方括号
	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	// 解析列表元素
	var result []string
	var current strings.Builder
	inString := false
	stringQuote := byte(0)
	depth := 0

	for i := 0; i < len(text); i++ {
		char := text[i]

		// 处理转义字符
		if inString && char == '\\' && i+1 < len(text) {
			next := text[i+1]
			switch next {
			case 'n':
				current.WriteByte('\n')
			case 't':
				current.WriteByte('\t')
			case '\\':
				current.WriteByte('\\')
			case '"', '\'':
				current.WriteByte(next)
			default:
				current.WriteByte(char)
				current.WriteByte(next)
			}
			i++
			continue
		}

		// 处理字符串边界
		if (char == '"' || char == '\'') && depth == 0 {
			if !inString {
				inString = true
				stringQuote = char
			} else if char == stringQuote {
				inString = false
				stringQuote = 0
				// 字符串结束，添加到结果
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteByte(char)
			}
			continue
		}

		// 在字符串内部
		if inString {
			current.WriteByte(char)
			continue
		}

		// 处理嵌套结构
		if char == '[' || char == '{' {
			depth++
			current.WriteByte(char)
		} else if char == ']' || char == '}' {
			depth--
			current.WriteByte(char)
		} else if char == ',' && depth == 0 {
			// 分隔符，跳过
			continue
		} else if !isWhitespace(char) {
			current.WriteByte(char)
		}
	}

	return result
}

// isWhitespace 检查字符是否是空白字符
func isWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

// Executor 执行器 - 负责按计划逐步执行
type Executor struct {
	llmClient      *core.HelloAgentsLLM
	promptTemplate string
}

// NewExecutor 创建执行器
func NewExecutor(llmClient *core.HelloAgentsLLM, promptTemplate string) *Executor {
	if promptTemplate == "" {
		promptTemplate = defaultExecutorPrompt
	}
	return &Executor{
		llmClient:      llmClient,
		promptTemplate: promptTemplate,
	}
}

// Execute 按计划执行任务
func (e *Executor) Execute(ctx context.Context, question string, plan []string) string {
	history := ""
	finalAnswer := ""

	fmt.Println("\n--- 正在执行计划 ---")
	for i, step := range plan {
		fmt.Printf("\n-> 正在执行步骤 %d/%d: %s\n", i+1, len(plan), step)

		// 构建提示词
		prompt := strings.ReplaceAll(e.promptTemplate, "{question}", question)
		prompt = strings.ReplaceAll(prompt, "{plan}", formatPlan(plan))
		if history == "" {
			prompt = strings.ReplaceAll(prompt, "{history}", "无")
		} else {
			prompt = strings.ReplaceAll(prompt, "{history}", history)
		}
		prompt = strings.ReplaceAll(prompt, "{current_step}", step)

		messages := []core.ChatMessage{
			{Role: core.RoleUser, Content: prompt},
		}

		responseText, err := e.llmClient.Invoke(ctx, messages, nil)
		if err != nil {
			responseText = fmt.Sprintf("执行失败: %s", err)
		}
		responseText = strings.TrimSpace(responseText)

		history += fmt.Sprintf("步骤 %d: %s\n结果: %s\n\n", i+1, step, responseText)
		finalAnswer = responseText
		fmt.Printf("✅ 步骤 %d 已完成，结果: %s\n", i+1, finalAnswer)
	}

	return finalAnswer
}

// formatPlan 将计划列表格式化为字符串
func formatPlan(plan []string) string {
	var sb strings.Builder
	for i, step := range plan {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	return strings.TrimSpace(sb.String())
}

// PlanAndSolveAgent 分解规划与逐步执行的智能体
type PlanAndSolveAgent struct {
	*core.BaseAgent
	planner  *Planner
	executor *Executor
}

// NewPlanAndSolveAgent 创建 PlanAndSolveAgent
func NewPlanAndSolveAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	config *core.Config,
	customPrompts map[string]string,
) *PlanAndSolveAgent {
	var plannerPrompt, executorPrompt string
	if customPrompts != nil {
		plannerPrompt = customPrompts["planner"]
		executorPrompt = customPrompts["executor"]
	}

	return &PlanAndSolveAgent{
		BaseAgent: core.NewBaseAgent(name, llm, systemPrompt, config),
		planner:   NewPlanner(llm, plannerPrompt),
		executor:  NewExecutor(llm, executorPrompt),
	}
}

// Run 运行 Plan and Solve Agent
func (a *PlanAndSolveAgent) Run(ctx context.Context, inputText string) (string, error) {
	fmt.Printf("\n🤖 %s 开始处理问题: %s\n", a.Name, inputText)

	// 1. 生成计划
	plan := a.planner.Plan(ctx, inputText)
	if len(plan) == 0 {
		finalAnswer := "无法生成有效的行动计划，任务终止。"
		fmt.Printf("\n--- 任务终止 ---\n%s\n", finalAnswer)

		// 保存到历史记录
		a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
		a.AddMessage(core.NewMessage(finalAnswer, core.RoleAssistant, core.Time{}, nil))

		return finalAnswer, nil
	}

	// 2. 执行计划
	finalAnswer := a.executor.Execute(ctx, inputText, plan)
	fmt.Printf("\n--- 任务完成 ---\n最终答案: %s\n", finalAnswer)

	// 保存到历史记录
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.RoleAssistant, core.Time{}, nil))

	return finalAnswer, nil
}
