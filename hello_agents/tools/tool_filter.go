package tools

import (
	"fmt"
)

// ToolFilter controls which tools are exposed to subagents.
type ToolFilter interface {
	Filter(allTools []string) []string
	IsAllowed(toolName string) bool
}

type ReadOnlyFilter struct {
	allowed map[string]struct{}
}

func NewReadOnlyFilter(additionalAllowed []string) *ReadOnlyFilter {
	base := map[string]struct{}{
		"Read":      {},
		"ReadTool":  {},
		"LS":        {},
		"LSTool":    {},
		"Glob":      {},
		"GlobTool":  {},
		"Grep":      {},
		"GrepTool":  {},
		"Skill":     {},
		"SkillTool": {},
	}
	for _, t := range additionalAllowed {
		base[t] = struct{}{}
	}
	return &ReadOnlyFilter{allowed: base}
}

func (f *ReadOnlyFilter) Filter(allTools []string) []string {
	out := make([]string, 0)
	for _, name := range allTools {
		if f.IsAllowed(name) {
			out = append(out, name)
		}
	}
	return out
}

func (f *ReadOnlyFilter) IsAllowed(toolName string) bool {
	_, ok := f.allowed[toolName]
	return ok
}

type FullAccessFilter struct {
	denied map[string]struct{}
}

func NewFullAccessFilter(additionalDenied []string) *FullAccessFilter {
	base := map[string]struct{}{
		"Bash":         {},
		"BashTool":     {},
		"Terminal":     {},
		"TerminalTool": {},
		"Execute":      {},
		"ExecuteTool":  {},
	}
	for _, t := range additionalDenied {
		base[t] = struct{}{}
	}
	return &FullAccessFilter{denied: base}
}

func (f *FullAccessFilter) Filter(allTools []string) []string {
	out := make([]string, 0, len(allTools))
	for _, name := range allTools {
		if f.IsAllowed(name) {
			out = append(out, name)
		}
	}
	return out
}

func (f *FullAccessFilter) IsAllowed(toolName string) bool {
	_, denied := f.denied[toolName]
	return !denied
}

type CustomFilter struct {
	allowed map[string]struct{}
	denied  map[string]struct{}
	mode    string
}

func NewCustomFilter(allowed []string, denied []string) *CustomFilter {
	filter, err := NewCustomFilterWithMode(allowed, denied, "whitelist")
	if err != nil {
		return &CustomFilter{
			allowed: map[string]struct{}{},
			denied:  map[string]struct{}{},
			mode:    "whitelist",
		}
	}
	return filter
}

func NewCustomFilterWithMode(allowed []string, denied []string, mode string) (*CustomFilter, error) {
	if mode != "whitelist" && mode != "blacklist" {
		return nil, fmt.Errorf("Invalid mode: %s. Must be 'whitelist' or 'blacklist'", mode)
	}

	allowSet := map[string]struct{}{}
	for _, t := range allowed {
		allowSet[t] = struct{}{}
	}
	denySet := map[string]struct{}{}
	for _, t := range denied {
		denySet[t] = struct{}{}
	}
	return &CustomFilter{
		allowed: allowSet,
		denied:  denySet,
		mode:    mode,
	}, nil
}

func (f *CustomFilter) Filter(allTools []string) []string {
	out := make([]string, 0)
	for _, name := range allTools {
		if f.IsAllowed(name) {
			out = append(out, name)
		}
	}
	return out
}

func (f *CustomFilter) IsAllowed(toolName string) bool {
	if f.mode == "blacklist" {
		_, denied := f.denied[toolName]
		return !denied
	}

	// Default whitelist behavior (parity with Python).
	if len(f.allowed) == 0 {
		return false
	}
	if len(f.allowed) > 0 {
		_, ok := f.allowed[toolName]
		if !ok {
			return false
		}
	}
	return true
}
