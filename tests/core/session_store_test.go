package core_test

import (
	"path/filepath"
	"strings"
	"testing"

	"helloagents-go/hello_agents/core"
)

func TestPythonValueEqualMatchesPythonLikeSemantics(t *testing.T) {
	cases := []struct {
		a, b any
		want bool
	}{
		{1, 1.0, true},
		{true, 1, true},
		{false, 0, true},
		{"1", 1, false},
		{"abc", "abc", true},
		{"abc", "xyz", false},
	}

	for _, tc := range cases {
		if got := core.ExportPythonValueEqual(tc.a, tc.b); got != tc.want {
			t.Fatalf("pythonValueEqual(%#v, %#v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckConfigConsistencyUsesPythonLikeComparison(t *testing.T) {
	store := &core.SessionStore{}

	check := store.CheckConfigConsistency(
		map[string]any{"max_steps": 1},
		map[string]any{"max_steps": 1.0},
	)

	if consistent, _ := check["consistent"].(bool); !consistent {
		t.Fatalf("max_steps int/float should be considered equal")
	}

	check = store.CheckConfigConsistency(
		map[string]any{"max_steps": "1"},
		map[string]any{"max_steps": 1},
	)
	if consistent, _ := check["consistent"].(bool); consistent {
		t.Fatalf("max_steps string/int should be considered different")
	}
}

func TestSessionStoreSaveKeepsWhitespaceSessionNameLikePythonTruthy(t *testing.T) {
	tmp := t.TempDir()
	store, err := core.NewSessionStore(tmp)
	if err != nil {
		t.Fatalf("NewSessionStore() error = %v", err)
	}

	path, err := store.Save(
		map[string]any{"name": "a"},
		nil,
		"hash",
		map[string]map[string]any{},
		map[string]any{},
		"   ",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	want := filepath.Join(tmp, "   .json")
	if path != want {
		t.Fatalf("Save() path = %q, want %q", path, want)
	}
}

func TestNewSessionStoreEmptyDirUsesCurrentDirectoryLikePathlib(t *testing.T) {
	store, err := core.NewSessionStore("")
	if err != nil {
		t.Fatalf("NewSessionStore(\"\") error = %v", err)
	}
	if store.SessionDir != "." {
		t.Fatalf("SessionDir = %q, want %q", store.SessionDir, ".")
	}
}

func TestSessionStoreSavePreservesExplicitEmptyCreatedAt(t *testing.T) {
	tmp := t.TempDir()
	store, err := core.NewSessionStore(tmp)
	if err != nil {
		t.Fatalf("NewSessionStore() error = %v", err)
	}

	path, err := store.Save(
		map[string]any{"name": "a"},
		nil,
		"hash",
		map[string]map[string]any{},
		map[string]any{"created_at": ""},
		"created-at-empty",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	createdAt, _ := data.CreatedAt.(string)
	if createdAt != "" {
		t.Fatalf("CreatedAt = %q, want explicit empty string", createdAt)
	}
}

func TestSessionStoreSaveUsesPythonISOStyleTimestamps(t *testing.T) {
	tmp := t.TempDir()
	store, err := core.NewSessionStore(tmp)
	if err != nil {
		t.Fatalf("NewSessionStore() error = %v", err)
	}

	path, err := store.Save(
		map[string]any{"name": "a"},
		nil,
		"hash",
		map[string]map[string]any{},
		map[string]any{},
		"iso-ts",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if strings.Contains(data.SavedAt, "Z") {
		t.Fatalf("SavedAt should be python-style naive timestamp, got %q", data.SavedAt)
	}
	if _, err := core.ExportParsePythonISOTime(data.SavedAt); err != nil {
		t.Fatalf("SavedAt parse error = %v, value=%q", err, data.SavedAt)
	}
	createdAt, ok := data.CreatedAt.(string)
	if !ok {
		t.Fatalf("CreatedAt type = %T, want string for default-created value", data.CreatedAt)
	}
	if _, err := core.ExportParsePythonISOTime(createdAt); err != nil {
		t.Fatalf("CreatedAt parse error = %v, value=%v", err, data.CreatedAt)
	}
}

func TestSessionStoreSaveKeepsCreatedAtRawTypeLikePython(t *testing.T) {
	tmp := t.TempDir()
	store, err := core.NewSessionStore(tmp)
	if err != nil {
		t.Fatalf("NewSessionStore() error = %v", err)
	}

	path, err := store.Save(
		map[string]any{"name": "a"},
		nil,
		"hash",
		map[string]map[string]any{},
		map[string]any{"created_at": 123},
		"created-at-raw",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.CreatedAt != float64(123) {
		t.Fatalf("CreatedAt = %#v, want raw numeric value 123", data.CreatedAt)
	}
}
