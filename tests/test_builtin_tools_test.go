package tests_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

// ---------------------------------------------------------------------------
// BashTool
// ---------------------------------------------------------------------------

func TestBashTool_EmptyCommandRejected(t *testing.T) {
	tool := builtin.NewBashTool()
	resp := tool.Run(map[string]any{"command": "   "})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error for empty command, got status=%s text=%q", resp.Status, resp.Text)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("expected INVALID_PARAM, got %q", resp.ErrorInfo["code"])
	}
}

func TestBashTool_EchoCommand(t *testing.T) {
	tool := builtin.NewBashTool()
	resp := tool.Run(map[string]any{"command": "echo hello-agents"})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Text)
	}
	stdout, _ := resp.Data["stdout"].(string)
	if !strings.Contains(stdout, "hello-agents") {
		t.Fatalf("expected stdout to contain 'hello-agents', got %q", stdout)
	}
	if code, _ := resp.Data["exit_code"].(int); code != 0 {
		t.Fatalf("expected exit_code 0, got %d", code)
	}
	if _, ok := resp.Stats["time_ms"]; !ok {
		t.Fatalf("expected time_ms stat, got %v", resp.Stats)
	}
}

func TestBashTool_NonZeroExitCode(t *testing.T) {
	tool := builtin.NewBashTool()
	resp := tool.Run(map[string]any{"command": "exit 7"})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error status for non-zero exit, got %s", resp.Status)
	}
	if code, _ := resp.Data["exit_code"].(int); code != 7 {
		t.Fatalf("expected exit_code 7, got %d", code)
	}
}

func TestBashTool_BlockedCommandRejected(t *testing.T) {
	tool := builtin.NewBashTool()
	resp := tool.Run(map[string]any{"command": "rm -rf /"})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error for blocked command, got %s", resp.Status)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeAccessDenied {
		t.Fatalf("expected ACCESS_DENIED, got %q", resp.ErrorInfo["code"])
	}
}

func TestBashTool_AllowlistEnforced(t *testing.T) {
	tool := builtin.NewBashToolWithOptions("", []string{"echo", "pwd"}, nil, 10)

	// Allowed command.
	resp := tool.Run(map[string]any{"command": "echo allowed"})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success for allowlisted command, got %s: %s", resp.Status, resp.Text)
	}

	// Disallowed command.
	resp = tool.Run(map[string]any{"command": "ls"})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error for non-allowlisted command, got %s", resp.Status)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeAccessDenied {
		t.Fatalf("expected ACCESS_DENIED, got %q", resp.ErrorInfo["code"])
	}
}

func TestBashTool_Timeout(t *testing.T) {
	tool := builtin.NewBashToolWithOptions("", nil, nil, 1)
	resp := tool.Run(map[string]any{"command": "sleep 5"})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error on timeout, got %s", resp.Status)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeTimeout {
		t.Fatalf("expected TIMEOUT code, got %q", resp.ErrorInfo["code"])
	}
}

func TestBashTool_WorkingDir(t *testing.T) {
	tool := builtin.NewBashToolWithOptions(t.TempDir(), nil, nil, 10)
	resp := tool.Run(map[string]any{"command": "pwd"})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Text)
	}
	stdout, _ := resp.Data["stdout"].(string)
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected non-empty pwd output")
	}
}

// ---------------------------------------------------------------------------
// HTTPTool
// ---------------------------------------------------------------------------

func TestHTTPTool_InvalidURL(t *testing.T) {
	tool := builtin.NewHTTPTool()

	// Empty URL.
	resp := tool.Run(map[string]any{"url": ""})
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("expected INVALID_PARAM for empty url, got %q", resp.ErrorInfo["code"])
	}

	// Bad scheme.
	resp = tool.Run(map[string]any{"url": "ftp://example.com"})
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidFormat {
		t.Fatalf("expected INVALID_FORMAT for non-http url, got %q", resp.ErrorInfo["code"])
	}

	// Bad method.
	resp = tool.Run(map[string]any{"url": "https://example.com", "method": "TRACE"})
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("expected INVALID_PARAM for bad method, got %q", resp.ErrorInfo["code"])
	}
}

func TestHTTPTool_LocalServerGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "hello" {
			t.Errorf("expected X-Test header to be forwarded, got %q", r.Header.Get("X-Test"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok","count":42}`))
	}))
	defer srv.Close()

	tool := builtin.NewHTTPToolWithOptions(5, 1<<16, "test-agent")
	resp := tool.Run(map[string]any{
		"url":     srv.URL,
		"method":  "GET",
		"headers": map[string]any{"X-Test": "hello"},
	})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Text)
	}
	if code, _ := resp.Data["status_code"].(int); code != 200 {
		t.Fatalf("expected status 200, got %d", code)
	}
	body, _ := resp.Data["body"].(string)
	if !strings.Contains(body, "ok") {
		t.Fatalf("expected body to contain 'ok', got %q", body)
	}
	if _, ok := resp.Data["json"]; !ok {
		t.Fatalf("expected parsed JSON payload in data.json")
	}
}

