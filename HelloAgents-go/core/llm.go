package core

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

// 支持的 LLM 提供商
const (
	ProviderOpenAI     = "openai"
	ProviderDeepSeek   = "deepseek"
	ProviderQwen       = "qwen"
	ProviderModelScope = "modelscope"
	ProviderKimi       = "kimi"
	ProviderZhipu      = "zhipu"
	ProviderOllama     = "ollama"
	ProviderVLLM       = "vllm"
	ProviderLocal      = "local"
	ProviderAuto       = "auto"
	ProviderCustom     = "custom"
)

// ChatMessage 表示单条对话消息，与 Python 的 dict[str, str] (role/content) 对应。
type ChatMessage struct {
	Role    string
	Content string
}

// HelloAgentsLLM 是 HelloAgents 的 LLM 客户端，用于调用任意兼容 OpenAI 接口的服务。
// 设计：参数优先、环境变量兜底；默认支持流式；多提供商统一接口。
type HelloAgentsLLM struct {
	Model       string
	APIKey      string
	BaseURL     string
	Provider    string
	Temperature float32
	MaxTokens   int
	Timeout     int
	client      *openai.Client
}

// InvokeOptions 用于 Invoke/StreamInvoke 的额外参数。
type InvokeOptions struct {
	Temperature *float32
	MaxTokens   *int
}

// NewHelloAgentsLLM 初始化 LLM 客户端。优先使用传入参数，未提供则从环境变量读取。
// 会尝试加载当前目录下的 .env 文件（若存在），与 Python 版行为一致。
// provider 为空时自动检测；设为 "custom" 时强制使用 LLM_API_KEY / LLM_BASE_URL。
func NewHelloAgentsLLM(model, apiKey, baseURL, provider string, temperature float32, maxTokens, timeout *int) (*HelloAgentsLLM, error) {
	_ = godotenv.Load(".env")
	llm := &HelloAgentsLLM{
		Model:       firstNonEmptyOrLast(model, os.Getenv("LLM_MODEL_ID")),
		Temperature: temperature,
	}
	if temperature == 0 {
		llm.Temperature = 0.7
	}
	if maxTokens != nil {
		llm.MaxTokens = *maxTokens
	}
	if timeout != nil {
		llm.Timeout = *timeout
	} else if t := os.Getenv("LLM_TIMEOUT"); t != "" {
		if n, err := strconv.Atoi(t); err == nil {
			llm.Timeout = n
		}
	}
	if llm.Timeout <= 0 {
		llm.Timeout = 60
	}

	reqProvider := strings.ToLower(strings.TrimSpace(provider))
	llm.Provider = autoDetectProvider(apiKey, baseURL)
	if reqProvider == ProviderCustom {
		llm.Provider = ProviderCustom
		llm.APIKey = firstNonEmptyOrLast(apiKey, os.Getenv("LLM_API_KEY"))
		llm.BaseURL = firstNonEmptyOrLast(baseURL, os.Getenv("LLM_BASE_URL"))
	} else if reqProvider != "" {
		llm.Provider = reqProvider
	}
	if llm.Provider != ProviderCustom {
		resolvedKey, resolvedURL := resolveCredentials(llm.Provider, apiKey, baseURL)
		llm.APIKey = resolvedKey
		llm.BaseURL = resolvedURL
	}

	if llm.Model == "" {
		llm.Model = getDefaultModel(llm.Provider, llm.Model)
	}
	if llm.APIKey == "" || llm.BaseURL == "" {
		return nil, &HelloAgentsError{Msg: "API密钥和服务地址必须被提供或在环境变量中定义"}
	}

	llm.client = createClient(llm.APIKey, llm.BaseURL)
	return llm, nil
}

// createClient 根据 apiKey 与 baseURL 创建 go-openai 客户端，与 Python 的 _create_client 对应。
func createClient(apiKey, baseURL string) *openai.Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = strings.TrimSuffix(baseURL, "/")
	if cfg.BaseURL != "" && !strings.HasPrefix(cfg.BaseURL, "http") {
		cfg.BaseURL = "https://" + cfg.BaseURL
	}
	return openai.NewClientWithConfig(cfg)
}

// firstNonEmptyOrLast 返回第一个非空字符串；若均空则返回最后一个（作为默认值）。
func firstNonEmptyOrLast(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	if len(strs) > 0 {
		return strs[len(strs)-1]
	}
	return ""
}

