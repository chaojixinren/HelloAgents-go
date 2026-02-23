package core

import "fmt"

// HelloAgentsException hierarchy mirrored with typed errors.
type HelloAgentsException struct {
	Message string
}

func (e HelloAgentsException) Error() string { return e.Message }

type LLMException struct{ HelloAgentsException }
type AgentException struct{ HelloAgentsException }
type ConfigException struct{ HelloAgentsException }
type ToolException struct{ HelloAgentsException }

func NewHelloAgentsException(msg string) error { return HelloAgentsException{Message: msg} }
func NewLLMException(msg string) error         { return LLMException{HelloAgentsException{Message: msg}} }
func NewAgentException(msg string) error       { return AgentException{HelloAgentsException{Message: msg}} }
func NewConfigException(msg string) error      { return ConfigException{HelloAgentsException{Message: msg}} }
func NewToolException(msg string) error        { return ToolException{HelloAgentsException{Message: msg}} }

func WrapError(prefix string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
