package tools

import (
	"fmt"
	"strings"
)

// step 工具链步骤
type step struct {
	toolName      string
	inputTemplate string
	outputKey     string
}

// ToolChain 工具链 - 支持多个工具的顺序执行
type ToolChain struct {
	name        string
	description string
	steps       []step
}

// NewToolChain 创建工具链
func NewToolChain(name, description string) *ToolChain {
	return &ToolChain{
		name:        name,
		description: description,
		steps:       make([]step, 0),
	}
}

// AddStep 添加工具执行步骤
// toolName: 工具名称
// inputTemplate: 输入模板，支持变量替换，如 "{input}" 或 "{search_result}"
// outputKey: 输出结果的键名，用于后续步骤引用
func (c *ToolChain) AddStep(toolName, inputTemplate, outputKey string) {
	if outputKey == "" {
		outputKey = fmt.Sprintf("step_%d_result", len(c.steps))
	}

	c.steps = append(c.steps, step{
		toolName:      toolName,
		inputTemplate: inputTemplate,
		outputKey:     outputKey,
	})
	fmt.Printf("✅ 工具链 '%s' 添加步骤: %s\n", c.name, toolName)
}

// Execute 执行工具链
// registry: 工具注册表
// inputData: 初始输入数据
// context: 执行上下文，用于变量替换
// 返回: 最终执行结果
func (c *ToolChain) Execute(registry *ToolRegistry, inputData string, context map[string]interface{}) string {
	if len(c.steps) == 0 {
		return "❌ 工具链为空，无法执行"
	}

	fmt.Printf("🚀 开始执行工具链: %s\n", c.name)

	// 初始化上下文
	if context == nil {
		context = make(map[string]interface{})
	}
	context["input"] = inputData

	finalResult := inputData

	for i, s := range c.steps {
		fmt.Printf("📝 执行步骤 %d/%d: %s\n", i+1, len(c.steps), s.toolName)

		// 替换模板中的变量
		actualInput, err := c.replaceTemplate(s.inputTemplate, context)
		if err != nil {
			return fmt.Sprintf("❌ 模板变量替换失败: %s", err.Error())
		}

		// 执行工具
		result := registry.ExecuteTool(s.toolName, actualInput)
		context[s.outputKey] = result
		finalResult = result
		fmt.Printf("✅ 步骤 %d 完成\n", i+1)
	}

	fmt.Printf("🎉 工具链 '%s' 执行完成\n", c.name)
	return finalResult
}

// replaceTemplate 替换模板中的变量
func (c *ToolChain) replaceTemplate(template string, context map[string]interface{}) (string, error) {
	result := template

	// 简单的变量替换实现
	// 支持 {variable} 格式
	for key, value := range context {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	// 检查是否还有未替换的变量
	if strings.Contains(result, "{") && strings.Contains(result, "}") {
		// 找出未替换的变量
		start := strings.Index(result, "{")
		end := strings.Index(result, "}")
		if start != -1 && end != -1 && end > start {
			missingVar := result[start+1 : end]
			return "", fmt.Errorf("变量 '%s' 未找到", missingVar)
		}
	}

	return result, nil
}

// Name 返回工具链名称
func (c *ToolChain) Name() string {
	return c.name
}

// Description 返回工具链描述
func (c *ToolChain) Description() string {
	return c.description
}

// Steps 返回步骤数量
func (c *ToolChain) Steps() int {
	return len(c.steps)
}

// ToolChainManager 工具链管理器
type ToolChainManager struct {
	registry *ToolRegistry
	chains   map[string]*ToolChain
}

// NewToolChainManager 创建工具链管理器
func NewToolChainManager(registry *ToolRegistry) *ToolChainManager {
	return &ToolChainManager{
		registry: registry,
		chains:   make(map[string]*ToolChain),
	}
}

// RegisterChain 注册工具链
func (m *ToolChainManager) RegisterChain(chain *ToolChain) {
	m.chains[chain.name] = chain
	fmt.Printf("✅ 工具链 '%s' 已注册\n", chain.name)
}

// ExecuteChain 执行指定的工具链
func (m *ToolChainManager) ExecuteChain(chainName, inputData string, context map[string]interface{}) string {
	chain, exists := m.chains[chainName]
	if !exists {
		return fmt.Sprintf("❌ 工具链 '%s' 不存在", chainName)
	}

	return chain.Execute(m.registry, inputData, context)
}

// ListChains 列出所有已注册的工具链
func (m *ToolChainManager) ListChains() []string {
	names := make([]string, 0, len(m.chains))
	for name := range m.chains {
		names = append(names, name)
	}
	return names
}

// GetChainInfo 获取工具链信息
func (m *ToolChainManager) GetChainInfo(chainName string) map[string]interface{} {
	chain, exists := m.chains[chainName]
	if !exists {
		return nil
	}

	stepDetails := make([]map[string]interface{}, 0, len(chain.steps))
	for _, s := range chain.steps {
		stepDetails = append(stepDetails, map[string]interface{}{
			"tool_name":      s.toolName,
			"input_template": s.inputTemplate,
			"output_key":     s.outputKey,
		})
	}

	return map[string]interface{}{
		"name":         chain.name,
		"description":  chain.description,
		"steps":        len(chain.steps),
		"step_details": stepDetails,
	}
}

// ==================== 便捷函数 ====================

// CreateResearchChain 创建一个研究工具链：搜索 -> 计算 -> 总结
func CreateResearchChain() *ToolChain {
	chain := NewToolChain(
		"research_and_calculate",
		"搜索信息并进行相关计算",
	)

	// 步骤1：搜索信息
	chain.AddStep(
		"search",
		"{input}",
		"search_result",
	)

	// 步骤2：基于搜索结果进行计算
	chain.AddStep(
		"my_calculator",
		"2 + 2", // 简单的计算示例
		"calc_result",
	)

	return chain
}

// CreateSimpleChain 创建一个简单的工具链示例
func CreateSimpleChain() *ToolChain {
	chain := NewToolChain(
		"simple_demo",
		"简单的工具链演示",
	)

	// 只包含一个计算步骤
	chain.AddStep(
		"my_calculator",
		"{input}",
		"result",
	)

	return chain
}
