package core

import (
	"os"
	"strconv"
	"strings"
)

// Config 是 HelloAgents 配置，与 Python 的 Config 对应。
type Config struct {
	DefaultModel       string
	DefaultProvider    string
	Temperature        float64
	MaxTokens          *int
	Debug              bool
	LogLevel           string
	MaxHistoryLength   int
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		DefaultModel:     "gpt-3.5-turbo",
		DefaultProvider:  "openai",
		Temperature:      0.7,
		MaxTokens:        nil,
		Debug:            false,
		LogLevel:         "INFO",
		MaxHistoryLength:  100,
	}
}

// FromEnv 从环境变量创建配置，与 Python 的 Config.from_env 对应。
func FromEnv() *Config {
	c := DefaultConfig()
	if v := os.Getenv("DEBUG"); strings.ToLower(v) == "true" {
		c.Debug = true
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Temperature = f
		}
	}
	if v := os.Getenv("MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxTokens = &n
		}
	}
	return c
}
