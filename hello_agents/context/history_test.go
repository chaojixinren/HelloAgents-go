package context

import (
	"testing"
)

func TestHistoryManagerAppendAndGet(t *testing.T) {
	hm := NewHistoryManager[string](
		2,
		0.8,
		func(summary string) string { return "[summary] " + summary },
		func(msg string) string { return msg },
	)

	hm.Append("msg1")
	hm.Append("msg2")
	hm.Append("msg3")

	history := hm.GetHistory()
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}
}

func TestHistoryManagerClear(t *testing.T) {
	hm := NewHistoryManager[string](
		2, 0.8,
		func(summary string) string { return summary },
		func(msg string) string { return msg },
	)

	hm.Append("msg1")
	hm.Clear()

	if len(hm.GetHistory()) != 0 {
		t.Fatal("history should be empty after Clear()")
	}
}

func TestHistoryManagerEstimateRounds(t *testing.T) {
	hm := NewHistoryManager[string](
		2, 0.8,
		func(summary string) string { return summary },
		func(msg string) string {
			if msg == "user-msg" {
				return "user"
			}
			return "assistant"
		},
	)

	hm.Append("user-msg")
	hm.Append("assistant-msg")
	hm.Append("user-msg")
	hm.Append("assistant-msg")

	rounds := hm.EstimateRounds()
	if rounds != 2 {
		t.Fatalf("EstimateRounds() = %d, want 2", rounds)
	}
}

func TestHistoryManagerCompress(t *testing.T) {
	hm := NewHistoryManager[string](
		1, 0.8,
		func(summary string) string { return "[compressed] " + summary },
		func(msg string) string {
			if msg == "user" {
				return "user"
			}
			return "assistant"
		},
	)

	hm.Append("user")
	hm.Append("assistant")
	hm.Append("user")
	hm.Append("assistant")

	hm.Compress("summary of old messages")

	history := hm.GetHistory()
	if len(history) == 0 {
		t.Fatal("history should not be empty after compress")
	}
}

func TestTokenCounterCountMessage(t *testing.T) {
	tc := NewTokenCounter[string](
		"gpt-3.5-turbo",
		func(msg string) string { return msg },
		func(msg string) string { return msg },
	)

	count := tc.CountMessage("hello world")
	if count <= 0 {
		t.Fatalf("CountMessage returned %d, want > 0", count)
	}
}

func TestTokenCounterCountMessages(t *testing.T) {
	tc := NewTokenCounter[string](
		"gpt-3.5-turbo",
		func(msg string) string { return msg },
		func(msg string) string { return msg },
	)

	messages := []string{"hello", "world", "test"}
	count := tc.CountMessages(messages)
	if count <= 0 {
		t.Fatalf("CountMessages returned %d, want > 0", count)
	}
}
