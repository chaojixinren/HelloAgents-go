package tests_test

import (
	"testing"

	"helloagents-go/hello_agents/tools"
)

// ---------------------------------------------------------------------------
// ToolFilter tests (from tools/tool_filter_test.go)
// ---------------------------------------------------------------------------

func TestReadOnlyFilter(t *testing.T) {
	filter := tools.NewReadOnlyFilter(nil)
	all := []string{"Read", "Write", "Edit", "LS", "Grep", "Bash", "Skill", "SkillTool"}
	filtered := filter.Filter(all)

	expected := map[string]bool{
		"Read":      true,
		"LS":        true,
		"Grep":      true,
		"Skill":     true,
		"SkillTool": true,
	}
	for _, tool := range filtered {
		if !expected[tool] {
			t.Fatalf("unexpected tool in readonly result: %q", tool)
		}
	}
	if filter.IsAllowed("Write") {
		t.Fatalf("Write should not be allowed by ReadOnlyFilter")
	}
}

func TestReadOnlyFilterAdditionalAllowed(t *testing.T) {
	filter := tools.NewReadOnlyFilter([]string{"CustomTool"})
	if !filter.IsAllowed("CustomTool") {
		t.Fatalf("CustomTool should be allowed when passed in additional_allowed")
	}
}

func TestFullAccessFilter(t *testing.T) {
	filter := tools.NewFullAccessFilter(nil)
	all := []string{"Read", "Write", "Bash", "Terminal", "LS"}
	filtered := filter.Filter(all)
	for _, tool := range filtered {
		if tool == "Bash" || tool == "Terminal" {
			t.Fatalf("dangerous tool should be denied: %q", tool)
		}
	}
	if filter.IsAllowed("Bash") {
		t.Fatalf("Bash should be denied")
	}
}

func TestCustomFilterModes(t *testing.T) {
	whitelist, err := tools.NewCustomFilterWithMode([]string{"Read", "LS"}, nil, "whitelist")
	if err != nil {
		t.Fatalf("NewCustomFilterWithMode() error = %v", err)
	}
	filtered := whitelist.Filter([]string{"Read", "Write", "LS"})
	if len(filtered) != 2 || filtered[0] != "Read" || filtered[1] != "LS" {
		t.Fatalf("whitelist filter result = %#v, want [Read LS]", filtered)
	}

	blacklist, err := tools.NewCustomFilterWithMode(nil, []string{"Bash"}, "blacklist")
	if err != nil {
		t.Fatalf("NewCustomFilterWithMode() error = %v", err)
	}
	if blacklist.IsAllowed("Bash") {
		t.Fatalf("Bash should be denied in blacklist mode")
	}
	if !blacklist.IsAllowed("Read") {
		t.Fatalf("Read should be allowed in blacklist mode")
	}
}

func TestCustomFilterInvalidMode(t *testing.T) {
	_, err := tools.NewCustomFilterWithMode(nil, nil, "invalid")
	if err == nil {
		t.Fatalf("expected invalid mode error")
	}
	if err.Error() != "Invalid mode: invalid. Must be 'whitelist' or 'blacklist'" {
		t.Fatalf("error message = %q", err.Error())
	}
}
