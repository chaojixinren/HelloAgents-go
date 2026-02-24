package context

import "testing"

func TestNewHistoryManagerKeepsExplicitValues(t *testing.T) {
	h := NewHistoryManager[int](0, 0, nil, nil)
	if h.MinRetainRounds != 0 {
		t.Fatalf("MinRetainRounds = %d, want 0", h.MinRetainRounds)
	}
	if h.CompressionThreshold != 0 {
		t.Fatalf("CompressionThreshold = %v, want 0", h.CompressionThreshold)
	}
}

func TestNewTokenCounterKeepsEmptyModel(t *testing.T) {
	c := NewTokenCounter[string]("", func(s string) string { return s }, func(s string) string { return s })
	if c.Model != "" {
		t.Fatalf("Model = %q, want empty string", c.Model)
	}
}

func TestNewContextBuilderDoesNotNormalizeProvidedConfig(t *testing.T) {
	cfg := ContextConfig{
		MaxTokens:         0,
		ReserveRatio:      2,
		MinRelevance:      -1,
		MMRLambda:         2,
		EnableCompression: true,
	}
	builder := NewContextBuilder(cfg)
	if builder.Config.MaxTokens != 0 {
		t.Fatalf("MaxTokens = %d, want 0", builder.Config.MaxTokens)
	}
	if builder.Config.ReserveRatio != 2 {
		t.Fatalf("ReserveRatio = %v, want 2", builder.Config.ReserveRatio)
	}
	if builder.Config.MinRelevance != -1 {
		t.Fatalf("MinRelevance = %v, want -1", builder.Config.MinRelevance)
	}
	if builder.Config.MMRLambda != 2 {
		t.Fatalf("MMRLambda = %v, want 2", builder.Config.MMRLambda)
	}
}
