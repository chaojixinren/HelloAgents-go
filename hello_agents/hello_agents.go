// Package hello_agents keeps module boundaries aligned with HelloAgents-py.
package hello_agents

import (
	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

// Core exports.
type HelloAgentsLLM = core.HelloAgentsLLM
type Config = core.Config
type Message = core.Message
type HelloAgentsException = core.HelloAgentsException

// Agent exports.
type SimpleAgent = agents.SimpleAgent
type ReActAgent = agents.ReActAgent
type ReflectionAgent = agents.ReflectionAgent
type PlanSolveAgent = agents.PlanSolveAgent

// Tool exports.
type ToolRegistry = tools.ToolRegistry
type CalculatorTool = builtin.CalculatorTool

var GlobalRegistry = tools.GlobalRegistry

func Calculate(expression string) tools.ToolResponse {
	return builtin.Calculate(expression)
}
