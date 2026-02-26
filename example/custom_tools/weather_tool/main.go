// 自定义工具示例 - 天气查询工具
//
// 对应 Python 版本: examples/custom_tools/weather_tool.py
// 演示如何创建一个自定义的天气查询工具。
package main

import (
	"fmt"
	"math/rand"

	"helloagents-go/hello_agents/tools"
)

// WeatherTool 天气查询工具示例
type WeatherTool struct {
	tools.BaseTool
}

func NewWeatherTool() *WeatherTool {
	t := &WeatherTool{}
	t.Name = "WeatherQuery"
	t.Description = "查询指定城市的天气信息"
	t.Parameters = map[string]tools.ToolParameter{
		"city": {
			Name:        "city",
			Type:        "string",
			Description: "要查询天气的城市名称",
			Required:    true,
		},
		"unit": {
			Name:        "unit",
			Type:        "string",
			Description: "温度单位: celsius 或 fahrenheit",
			Required:    false,
			Default:     "celsius",
		},
	}
	return t
}

func (t *WeatherTool) GetName() string                            { return t.Name }
func (t *WeatherTool) GetDescription() string                     { return t.Description }
func (t *WeatherTool) GetParameters() []tools.ToolParameter       { return t.BaseTool.GetParameters() }
func (t *WeatherTool) RunWithTiming(p map[string]any) tools.ToolResponse { return t.BaseTool.RunWithTiming(p) }
func (t *WeatherTool) ARun(p map[string]any) tools.ToolResponse   { return t.Run(p) }
func (t *WeatherTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *WeatherTool) Run(parameters map[string]any) tools.ToolResponse {
	city, _ := parameters["city"].(string)
	if city == "" {
		return tools.Error("城市名称不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	unit, _ := parameters["unit"].(string)
	if unit == "" {
		unit = "celsius"
	}

	temp := 15 + rand.Intn(20)
	if unit == "fahrenheit" {
		temp = temp*9/5 + 32
	}

	conditions := []string{"晴", "多云", "阴", "小雨"}
	condition := conditions[rand.Intn(len(conditions))]

	return tools.Success(
		fmt.Sprintf("%s 天气: %s, 温度 %d°%s", city, condition, temp, map[string]string{"celsius": "C", "fahrenheit": "F"}[unit]),
		map[string]any{
			"city":        city,
			"temperature": temp,
			"condition":   condition,
			"unit":        unit,
		},
	)
}

func main() {
	fmt.Println("=== 自定义天气工具示例 ===")
	fmt.Println()

	tool := NewWeatherTool()
	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(tool, false)

	cities := []string{"北京", "上海", "广州", "深圳"}
	for _, city := range cities {
		resp := tool.Run(map[string]any{"city": city})
		fmt.Printf("  %s\n", resp.Text)
	}
	fmt.Println()

	// OpenAI Schema
	fmt.Println("OpenAI Function Schema:")
	schema := tool.ToOpenAISchema()
	fmt.Printf("  %v\n", schema)
}
