package tools

import "testing"

func TestBaseToolToMapMatchesPythonShape(t *testing.T) {
	base := NewBaseTool("x", "desc", true)
	m := base.ToMap()
	if _, ok := m["expandable"]; ok {
		t.Fatalf("ToMap() should not include 'expandable' key")
	}
	if m["name"] != "x" {
		t.Fatalf("name = %v, want x", m["name"])
	}
	if m["description"] != "desc" {
		t.Fatalf("description = %v, want desc", m["description"])
	}
}
