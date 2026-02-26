package core

import "time"

var ExportConvertAnthropicMessages = convertAnthropicMessages

var ExportConvertGeminiMessages = convertGeminiMessages

var ExportPythonValueEqual = pythonValueEqual

func ExportParsePythonISOTime(value string) (time.Time, error) {
	return parsePythonISOTime(value)
}

func ExportCopyMap(in map[string]any) map[string]any {
	return copyMap(in)
}

func (a *BaseAgent) ExportConvertParameterTypes(toolName string, input map[string]any) map[string]any {
	return a.convertParameterTypes(toolName, input)
}

func (a *BaseAgent) ExportMapParameterType(paramType string) string {
	return a.mapParameterType(paramType)
}

func (a *BaseAgent) ExportGetSummaryLLM() (*HelloAgentsLLM, error) {
	return a.getSummaryLLM()
}

func (a *BaseAgent) ExportExtractToolsFromHistory(history []Message) []string {
	return a.extractToolsFromHistory(history)
}

func NewOpenAIAdapterForTest(apiKey, baseURL string, timeout int, model string) *OpenAIAdapter {
	return &OpenAIAdapter{adapterBase: adapterBase{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: timeout,
		Model:   model,
	}}
}

func NewAnthropicAdapterForTest(apiKey, baseURL string, timeout int, model string) *AnthropicAdapter {
	return &AnthropicAdapter{adapterBase: adapterBase{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: timeout,
		Model:   model,
	}}
}

func NewGeminiAdapterForTest(apiKey, baseURL string, timeout int, model string) *GeminiAdapter {
	return &GeminiAdapter{adapterBase: adapterBase{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: timeout,
		Model:   model,
	}}
}
