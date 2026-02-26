// HTTP SSE 流式输出服务端示例
//
// 对应 Python 版本: examples/fastapi_sse_server.py
// 演示如何使用 Go 标准库 net/http 构建 SSE 流式服务。
//
// 运行方式:
//
//	go run example/sse_server_demo/main.go
//
// 测试方式:
//
//	curl -N http://localhost:8000/agent/stream -X POST \
//	  -H "Content-Type: application/json" \
//	  -d '{"input": "你好"}'
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// AgentRequest 请求体
type AgentRequest struct {
	Input     string `json:"input"`
	AgentType string `json:"agent_type"`
}

// SSEEvent 一个 SSE 事件
type SSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

// simulateAgentStream 模拟 Agent 流式输出
func simulateAgentStream(input string, agentType string) <-chan SSEEvent {
	ch := make(chan SSEEvent)
	go func() {
		defer close(ch)

		// 发送开始事件
		ch <- SSEEvent{
			Event: "start",
			Data: map[string]any{
				"agent_type": agentType,
				"input":      input,
			},
		}

		// 模拟思考过程
		thinkingSteps := []string{
			"分析用户输入...",
			"搜索相关信息...",
			"组织回复内容...",
		}
		for _, step := range thinkingSteps {
			time.Sleep(300 * time.Millisecond)
			ch <- SSEEvent{
				Event: "thinking",
				Data:  map[string]string{"step": step},
			}
		}

		// 模拟工具调用（ReAct 模式）
		if agentType == "react" {
			time.Sleep(200 * time.Millisecond)
			ch <- SSEEvent{
				Event: "tool_call",
				Data: map[string]any{
					"tool":   "Calculator",
					"params": map[string]string{"expression": "42 + 58"},
					"result": "100",
				},
			}
		}

		// 模拟流式文本输出
		responses := map[string]string{
			"react":      "根据工具计算结果，42 + 58 = 100。这是一个简单的加法运算。",
			"simple":     "你好！我是 HelloAgents 框架的 AI 助手，很高兴为你服务。",
			"reflection": "经过深入分析，人工智能正在快速发展，主要趋势包括大模型、多模态和 Agent 技术。",
			"plan":       "制定计划：1. 分析问题 2. 收集信息 3. 生成回答",
		}

		response := responses[agentType]
		if response == "" {
			response = responses["simple"]
		}

		words := strings.Fields(response)
		for i, word := range words {
			delay := time.Duration(50+rand.Intn(100)) * time.Millisecond
			time.Sleep(delay)
			ch <- SSEEvent{
				Event: "token",
				Data: map[string]any{
					"token":    word + " ",
					"index":    i,
					"finished": i == len(words)-1,
				},
			}
		}

		// 发送完成事件
		ch <- SSEEvent{
			Event: "done",
			Data: map[string]any{
				"agent_type":  agentType,
				"total_words": len(words),
			},
		}
	}()
	return ch
}

func handleAgentStream(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"只支持 POST 请求"}`, http.StatusMethodNotAllowed)
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"请求格式错误: %s"}`, err), http.StatusBadRequest)
		return
	}

	if req.AgentType == "" {
		req.AgentType = "react"
	}

	validTypes := map[string]bool{"react": true, "simple": true, "reflection": true, "plan": true}
	if !validTypes[req.AgentType] {
		http.Error(w, fmt.Sprintf(`{"error":"未知的 agent_type: %s"}`, req.AgentType), http.StatusBadRequest)
		return
	}

	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"不支持 SSE 流式输出"}`, http.StatusInternalServerError)
		return
	}

	events := simulateAgentStream(req.Input, req.AgentType)
	for event := range events {
		data, _ := json.Marshal(event.Data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, string(data))
		flusher.Flush()
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message": "HelloAgents SSE Demo (Go)",
		"endpoints": map[string]string{
			"stream": "/agent/stream",
			"root":   "/",
		},
		"supported_agents": []string{"react", "simple", "reflection", "plan"},
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/agent/stream", handleAgentStream)

	addr := ":8000"
	fmt.Println("=== HelloAgents SSE 服务端示例 ===")
	fmt.Println()
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Println()
	fmt.Println("端点:")
	fmt.Println("  GET  / ...................... 服务信息")
	fmt.Println("  POST /agent/stream ......... SSE 流式输出")
	fmt.Println()
	fmt.Println("测试命令:")
	fmt.Println(`  curl -N http://localhost:8000/agent/stream -X POST \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"input": "你好", "agent_type": "react"}'`)
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, mux))
}
