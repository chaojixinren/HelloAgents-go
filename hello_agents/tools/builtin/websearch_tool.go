package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

// SearchResult is a single web search hit.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source,omitempty"`
}

// SearchProvider abstracts the search backend so callers can plug in
// SerpAPI, Bing, Brave, etc. The built-in DuckDuckGoProvider is a no-key
// fallback intended for demos.
type SearchProvider interface {
	Search(query string, num int) ([]SearchResult, error)
	Name() string
}

// WebSearchTool performs web searches via a pluggable SearchProvider.
type WebSearchTool struct {
	tools.BaseTool
	Provider    SearchProvider
	MaxResults  int
	HTTPTimeout int
}

// NewWebSearchTool creates a WebSearchTool backed by the default
// DuckDuckGo Instant Answer provider. For production use, inject a real
// search backend with NewWebSearchToolWithProvider.
func NewWebSearchTool() *WebSearchTool {
	return NewWebSearchToolWithProvider(&DuckDuckGoProvider{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	})
}

// NewWebSearchToolWithProvider creates a WebSearchTool with a custom provider.
func NewWebSearchToolWithProvider(provider SearchProvider) *WebSearchTool {
	maxResults := 10
	base := tools.NewBaseTool(
		"WebSearch",
		"在网络上搜索实时信息，返回标题、URL 与摘要。适合获取最新资讯或验证事实。",
		false,
	)
	base.Parameters = map[string]tools.ToolParameter{
		"query": {
			Name:        "query",
			Type:        "string",
			Description: "搜索关键词",
			Required:    true,
		},
		"num": {
			Name:        "num",
			Type:        "integer",
			Description: "返回结果数量上限，默认 10",
			Required:    false,
			Default:     10,
		},
	}
	t := &WebSearchTool{
		BaseTool:   base,
		Provider:   provider,
		MaxResults: maxResults,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

// Run executes the search.
func (t *WebSearchTool) Run(parameters map[string]any) tools.ToolResponse {
	query, _ := parameters["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return tools.Error("query 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	num := t.MaxResults
	if v := intFromAny(parameters["num"]); v > 0 {
		num = v
	}

	if t.Provider == nil {
		return tools.Error(
			"未配置搜索 provider，请使用 NewWebSearchToolWithProvider 注入后端",
			tools.ToolErrorCodeInvalidParam,
			map[string]any{"query": query},
		)
	}

	start := time.Now()
	results, err := t.Provider.Search(query, num)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return tools.Error(
			fmt.Sprintf("搜索失败: %v", err),
			tools.ToolErrorCodeAPIError,
			map[string]any{"query": query, "provider": t.Provider.Name()},
		)
	}

	if len(results) > num {
		results = results[:num]
	}

	text := fmt.Sprintf("搜索完成，找到 %d 条结果", len(results))
	status := tools.ToolStatusSuccess
	if len(results) == 0 {
		status = tools.ToolStatusPartial
		text = "搜索完成，但未找到任何结果"
	}

	lines := make([]string, 0, len(results))
	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s\n   %s", i+1, r.Title, r.URL, r.Snippet))
	}

	return tools.ToolResponse{
		Status: status,
		Text:   text,
		Data: map[string]any{
			"query":   query,
			"results": results,
			"count":   len(results),
			"summary": strings.Join(lines, "\n\n"),
		},
		Stats: map[string]any{
			"time_ms":       elapsed,
			"provider":      t.Provider.Name(),
			"requested_num": num,
		},
		Context: map[string]any{
			"tool_name": "WebSearch",
		},
	}
}

// String provides a debug representation.
func (t *WebSearchTool) String() string {
	name := "none"
	if t.Provider != nil {
		name = t.Provider.Name()
	}
	return fmt.Sprintf("WebSearchTool(provider=%s, max=%d)", name, t.MaxResults)
}

// DuckDuckGoProvider queries the DuckDuckGo Instant Answer JSON API.
// It requires no API key, but only returns a limited set of abstract
// results (Wikipedia-like summaries). For richer results, inject a
// SerpAPI/Bing/Brave provider.
type DuckDuckGoProvider struct {
	HTTPClient *http.Client
	BaseURL    string
}

// Name returns the provider identifier.
func (p *DuckDuckGoProvider) Name() string { return "duckduckgo" }

// Search calls the DuckDuckGo Instant Answer API and flattens
// RelatedTopics into a flat result list.
func (p *DuckDuckGoProvider) Search(query string, num int) ([]SearchResult, error) {
	if p.HTTPClient == nil {
		p.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	base := p.BaseURL
	if base == "" {
		base = "https://api.duckduckgo.com/"
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("no_redirect", "1")
	params.Set("skip_disambig", "1")

	req, err := http.NewRequest("GET", base+"?"+params.Encode(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "HelloAgents-Go/1.0")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var ddg duckDuckGoResponse
	if err := json.Unmarshal(body, &ddg); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	results := make([]SearchResult, 0, num)
	source := "duckduckgo"

	// Abstract topic (top hit).
	if ddg.AbstractText != "" && ddg.AbstractURL != "" {
		results = append(results, SearchResult{
			Title:   ddg.Heading,
			URL:     ddg.AbstractURL,
			Snippet: ddg.AbstractText,
			Source:  source,
		})
	}

	// Flatten RelatedTopics (may be nested).
	for _, topic := range ddg.RelatedTopics {
		flattenDuckDuckGoTopic(topic, source, &results, num)
		if len(results) >= num {
			break
		}
	}

	if len(results) > num {
		results = results[:num]
	}
	return results, nil
}

func flattenDuckDuckGoTopic(topic duckDuckGoTopic, source string, out *[]SearchResult, limit int) {
	if len(*out) >= limit {
		return
	}
	if topic.Text != "" && topic.FirstURL != "" {
		title := topic.Text
		if idx := strings.Index(topic.Text, " - "); idx > 0 {
			title = topic.Text[:idx]
		}
		*out = append(*out, SearchResult{
			Title:   title,
			URL:     topic.FirstURL,
			Snippet: topic.Text,
			Source:  source,
		})
		return
	}
	for _, sub := range topic.Topics {
		flattenDuckDuckGoTopic(sub, source, out, limit)
		if len(*out) >= limit {
			return
		}
	}
}

type duckDuckGoResponse struct {
	Heading       string            `json:"Heading"`
	AbstractText  string            `json:"AbstractText"`
	AbstractURL   string            `json:"AbstractURL"`
	RelatedTopics []duckDuckGoTopic `json:"RelatedTopics"`
}

type duckDuckGoTopic struct {
	Text     string            `json:"Text"`
	FirstURL string            `json:"FirstURL"`
	Topics   []duckDuckGoTopic `json:"Topics"`
}
