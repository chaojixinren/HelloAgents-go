package observability

import "time"

func (l *TraceLogger) ExportComputeStats() map[string]any {
	return l.computeStats()
}

func ExportParseTraceTimestamp(value string) (time.Time, error) {
	return parseTraceTimestamp(value)
}
