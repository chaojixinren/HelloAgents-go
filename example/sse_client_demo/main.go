// SSE 客户端测试脚本
//
// 对应 Python 版本: examples/test_sse_client.py
// 演示如何使用 Go 标准库连接 SSE 服务端并解析事件流。
//
// 使用前需先启动 SSE 服务端:
//
//	go run example/sse_server_demo/main.go
//
// 然后运行客户端:
//
//	go run example/sse_client_demo/main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func testSSEStream(inputText, agentType, baseURL string) {
	url := baseURL + "/agent/stream"

	payload := map[string]string{
		"input":      inputText,
		"agent_type": agentType,
	}
	body, _ := json.Marshal(payload)

	fmt.Printf("🚀 发送请求: %s\n", inputText)
	fmt.Printf("📝 Agent类型: %s\n", agentType)
	fmt.Println("------------------------------------------------------------")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ 连接失败: %v\n", err)
		fmt.Println("   请确保 SSE 服务端已启动:")
		fmt.Println("   go run example/sse_server_demo/main.go")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 错误: %d\n", resp.StatusCode)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			printEvent(currentEvent, data)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("❌ 读取流错误: %v\n", err)
	}
}

func printEvent(event, data string) {
	switch event {
	case "start":
		fmt.Printf("  ▶ [start] %s\n", data)
	case "thinking":
		fmt.Printf("  💭 [thinking] %s\n", data)
	case "tool_call":
		fmt.Printf("  🔧 [tool_call] %s\n", data)
	case "token":
		var parsed map[string]any
		if err := json.Unmarshal([]byte(data), &parsed); err == nil {
			if token, ok := parsed["token"].(string); ok {
				fmt.Print(token)
				if finished, ok := parsed["finished"].(bool); ok && finished {
					fmt.Println()
				}
			}
		}
	case "done":
		fmt.Printf("  ✅ [done] %s\n", data)
	case "error":
		fmt.Printf("  ❌ [error] %s\n", data)
	default:
		fmt.Printf("  [%s] %s\n", event, data)
	}
}

func main() {
	fmt.Println("=== SSE 客户端测试 ===")
	fmt.Println()

	baseURL := "http://localhost:8000"

	testCases := []struct {
		input     string
		agentType string
	}{
		{"计算 42 + 58", "react"},
		{"你好，介绍一下你自己", "simple"},
		{"分析一下人工智能的发展趋势", "reflection"},
	}

	for _, tc := range testCases {
		testSSEStream(tc.input, tc.agentType, baseURL)
		fmt.Println()
		fmt.Println("============================================================")
		fmt.Println()
	}
}
