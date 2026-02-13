package agents

import (
	"context"
	"fmt"
	"strings"

	"helloagents-go/HelloAgents-go/core"
)

// 默认提示词模板
var defaultPrompts = map[string]string{
	"initial": `请根据以下要求完成任务：

任务: {task}

请提供一个完整、准确的回答。`,
	"reflect": `请仔细审查以下回答，并找出可能的问题或改进空间：

# 原始任务:
{task}

# 当前回答:
{content}

请分析这个回答的质量，指出不足之处，并提出具体的改进建议。
如果回答已经很好，请回答"无需改进"。`,
	"refine": `请根据反馈意见改进你的回答：

# 原始任务:
{task}

# 上一轮回答:
{last_attempt}

# 反馈意见:
{feedback}

请提供一个改进后的回答。`,
}

// memoryRecord 记忆记录
type memoryRecord struct {
	Type    string
	Content string
}

// Memory 简单的短期记忆模块，用于存储智能体的行动与反思轨迹。
type Memory struct {
	records []memoryRecord
}

// NewMemory 创建新的记忆模块。
func NewMemory() *Memory {
	return &Memory{
		records: make([]memoryRecord, 0),
	}
}

// addRecord 向记忆中添加一条新记录。
func (m *Memory) addRecord(recordType, content string) {
	m.records = append(m.records, memoryRecord{Type: recordType, Content: content})
	fmt.Printf("📝 记忆已更新，新增一条 '%s' 记录。\n", recordType)
}

// getTrajectory 将所有记忆记录格式化为一个连贯的字符串文本。
func (m *Memory) getTrajectory() string {
	var sb strings.Builder
	for _, record := range m.records {
		if record.Type == "execution" {
			sb.WriteString("--- 上一轮尝试 (代码) ---\n")
			sb.WriteString(record.Content)
			sb.WriteString("\n\n")
		} else if record.Type == "reflection" {
			sb.WriteString("--- 评审员反馈 ---\n")
			sb.WriteString(record.Content)
			sb.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// getLastExecution 获取最近一次的执行结果。
func (m *Memory) getLastExecution() string {
	for i := len(m.records) - 1; i >= 0; i-- {
		if m.records[i].Type == "execution" {
			return m.records[i].Content
		}
	}
	return ""
}

// ReflectionAgent 自我反思与迭代优化的智能体。
type ReflectionAgent struct {
	*core.BaseAgent
	maxIterations int
	memory        *Memory
	prompts       map[string]string
}

// NewReflectionAgent 创建 ReflectionAgent。
func NewReflectionAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	config *core.Config,
	maxIterations int,
	customPrompts map[string]string,
) *ReflectionAgent {
	if maxIterations <= 0 {
		maxIterations = 3
	}

	prompts := make(map[string]string)
	if customPrompts != nil {
		for k, v := range customPrompts {
			prompts[k] = v
		}
	}
	for k, v := range defaultPrompts {
		if _, exists := prompts[k]; !exists {
			prompts[k] = v
		}
	}

	return &ReflectionAgent{
		BaseAgent:     core.NewBaseAgent(name, llm, systemPrompt, config),
		maxIterations: maxIterations,
		memory:        NewMemory(),
		prompts:       prompts,
	}
}

// Run 运行 Reflection Agent。
func (a *ReflectionAgent) Run(ctx context.Context, inputText string) (string, error) {
	fmt.Printf("\n🤖 %s 开始处理任务: %s\n", a.Name, inputText)

	// 重置记忆
	a.memory = NewMemory()

	// 1. 初始执行
	fmt.Println("\n--- 正在进行初始尝试 ---")
	initialPrompt := strings.ReplaceAll(a.prompts["initial"], "{task}", inputText)
	initialResult, err := a.getLLMResponse(ctx, initialPrompt)
	if err != nil {
		return "", fmt.Errorf("初始执行失败: %w", err)
	}
	a.memory.addRecord("execution", initialResult)

	// 2. 迭代循环：反思与优化
	for i := 0; i < a.maxIterations; i++ {
		fmt.Printf("\n--- 第 %d/%d 轮迭代 ---\n", i+1, a.maxIterations)

		// a. 反思
		fmt.Println("\n-> 正在进行反思...")
		lastResult := a.memory.getLastExecution()
		reflectPrompt := strings.ReplaceAll(a.prompts["reflect"], "{task}", inputText)
		reflectPrompt = strings.ReplaceAll(reflectPrompt, "{content}", lastResult)

		feedback, err := a.getLLMResponse(ctx, reflectPrompt)
		if err != nil {
			return "", fmt.Errorf("反思失败: %w", err)
		}
		a.memory.addRecord("reflection", feedback)

		// b. 检查是否需要停止
		if strings.Contains(feedback, "无需改进") ||
			strings.Contains(strings.ToLower(feedback), "no need for improvement") {
			fmt.Println("\n✅ 反思认为结果已无需改进，任务完成。")
			break
		}

		// c. 优化
		fmt.Println("\n-> 正在进行优化...")
		refinePrompt := strings.ReplaceAll(a.prompts["refine"], "{task}", inputText)
		refinePrompt = strings.ReplaceAll(refinePrompt, "{last_attempt}", lastResult)
		refinePrompt = strings.ReplaceAll(refinePrompt, "{feedback}", feedback)

		refinedResult, err := a.getLLMResponse(ctx, refinePrompt)
		if err != nil {
			return "", fmt.Errorf("优化失败: %w", err)
		}
		a.memory.addRecord("execution", refinedResult)
	}

	finalResult := a.memory.getLastExecution()
	fmt.Printf("\n--- 任务完成 ---\n最终结果:\n%s\n", finalResult)

	// 保存到历史记录
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalResult, core.RoleAssistant, core.Time{}, nil))

	return finalResult, nil
}

// getLLMResponse 调用 LLM 并获取完整响应。
func (a *ReflectionAgent) getLLMResponse(ctx context.Context, prompt string) (string, error) {
	messages := []core.ChatMessage{
		{Role: core.RoleUser, Content: prompt},
	}
	return a.LLM.Invoke(ctx, messages, nil)
}