func autoDetectProvider(apiKey, baseURL string) string {
	key := firstNonEmptyOrLast(apiKey, os.Getenv("LLM_API_KEY"))
	url := firstNonEmptyOrLast(baseURL, os.Getenv("LLM_BASE_URL"))

	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI
	}
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		return ProviderDeepSeek
	}
	if os.Getenv("DASHSCOPE_API_KEY") != "" {
		return ProviderQwen
	}
	if os.Getenv("MODELSCOPE_API_KEY") != "" {
		return ProviderModelScope
	}
	if os.Getenv("KIMI_API_KEY") != "" || os.Getenv("MOONSHOT_API_KEY") != "" {
		return ProviderKimi
	}
	if os.Getenv("ZHIPU_API_KEY") != "" || os.Getenv("GLM_API_KEY") != "" {
		return ProviderZhipu
	}
	if os.Getenv("OLLAMA_API_KEY") != "" || os.Getenv("OLLAMA_HOST") != "" {
		return ProviderOllama
	}
	if os.Getenv("VLLM_API_KEY") != "" || os.Getenv("VLLM_HOST") != "" {
		return ProviderVLLM
	}

	keyLower := strings.ToLower(key)
	if strings.HasPrefix(key, "ms-") {
		return ProviderModelScope
	}
	if keyLower == "ollama" {
		return ProviderOllama
	}
	if keyLower == "vllm" {
		return ProviderVLLM
	}
	if keyLower == "local" {
		return ProviderLocal
	}
	if len(key) > 50 && strings.HasPrefix(key, "sk-") {
		// 可能是 OpenAI/DeepSeek/Kimi，不在此处区分
	}
	if strings.Contains(key, ".") && (strings.HasSuffix(key, ".") || strings.Contains(key[len(key)-min(20, len(key)):], ".")) {
		return ProviderZhipu
	}

	urlLower := strings.ToLower(url)
	if strings.Contains(urlLower, "api.openai.com") {
		return ProviderOpenAI
	}
	if strings.Contains(urlLower, "api.deepseek.com") {
		return ProviderDeepSeek
	}
	if strings.Contains(urlLower, "dashscope.aliyuncs.com") {
		return ProviderQwen
	}
	if strings.Contains(urlLower, "api-inference.modelscope.cn") {
		return ProviderModelScope
	}
	if strings.Contains(urlLower, "api.moonshot.cn") {
		return ProviderKimi
	}
	if strings.Contains(urlLower, "open.bigmodel.cn") {
		return ProviderZhipu
	}
	if strings.Contains(urlLower, "localhost") || strings.Contains(urlLower, "127.0.0.1") {
		if strings.Contains(urlLower, ":11434") || strings.Contains(urlLower, "ollama") {
			return ProviderOllama
		}
		if strings.Contains(urlLower, ":8000") && strings.Contains(urlLower, "vllm") {
			return ProviderVLLM
		}
		if strings.Contains(urlLower, ":8080") || strings.Contains(urlLower, ":7860") {
			return ProviderLocal
		}
		if keyLower == "ollama" {
			return ProviderOllama
		}
		if keyLower == "vllm" {
			return ProviderVLLM
		}
		return ProviderLocal
	}
	if strings.Contains(urlLower, ":8080") || strings.Contains(urlLower, ":7860") || strings.Contains(urlLower, ":5000") {
		return ProviderLocal
	}
	return ProviderAuto
}

