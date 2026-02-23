package tools

var (
	ToolErrorCodeNotFound         = "NOT_FOUND"
	ToolErrorCodeAccessDenied     = "ACCESS_DENIED"
	ToolErrorCodePermissionDenied = "PERMISSION_DENIED"
	ToolErrorCodeIsDirectory      = "IS_DIRECTORY"
	ToolErrorCodeBinaryFile       = "BINARY_FILE"

	ToolErrorCodeInvalidParam  = "INVALID_PARAM"
	ToolErrorCodeInvalidFormat = "INVALID_FORMAT"

	ToolErrorCodeExecutionError = "EXECUTION_ERROR"
	ToolErrorCodeTimeout        = "TIMEOUT"
	ToolErrorCodeInternalError  = "INTERNAL_ERROR"

	ToolErrorCodeConflict    = "CONFLICT"
	ToolErrorCodeCircuitOpen = "CIRCUIT_OPEN"

	ToolErrorCodeNetworkError = "NETWORK_ERROR"
	ToolErrorCodeAPIError     = "API_ERROR"
	ToolErrorCodeRateLimit    = "RATE_LIMIT"
)

func GetAllCodes() []string {
	return []string{
		ToolErrorCodeNotFound,
		ToolErrorCodeAccessDenied,
		ToolErrorCodePermissionDenied,
		ToolErrorCodeIsDirectory,
		ToolErrorCodeBinaryFile,
		ToolErrorCodeInvalidParam,
		ToolErrorCodeInvalidFormat,
		ToolErrorCodeExecutionError,
		ToolErrorCodeTimeout,
		ToolErrorCodeInternalError,
		ToolErrorCodeConflict,
		ToolErrorCodeCircuitOpen,
		ToolErrorCodeNetworkError,
		ToolErrorCodeAPIError,
		ToolErrorCodeRateLimit,
	}
}

func IsValidCode(code string) bool {
	for _, c := range GetAllCodes() {
		if c == code {
			return true
		}
	}
	return false
}
