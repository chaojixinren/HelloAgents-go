package builtin

import "time"

const pythonISOLayout = "2006-01-02T15:04:05.999999"

func nowPythonISOTime() string {
	return time.Now().Format(pythonISOLayout)
}