func resolveCredentials(provider, apiKey, baseURL string) (key, base string) {
	get := func(keys ...string) string {
		for _, k := range keys {
			if v := os.Getenv(k); v != "" {
				return v
			}
		}
		return ""
	}
	switch provider {
	case ProviderOpenAI:
		return firstNonEmptyOrLast(apiKey, get("OPENAI_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL", "OPENAI_BASE_URL"), "https://api.openai.com/v1")
	case ProviderDeepSeek:
		return firstNonEmptyOrLast(apiKey, get("DEEPSEEK_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "https://api.deepseek.com")
	case ProviderQwen:
		return firstNonEmptyOrLast(apiKey, get("DASHSCOPE_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "https://dashscope.aliyuncs.com/compatible-mode/v1")
	case ProviderModelScope:
		return firstNonEmptyOrLast(apiKey, get("MODELSCOPE_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "https://api-inference.modelscope.cn/v1/")
	case ProviderKimi:
		return firstNonEmptyOrLast(apiKey, get("KIMI_API_KEY", "MOONSHOT_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "https://api.moonshot.cn/v1")
	case ProviderZhipu:
		return firstNonEmptyOrLast(apiKey, get("ZHIPU_API_KEY", "GLM_API_KEY", "LLM_API_KEY")),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "https://open.bigmodel.cn/api/paas/v4")
	case ProviderOllama:
		return firstNonEmptyOrLast(apiKey, get("OLLAMA_API_KEY", "LLM_API_KEY"), "ollama"),
			firstNonEmptyOrLast(baseURL, get("OLLAMA_HOST", "LLM_BASE_URL"), "http://localhost:11434/v1")
	case ProviderVLLM:
		return firstNonEmptyOrLast(apiKey, get("VLLM_API_KEY", "LLM_API_KEY"), "vllm"),
			firstNonEmptyOrLast(baseURL, get("VLLM_HOST", "LLM_BASE_URL"), "http://localhost:8000/v1")
	case ProviderLocal:
		return firstNonEmptyOrLast(apiKey, get("LLM_API_KEY"), "local"),
			firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"), "http://localhost:8000/v1")
	case ProviderCustom:
		return firstNonEmptyOrLast(apiKey, get("LLM_API_KEY")), firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"))
	default:
		return firstNonEmptyOrLast(apiKey, get("LLM_API_KEY")), firstNonEmptyOrLast(baseURL, get("LLM_BASE_URL"))
	}
}

func getDefaultModel(provider, fallbackModel string) string {
	switch provider {
	case ProviderOpenAI:
		return "gpt-3.5-turbo"
	case ProviderDeepSeek:
		return "deepseek-chat"
	case ProviderQwen:
		return "qwen-plus"
	case ProviderModelScope:
		return "Qwen/Qwen2.5-72B-Instruct"
	case ProviderKimi:
		return "moonshot-v1-8k"
	case ProviderZhipu:
		return "glm-4.7"
	case ProviderOllama:
		return "llama3.2"
	case ProviderVLLM:
		return "meta-llama/Llama-2-7b-chat-hf"
	case ProviderLocal:
		return "local-model"
	case ProviderCustom:
		if fallbackModel != "" {
			return fallbackModel
		}
		return "gpt-3.5-turbo"
	default:
		base := os.Getenv("LLM_BASE_URL")
		bl := strings.ToLower(base)
		switch {
		case strings.Contains(bl, "modelscope"):
			return "Qwen/Qwen2.5-72B-Instruct"
		case strings.Contains(bl, "deepseek"):
			return "deepseek-chat"
		case strings.Contains(bl, "dashscope"):
			return "qwen-plus"
		case strings.Contains(bl, "moonshot"):
			return "moonshot-v1-8k"
		case strings.Contains(bl, "bigmodel"):
			return "glm-4"
		case strings.Contains(bl, "ollama") || strings.Contains(bl, ":11434"):
			return "llama3.2"
		case strings.Contains(bl, ":8000") || strings.Contains(bl, "vllm"):
			return "meta-llama/Llama-2-7b-chat-hf"
		case strings.Contains(bl, "localhost") || strings.Contains(bl, "127.0.0.1"):
			return "local-model"
		default:
			return "gpt-3.5-turbo"
		}
	}
}

// messagesToOpenAI 将 ChatMessage 列表转为 go-openai 的 Messages。
func messagesToOpenAI(msgs []ChatMessage) []openai.ChatCompletionMessage {
	out := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, m := range msgs {
		role := openai.ChatMessageRoleUser
		switch strings.ToLower(m.Role) {
		case "system":
			role = openai.ChatMessageRoleSystem
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		case "user":
			role = openai.ChatMessageRoleUser
		}
		out = append(out, openai.ChatCompletionMessage{Role: role, Content: m.Content})
	}
	return out
}

// Think 调用 LLM 并流式返回内容。temperature 为 nil 时使用初始化时的值。
func (l *HelloAgentsLLM) Think(ctx context.Context, messages []ChatMessage, temperature *float32) (<-chan string, <-chan error) {
	ch := make(chan string, 32)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errCh)
		temp := l.Temperature
		if temperature != nil {
			temp = *temperature
		}
		req := openai.ChatCompletionRequest{
			Model:       l.Model,
			Messages:    messagesToOpenAI(messages),
			Temperature: temp,
			Stream:      true,
		}
		if l.MaxTokens > 0 {
			req.MaxTokens = l.MaxTokens
		}
		if ctx == nil {
			ctx = context.Background()
		}
		if l.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(l.Timeout)*time.Second)
			defer cancel()
		}
		stream, err := l.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			errCh <- &LLMError{Msg: "LLM调用失败", Err: err}
			return
		}
		defer stream.Close()
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errCh <- &LLMError{Msg: "LLM流式调用失败", Err: err}
				return
			}
			if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
				ch <- resp.Choices[0].Delta.Content
			}
		}
	}()
	return ch, errCh
}

// Invoke 非流式调用，返回完整回复文本。
func (l *HelloAgentsLLM) Invoke(ctx context.Context, messages []ChatMessage, opts *InvokeOptions) (string, error) {
	temp := l.Temperature
	maxTok := l.MaxTokens
	if opts != nil {
		if opts.Temperature != nil {
			temp = *opts.Temperature
		}
		if opts.MaxTokens != nil {
			maxTok = *opts.MaxTokens
		}
	}
	req := openai.ChatCompletionRequest{
		Model:       l.Model,
		Messages:    messagesToOpenAI(messages),
		Temperature: temp,
	}
	if maxTok > 0 {
		req.MaxTokens = maxTok
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if l.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(l.Timeout)*time.Second)
		defer cancel()
	}
	resp, err := l.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", &LLMError{Msg: "LLM调用失败", Err: err}
	}
	if len(resp.Choices) == 0 {
		return "", &LLMError{Msg: "LLM返回无内容"}
	}
	content := resp.Choices[0].Message.Content
	return content, nil
}

// StreamInvoke 流式调用的别名，与 Think 行为一致。
func (l *HelloAgentsLLM) StreamInvoke(ctx context.Context, messages []ChatMessage, opts *InvokeOptions) (<-chan string, <-chan error) {
	var temp *float32
	if opts != nil && opts.Temperature != nil {
		temp = opts.Temperature
	}
	return l.Think(ctx, messages, temp)
}
