package builtin

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"helloagents-go/HelloAgents-go/tools"
)

// Terminal is a tool for executing shell commands with a security whitelist.
type Terminal struct {
	*tools.BaseTool
	allowedCommands map[string]bool
	mu              sync.RWMutex
}

// NewTerminal creates a new Terminal tool with default allowed commands.
func NewTerminal() *Terminal {
	// Default safe commands for each platform
	defaultAllowed := getDefaultAllowedCommands()

	return &Terminal{
		BaseTool: tools.NewBaseTool(
			"terminal",
			"Executes shell commands from a predefined whitelist of safe commands. "+
				"Cross-platform support for Windows, Linux, and macOS. "+
				"Useful for running file operations, system information commands, and development tools.",
			[]tools.ToolParameter{
				{
					Name:        "command",
					Type:        "string",
					Description: "The shell command to execute. Must be in the allowed command whitelist.",
					Required:    true,
				},
			},
		),
		allowedCommands: defaultAllowed,
	}
}

// NewTerminalWithWhitelist creates a new Terminal tool with a custom command whitelist.
func NewTerminalWithWhitelist(allowedCommands []string) *Terminal {
	whitelist := make(map[string]bool)
	for _, cmd := range allowedCommands {
		whitelist[strings.ToLower(strings.TrimSpace(cmd))] = true
	}

	return &Terminal{
		BaseTool: tools.NewBaseTool(
			"terminal",
			"Executes shell commands from a predefined whitelist of safe commands.",
			[]tools.ToolParameter{
				{
					Name:        "command",
					Type:        "string",
					Description: "The shell command to execute. Must be in the allowed command whitelist.",
					Required:    true,
				},
			},
		),
		allowedCommands: whitelist,
	}
}

// Run executes a shell command if it's in the whitelist.
func (t *Terminal) Run(parameters map[string]interface{}) (string, error) {
	cmdParam, exists := parameters["command"]
	if !exists {
		return "", fmt.Errorf("command parameter is required")
	}

	cmdStr, ok := cmdParam.(string)
	if !ok {
		return "", fmt.Errorf("command must be a string")
	}

	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Parse the command to get the base command name
	baseCommand := parseBaseCommand(cmdStr)

	// Check if command is allowed
	t.mu.RLock()
	allowed := t.allowedCommands[strings.ToLower(baseCommand)]
	t.mu.RUnlock()

	if !allowed {
		return "", fmt.Errorf("command '%s' is not in the allowed whitelist", baseCommand)
	}

	// Execute the command
	output, err := t.executeCommand(cmdStr)
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

// AddAllowedCommand adds a command to the whitelist.
func (t *Terminal) AddAllowedCommand(command string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := strings.ToLower(strings.TrimSpace(command))
	if cmd != "" {
		t.allowedCommands[cmd] = true
	}
}

// RemoveAllowedCommand removes a command from the whitelist.
func (t *Terminal) RemoveAllowedCommand(command string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := strings.ToLower(strings.TrimSpace(command))
	delete(t.allowedCommands, cmd)
}

// IsAllowed checks if a command is in the whitelist.
func (t *Terminal) IsAllowed(command string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	baseCmd := parseBaseCommand(command)
	return t.allowedCommands[strings.ToLower(baseCmd)]
}

// ListAllowedCommands returns a list of all allowed commands.
func (t *Terminal) ListAllowedCommands() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	commands := make([]string, 0, len(t.allowedCommands))
	for cmd := range t.allowedCommands {
		commands = append(commands, cmd)
	}

	return commands
}

