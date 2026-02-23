package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type StreamEventType string

const (
	StreamAgentStart     StreamEventType = "agent_start"
	StreamAgentFinish    StreamEventType = "agent_finish"
	StreamStepStart      StreamEventType = "step_start"
	StreamStepFinish     StreamEventType = "step_finish"
	StreamToolCallStart  StreamEventType = "tool_call_start"
	StreamToolCallFinish StreamEventType = "tool_call_finish"
	StreamLLMChunk       StreamEventType = "llm_chunk"
	StreamThinking       StreamEventType = "thinking"
	StreamError          StreamEventType = "error"
)

type StreamEvent struct {
	Type      StreamEventType `json:"type"`
	Timestamp float64         `json:"timestamp"`
	AgentName string          `json:"agent_name"`
	Data      map[string]any  `json:"data"`
}

func NewStreamEvent(eventType StreamEventType, agentName string, data map[string]any) StreamEvent {
	if data == nil {
		data = map[string]any{}
	}
	return StreamEvent{
		Type:      eventType,
		Timestamp: float64(time.Now().UnixNano()) / 1e9,
		AgentName: agentName,
		Data:      data,
	}
}

func (e StreamEvent) ToSSE() string {
	payload, _ := json.Marshal(e.ToMap())
	lines := []string{
		fmt.Sprintf("event: %s", e.Type),
		fmt.Sprintf("data: %s", string(payload)),
		"",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (e StreamEvent) ToMap() map[string]any {
	return map[string]any{
		"type":       string(e.Type),
		"timestamp":  e.Timestamp,
		"agent_name": e.AgentName,
		"data":       e.Data,
	}
}

// ToDict keeps naming parity with Python StreamEvent.to_dict().
func (e StreamEvent) ToDict() map[string]any {
	return e.ToMap()
}

// StreamBuffer stores recent stream events with bounded size.
type StreamBuffer struct {
	MaxBufferSize int
	mu            sync.RWMutex
	events        []StreamEvent
}

func NewStreamBuffer(maxBufferSize int) *StreamBuffer {
	if maxBufferSize <= 0 {
		maxBufferSize = 100
	}
	return &StreamBuffer{
		MaxBufferSize: maxBufferSize,
		events:        make([]StreamEvent, 0, maxBufferSize),
	}
}

func (b *StreamBuffer) Add(event StreamEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = append(b.events, event)
	if len(b.events) > b.MaxBufferSize {
		b.events = b.events[1:]
	}
}

func (b *StreamBuffer) GetAll() []StreamEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]StreamEvent, len(b.events))
	copy(out, b.events)
	return out
}

func (b *StreamBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = b.events[:0]
}

func (b *StreamBuffer) FilterByType(eventType StreamEventType) []StreamEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	res := make([]StreamEvent, 0)
	for _, e := range b.events {
		if e.Type == eventType {
			res = append(res, e)
		}
	}
	return res
}

// StreamToSSE converts an event channel to SSE-formatted channel.
func StreamToSSE(eventStream <-chan StreamEvent, includeTypes map[StreamEventType]bool) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for event := range eventStream {
			if includeTypes != nil && !includeTypes[event.Type] {
				continue
			}
			out <- event.ToSSE()
		}
	}()
	return out
}

// StreamToJSON converts an event channel to jsonl strings.
func StreamToJSON(eventStream <-chan StreamEvent, includeTypes map[StreamEventType]bool) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for event := range eventStream {
			if includeTypes != nil && !includeTypes[event.Type] {
				continue
			}
			payload, _ := json.Marshal(event.ToMap())
			out <- string(payload) + "\n"
		}
	}()
	return out
}
