package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type historyEntry struct {
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
}

// LastActiveByProject reads ~/.claude/history.jsonl and returns
// a map of project path -> last activity time.
func LastActiveByProject() map[string]time.Time {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	f, err := os.Open(filepath.Join(home, ".claude", "history.jsonl"))
	if err != nil {
		return nil
	}
	defer f.Close()

	result := make(map[string]time.Time)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		var entry historyEntry
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}
		if entry.Project == "" || entry.Timestamp == 0 {
			continue
		}
		t := time.UnixMilli(entry.Timestamp)
		if existing, ok := result[entry.Project]; !ok || t.After(existing) {
			result[entry.Project] = t
		}
	}
	return result
}
