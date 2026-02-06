package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PaneStatus represents the state of a Claude pane.
type PaneStatus int

const (
	StatusIdle           PaneStatus = iota // waiting for user input
	StatusBusy                             // Claude is working
	StatusNeedsAttention                   // Claude needs user attention
)

// ClaudePane represents a tmux pane running Claude.
type ClaudePane struct {
	Target     string // e.g. "main:2.1"
	Session    string
	Window     string
	Pane       string
	Path       string
	PID        int
	Status     PaneStatus
	LastActive time.Time
}

// Workspace groups panes by working directory.
type Workspace struct {
	Path      string
	ShortPath string
	Panes     []ClaudePane
}

// GroupByWorkspace groups panes by their working directory.
func GroupByWorkspace(panes []ClaudePane) []Workspace {
	home, _ := os.UserHomeDir()
	groups := make(map[string][]ClaudePane)
	for _, p := range panes {
		groups[p.Path] = append(groups[p.Path], p)
	}

	var workspaces []Workspace
	for path, ps := range groups {
		short := filepath.Base(path)
		if short == "." || short == "/" {
			short = path
			if home != "" && strings.HasPrefix(short, home) {
				short = "~" + strings.TrimPrefix(short, home)
			}
		}
		workspaces = append(workspaces, Workspace{
			Path:      path,
			ShortPath: short,
			Panes:     ps,
		})
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Path < workspaces[j].Path
	})
	return workspaces
}
