package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"
)

type SessionStore struct {
	SessionDir string
}

type SessionData struct {
	SessionID      string                    `json:"session_id"`
	CreatedAt      any                       `json:"created_at"`
	SavedAt        string                    `json:"saved_at"`
	AgentConfig    map[string]any            `json:"agent_config"`
	History        []map[string]any          `json:"history"`
	ToolSchemaHash string                    `json:"tool_schema_hash"`
	ReadCache      map[string]map[string]any `json:"read_cache"`
	Metadata       map[string]any            `json:"metadata"`
}

func NewSessionStore(sessionDir string) (*SessionStore, error) {
	if sessionDir == "" {
		sessionDir = "."
	}
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}
	return &SessionStore{SessionDir: sessionDir}, nil
}

func (s *SessionStore) generateSessionID() string {
	now := time.Now().Format("20060102-150405")
	suffixBytes := make([]byte, 4)
	if _, err := rand.Read(suffixBytes); err != nil {
		h := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		return fmt.Sprintf("s-%s-%s", now, hex.EncodeToString(h[:])[:8])
	}
	return fmt.Sprintf("s-%s-%s", now, hex.EncodeToString(suffixBytes))
}

func (s *SessionStore) Save(
	agentConfig map[string]any,
	history []Message,
	toolSchemaHash string,
	readCache map[string]map[string]any,
	metadata map[string]any,
	sessionName string,
) (string, error) {
	sessionID := s.generateSessionID()
	filename := fmt.Sprintf("session-%s.json", sessionID)
	if sessionName != "" {
		filename = sessionName + ".json"
	}

	if metadata == nil {
		metadata = map[string]any{}
	}

	historyMaps := make([]map[string]any, 0, len(history))
	for _, msg := range history {
		historyMaps = append(historyMaps, msg.ToMap())
	}

	createdAt := any(nowPythonISOTime())
	if rawCreatedAt, exists := metadata["created_at"]; exists {
		createdAt = rawCreatedAt
	}

	record := SessionData{
		SessionID:      sessionID,
		CreatedAt:      createdAt,
		SavedAt:        nowPythonISOTime(),
		AgentConfig:    agentConfig,
		History:        historyMaps,
		ToolSchemaHash: toolSchemaHash,
		ReadCache:      readCache,
		Metadata:       metadata,
	}

	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}

	outPath := filepath.Join(s.SessionDir, filename)
	tmpPath := outPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func (s *SessionStore) Load(filepath string) (SessionData, error) {
	var out SessionData
	data, err := os.ReadFile(filepath)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (s *SessionStore) ListSessions() ([]map[string]any, error) {
	entries, err := os.ReadDir(s.SessionDir)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || len(name) < 5 || name[len(name)-5:] != ".json" {
			continue
		}

		fullpath := filepath.Join(s.SessionDir, entry.Name())
		record, err := s.Load(fullpath)
		if err != nil {
			fmt.Printf("⚠️ 警告：无法读取 %s: %v\n", fullpath, err)
			continue
		}

		items = append(items, map[string]any{
			"filename":   entry.Name(),
			"filepath":   fullpath,
			"session_id": record.SessionID,
			"created_at": record.CreatedAt,
			"saved_at":   record.SavedAt,
			"metadata":   record.Metadata,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		a, _ := items[i]["saved_at"].(string)
		b, _ := items[j]["saved_at"].(string)
		return a > b
	})
	return items, nil
}

func (s *SessionStore) Delete(sessionName string) bool {
	path := filepath.Join(s.SessionDir, sessionName+".json")
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return os.Remove(path) == nil
}

func (s *SessionStore) CheckConfigConsistency(savedConfig, currentConfig map[string]any) map[string]any {
	warnings := make([]string, 0)
	pairs := [][2]string{{"llm_provider", "LLM 提供商变化"}, {"llm_model", "模型变化"}, {"max_steps", "最大步数变化"}}
	for _, pair := range pairs {
		k := pair[0]
		before := savedConfig[k]
		after := currentConfig[k]
		if !pythonValueEqual(before, after) {
			warnings = append(warnings, fmt.Sprintf("%s: %s → %s", pair[1], before, after))
		}
	}
	return map[string]any{
		"consistent": len(warnings) == 0,
		"warnings":   warnings,
	}
}

func (s *SessionStore) CheckToolSchemaConsistency(savedHash, currentHash string) map[string]any {
	changed := savedHash != currentHash
	recommendation := "可以安全恢复"
	if changed {
		recommendation = "建议重新读取文件"
	}
	return map[string]any{
		"changed":        changed,
		"saved_hash":     savedHash,
		"current_hash":   currentHash,
		"recommendation": recommendation,
	}
}

func pythonValueEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	if af, aok := numericValue(a); aok {
		if bf, bok := numericValue(b); bok {
			return af == bf
		}
	}

	if as, aok := a.(string); aok {
		bs, bok := b.(string)
		return bok && as == bs
	}

	if ab, aok := a.(bool); aok {
		if bb, bok := b.(bool); bok {
			return ab == bb
		}
	}

	return reflect.DeepEqual(a, b)
}

func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}
