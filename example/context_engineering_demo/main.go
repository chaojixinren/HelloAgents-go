// 上下文工程使用示例
//
// 对应 Python 版本: examples/context_engineering_demo.py
// 演示 HistoryManager、ObservationTruncator 和 TokenCounter。
package main

import (
	"fmt"
	"os"
	"strings"

	haContext "helloagents-go/hello_agents/context"
	"helloagents-go/hello_agents/core"
)

func main() {
	fmt.Println("=== 上下文工程示例 ===")
	fmt.Println()

	demoTokenCounter()
	demoHistoryManager()
	demoTruncator()
}

func demoTokenCounter() {
	fmt.Println("示例 1: Token 计数器")
	fmt.Println(strings.Repeat("=", 50))

	tc := haContext.NewTokenCounter[core.Message](
		"gpt-3.5-turbo",
		func(msg core.Message) string { return msg.Content },
		func(msg core.Message) string { return string(msg.Role) + ":" + msg.Content },
	)

	msg := core.NewMessage("这是一条测试消息，用于展示 Token 计数功能。", core.MessageRoleUser, nil)
	count := tc.CountMessage(msg)
	fmt.Printf("  单条消息 Token 数: %d\n", count)

	messages := []core.Message{
		core.NewMessage("你好", core.MessageRoleUser, nil),
		core.NewMessage("你好！有什么可以帮助你的吗？", core.MessageRoleAssistant, nil),
		core.NewMessage("请计算 2+2", core.MessageRoleUser, nil),
	}
	total := tc.CountMessages(messages)
	fmt.Printf("  %d 条消息总 Token 数: %d\n", len(messages), total)
	fmt.Println()
}

func demoHistoryManager() {
	fmt.Println("示例 2: 历史管理器")
	fmt.Println(strings.Repeat("=", 50))

	hm := haContext.NewHistoryManager[core.Message](
		2,  // 压缩时保留最近 2 轮
		0.8,
		func(summary string) core.Message {
			return core.NewMessage("[摘要] "+summary, core.MessageRoleSummary, nil)
		},
		func(msg core.Message) string { return string(msg.Role) },
	)

	// 添加多轮对话
	rounds := []struct{ user, assistant string }{
		{"你好", "你好！很高兴见到你。"},
		{"今天天气怎么样？", "今天晴天，温度适宜。"},
		{"推荐一家餐厅", "推荐'味道'中餐馆，性价比很高。"},
		{"它在哪里？", "在市中心的商业街上。"},
	}
	for _, r := range rounds {
		hm.Append(core.NewMessage(r.user, core.MessageRoleUser, nil))
		hm.Append(core.NewMessage(r.assistant, core.MessageRoleAssistant, nil))
	}

	fmt.Printf("  当前历史消息数: %d\n", len(hm.GetHistory()))
	fmt.Printf("  预估轮次: %d\n", hm.EstimateRounds())

	// 压缩
	hm.Compress("用户与助手进行了关于天气和餐厅的闲聊。")
	fmt.Printf("  压缩后消息数: %d\n", len(hm.GetHistory()))
	for i, msg := range hm.GetHistory() {
		preview := msg.Content
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		fmt.Printf("    [%d] %s: %s\n", i, msg.Role, preview)
	}
	fmt.Println()
}

func demoTruncator() {
	fmt.Println("示例 3: 输出截断器")
	fmt.Println(strings.Repeat("=", 50))

	tmpDir, _ := os.MkdirTemp("", "truncator_demo")
	defer os.RemoveAll(tmpDir)

	truncator := haContext.NewObservationTruncator(5, 4096, "head", tmpDir)

	// 短输出 - 不截断
	shortContent := "line1\nline2\nline3"
	preview, result := truncator.Truncate(shortContent, "ShortTool")
	fmt.Printf("  短输出 (%d 行): truncated=%v\n", 3, result["truncated"])
	fmt.Printf("  预览: %s\n", preview)
	fmt.Println()

	// 长输出 - 截断
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("第 %d 行: 这是一些数据内容 %s", i+1, strings.Repeat("x", 20)))
	}
	longContent := strings.Join(lines, "\n")
	preview, result = truncator.Truncate(longContent, "LongTool")
	stats := result["stats"].(map[string]any)
	fmt.Printf("  长输出 (%d 行): truncated=%v\n", stats["original_lines"], result["truncated"])
	fmt.Printf("  保留行数: %v\n", stats["kept_lines"])
	fmt.Printf("  完整输出保存路径: %v\n", result["full_output_path"])
}
