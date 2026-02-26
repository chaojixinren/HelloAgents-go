package observability

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"helloagents-go/hello_agents/logging"
)

// TraceLogger provides dual-format auditing trace output (JSONL + HTML).
type TraceLogger struct {
	OutputDir      string
	Sanitize       bool
	HTMLIncludeRaw bool

	SessionID string
	JSONLPath string
	HTMLPath  string

	mu        sync.Mutex
	events    []map[string]any
	jsonlFile *os.File
	htmlFile  *os.File
	closed    bool
}

func NewTraceLogger(outputDir string, sanitize, htmlIncludeRawResponse bool) (*TraceLogger, error) {
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	l := &TraceLogger{
		OutputDir:      outputDir,
		Sanitize:       sanitize,
		HTMLIncludeRaw: htmlIncludeRawResponse,
		events:         make([]map[string]any, 0, 128),
	}
	l.SessionID = l.generateSessionID()
	l.JSONLPath = filepath.Join(outputDir, fmt.Sprintf("trace-%s.jsonl", l.SessionID))
	l.HTMLPath = filepath.Join(outputDir, fmt.Sprintf("trace-%s.html", l.SessionID))

	jsonlFile, err := os.OpenFile(l.JSONLPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	htmlFile, err := os.OpenFile(l.HTMLPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		_ = jsonlFile.Close()
		return nil, err
	}

	l.jsonlFile = jsonlFile
	l.htmlFile = htmlFile

	if err := l.writeHTMLHeader(); err != nil {
		_ = l.jsonlFile.Close()
		_ = l.htmlFile.Close()
		return nil, err
	}
	return l, nil
}

func (l *TraceLogger) generateSessionID() string {
	now := time.Now().Format("20060102-150405")
	suffixBytes := make([]byte, 2)
	if _, err := rand.Read(suffixBytes); err != nil {
		return fmt.Sprintf("s-%s-%04x", now, time.Now().UnixNano()&0xffff)
	}
	return fmt.Sprintf("s-%s-%02x%02x", now, suffixBytes[0], suffixBytes[1])
}

func (l *TraceLogger) LogEvent(event string, payload map[string]any, step *int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}

	eventObj := map[string]any{
		"ts":         nowPythonISOTime(),
		"session_id": l.SessionID,
		"step":       nil,
		"event":      event,
		"payload":    payload,
	}
	if step != nil {
		eventObj["step"] = *step
	}
	if l.Sanitize {
		eventObj = l.sanitizeEvent(eventObj)
	}

	l.events = append(l.events, eventObj)
	line, _ := json.Marshal(eventObj)
	_, _ = l.jsonlFile.WriteString(string(line) + "\n")
	_ = l.jsonlFile.Sync()

	_ = l.writeHTMLEvent(eventObj)
}

func (l *TraceLogger) sanitizeEvent(event map[string]any) map[string]any {
	deepCopy, ok := deepClone(event).(map[string]any)
	if !ok {
		return event
	}
	deepCopy["payload"] = l.sanitizeValue(deepCopy["payload"])
	return deepCopy
}

func (l *TraceLogger) sanitizeValue(value any) any {
	switch v := value.(type) {
	case string:
		apiKeyRE := regexp.MustCompile(`sk-[a-zA-Z0-9]+`)
		bearerRE := regexp.MustCompile(`Bearer\s+[a-zA-Z0-9_\-]+`)
		pathRE := regexp.MustCompile(`(/Users/|/home/|C:\\Users\\)[^/\\]+`)

		v = apiKeyRE.ReplaceAllString(v, "sk-***")
		v = bearerRE.ReplaceAllString(v, "Bearer ***")
		v = pathRE.ReplaceAllString(v, `${1}***`)
		return v
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[k] = l.sanitizeValue(vv)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, vv := range v {
			out[k] = l.sanitizeValue(vv).(string)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, l.sanitizeValue(item))
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, l.sanitizeValue(item).(string))
		}
		return out
	default:
		return value
	}
}

func deepClone(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

func (l *TraceLogger) Finalize() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}

	stats := l.computeStats()
	if err := l.writeHTMLFooter(stats); err != nil {
		return err
	}

	if err := l.jsonlFile.Close(); err != nil {
		return err
	}
	if err := l.htmlFile.Close(); err != nil {
		return err
	}

	l.closed = true
	logging.Info("Trace 已保存", "jsonl", l.JSONLPath, "html", l.HTMLPath)
	return nil
}

