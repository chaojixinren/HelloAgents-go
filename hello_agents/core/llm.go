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
	if timeout != nil && *timeout != 0 {
		timeoutValue = *timeout
	} else if s := os.Getenv("LLM_TIMEOUT"); s != "" {
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return nil, NewHelloAgentsException(fmt.Sprintf("LLM_TIMEOUT 环境变量格式错误: %v", err))
		}
		timeoutValue = parsed
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

	if maxTokens != nil && *maxTokens == 0 {
		maxTokens = nil
	}

	llm := &HelloAgentsLLM{
		Model:       model,
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
	callKwargs := copyMap(kwargs)
	if _, ok := callKwargs["temperature"]; !ok {
		callKwargs["temperature"] = l.Temperature
	}
	if l.MaxTokens != nil {
		if _, ok := callKwargs["max_tokens"]; !ok {
			callKwargs["max_tokens"] = *l.MaxTokens
		}
	}
	resp, err := l.adapter.Invoke(messages, callKwargs)
	if err != nil {
		return LLMResponse{}, WrapError("llm invoke failed", err)
	}
	return resp, nil
}

func (l *HelloAgentsLLM) StreamInvoke(messages []map[string]any, kwargs map[string]any) (<-chan string, <-chan error) {
	callKwargs := copyMap(kwargs)

	temperature, hasTemperature := callKwargs["temperature"]
	if hasTemperature {
		delete(callKwargs, "temperature")
	}

	adapterKwargs := map[string]any{}
	if hasTemperature {
		adapterKwargs["temperature"] = temperature
	}

	if l.MaxTokens != nil {
		if maxTokens, ok := callKwargs["max_tokens"]; ok {
			adapterKwargs["max_tokens"] = maxTokens
			delete(callKwargs, "max_tokens")
		} else {
			adapterKwargs["max_tokens"] = *l.MaxTokens
		}
	}
	for k, v := range callKwargs {
		adapterKwargs[k] = v
	}

	innerStream, innerErr := l.adapter.StreamInvoke(messages, adapterKwargs)
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
	callKwargs := map[string]any{
		"temperature": l.Temperature,
		"tool_choice": toolChoice,
	}
	for k, v := range copyMap(kwargs) {
		callKwargs[k] = v
	}

	if l.MaxTokens != nil {
		if _, ok := callKwargs["max_tokens"]; !ok {
			callKwargs["max_tokens"] = *l.MaxTokens
		}
	}
	return l.adapter.InvokeWithTools(messages, tools, callKwargs)
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

// NewLLMFromAdapter creates a HelloAgentsLLM with a custom adapter for testing.
func NewLLMFromAdapter(model, apiKey, baseURL string, timeout int, temperature float64, adapter BaseLLMAdapter) *HelloAgentsLLM {
	return &HelloAgentsLLM{
		Model:       model,
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Timeout:     timeout,
		Temperature: temperature,
		adapter:     adapter,
	}
}
