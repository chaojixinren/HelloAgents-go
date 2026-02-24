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

func TestFromEnvPanicsOnInvalidTemperature(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "bad-number")
	t.Setenv("MAX_TOKENS", "")

	defer func() {
		if recover() == nil {
			t.Fatalf("FromEnv() should panic when TEMPERATURE is invalid")
		}
	}()
	_ = FromEnv()
}

func TestFromEnvPanicsOnInvalidMaxTokens(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "0.7")
	t.Setenv("MAX_TOKENS", "not-int")

	defer func() {
		if recover() == nil {
			t.Fatalf("FromEnv() should panic when MAX_TOKENS is invalid")
		}
	}()
	_ = FromEnv()
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

func TestFromEnvPanicsWhenTemperatureIsExplicitEmpty(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("TEMPERATURE", "")
	t.Setenv("MAX_TOKENS", "")

	defer func() {
		if recover() == nil {
			t.Fatalf("FromEnv() should panic when TEMPERATURE is explicitly empty")
		}
	}()
	_ = FromEnv()
}
