package builtin

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"helloagents-go/HelloAgents-go/tools"
)

// NoteTool provides note-taking functionality (interface definition with basic in-memory implementation).
// This can be extended to use persistent storage like databases, file systems, or cloud services.
type NoteTool struct {
	*tools.BaseTool
	notes map[string]NoteEntry
	mu    sync.RWMutex
}

// NoteEntry represents a single note entry.
type NoteEntry struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewNoteTool creates a new NoteTool with in-memory storage.
func NewNoteTool() *NoteTool {
	return &NoteTool{
		BaseTool: tools.NewBaseTool(
			"note",
			"Manages notes with CRUD operations. Supports adding, reading, updating, deleting, and listing notes. "+
				"Currently uses in-memory storage (can be extended for persistence).",
			[]tools.ToolParameter{
				{
					Name:        "action",
					Type:        "string",
					Description: "Action to perform: add, read, update, delete, list, search",
					Required:    true,
					Enum:        []string{"add", "read", "update", "delete", "list", "search"},
				},
				{
					Name:        "id",
					Type:        "string",
					Description: "Note ID (for read, update, delete operations)",
					Required:    false,
				},
				{
					Name:        "title",
					Type:        "string",
					Description: "Note title (for add, update operations)",
					Required:    false,
				},
				{
					Name:        "content",
					Type:        "string",
					Description: "Note content (for add, update operations)",
					Required:    false,
				},
				{
					Name:        "tags",
					Type:        "string",
					Description: "Comma-separated tags (for add, update operations)",
					Required:    false,
				},
				{
					Name:        "query",
					Type:        "string",
					Description: "Search query (for search operation)",
					Required:    false,
				},
			},
		),
		notes: make(map[string]NoteEntry),
	}
}

// Run executes a note operation.
func (nt *NoteTool) Run(parameters map[string]interface{}) (string, error) {
	action, ok := parameters["action"].(string)
	if !ok {
		return "", fmt.Errorf("action parameter is required")
	}

	switch strings.ToLower(action) {
	case "add":
		return nt.addNote(parameters)
	case "read":
		return nt.readNote(parameters)
	case "update":
		return nt.updateNote(parameters)
	case "delete":
		return nt.deleteNote(parameters)
	case "list":
		return nt.listNotes()
	case "search":
		return nt.searchNotes(parameters)
	default:
		return "", fmt.Errorf("unknown action: %s. Valid actions: add, read, update, delete, list, search", action)
	}
}

// addNote adds a new note.
func (nt *NoteTool) addNote(parameters map[string]interface{}) (string, error) {
	title, _ := parameters["title"].(string)
	content, _ := parameters["content"].(string)

	if title == "" && content == "" {
		return "", fmt.Errorf("at least title or content must be provided")
	}

	// Parse tags
	var tags []string
	if tagsStr, ok := parameters["tags"].(string); ok {
		tags = splitTags(tagsStr)
	}

	// Generate ID
	id := generateNoteID()

	// Create note entry
	entry := NoteEntry{
		ID:       id,
		Title:    title,
		Content:  content,
		Tags:     tags,
		Metadata: make(map[string]interface{}),
	}

	// Store note
	nt.mu.Lock()
	nt.notes[id] = entry
	nt.mu.Unlock()

	return fmt.Sprintf("Note added successfully with ID: %s", id), nil
}

// readNote reads a note by ID.
func (nt *NoteTool) readNote(parameters map[string]interface{}) (string, error) {
	id, ok := parameters["id"].(string)
	if !ok {
		return "", fmt.Errorf("id parameter is required for read operation")
	}

	nt.mu.RLock()
	entry, exists := nt.notes[id]
	nt.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("note not found: %s", id)
	}

	result := fmt.Sprintf("Note ID: %s\n", entry.ID)
	if entry.Title != "" {
		result += fmt.Sprintf("Title: %s\n", entry.Title)
	}
	if entry.Content != "" {
		result += fmt.Sprintf("Content: %s\n", entry.Content)
	}
	if len(entry.Tags) > 0 {
		result += fmt.Sprintf("Tags: %v\n", entry.Tags)
	}

	return result, nil
}