func (l *TraceLogger) computeStats() map[string]any {
	stats := map[string]any{
		"total_steps":      0,
		"total_tokens":     0,
		"total_cost":       0.0,
		"tool_calls":       map[string]int{},
		"errors":           []map[string]any{},
		"duration_seconds": 0.0,
		"model_calls":      0,
	}

	var sessionStart time.Time
	var sessionEnd time.Time

	for _, event := range l.events {
		eventType, _ := event["event"].(string)

		if tsStr, ok := event["ts"].(string); ok {
			ts, err := parseTraceTimestamp(tsStr)
			if err == nil {
				if eventType == "session_start" {
					sessionStart = ts
				}
				if eventType == "session_end" {
					sessionEnd = ts
				}
			}
		}

		if rawStep, exists := event["step"]; exists && rawStep != nil {
			current := intFromAny(rawStep)
			if current > stats["total_steps"].(int) {
				stats["total_steps"] = current
			}
		}

		payload, _ := event["payload"].(map[string]any)
		if eventType == "model_output" && payload != nil {
			if usage, ok := payload["usage"].(map[string]any); ok {
				stats["total_tokens"] = stats["total_tokens"].(int) + intFromAny(usage["total_tokens"])
				stats["total_cost"] = stats["total_cost"].(float64) + floatFromAny(usage["cost"])
			}
			stats["model_calls"] = stats["model_calls"].(int) + 1
		}

		if eventType == "tool_call" && payload != nil {
			toolName := "unknown"
			if name, ok := payload["tool_name"].(string); ok {
				toolName = name
			}
			calls := stats["tool_calls"].(map[string]int)
			calls[toolName]++
		}

		if eventType == "error" && payload != nil {
			errItem := map[string]any{
				"step":    event["step"],
				"type":    payload["error_type"],
				"message": payload["message"],
			}
			errs := stats["errors"].([]map[string]any)
			errs = append(errs, errItem)
			stats["errors"] = errs
		}
	}

	if !sessionStart.IsZero() && !sessionEnd.IsZero() {
		stats["duration_seconds"] = sessionEnd.Sub(sessionStart).Seconds()
	}

	return stats
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case int16:
		return int(v)
	case int8:
		return int(v)
	case uint:
		return int(v)
	case uint64:
		return int(v)
	case uint32:
		return int(v)
	case uint16:
		return int(v)
	case uint8:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	default:
		return 0
	}
}

func nowPythonISOTime() string {
	return time.Now().Format("2006-01-02T15:04:05.999999")
}

func parseTraceTimestamp(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func (l *TraceLogger) writeHTMLHeader() error {
	head := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Trace: %s</title>
  <style>
    body { font-family: ui-monospace, Menlo, Monaco, monospace; background: #121417; color: #eceff4; margin: 0; padding: 24px; }
    .panel { background: #1b2028; border: 1px solid #2f3743; border-radius: 8px; padding: 16px; margin-bottom: 16px; }
    h1 { color: #7ee787; margin: 0 0 8px 0; }
    .events { display: grid; gap: 8px; }
    .event { background: #0f141b; border-left: 3px solid #58a6ff; padding: 10px; border-radius: 6px; }
    .event-header { color: #8b949e; font-size: 12px; margin-bottom: 4px; }
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; }
    table { width: 100%%; border-collapse: collapse; }
    td, th { border-bottom: 1px solid #2f3743; padding: 6px 4px; text-align: left; }
  </style>
</head>
<body>
  <div class="panel">
    <h1>Trace Session</h1>
    <div>session_id: %s</div>
    <div>generated_at: %s</div>
  </div>
  <div class="panel" id="stats-panel"><h2>Stats</h2><div>pending finalize...</div></div>
  <div class="panel"><h2>Events</h2><div class="events" id="events">`, l.SessionID, l.SessionID, time.Now().Format("2006-01-02 15:04:05"))

	_, err := l.htmlFile.WriteString(head)
	return err
}

func (l *TraceLogger) writeHTMLEvent(event map[string]any) error {
	eventJSON, _ := json.MarshalIndent(event, "", "  ")
	esc := template.HTMLEscapeString(string(eventJSON))
	line := fmt.Sprintf(`<div class="event"><div class="event-header">%s</div><pre>%s</pre></div>`, template.HTMLEscapeString(fmt.Sprintf("%v", event["event"])), esc)
	_, err := l.htmlFile.WriteString(line)
	return err
}

func (l *TraceLogger) writeHTMLFooter(stats map[string]any) error {
	toolRows := ""
	toolCalls, _ := stats["tool_calls"].(map[string]int)
	toolNames := make([]string, 0, len(toolCalls))
	for name := range toolCalls {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)
	for _, name := range toolNames {
		toolRows += fmt.Sprintf("<tr><td>%s</td><td>%d</td></tr>", template.HTMLEscapeString(name), toolCalls[name])
	}
	if toolRows == "" {
		toolRows = "<tr><td colspan=\"2\">none</td></tr>"
	}

	errItems := ""
	errorsList, _ := stats["errors"].([]map[string]any)
	if len(errorsList) == 0 {
		errItems = "<li>none</li>"
	} else {
		parts := make([]string, 0, len(errorsList))
		for _, item := range errorsList {
			parts = append(parts, fmt.Sprintf("<li>[%v] %v: %v</li>", item["step"], item["type"], template.HTMLEscapeString(fmt.Sprintf("%v", item["message"]))))
		}
		errItems = strings.Join(parts, "")
	}

	statsHTML := fmt.Sprintf(`<div><strong>total_steps</strong>: %v</div>
<div><strong>total_tokens</strong>: %v</div>
<div><strong>total_cost</strong>: %.6f</div>
<div><strong>duration_seconds</strong>: %.2f</div>
<div><strong>model_calls</strong>: %v</div>
<h3>Tool Calls</h3>
<table><tr><th>tool</th><th>count</th></tr>%s</table>
<h3>Errors</h3>
<ul>%s</ul>`,
		stats["total_steps"],
		stats["total_tokens"],
		stats["total_cost"].(float64),
		stats["duration_seconds"].(float64),
		stats["model_calls"],
		toolRows,
		errItems,
	)

	footer := fmt.Sprintf(`</div></div>
<script>document.getElementById('stats-panel').innerHTML = %q;</script>
</body>
</html>`, statsHTML)
	_, err := l.htmlFile.WriteString(footer)
	return err
}
