package agents

import (
	"strings"

	"helloagents-go/hello_agents/core"
)

func streamLLMResponse(
	llm *core.HelloAgentsLLM,
	messages []map[string]any,
	kwargs map[string]any,
	onChunk func(string),
) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}

	chunks, errs := llm.AStreamInvoke(messages, kwargs)
	var builder strings.Builder

	for chunks != nil || errs != nil {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				chunks = nil
				continue
			}
			builder.WriteString(chunk)
			if onChunk != nil {
				onChunk(chunk)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return builder.String(), err
			}
		}
	}

	return builder.String(), nil
}
