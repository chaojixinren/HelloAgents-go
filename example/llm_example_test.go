package example

import (
	"errors"
	"testing"

	"helloagents-go/core"
)

func TestNewHelloAgentsLLM_Validation(t *testing.T) {
	// provider=custom 且未提供 API Key/BaseURL 时应返回错误
	_, err := core.NewHelloAgentsLLM("model", "", "", "custom", 0.7, nil, nil)
	if err == nil {
		t.Fatal("expected error when api key and base url are empty with provider custom")
	}
	var he *core.HelloAgentsError
	if !errors.As(err, &he) {
		t.Log("expected HelloAgentsError or wrapped:", err)
	}
}
