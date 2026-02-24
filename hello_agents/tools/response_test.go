package tools

import (
	"encoding/json"
	"testing"
)

func TestToolResponseHelpers(t *testing.T) {
	success := Success("ok", map[string]any{"value": 42})
	if success.Status != ToolStatusSuccess {
		t.Fatalf("success status = %q, want success", success.Status)
	}
	if success.Data["value"] != 42 {
		t.Fatalf("success data value = %v, want 42", success.Data["value"])
	}

	partial := Partial("partial", map[string]any{"truncated": true})
	if partial.Status != ToolStatusPartial {
		t.Fatalf("partial status = %q, want partial", partial.Status)
	}

	errResp := Error("failed", ToolErrorCodeInvalidParam, nil)
	if errResp.Status != ToolStatusError {
		t.Fatalf("error status = %q, want error", errResp.Status)
	}
	if errResp.ErrorInfo["code"] != ToolErrorCodeInvalidParam {
		t.Fatalf("error code = %q, want %q", errResp.ErrorInfo["code"], ToolErrorCodeInvalidParam)
	}
}

func TestToolResponseJSONRoundTrip(t *testing.T) {
	resp := Success("OK", map[string]any{"value": 123})
	jsonStr := resp.ToJSON()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("ToJSON() output is not valid json: %v", err)
	}
	if parsed["status"] != "success" {
		t.Fatalf("status = %v, want success", parsed["status"])
	}
}

func TestToolResponseFromMapAcceptsStringMapError(t *testing.T) {
	resp := ToolResponseFromMap(map[string]any{
		"status": "error",
		"text":   "失败",
		"data":   map[string]any{},
		"error":  map[string]string{"code": "TEST_ERROR", "message": "失败"},
	})
	if resp.Status != ToolStatusError {
		t.Fatalf("status = %q, want error", resp.Status)
	}
	if resp.ErrorInfo["code"] != "TEST_ERROR" {
		t.Fatalf("error code = %q, want TEST_ERROR", resp.ErrorInfo["code"])
	}
}

func TestToolResponseFromJSONPanicsOnInvalidJSONLikePython(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("ToolResponseFromJSON() should panic on invalid json")
		}
	}()
	_ = ToolResponseFromJSON("not-json")
}

func TestToolResponseFromMapPanicsOnInvalidStatusLikePythonEnum(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("ToolResponseFromMap() should panic on invalid status")
		}
	}()
	_ = ToolResponseFromMap(map[string]any{
		"status": "unknown",
		"text":   "x",
	})
}
