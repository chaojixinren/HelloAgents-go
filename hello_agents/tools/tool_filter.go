package tools

import "maps"

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
		"Grep":      {},
		"LS":        {},
		"Glob":      {},
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
	base := map[string]struct{}{}
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
}

func NewCustomFilter(allowed []string, denied []string) *CustomFilter {
	allowSet := map[string]struct{}{}
	for _, t := range allowed {
		allowSet[t] = struct{}{}
	}
	denySet := map[string]struct{}{}
	for _, t := range denied {
		denySet[t] = struct{}{}
	}
	return &CustomFilter{allowed: allowSet, denied: denySet}
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
	if len(f.allowed) > 0 {
		_, ok := f.allowed[toolName]
		if !ok {
			return false
		}
	}
	_, denied := f.denied[toolName]
	return !denied
}

func cloneSet(in map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	maps.Copy(out, in)
	return out
}
