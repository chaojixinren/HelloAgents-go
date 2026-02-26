// 高级工具模板 - 完整特性
//
// 对应 Python 版本: examples/custom_tools/advanced_tool_template.py
// 展示完整的自定义工具特性：参数验证、缓存、重试、统计信息。
package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"helloagents-go/hello_agents/tools"
)

// AdvancedTool 高级工具模板
type AdvancedTool struct {
	tools.BaseTool
	apiKey     string
	maxRetries int
	timeout    int
	cache      map[string]tools.ToolResponse
	stats      map[string]int
}

func NewAdvancedTool(apiKey string, maxRetries int, timeout int, enableCache bool) *AdvancedTool {
	var cache map[string]tools.ToolResponse
	if enableCache {
		cache = make(map[string]tools.ToolResponse)
	}
	t := &AdvancedTool{
		apiKey:     apiKey,
		maxRetries: maxRetries,
		timeout:    timeout,
		cache:      cache,
		stats: map[string]int{
			"total_calls":   0,
			"success_calls": 0,
			"error_calls":   0,
			"cache_hits":    0,
		},
	}
	t.Name = "advanced_tool"
	t.Description = "高级工具模板，展示完整的工具特性"
	t.Parameters = map[string]tools.ToolParameter{
		"query": {
			Name: "query", Type: "string",
			Description: "查询字符串", Required: true,
		},
		"timeout": {
			Name: "timeout", Type: "integer",
			Description: "超时时间（秒）", Required: false, Default: 30,
		},
		"format": {
			Name: "format", Type: "string",
			Description: "输出格式 (json/text)", Required: false, Default: "json",
		},
	}
	return t
}

func (t *AdvancedTool) GetName() string                      { return t.Name }
func (t *AdvancedTool) GetDescription() string               { return t.Description }
func (t *AdvancedTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *AdvancedTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *AdvancedTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *AdvancedTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *AdvancedTool) Run(parameters map[string]any) tools.ToolResponse {
	t.stats["total_calls"]++
	startTime := time.Now()

	// 1. 参数验证
	if errResp := t.validateParameters(parameters); errResp != nil {
		t.stats["error_calls"]++
		return *errResp
	}

	// 2. 检查缓存
	if t.cache != nil {
		cacheKey := t.getCacheKey(parameters)
		if cached, ok := t.cache[cacheKey]; ok {
			t.stats["cache_hits"]++
			return cached
		}
	}

	// 3. 执行业务逻辑（带重试）
	for attempt := 0; attempt < t.maxRetries; attempt++ {
		result, err := t.executeLogic(parameters)
		if err != nil {
			if attempt == t.maxRetries-1 {
				t.stats["error_calls"]++
				return tools.Error(
					fmt.Sprintf("执行失败: %v", err),
					tools.ToolErrorCodeExecutionError,
					map[string]any{"parameters": parameters, "retries": attempt + 1},
				)
			}
			time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
			continue
		}

		elapsed := time.Since(startTime)
		resp := tools.Success(
			fmt.Sprintf("执行成功: %s", result),
			map[string]any{
				"result":     result,
				"parameters": parameters,
				"attempt":    attempt + 1,
			},
			map[string]any{
				"time_ms":   elapsed.Milliseconds(),
				"retries":   attempt,
				"cache_hit": false,
			},
		)

		// 缓存结果
		if t.cache != nil {
			cacheKey := t.getCacheKey(parameters)
			t.cache[cacheKey] = resp
		}

		t.stats["success_calls"]++
		return resp
	}

	return tools.Error("未知错误", tools.ToolErrorCodeInternalError, nil)
}

func (t *AdvancedTool) validateParameters(parameters map[string]any) *tools.ToolResponse {
	query, _ := parameters["query"].(string)
	if query == "" {
		resp := tools.Error("参数 'query' 不能为空", tools.ToolErrorCodeInvalidParam, nil)
		return &resp
	}

	if timeoutVal, ok := parameters["timeout"]; ok {
		timeout, isNum := toInt(timeoutVal)
		if !isNum {
			resp := tools.Error("参数 'timeout' 必须是整数", tools.ToolErrorCodeInvalidParam, nil)
			return &resp
		}
		if timeout <= 0 {
			resp := tools.Error("参数 'timeout' 必须大于 0", tools.ToolErrorCodeInvalidParam, nil)
			return &resp
		}
	}

	if fmtVal, ok := parameters["format"].(string); ok {
		if fmtVal != "json" && fmtVal != "text" {
			resp := tools.Error("参数 'format' 必须是 'json' 或 'text'", tools.ToolErrorCodeInvalidParam, nil)
			return &resp
		}
	}

	return nil
}

func (t *AdvancedTool) getCacheKey(parameters map[string]any) string {
	keys := make([]string, 0, len(parameters))
	for k := range parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	data, _ := json.Marshal(parameters)
	return fmt.Sprintf("%x", md5.Sum(data))
}

func (t *AdvancedTool) executeLogic(parameters map[string]any) (string, error) {
	query, _ := parameters["query"].(string)
	time.Sleep(10 * time.Millisecond)
	return fmt.Sprintf("处理查询 '%s' 的结果", query), nil
}

func (t *AdvancedTool) GetStats() map[string]int {
	out := make(map[string]int)
	for k, v := range t.stats {
		out[k] = v
	}
	return out
}

func (t *AdvancedTool) ClearCache() {
	if t.cache != nil {
		t.cache = make(map[string]tools.ToolResponse)
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func main() {
	// 1. 基本使用
	fmt.Println("=== 基本使用 ===")
	tool := NewAdvancedTool("test_key", 3, 30, true)

	resp := tool.Run(map[string]any{"query": "test query", "timeout": 10})
	fmt.Printf("状态: %s\n", resp.Status)
	fmt.Printf("文本: %s\n", resp.Text)
	fmt.Printf("统计: %v\n", resp.Stats)
	fmt.Println()

	// 2. 缓存测试
	fmt.Println("=== 缓存测试 ===")
	resp2 := tool.Run(map[string]any{"query": "test query", "timeout": 10})
	fmt.Printf("第二次调用结果: %s\n", resp2.Text)
	fmt.Printf("工具统计: %v\n", tool.GetStats())
	fmt.Println()

	// 3. 参数验证测试
	fmt.Println("=== 参数验证测试 ===")
	resp3 := tool.Run(map[string]any{"query": ""})
	fmt.Printf("空查询: status=%s, text=%s\n", resp3.Status, resp3.Text)

	resp4 := tool.Run(map[string]any{"query": "test", "format": "xml"})
	fmt.Printf("无效格式: status=%s, text=%s\n", resp4.Status, resp4.Text)
	fmt.Println()

	// 4. 清除缓存
	fmt.Println("=== 清除缓存后 ===")
	tool.ClearCache()
	resp5 := tool.Run(map[string]any{"query": "test query", "timeout": 10})
	fmt.Printf("清除缓存后: %s\n", resp5.Text)
	fmt.Printf("最终统计: %v\n", tool.GetStats())
}
