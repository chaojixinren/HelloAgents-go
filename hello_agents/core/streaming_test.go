package core

import "testing"

func TestStreamBufferKeepsExplicitZeroSizeLikePython(t *testing.T) {
	buffer := NewStreamBuffer(0)
	if buffer.MaxBufferSize != 0 {
		t.Fatalf("MaxBufferSize = %d, want 0", buffer.MaxBufferSize)
	}
	buffer.Add(NewStreamEvent(StreamLLMChunk, "agent", map[string]any{"chunk": "x"}))
	if len(buffer.GetAll()) != 0 {
		t.Fatalf("buffer with max size 0 should keep 0 events")
	}
}

func TestStreamBufferKeepsExplicitNegativeSizeLikePythonBehavior(t *testing.T) {
	buffer := NewStreamBuffer(-1)
	buffer.Add(NewStreamEvent(StreamLLMChunk, "agent", map[string]any{"chunk": "x"}))
	if len(buffer.GetAll()) != 0 {
		t.Fatalf("buffer with negative max size should keep 0 events due immediate drop")
	}
}
