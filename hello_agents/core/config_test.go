package core

import "testing"

func TestFromEnvDebugMatchesPythonTrueOnlyRule(t *testing.T) {
	t.Setenv("DEBUG", "1")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "")

	cfg := FromEnv()
	if cfg.Debug {
		t.Fatalf("FromEnv().Debug = true, want false when DEBUG=1")
	}
}

func TestFromEnvFallsBackOnInvalidTemperature(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "bad-number")
	t.Setenv("MAX_TOKENS", "")

	cfg := FromEnv()
	if cfg.Temperature != 0.7 {
		t.Fatalf("FromEnv().Temperature = %v, want 0.7 fallback on invalid input", cfg.Temperature)
	}
}

func TestFromEnvIgnoresInvalidMaxTokens(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "not-int")

	cfg := FromEnv()
	if cfg.MaxTokens != nil {
		t.Fatalf("FromEnv().MaxTokens = %v, want nil on invalid input", *cfg.MaxTokens)
	}
}

func TestFromEnvKeepsExplicitEmptyLogLevelLikeOsGetenv(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "")

	cfg := FromEnv()
	if cfg.LogLevel != "" {
		t.Fatalf("FromEnv().LogLevel = %q, want explicit empty string", cfg.LogLevel)
	}
}

func TestFromEnvFallsBackWhenTemperatureIsExplicitEmpty(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "")
	t.Setenv("MAX_TOKENS", "")

	cfg := FromEnv()
	if cfg.Temperature != 0.7 {
		t.Fatalf("FromEnv().Temperature = %v, want 0.7 fallback on empty string", cfg.Temperature)
	}
}
