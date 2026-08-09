package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

// BashTool executes shell commands in a constrained environment.
//
// Safety controls:
//   - BlockedCommands: a deny-list of substrings that always reject a command.
//     Defaults to a small set of catastrophic patterns (rm -rf /, mkfs, ...).
//   - AllowedCommands: when non-empty, only commands whose first token appears
//     in this set are permitted. Empty means "allow any (still subject to the
//     deny-list)".
//   - WorkingDir: restricts where the command runs.
//   - DefaultTimeout: per-command deadline in seconds.
type BashTool struct {
	tools.BaseTool
	WorkingDir      string
	AllowedCommands map[string]bool
	BlockedCommands []string
	DefaultTimeout  int
}

// NewBashTool creates a BashTool with safe defaults (deny-list active, no
// allow-list, 30s timeout, current working directory).
func NewBashTool() *BashTool {
	return NewBashToolWithOptions("", nil, nil, 30)
}

// NewBashToolWithOptions builds a BashTool with explicit safety controls.
func NewBashToolWithOptions(workingDir string, allowedCommands, blockedCommands []string, defaultTimeout int) *BashTool {
	if defaultTimeout <= 0 {
		defaultTimeout = 30
	}
	allowed := map[string]bool{}
	for _, c := range allowedCommands {
		allowed[strings.TrimSpace(c)] = true
	}
	if len(blockedCommands) == 0 {
		blockedCommands = defaultBlockedCommands()
	}
	base := tools.NewBaseTool(
		"Bash",
		"在受限环境中执行 shell 命令。支持管道与重定向，受超时、工作目录与命令黑/白名单限制。仅用于受信任的任务。",
		false,
	)
	base.Parameters = map[string]tools.ToolParameter{
		"command": {
			Name:        "command",
			Type:        "string",
			Description: "要执行的 shell 命令",
			Required:    true,
		},
		"timeout": {
			Name:        "timeout",
			Type:        "integer",
			Description: "超时时间（秒），默认 30",
			Required:    false,
			Default:     30,
		},
	}
	t := &BashTool{
		BaseTool:        base,
		WorkingDir:      workingDir,
		AllowedCommands: allowed,
		BlockedCommands: blockedCommands,
		DefaultTimeout:  defaultTimeout,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func defaultBlockedCommands() []string {
	return []string{
		"rm -rf /",
		"rm -rf ~",
		"rm -rf *",
		"mkfs",
		"dd if=/dev/zero of=/dev/sd",
		"dd if=/dev/zero of=/dev/nvme",
		":(){:|:&};:",
		"shutdown",
		"reboot",
		"halt",
		"init 0",
		"init 6",
	}
}

// Run executes the command and returns stdout/stderr/exit code.
func (t *BashTool) Run(parameters map[string]any) tools.ToolResponse {
	command, _ := parameters["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return tools.Error("command 不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	if reason := t.checkBlocked(command); reason != "" {
		return tools.Error(
			fmt.Sprintf("命令被安全策略拒绝: %s", reason),
			tools.ToolErrorCodeAccessDenied,
			map[string]any{"command": command},
		)
	}
	if reason := t.checkAllowed(command); reason != "" {
		return tools.Error(
			fmt.Sprintf("命令不在白名单中: %s", reason),
			tools.ToolErrorCodeAccessDenied,
			map[string]any{"command": command},
		)
	}

	timeoutSec := t.DefaultTimeout
	if v := intFromAny(parameters["timeout"]); v > 0 {
		timeoutSec = v
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if t.WorkingDir != "" {
		cmd.Dir = t.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return tools.Error(
				fmt.Sprintf("命令执行超时（%d 秒）", timeoutSec),
				tools.ToolErrorCodeTimeout,
				map[string]any{
					"command": command,
					"timeout": timeoutSec,
					"stdout":  stdout.String(),
					"stderr":  stderr.String(),
				},
			)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return tools.Error(
				fmt.Sprintf("命令执行失败: %v", err),
				tools.ToolErrorCodeExecutionError,
				map[string]any{"command": command},
			)
		}
	}

	status := tools.ToolStatusSuccess
	text := "命令执行成功"
	if exitCode != 0 {
		status = tools.ToolStatusError
		text = fmt.Sprintf("命令以非零状态码退出: %d", exitCode)
	} else if stderr.Len() > 0 && stdout.Len() == 0 {
		status = tools.ToolStatusPartial
		text = "命令执行完成，但仅在 stderr 产生输出"
	}

	return tools.ToolResponse{
		Status: status,
		Text:   text,
		Data: map[string]any{
			"command":    command,
			"stdout":     stdout.String(),
			"stderr":     stderr.String(),
			"exit_code":  exitCode,
			"stdout_len": stdout.Len(),
			"stderr_len": stderr.Len(),
		},
		Stats: map[string]any{
			"time_ms":   elapsed,
			"timeout_s": timeoutSec,
		},
		Context: map[string]any{
			"working_dir": t.WorkingDir,
		},
	}
}

// checkBlocked returns a non-empty reason string when the command matches a
// deny-listed pattern.
func (t *BashTool) checkBlocked(command string) string {
	lower := strings.ToLower(command)
	for _, pattern := range t.BlockedCommands {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return fmt.Sprintf("匹配黑名单模式 %q", pattern)
		}
	}
	return ""
}

// checkAllowed returns a non-empty reason string when an allow-list is
// configured and the command's first token is not in it.
func (t *BashTool) checkAllowed(command string) string {
	if len(t.AllowedCommands) == 0 {
		return ""
	}
	first := firstCommandToken(command)
	if first == "" {
		return "无法解析命令"
	}
	if t.AllowedCommands[first] {
		return ""
	}
	return fmt.Sprintf("首个命令 %q 不在白名单", first)
}

// firstCommandToken extracts the executable name, skipping any leading
// VAR=value environment assignments.
func firstCommandToken(command string) string {
	fields := strings.Fields(command)
	for _, f := range fields {
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "-") {
			continue
		}
		return f
	}
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// String provides a debug representation.
func (t *BashTool) String() string {
	return fmt.Sprintf("BashTool(working_dir=%s, allowed=%d, blocked=%d)",
		t.WorkingDir, len(t.AllowedCommands), len(t.BlockedCommands))
}
