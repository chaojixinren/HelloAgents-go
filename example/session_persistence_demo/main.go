// 会话持久化使用示例
//
// 对应 Python 版本: examples/session_persistence_demo.py
// 演示如何使用 SessionStore 保存和恢复会话。
package main

import (
	"fmt"
	"os"

	"helloagents-go/hello_agents/core"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "session_demo")
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== 会话持久化示例 ===")
	fmt.Println()

	store, err := core.NewSessionStore(tmpDir)
	if err != nil {
		fmt.Printf("❌ 创建 SessionStore 失败: %v\n", err)
		return
	}

	// 1. 保存会话
	fmt.Println("1. 保存会话:")
	agentConfig := map[string]any{
		"name":         "demo-agent",
		"agent_type":   "SimpleAgent",
		"llm_provider": "openai",
		"llm_model":    "gpt-4",
	}
	history := []core.Message{
		core.NewMessage("你好", core.MessageRoleUser, nil),
		core.NewMessage("你好！有什么可以帮助你的吗？", core.MessageRoleAssistant, nil),
	}
	metadata := map[string]any{
		"total_tokens": 100,
		"total_steps":  1,
	}

	filepath, err := store.Save(agentConfig, history, "hash-123", nil, metadata, "demo-session")
	if err != nil {
		fmt.Printf("   ❌ 保存失败: %v\n", err)
		return
	}
	fmt.Printf("   ✅ 已保存到: %s\n", filepath)
	fmt.Println()

	// 2. 列出会话
	fmt.Println("2. 列出所有会话:")
	sessions, _ := store.ListSessions()
	for _, s := range sessions {
		fmt.Printf("   - %s (session_id: %v)\n", s["filename"], s["session_id"])
	}
	fmt.Println()

	// 3. 加载会话
	fmt.Println("3. 加载会话:")
	record, err := store.Load(filepath)
	if err != nil {
		fmt.Printf("   ❌ 加载失败: %v\n", err)
		return
	}
	fmt.Printf("   Session ID: %s\n", record.SessionID)
	fmt.Printf("   Agent Config: %v\n", record.AgentConfig)
	fmt.Printf("   History Length: %d\n", len(record.History))
	fmt.Printf("   Metadata: %v\n", record.Metadata)
	fmt.Println()

	// 4. 一致性检查
	fmt.Println("4. 配置一致性检查:")
	check := store.CheckConfigConsistency(record.AgentConfig, agentConfig)
	fmt.Printf("   一致: %v\n", check["consistent"])

	differentConfig := map[string]any{
		"name":         "demo-agent",
		"agent_type":   "ReActAgent",
		"llm_provider": "anthropic",
		"llm_model":    "claude-3",
	}
	check = store.CheckConfigConsistency(record.AgentConfig, differentConfig)
	fmt.Printf("   不同配置一致性: %v\n", check["consistent"])
	if warnings, ok := check["warnings"].([]string); ok {
		for _, w := range warnings {
			fmt.Printf("     ⚠️  %s\n", w)
		}
	}
}
