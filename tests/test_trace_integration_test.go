package tests_test

import (
	"testing"

	"helloagents-go/hello_agents/core"
)

// ---------------------------------------------------------------------------
// StreamBuffer / StreamEvent tests (from core/streaming_test.go)
// ---------------------------------------------------------------------------

func TestStreamBufferKeepsExplicitZeroSizeLikePython(t *testing.T) {
	buffer := core.NewStreamBuffer(0)
	if buffer.MaxBufferSize != 0 {
		t.Fatalf("MaxBufferSize = %d, want 0", buffer.MaxBufferSize)
	}
	buffer.Add(core.NewStreamEvent(core.StreamLLMChunk, "agent", map[string]any{"chunk": "x"}))
	if len(buffer.GetAll()) != 0 {
		t.Fatalf("buffer with max size 0 should keep 0 events")
	}
}

func TestStreamBufferKeepsExplicitNegativeSizeLikePythonBehavior(t *testing.T) {
	buffer := core.NewStreamBuffer(-1)
	buffer.Add(core.NewStreamEvent(core.StreamLLMChunk, "agent", map[string]any{"chunk": "x"}))
	if len(buffer.GetAll()) != 0 {
		t.Fatalf("buffer with negative max size should keep 0 events due immediate drop")
	}
}
