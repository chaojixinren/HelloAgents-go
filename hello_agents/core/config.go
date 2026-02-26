package core

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config mirrors hello_agents.core.config.Config in Python.
type Config struct {
	// LLM
	DefaultModel    string  `json:"default_model"`
	DefaultProvider string  `json:"default_provider"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       *int    `json:"max_tokens"`

	// System
	Debug    bool   `json:"debug"`
	LogLevel string `json:"log_level"`

	// History
	MaxHistoryLength int `json:"max_history_length"`

	// Context engineering
	ContextWindow          int     `json:"context_window"`
	CompressionThreshold   float64 `json:"compression_threshold"`
	MinRetainRounds        int     `json:"min_retain_rounds"`
	EnableSmartCompression bool    `json:"enable_smart_compression"`

	// Smart summary
	SummaryLLMProvider string  `json:"summary_llm_provider"`
	SummaryLLMModel    string  `json:"summary_llm_model"`
	SummaryMaxTokens   int     `json:"summary_max_tokens"`
	SummaryTemperature float64 `json:"summary_temperature"`

	// Tool output truncation
	ToolOutputMaxLines          int    `json:"tool_output_max_lines"`
	ToolOutputMaxBytes          int    `json:"tool_output_max_bytes"`
	ToolOutputDir               string `json:"tool_output_dir"`
	ToolOutputTruncateDirection string `json:"tool_output_truncate_direction"`

	// Observability
	TraceEnabled                bool   `json:"trace_enabled"`
	TraceDir                    string `json:"trace_dir"`
	TraceSanitize               bool   `json:"trace_sanitize"`
	TraceHTMLIncludeRawResponse bool   `json:"trace_html_include_raw_response"`

	// Skills
	SkillsEnabled      bool   `json:"skills_enabled"`
	SkillsDir          string `json:"skills_dir"`
	SkillsAutoRegister bool   `json:"skills_auto_register"`

	// Circuit breaker
	CircuitEnabled          bool `json:"circuit_enabled"`
	CircuitFailureThreshold int  `json:"circuit_failure_threshold"`
	CircuitRecoveryTimeout  int  `json:"circuit_recovery_timeout"`

	// Session persistence
	SessionEnabled   bool   `json:"session_enabled"`
	SessionDir       string `json:"session_dir"`
	AutoSaveEnabled  bool   `json:"auto_save_enabled"`
	AutoSaveInterval int    `json:"auto_save_interval"`

	// Subagent
	SubagentEnabled          bool   `json:"subagent_enabled"`
	SubagentMaxSteps         int    `json:"subagent_max_steps"`
	SubagentUseLightLLM      bool   `json:"subagent_use_light_llm"`
	SubagentLightLLMProvider string `json:"subagent_light_llm_provider"`
	SubagentLightLLMModel    string `json:"subagent_light_llm_model"`

	// TodoWrite
	TodoWriteEnabled        bool   `json:"todowrite_enabled"`
	TodoWritePersistenceDir string `json:"todowrite_persistence_dir"`

	// DevLog
	DevLogEnabled        bool   `json:"devlog_enabled"`
	DevLogPersistenceDir string `json:"devlog_persistence_dir"`

	// Async lifecycle
	AsyncEnabled       bool    `json:"async_enabled"`
	MaxConcurrentTools int     `json:"max_concurrent_tools"`
	HookTimeoutSeconds float64 `json:"hook_timeout_seconds"`
	LLMAsyncTimeout    int     `json:"llm_async_timeout"`
	ToolAsyncTimeout   int     `json:"tool_async_timeout"`

	// Streaming
	StreamEnabled          bool `json:"stream_enabled"`
	StreamBufferSize       int  `json:"stream_buffer_size"`
	StreamIncludeThinking  bool `json:"stream_include_thinking"`
	StreamIncludeToolCalls bool `json:"stream_include_tool_calls"`
}

var dotenvOnce sync.Once

func ensureDotEnvLoaded() {
	dotenvOnce.Do(func() {
		_ = LoadDotEnv(".env")
	})
}

func DefaultConfig() Config {
	return Config{
		DefaultModel:    "gpt-3.5-turbo",
		DefaultProvider: "openai",
		Temperature:     0.7,

		Debug:    false,
		LogLevel: "INFO",

		MaxHistoryLength: 100,

		ContextWindow:          128000,
		CompressionThreshold:   0.8,
		MinRetainRounds:        10,
		EnableSmartCompression: false,

		SummaryLLMProvider: "deepseek",
		SummaryLLMModel:    "deepseek-chat",
		SummaryMaxTokens:   800,
		SummaryTemperature: 0.3,

		ToolOutputMaxLines:          2000,
		ToolOutputMaxBytes:          51200,
		ToolOutputDir:               "tool-output",
		ToolOutputTruncateDirection: "head",

		TraceEnabled:                true,
		TraceDir:                    "memory/traces",
		TraceSanitize:               true,
		TraceHTMLIncludeRawResponse: false,

		SkillsEnabled:      true,
		SkillsDir:          "skills",
		SkillsAutoRegister: true,

		CircuitEnabled:          true,
		CircuitFailureThreshold: 3,
		CircuitRecoveryTimeout:  300,

		SessionEnabled:   true,
		SessionDir:       "memory/sessions",
		AutoSaveEnabled:  false,
		AutoSaveInterval: 10,

		SubagentEnabled:          true,
		SubagentMaxSteps:         15,
		SubagentUseLightLLM:      false,
		SubagentLightLLMProvider: "deepseek",
		SubagentLightLLMModel:    "deepseek-chat",

		TodoWriteEnabled:        true,
		TodoWritePersistenceDir: "memory/todos",

		DevLogEnabled:        true,
		DevLogPersistenceDir: "memory/devlogs",

		AsyncEnabled:       true,
		MaxConcurrentTools: 3,
		HookTimeoutSeconds: 5.0,
		LLMAsyncTimeout:    120,
		ToolAsyncTimeout:   30,

		StreamEnabled:          true,
		StreamBufferSize:       100,
		StreamIncludeThinking:  true,
		StreamIncludeToolCalls: true,
	}
}

// FromEnv creates config from environment variables while preserving Python defaults.
func FromEnv() Config {
	ensureDotEnvLoaded()

	cfg := DefaultConfig()

	// Python parity:
	// debug=os.getenv("DEBUG", "false").lower() == "true"
	cfg.Debug = strings.ToLower(os.Getenv("DEBUG")) == "true"
	logLevelRaw, hasLogLevel := os.LookupEnv("LOG_LEVEL")
	if hasLogLevel {
		cfg.LogLevel = logLevelRaw
	} else {
		cfg.LogLevel = "INFO"
	}

	temperatureRaw, hasTemperature := os.LookupEnv("TEMPERATURE")
	if !hasTemperature {
		temperatureRaw = "0.7"
	}
	temperatureValue, err := strconv.ParseFloat(temperatureRaw, 64)
	if err != nil {
		temperatureValue = 0.7
	}
	cfg.Temperature = temperatureValue

	maxTokensRaw, hasMaxTokens := os.LookupEnv("MAX_TOKENS")
	if hasMaxTokens && maxTokensRaw != "" {
		maxTokensValue, err := strconv.Atoi(maxTokensRaw)
		if err == nil {
			cfg.MaxTokens = &maxTokensValue
		}
	} else {
		cfg.MaxTokens = nil
	}

	return cfg
}

func (c Config) ToMap() map[string]any {
	return map[string]any{
		"default_model":                   c.DefaultModel,
		"default_provider":                c.DefaultProvider,
		"temperature":                     c.Temperature,
		"max_tokens":                      c.MaxTokens,
		"debug":                           c.Debug,
		"log_level":                       c.LogLevel,
		"max_history_length":              c.MaxHistoryLength,
		"context_window":                  c.ContextWindow,
		"compression_threshold":           c.CompressionThreshold,
		"min_retain_rounds":               c.MinRetainRounds,
		"enable_smart_compression":        c.EnableSmartCompression,
		"summary_llm_provider":            c.SummaryLLMProvider,
		"summary_llm_model":               c.SummaryLLMModel,
		"summary_max_tokens":              c.SummaryMaxTokens,
		"summary_temperature":             c.SummaryTemperature,
		"tool_output_max_lines":           c.ToolOutputMaxLines,
		"tool_output_max_bytes":           c.ToolOutputMaxBytes,
		"tool_output_dir":                 c.ToolOutputDir,
		"tool_output_truncate_direction":  c.ToolOutputTruncateDirection,
		"trace_enabled":                   c.TraceEnabled,
		"trace_dir":                       c.TraceDir,
		"trace_sanitize":                  c.TraceSanitize,
		"trace_html_include_raw_response": c.TraceHTMLIncludeRawResponse,
		"skills_enabled":                  c.SkillsEnabled,
		"skills_dir":                      c.SkillsDir,
		"skills_auto_register":            c.SkillsAutoRegister,
		"circuit_enabled":                 c.CircuitEnabled,
		"circuit_failure_threshold":       c.CircuitFailureThreshold,
		"circuit_recovery_timeout":        c.CircuitRecoveryTimeout,
		"session_enabled":                 c.SessionEnabled,
		"session_dir":                     c.SessionDir,
		"auto_save_enabled":               c.AutoSaveEnabled,
		"auto_save_interval":              c.AutoSaveInterval,
		"subagent_enabled":                c.SubagentEnabled,
		"subagent_max_steps":              c.SubagentMaxSteps,
		"subagent_use_light_llm":          c.SubagentUseLightLLM,
		"subagent_light_llm_provider":     c.SubagentLightLLMProvider,
		"subagent_light_llm_model":        c.SubagentLightLLMModel,
		"todowrite_enabled":               c.TodoWriteEnabled,
		"todowrite_persistence_dir":       c.TodoWritePersistenceDir,
		"devlog_enabled":                  c.DevLogEnabled,
		"devlog_persistence_dir":          c.DevLogPersistenceDir,
		"async_enabled":                   c.AsyncEnabled,
		"max_concurrent_tools":            c.MaxConcurrentTools,
		"hook_timeout_seconds":            c.HookTimeoutSeconds,
		"llm_async_timeout":               c.LLMAsyncTimeout,
		"tool_async_timeout":              c.ToolAsyncTimeout,
		"stream_enabled":                  c.StreamEnabled,
		"stream_buffer_size":              c.StreamBufferSize,
		"stream_include_thinking":         c.StreamIncludeThinking,
		"stream_include_tool_calls":       c.StreamIncludeToolCalls,
	}
}

// ToDict keeps naming parity with Python Config.to_dict().
func (c Config) ToDict() map[string]any {
	return c.ToMap()
}

// LoadDotEnv loads KEY=VALUE pairs from .env file to process env if key does not exist.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexRune(line, '=')
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"")
		val = strings.Trim(val, "'")

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}

	return scanner.Err()
}
