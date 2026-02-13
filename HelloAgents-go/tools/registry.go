package tools

import (
	"fmt"
	"strings"
	"sync"
)

// functionInfo 函数工具信息
type functionInfo struct {
	description string
	func_       func(string) string
}

// ToolRegistry 工具注册表
// 提供工具的注册、管理和执行功能。
// 支持两种工具注册方式：
// 1. Tool对象注册（推荐）
// 2. 函数直接注册（简便）
type ToolRegistry struct {
	mu         sync.RWMutex // 互斥锁，用于保护工具注册表的并发访问 读多写少
	_tools     map[string]Tool
	_functions map[string]functionInfo
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		_tools:     make(map[string]Tool),
		_functions: make(map[string]functionInfo),
	}
}

// RegisterTool 注册Tool对象
// autoExpand: 是否自动展开可展开的工具（默认true）
func (r *ToolRegistry) RegisterTool(tool Tool, autoExpand bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查工具是否可展开
	if autoExpand {
		expandedTools := tool.GetExpandedTools()
		if len(expandedTools) > 0 {
			// 注册所有展开的子工具
			for _, subTool := range expandedTools {
				subName := subTool.Name()
				if _, exists := r._tools[subName]; exists {
					fmt.Printf("⚠️ 警告：工具 '%s' 已存在，将被覆盖。\n", subName)
				}
				r._tools[subName] = subTool
			}
			fmt.Printf("✅ 工具 '%s' 已展开为 %d 个独立工具\n", tool.Name(), len(expandedTools))
			return
		}
	}

	// 普通工具或不展开的工具
	name := tool.Name()
	if _, exists := r._tools[name]; exists {
		fmt.Printf("⚠️ 警告：工具 '%s' 已存在，将被覆盖。\n", name)
	}

	r._tools[name] = tool
	fmt.Printf("✅ 工具 '%s' 已注册。\n", name)
}

// RegisterFunction 直接注册函数作为工具（简便方式）
// name: 工具名称
// description: 工具描述
// func_: 工具函数，接受字符串参数，返回字符串结果
func (r *ToolRegistry) RegisterFunction(name string, description string, func_ func(string) string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r._functions[name]; exists {
		fmt.Printf("⚠️ 警告：工具 '%s' 已存在，将被覆盖。\n", name)
	}

	r._functions[name] = functionInfo{
		description: description,
		func_:       func_,
	}
	fmt.Printf("✅ 工具 '%s' 已注册。\n", name)
}

// Unregister 注销工具
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r._tools[name]; exists {
		delete(r._tools, name)
		fmt.Printf("🗑️ 工具 '%s' 已注销。\n", name)
	} else if _, exists := r._functions[name]; exists {
		delete(r._functions, name)
		fmt.Printf("🗑️ 工具 '%s' 已注销。\n", name)
	} else {
		fmt.Printf("⚠️ 工具 '%s' 不存在。\n", name)
	}
}

// GetTool 获取Tool对象
func (r *ToolRegistry) GetTool(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r._tools[name]
}

// GetFunction 获取工具函数
func (r *ToolRegistry) GetFunction(name string) func(string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, exists := r._functions[name]; exists {
		return info.func_
	}
	return nil
}

// ExecuteTool 执行工具
// name: 工具名称
// inputText: 输入参数
// 返回: 工具执行结果
func (r *ToolRegistry) ExecuteTool(name string, inputText string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 优先查找Tool对象
	if tool, exists := r._tools[name]; exists {
		// 简化参数传递，直接传入字符串
		result, err := tool.Run(map[string]interface{}{"input": inputText})
		if err != nil {
			return fmt.Sprintf("错误：执行工具 '%s' 时发生异常: %s", name, err.Error())
		}
		return result
	}

	// 查找函数工具
	if info, exists := r._functions[name]; exists {
		result, err := info.func_(inputText), error(nil)
		// Go 的函数签名是 func(string) string，不会返回 error
		// 但为了与 Python 一致，这里保留结构
		_ = err // 忽略
		if result == "" {
			// 如果函数返回空字符串，可能是错误
		}
		return result
	}

	return fmt.Sprintf("错误：未找到名为 '%s' 的工具。", name)
}

// GetToolsDescription 获取所有可用工具的格式化描述字符串
// 返回: 工具描述字符串，用于构建提示词
func (r *ToolRegistry) GetToolsDescription() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descriptions := make([]string, 0)

	// Tool对象描述
	for _, tool := range r._tools {
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}

	// 函数工具描述
	for name, info := range r._functions {
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", name, info.description))
	}

	if len(descriptions) == 0 {
		return "暂无可用工具"
	}

	return strings.Join(descriptions, "\n")
}

// ListTools 列出所有工具名称
func (r *ToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r._tools)+len(r._functions))
	for name := range r._tools {
		names = append(names, name)
	}
	for name := range r._functions {
		names = append(names, name)
	}
	return names
}

// GetAllTools 获取所有Tool对象
func (r *ToolRegistry) GetAllTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r._tools))
	for _, tool := range r._tools {
		tools = append(tools, tool)
	}
	return tools
}

// Clear 清空所有工具
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r._tools = make(map[string]Tool)
	r._functions = make(map[string]functionInfo)
	fmt.Println("🧹 所有工具已清空。")
}

// GlobalRegistry 全局工具注册表
var GlobalRegistry = NewToolRegistry()