// updateNote updates an existing note.
func (nt *NoteTool) updateNote(parameters map[string]interface{}) (string, error) {
	id, ok := parameters["id"].(string)
	if !ok {
		return "", fmt.Errorf("id parameter is required for update operation")
	}

	nt.mu.Lock()
	defer nt.mu.Unlock()

	entry, exists := nt.notes[id]
	if !exists {
		return "", fmt.Errorf("note not found: %s", id)
	}

	// Update fields
	if title, ok := parameters["title"].(string); ok {
		entry.Title = title
	}
	if content, ok := parameters["content"].(string); ok {
		entry.Content = content
	}
	if tagsStr, ok := parameters["tags"].(string); ok {
		entry.Tags = splitTags(tagsStr)
	}

	nt.notes[id] = entry

	return fmt.Sprintf("Note %s updated successfully", id), nil
}

// deleteNote deletes a note by ID.
func (nt *NoteTool) deleteNote(parameters map[string]interface{}) (string, error) {
	id, ok := parameters["id"].(string)
	if !ok {
		return "", fmt.Errorf("id parameter is required for delete operation")
	}

	nt.mu.Lock()
	defer nt.mu.Unlock()

	if _, exists := nt.notes[id]; !exists {
		return "", fmt.Errorf("note not found: %s", id)
	}

	delete(nt.notes, id)

	return fmt.Sprintf("Note %s deleted successfully", id), nil
}

// listNotes lists all notes.
func (nt *NoteTool) listNotes() (string, error) {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	if len(nt.notes) == 0 {
		return "No notes found", nil
	}

	result := fmt.Sprintf("Found %d notes:\n\n", len(nt.notes))

	for _, entry := range nt.notes {
		result += fmt.Sprintf("- ID: %s\n", entry.ID)
		if entry.Title != "" {
			result += fmt.Sprintf("  Title: %s\n", entry.Title)
		}
		if len(entry.Tags) > 0 {
			result += fmt.Sprintf("  Tags: %v\n", entry.Tags)
		}
		if entry.Content != "" {
			preview := entry.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			result += fmt.Sprintf("  Preview: %s\n", preview)
		}
		result += "\n"
	}

	return result, nil
}

// searchNotes searches for notes by query.
func (nt *NoteTool) searchNotes(parameters map[string]interface{}) (string, error) {
	query, ok := parameters["query"].(string)
	if !ok {
		return "", fmt.Errorf("query parameter is required for search operation")
	}

	nt.mu.RLock()
	defer nt.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]NoteEntry, 0)

	for _, entry := range nt.notes {
		// Search in title, content, and tags
		if strings.Contains(strings.ToLower(entry.Title), query) ||
			strings.Contains(strings.ToLower(entry.Content), query) {
			results = append(results, entry)
			continue
		}

		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, entry)
				break
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No notes found matching query: %s", query), nil
	}

	result := fmt.Sprintf("Found %d notes matching '%s':\n\n", len(results), query)

	for _, entry := range results {
		result += fmt.Sprintf("- ID: %s\n", entry.ID)
		if entry.Title != "" {
			result += fmt.Sprintf("  Title: %s\n", entry.Title)
		}
		result += "\n"
	}

	return result, nil
}

// GetNoteCount returns the total number of notes.
func (nt *NoteTool) GetNoteCount() int {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	return len(nt.notes)
}

// ClearAllNotes removes all notes.
func (nt *NoteTool) ClearAllNotes() {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.notes = make(map[string]NoteEntry)
}

// ExportNotes exports all notes as a string (for future persistence implementation).
func (nt *NoteTool) ExportNotes() string {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	result := fmt.Sprintf("# Notes Export (%d notes)\n\n", len(nt.notes))

	for _, entry := range nt.notes {
		result += fmt.Sprintf("## %s (ID: %s)\n", entry.Title, entry.ID)
		if len(entry.Tags) > 0 {
			result += fmt.Sprintf("Tags: %v\n", entry.Tags)
		}
		result += fmt.Sprintf("\n%s\n\n", entry.Content)
	}

	return result
}

// generateNoteID generates a unique note ID.
func generateNoteID() string {
	return fmt.Sprintf("note_%d", time.Now().UnixNano())
}

// Helper function to split tags (reused from memory.go)
func splitTags(tagsStr string) []string {
	// Simple implementation - split by comma
	tags := make([]string, 0)
	current := ""
	inQuotes := false

	for _, ch := range tagsStr {
		switch ch {
		case '"', '\'':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				if current != "" {
					tags = append(tags, current)
					current = ""
				}
				continue
			}
			fallthrough
		default:
			current += string(ch)
		}
	}

	if current != "" {
		tags = append(tags, current)
	}

	return tags
}
