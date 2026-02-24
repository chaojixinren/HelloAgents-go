package core

import "time"

const pythonISOLayout = "2006-01-02T15:04:05.999999"

// formatPythonISOTime mirrors Python datetime.now().isoformat() default style.
func formatPythonISOTime(ts time.Time) string {
	return ts.Format(pythonISOLayout)
}

func nowPythonISOTime() string {
	return formatPythonISOTime(time.Now())
}

func parsePythonISOTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		pythonISOLayout,
		"2006-01-02T15:04:05",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
