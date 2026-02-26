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

func TestToolResponseFromJSONReturnsErrorOnInvalidJSON(t *testing.T) {
	resp := ToolResponseFromJSON("not-json")
	if resp.Status != ToolStatusError {
		t.Fatalf("ToolResponseFromJSON() status = %q, want error on invalid json", resp.Status)
	}
}

func TestToolResponseFromMapDefaultsToErrorOnInvalidStatus(t *testing.T) {
	resp := ToolResponseFromMap(map[string]any{
		"status": "unknown",
		"text":   "x",
	})
	if resp.Status != ToolStatusError {
		t.Fatalf("ToolResponseFromMap() status = %q, want error on invalid status", resp.Status)
	}
}
