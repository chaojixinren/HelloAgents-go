package core

import (
	"fmt"
	"os"
	"strconv"
)

// HelloAgentsLLM mirrors hello_agents.core.llm.HelloAgentsLLM.
type HelloAgentsLLM struct {
	Model       string
	Provider    string
	APIKey      string
	BaseURL     string
	Timeout     int
	Temperature float64
	MaxTokens   *int
	Kwargs      map[string]any

	adapter       BaseLLMAdapter
	LastCallStats *StreamStats
}

func NewHelloAgentsLLM(model, apiKey, baseURL string, temperature float64, maxTokens *int, timeout *int, kwargs map[string]any) (*HelloAgentsLLM, error) {
	ensureDotEnvLoaded()

	if kwargs == nil {
		kwargs = map[string]any{}
	}

	if model == "" {
		model = os.Getenv("LLM_MODEL_ID")
	}
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}

	timeoutValue := 60
	if timeout != nil {
		timeoutValue = *timeout
	} else if s := os.Getenv("LLM_TIMEOUT"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil {
			timeoutValue = parsed
		}
	}

	if model == "" {
		return nil, NewHelloAgentsException("必须提供模型名称（model参数或LLM_MODEL_ID环境变量）")
	}
	if apiKey == "" {
		return nil, NewHelloAgentsException("必须提供API密钥（api_key参数或LLM_API_KEY环境变量）")
	}
	if baseURL == "" {
		return nil, NewHelloAgentsException("必须提供服务地址（base_url参数或LLM_BASE_URL环境变量）")
	}

	provider := ""
	if v, ok := kwargs["provider"].(string); ok {
		provider = v
	}

	llm := &HelloAgentsLLM{
		Model:       model,
		Provider:    provider,
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Timeout:     timeoutValue,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Kwargs:      kwargs,
	}
	llm.adapter = CreateAdapter(apiKey, baseURL, timeoutValue, model)
	return llm, nil
}

func (l *HelloAgentsLLM) Think(messages []map[string]any, temperature *float64) (<-chan string, <-chan error) {
	kwargs := map[string]any{}
	if temperature != nil {
		kwargs["temperature"] = *temperature
	} else {
		kwargs["temperature"] = l.Temperature
	}
	if l.MaxTokens != nil {
		kwargs["max_tokens"] = *l.MaxTokens
	}
	return l.StreamInvoke(messages, kwargs)
}

func (l *HelloAgentsLLM) Invoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	if _, ok := kwargs["temperature"]; !ok {
		kwargs["temperature"] = l.Temperature
	}
	if l.MaxTokens != nil {
		if _, ok := kwargs["max_tokens"]; !ok {
			kwargs["max_tokens"] = *l.MaxTokens
		}
	}
	resp, err := l.adapter.Invoke(messages, kwargs)
	if err != nil {
		return LLMResponse{}, WrapError("llm invoke failed", err)
	}
	return resp, nil
}

func (l *HelloAgentsLLM) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	if _, ok := kwargs["temperature"]; !ok {
		kwargs["temperature"] = l.Temperature
	}
	if l.MaxTokens != nil {
		if _, ok := kwargs["max_tokens"]; !ok {
			kwargs["max_tokens"] = *l.MaxTokens
		}
	}

	innerStream, innerErr := l.adapter.StreamInvoke(messages, kwargs)
	out := make(chan string)
	errOut := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errOut)

		for chunk := range innerStream {
			out <- chunk
		}

		for err := range innerErr {
			if err != nil {
				errOut <- err
				return
			}
		}

		if stats := l.adapter.LastStats(); stats != nil {
			l.LastCallStats = stats
		}
	}()

	return out, errOut
}

func (l *HelloAgentsLLM) InvokeWithTools(messages []map[string]any, tools []map[string]any, toolChoice any, kwargs map[string]any) (map[string]any, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	kwargs["tool_choice"] = toolChoice
	if _, ok := kwargs["temperature"]; !ok {
		kwargs["temperature"] = l.Temperature
	}
	if l.MaxTokens != nil {
		if _, ok := kwargs["max_tokens"]; !ok {
			kwargs["max_tokens"] = *l.MaxTokens
		}
	}
	return l.adapter.InvokeWithTools(messages, tools, kwargs)
}

func (l *HelloAgentsLLM) AInvoke(messages []map[string]any, kwargs map[string]any) (LLMResponse, error) {
	return l.Invoke(messages, kwargs)
}

func (l *HelloAgentsLLM) AStreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	return l.StreamInvoke(messages, kwargs)
}

func (l *HelloAgentsLLM) AInvokeWithTools(messages []map[string]any, tools []map[string]any, toolChoice any, kwargs map[string]any) (map[string]any, error) {
	return l.InvokeWithTools(messages, tools, toolChoice, kwargs)
}

func (l *HelloAgentsLLM) Validate() error {
	if l.Model == "" || l.APIKey == "" || l.BaseURL == "" {
		return fmt.Errorf("llm client is not fully configured")
	}
	return nil
}
