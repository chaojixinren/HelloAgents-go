package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ObservationTruncator truncates large tool outputs while keeping a recoverable full copy.
type ObservationTruncator struct {
	MaxLines          int
	MaxBytes          int
	TruncateDirection string
	OutputDir         string
}

func NewObservationTruncator(maxLines, maxBytes int, truncateDirection, outputDir string) *ObservationTruncator {
	if maxLines <= 0 {
		maxLines = 2000
	}
	if maxBytes <= 0 {
		maxBytes = 51200
	}
	if truncateDirection == "" {
		truncateDirection = "head"
	}
	if outputDir == "" {
		outputDir = "tool-output"
	}
	_ = os.MkdirAll(outputDir, 0o755)
	return &ObservationTruncator{
		MaxLines:          maxLines,
		MaxBytes:          maxBytes,
		TruncateDirection: truncateDirection,
		OutputDir:         outputDir,
	}
}

func (t *ObservationTruncator) Truncate(content string, toolName string) (string, map[string]any) {
	return t.TruncateWithMetadata(content, toolName, nil)
}

func (t *ObservationTruncator) TruncateWithMetadata(content string, toolName string, metadata map[string]any) (string, map[string]any) {
	start := time.Now()
	lines := []string{}
	if content != "" {
		lines = strings.Split(content, "\n")
	}
	bytesSize := len([]byte(content))

	if len(lines) <= t.MaxLines && bytesSize <= t.MaxBytes {
		result := map[string]any{
			"truncated":        false,
			"preview":          content,
			"full_output_path": nil,
			"stats": map[string]any{
				"original_lines": len(lines),
				"original_bytes": bytesSize,
				"time_ms":        time.Since(start).Milliseconds(),
			},
		}
		return content, result
	}

	truncatedLines := t.truncateLines(lines)
	preview := strings.Join(truncatedLines, "\n")
	if len([]byte(preview)) > t.MaxBytes {
		b := []byte(preview)
		switch t.TruncateDirection {
		case "tail":
			preview = string(b[len(b)-t.MaxBytes:])
		case "head_tail":
			half := t.MaxBytes / 2
			preview = string(b[:half]) + "\n...(中间省略)...\n" + string(b[len(b)-half:])
		default:
			preview = string(b[:t.MaxBytes])
		}
	}

	fullOutputPath := t.saveFullOutput(content, toolName, metadata)
	result := map[string]any{
		"truncated":        true,
		"preview":          preview,
		"full_output_path": fullOutputPath,
		"stats": map[string]any{
			"direction":      t.TruncateDirection,
			"original_lines": len(lines),
			"original_bytes": bytesSize,
			"kept_lines": func() int {
				if preview == "" {
					return 0
				}
				return len(strings.Split(preview, "\n"))
			}(),
			"kept_bytes": len([]byte(preview)),
			"time_ms":    time.Since(start).Milliseconds(),
		},
	}
	return preview, result
}

func (t *ObservationTruncator) truncateLines(lines []string) []string {
	if len(lines) <= t.MaxLines {
		return lines
	}
	if t.TruncateDirection == "tail" {
		return lines[len(lines)-t.MaxLines:]
	}
	if t.TruncateDirection == "head_tail" {
		half := t.MaxLines / 2
		res := make([]string, 0, t.MaxLines+1)
		res = append(res, lines[:half]...)
		res = append(res, "...(中间省略)...")
		res = append(res, lines[len(lines)-half:]...)
		return res
	}
	return lines[:t.MaxLines]
}

func (t *ObservationTruncator) saveFullOutput(content, toolName string, metadata map[string]any) string {
	timestamp := time.Now().Format("20060102_150405_000000")
	filename := fmt.Sprintf("tool_%s_%s.json", timestamp, toolName)
	path := filepath.Join(t.OutputDir, filename)
	payload := map[string]any{
		"tool":      toolName,
		"output":    content,
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"metadata":  metadata,
	}
	if payload["metadata"] == nil {
		payload["metadata"] = map[string]any{}
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
	return path
}
