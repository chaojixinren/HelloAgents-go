package core

import "fmt"

// HelloAgentsError 是 HelloAgents 基础错误类型，对应 Python 的 HelloAgentsException。
type HelloAgentsError struct {
	Msg string
}

func (e *HelloAgentsError) Error() string {
	return e.Msg
}

// LLMError 表示 LLM 调用或配置相关错误，对应 Python 的 LLMException。
type LLMError struct {
	Msg string
	Err error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

// AgentError 表示 Agent 相关错误，对应 Python 的 AgentException。
type AgentError struct {
	Msg string
	Err error
}

func (e *AgentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

// ConfigError 表示配置相关错误，对应 Python 的 ConfigException。
type ConfigError struct {
	Msg string
	Err error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ToolError 表示工具相关错误，对应 Python 的 ToolException。
type ToolError struct {
	Msg string
	Err error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *ToolError) Unwrap() error {
	return e.Err
}