func TestHTTPTool_PostBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("expected JSON content-type, got %q", ct)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte("created"))
	}))
	defer srv.Close()

	tool := builtin.NewHTTPTool()
	resp := tool.Run(map[string]any{
		"url":    srv.URL,
		"method": "POST",
		"body":   `{"name":"agent"}`,
	})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Text)
	}
	if code, _ := resp.Data["status_code"].(int); code != 201 {
		t.Fatalf("expected status 201, got %d", code)
	}
}

func TestHTTPTool_ServerErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	tool := builtin.NewHTTPTool()
	resp := tool.Run(map[string]any{"url": srv.URL})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error status for HTTP 500, got %s", resp.Status)
	}
	if code, _ := resp.Data["status_code"].(int); code != 500 {
		t.Fatalf("expected status 500, got %d", code)
	}
}

func TestHTTPTool_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("slow"))
	}))
	defer srv.Close()

	tool := builtin.NewHTTPToolWithOptions(1, 1<<16, "test")
	// Override client timeout to be aggressive for the test.
	tool.Client.Timeout = 100 * time.Millisecond
	resp := tool.Run(map[string]any{"url": srv.URL})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected error on timeout, got %s", resp.Status)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeNetworkError {
		t.Fatalf("expected NETWORK_ERROR, got %q", resp.ErrorInfo["code"])
	}
}

// ---------------------------------------------------------------------------
// WebSearchTool
// ---------------------------------------------------------------------------

type mockSearchProvider struct {
	results []builtin.SearchResult
	err     error
	calls   int
	lastQ   string
	lastNum int
}

func (m *mockSearchProvider) Search(query string, num int) ([]builtin.SearchResult, error) {
	m.calls++
	m.lastQ = query
	m.lastNum = num
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockSearchProvider) Name() string { return "mock" }

func TestWebSearchTool_EmptyQuery(t *testing.T) {
	tool := builtin.NewWebSearchToolWithProvider(&mockSearchProvider{})
	resp := tool.Run(map[string]any{"query": "  "})
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("expected INVALID_PARAM for empty query, got %q", resp.ErrorInfo["code"])
	}
}

func TestWebSearchTool_NoProviderConfigured(t *testing.T) {
	tool := builtin.NewWebSearchToolWithProvider(nil)
	resp := tool.Run(map[string]any{"query": "golang"})
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("expected INVALID_PARAM when no provider, got %q", resp.ErrorInfo["code"])
	}
}

func TestWebSearchTool_MockResults(t *testing.T) {
	mock := &mockSearchProvider{
		results: []builtin.SearchResult{
			{Title: "Go", URL: "https://go.dev", Snippet: "Go programming language", Source: "mock"},
			{Title: "Effective Go", URL: "https://go.dev/doc/effective_go", Snippet: "tips", Source: "mock"},
		},
	}
	tool := builtin.NewWebSearchToolWithProvider(mock)
	resp := tool.Run(map[string]any{"query": "golang", "num": 5})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Text)
	}
	if mock.lastQ != "golang" || mock.lastNum != 5 {
		t.Fatalf("provider got query=%q num=%d, want golang/5", mock.lastQ, mock.lastNum)
	}
	if count, _ := resp.Data["count"].(int); count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	summary, _ := resp.Data["summary"].(string)
	if !strings.Contains(summary, "go.dev") {
		t.Fatalf("expected summary to contain go.dev, got %q", summary)
	}
}

func TestWebSearchTool_ZeroResultsPartial(t *testing.T) {
	mock := &mockSearchProvider{results: nil}
	tool := builtin.NewWebSearchToolWithProvider(mock)
	resp := tool.Run(map[string]any{"query": "obscure term"})
	if resp.Status != tools.ToolStatusPartial {
		t.Fatalf("expected PARTIAL for zero results, got %s", resp.Status)
	}
	if count, _ := resp.Data["count"].(int); count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}
}

func TestWebSearchTool_ProviderError(t *testing.T) {
	mock := &mockSearchProvider{err: http.ErrAbortHandler}
	tool := builtin.NewWebSearchToolWithProvider(mock)
	resp := tool.Run(map[string]any{"query": "fail"})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("expected ERROR on provider failure, got %s", resp.Status)
	}
	if resp.ErrorInfo["code"] != tools.ToolErrorCodeAPIError {
		t.Fatalf("expected API_ERROR, got %q", resp.ErrorInfo["code"])
	}
}