// executeCommand executes a shell command and returns the output.
func (t *Terminal) executeCommand(cmdStr string) (string, error) {
	var cmd *exec.Cmd

	// Detect OS and create appropriate command
	switch runtime.GOOS {
	case "windows":
		// On Windows, use cmd /c
		cmd = exec.Command("cmd", "/c", cmdStr)
	default:
		// On Unix-like systems, use sh -c
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// parseBaseCommand extracts the base command name from a command string.
func parseBaseCommand(cmdStr string) string {
	cmdStr = strings.TrimSpace(cmdStr)

	// Handle quoted commands
	if strings.HasPrefix(cmdStr, "\"") || strings.HasPrefix(cmdStr, "'") {
		// Find the closing quote
		quote := rune(cmdStr[0])
		endQuote := strings.IndexRune(cmdStr[1:], quote)
		if endQuote > 0 {
			return cmdStr[1 : endQuote+1]
		}
	}

	// Split by whitespace and get the first part
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return ""
	}

	// Remove path separators to get just the command name
	baseCmd := parts[0]
	if idx := strings.LastIndex(baseCmd, "/"); idx >= 0 {
		baseCmd = baseCmd[idx+1:]
	}
	if idx := strings.LastIndex(baseCmd, "\\"); idx >= 0 {
		baseCmd = baseCmd[idx+1:]
	}

	return baseCmd
}

// getDefaultAllowedCommands returns the default whitelist based on the OS.
func getDefaultAllowedCommands() map[string]bool {
	allowed := make(map[string]bool)

	// Common safe commands (platform-independent)
	allowed["echo"] = true
	allowed["pwd"] = true
	allowed["date"] = true
	allowed["hostname"] = true
	allowed["whoami"] = true
	allowed["uptime"] = true
	allowed["env"] = true
	allowed["printenv"] = true

	// File operations (read-only)
	allowed["ls"] = true
	allowed["dir"] = true // Windows
	allowed["cat"] = true
	allowed["type"] = true // Windows
	allowed["head"] = true
	allowed["tail"] = true
	allowed["grep"] = true
	allowed["find"] = true
	allowed["wc"] = true
	allowed["file"] = true
	allowed["stat"] = true

	// Development tools
	allowed["git"] = true
	allowed["node"] = true
	allowed["python"] = true
	allowed["python3"] = true
	allowed["pip"] = true
	allowed["pip3"] = true
	allowed["npm"] = true
	allowed["yarn"] = true
	allowed["go"] = true
	allowed["rustc"] = true
	allowed["cargo"] = true
	allowed["java"] = true
	allowed["javac"] = true
	allowed["mvn"] = true
	allowed["gradle"] = true
	allowed["docker"] = true
	allowed["kubectl"] = true
	allowed["terraform"] = true
	allowed["make"] = true
	allowed["cmake"] = true
	allowed["gcc"] = true
	allowed["g++"] = true
	allowed["clang"] = true
	allowed["clang++"] = true

	// System information (read-only)
	allowed["uname"] = true
	allowed["df"] = true
	allowed["du"] = true
	allowed["ps"] = true
	allowed["top"] = true
	allowed["htop"] = true
	allowed["free"] = true
	allowed["vmstat"] = true
	allowed["iostat"] = true
	allowed["mpstat"] = true
	allowed["netstat"] = true
	allowed["ss"] = true
	allowed["lscpu"] = true
	allowed["lsblk"] = true
	allowed["lsusb"] = true
	allowed["lspci"] = true

	// Network tools (read-only)
	allowed["ping"] = true
	allowed["traceroute"] = true
	allowed["tracepath"] = true
	allowed["nslookup"] = true
	allowed["dig"] = true
	allowed["host"] = true
	allowed["curl"] = true
	allowed["wget"] = true
	allowed["ssh"] = true
	allowed["scp"] = true
	allowed["rsync"] = true

	// Compression and archiving
	allowed["tar"] = true
	allowed["gzip"] = true
	allowed["gunzip"] = true
	allowed["zip"] = true
	allowed["unzip"] = true

	// Text processing
	allowed["sed"] = true
	allowed["awk"] = true
	allowed["sort"] = true
	allowed["uniq"] = true
	allowed["cut"] = true
	allowed["tr"] = true
	allowed["diff"] = true
	allowed["xargs"] = true

	// Windows-specific commands
	if runtime.GOOS == "windows" {
		allowed["cmd"] = true
		allowed["powershell"] = true
		allowed["powershell.exe"] = true
		allowed["pwsh"] = true
		allowed["systeminfo"] = true
		allowed["tasklist"] = true
		allowed["taskmgr"] = true
		allowed["ipconfig"] = true
		allowed["getmac"] = true
		allowed["chdir"] = true
		allowed["cd"] = true
		allowed["cls"] = true
		allowed["copy"] = true
		allowed["xcopy"] = true
		allowed["move"] = true
		allowed["rename"] = true
		allowed["del"] = true
		allowed["erase"] = true
		allowed["mkdir"] = true
		allowed["rmdir"] = true
		allowed["tree"] = true
		allowed["where"] = true
		allowed["driverquery"] = true
	}

	// Unix/Linux-specific commands
	if runtime.GOOS != "windows" {
		allowed["man"] = true
		allowed["which"] = true
		allowed["whereis"] = true
		allowed["what"] = true
		allowed["sh"] = true
		allowed["bash"] = true
		allowed["zsh"] = true
		allowed["fish"] = true
		allowed["clear"] = true
		allowed["history"] = true
		allowed["jobs"] = true
		allowed["bg"] = true
		allowed["fg"] = true
		allowed["kill"] = true
		allowed["killall"] = true
		allowed["pkill"] = true
		allowed["pgrep"] = true
		allowed["nohup"] = true
		allowed["screen"] = true
		allowed["tmux"] = true
		allowed["watch"] = true
		allowed["time"] = true
		allowed["timeout"] = true
		allowed["cp"] = true
		allowed["mv"] = true
		allowed["rm"] = true
		allowed["mkdir"] = true
		allowed["rmdir"] = true
		allowed["chmod"] = true
		allowed["chown"] = true
		allowed["ln"] = true
		allowed["touch"] = true
		allowed["tree"] = true
		allowed["fdisk"] = true
		allowed["mount"] = true
		allowed["umount"] = true
		allowed["systemctl"] = true
		allowed["service"] = true
		allowed["journalctl"] = true
		allowed["dmesg"] = true
		allowed["sysctl"] = true
	}

	return allowed
}
