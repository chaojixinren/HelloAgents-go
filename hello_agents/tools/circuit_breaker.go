package tools

import (
	"sync"
	"time"

	"helloagents-go/hello_agents/logging"
)

// CircuitBreaker prevents endless retries on repeatedly failing tools.
type CircuitBreaker struct {
	FailureThreshold int
	RecoveryTimeout  int
	Enabled          bool

	mu             sync.Mutex
	failureCounts  map[string]int
	openTimestamps map[string]time.Time
}

func NewCircuitBreaker(failureThreshold, recoveryTimeout int, enabled bool) *CircuitBreaker {
	return &CircuitBreaker{
		FailureThreshold: failureThreshold,
		RecoveryTimeout:  recoveryTimeout,
		Enabled:          enabled,
		failureCounts:    map[string]int{},
		openTimestamps:   map[string]time.Time{},
	}
}

func (c *CircuitBreaker) IsOpen(toolName string) bool {
	if !c.Enabled {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	openAt, exists := c.openTimestamps[toolName]
	if !exists {
		return false
	}
	if time.Since(openAt) > time.Duration(c.RecoveryTimeout)*time.Second {
		delete(c.openTimestamps, toolName)
		c.failureCounts[toolName] = 0
		return false
	}
	return true
}

func (c *CircuitBreaker) RecordResult(toolName string, response ToolResponse) {
	if !c.Enabled {
		return
	}
	if response.Status == ToolStatusError {
		c.onFailure(toolName)
		return
	}
	c.onSuccess(toolName)
}

func (c *CircuitBreaker) onFailure(toolName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCounts[toolName]++
	if c.failureCounts[toolName] >= c.FailureThreshold {
		c.openTimestamps[toolName] = time.Now()
		logging.Warn("Circuit Breaker: 工具已熔断", "tool", toolName, "failures", c.failureCounts[toolName])
	}
}

func (c *CircuitBreaker) onSuccess(toolName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCounts[toolName] = 0
}

func (c *CircuitBreaker) Open(toolName string) {
	if !c.Enabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.openTimestamps[toolName] = time.Now()
	logging.Warn("Circuit Breaker: 工具已手动熔断", "tool", toolName)
}

func (c *CircuitBreaker) Close(toolName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCounts[toolName] = 0
	delete(c.openTimestamps, toolName)
	logging.Info("Circuit Breaker: 工具已恢复", "tool", toolName)
}

func (c *CircuitBreaker) GetStatus(toolName string) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	if openAt, ok := c.openTimestamps[toolName]; ok {
		remaining := time.Duration(c.RecoveryTimeout)*time.Second - time.Since(openAt)
		if remaining < 0 {
			remaining = 0
		}
		return map[string]any{
			"state":              "open",
			"failure_count":      c.failureCounts[toolName],
			"open_since":         float64(openAt.UnixNano()) / 1e9,
			"recover_in_seconds": int(remaining.Seconds()),
		}
	}
	return map[string]any{
		"state":         "closed",
		"failure_count": c.failureCounts[toolName],
	}
}

func (c *CircuitBreaker) GetAllStatus() map[string]map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	all := map[string]map[string]any{}
	for name := range c.failureCounts {
		all[name] = c.getStatusLocked(name)
	}
	for name := range c.openTimestamps {
		if _, ok := all[name]; !ok {
			all[name] = c.getStatusLocked(name)
		}
	}
	return all
}

func (c *CircuitBreaker) getStatusLocked(toolName string) map[string]any {
	if openAt, ok := c.openTimestamps[toolName]; ok {
		remaining := time.Duration(c.RecoveryTimeout)*time.Second - time.Since(openAt)
		if remaining < 0 {
			remaining = 0
		}
		return map[string]any{
			"state":              "open",
			"failure_count":      c.failureCounts[toolName],
			"open_since":         float64(openAt.UnixNano()) / 1e9,
			"recover_in_seconds": int(remaining.Seconds()),
		}
	}
	return map[string]any{
		"state":         "closed",
		"failure_count": c.failureCounts[toolName],
	}
}
