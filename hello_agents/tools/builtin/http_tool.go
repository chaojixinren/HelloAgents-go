package builtin

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

// HTTPTool issues HTTP/HTTPS requests and returns the response body, status
// code and selected headers. It is a generic primitive — agents can use it to
// fetch web pages, call REST APIs, or download JSON.
type HTTPTool struct {
	tools.BaseTool
	Client         *http.Client
	DefaultTimeout int
	MaxBodyBytes   int
	UserAgent      string
}

// NewHTTPTool creates an HTTPTool with sensible defaults (30s timeout, 1MB body cap).
func NewHTTPTool() *HTTPTool {
	return NewHTTPToolWithOptions(30, 1<<20, "HelloAgents-Go/1.0")
}

// NewHTTPToolWithOptions builds an HTTPTool with explicit settings.
func NewHTTPToolWithOptions(timeoutSec, maxBodyBytes int, userAgent string) *HTTPTool {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}
	if userAgent == "" {
		userAgent = "HelloAgents-Go/1.0"
	}
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}
	base := tools.NewBaseTool(
		"HTTP",
		"发送 HTTP/HTTPS 请求并返回响应体、状态码与响应头。支持 GET/POST/PUT/DELETE 等方法、自定义请求头与请求体。",
		false,
	)
	base.Parameters = map[string]tools.ToolParameter{
		"url": {
			Name:        "url",
			Type:        "string",
			Description: "请求的完整 URL（必须以 http:// 或 https:// 开头）",
			Required:    true,
		},
		"method": {
			Name:        "method",
			Type:        "string",
			Description: "HTTP 方法，默认 GET",
			Required:    false,
			Default:     "GET",
		},
		"headers": {
			Name:        "headers",
			Type:        "object",
			Description: "自定义请求头键值对",
			Required:    false,
		},
		"body": {
			Name:        "body",
			Type:        "string",
			Description: "请求体内容（用于 POST/PUT 等）",
			Required:    false,
		},
		"timeout": {
			Name:        "timeout",
			Type:        "integer",
			Description: "请求超时（秒），默认 30",
			Required:    false,
			Default:     30,
		},
	}
	t := &HTTPTool{
		BaseTool:       base,
		Client:         client,
		DefaultTimeout: timeoutSec,
		MaxBodyBytes:   maxBodyBytes,
		UserAgent:      userAgent,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

// Run executes the HTTP request.
func (t *HTTPTool) Run(parameters map[string]any) tools.ToolResponse {
	url, _ := parameters["url"].(string)
	url = strings.TrimSpace(url)
	if url == "" {
		return tools.Error("url 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}
	if !strings.HasPrefix(strings.ToLower(url), "http://") &&
		!strings.HasPrefix(strings.ToLower(url), "https://") {
		return tools.Error("url 必须以 http:// 或 https:// 开头", tools.ToolErrorCodeInvalidFormat, map[string]any{"url": url})
	}

	method, _ := parameters["method"].(string)
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return tools.Error("不支持的 HTTP 方法: "+method, tools.ToolErrorCodeInvalidParam, map[string]any{"method": method})
	}

	bodyStr, _ := parameters["body"].(string)
	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return tools.Error(fmt.Sprintf("构造请求失败: %v", err), tools.ToolErrorCodeInvalidFormat, map[string]any{"url": url})
	}
	req.Header.Set("User-Agent", t.UserAgent)
	if bodyStr != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers, ok := parameters["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, asString(v))
		}
	}

	// Only override the client timeout when the caller explicitly asks for
	// one; otherwise respect whatever the constructor (or test) set on
	// t.Client.Timeout.
	if v := intFromAny(parameters["timeout"]); v > 0 {
		t.Client.Timeout = time.Duration(v) * time.Second
	}
	timeoutSec := int(t.Client.Timeout / time.Second)
	if timeoutSec <= 0 {
		timeoutSec = t.DefaultTimeout
	}

	start := time.Now()
	resp, err := t.Client.Do(req)
	if err != nil {
		return tools.Error(
			fmt.Sprintf("HTTP 请求失败: %v", err),
			tools.ToolErrorCodeNetworkError,
			map[string]any{"url": url, "method": method},
		)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, int64(t.MaxBodyBytes)+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return tools.Error(
			fmt.Sprintf("读取响应体失败: %v", err),
			tools.ToolErrorCodeNetworkError,
			map[string]any{"url": url, "status": resp.StatusCode},
		)
	}
	elapsed := time.Since(start).Milliseconds()

	truncated := len(bodyBytes) > t.MaxBodyBytes
	bodyStr = string(bodyBytes)
	if truncated {
		bodyStr = bodyStr[:t.MaxBodyBytes]
	}

	// Detect JSON content-type so callers get a parsed payload too.
	contentType := resp.Header.Get("Content-Type")
	isJSON := strings.Contains(strings.ToLower(contentType), "application/json")
	var parsed any
	if isJSON && !truncated {
		_ = json.Unmarshal(bodyBytes, &parsed)
	}

	respHeaders := map[string]string{}
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	status := tools.ToolStatusSuccess
	text := fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if resp.StatusCode >= 400 {
		status = tools.ToolStatusError
		text = fmt.Sprintf("HTTP 请求返回错误状态码: %d", resp.StatusCode)
	} else if truncated {
		status = tools.ToolStatusPartial
		text = fmt.Sprintf("HTTP %d（响应体已截断到 %d 字节）", resp.StatusCode, t.MaxBodyBytes)
	}

	data := map[string]any{
		"url":          url,
		"method":       method,
		"status_code":  resp.StatusCode,
		"status_text":  http.StatusText(resp.StatusCode),
		"headers":      respHeaders,
		"body":         bodyStr,
		"body_size":    len(bodyBytes),
		"truncated":    truncated,
		"content_type": contentType,
	}
	if parsed != nil {
		data["json"] = parsed
	}

	return tools.ToolResponse{
		Status: status,
		Text:   text,
		Data:   data,
		Stats: map[string]any{
			"time_ms":   elapsed,
			"timeout_s": timeoutSec,
		},
		Context: map[string]any{
			"tool_name": "HTTP",
		},
	}
}

// asString coerces common scalar types to string.
func asString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// String provides a debug representation.
func (t *HTTPTool) String() string {
	return fmt.Sprintf("HTTPTool(timeout=%ds, max_body=%d)", t.DefaultTimeout, t.MaxBodyBytes)
}
