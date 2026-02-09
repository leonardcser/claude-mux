package provider

import "strings"

func init() { Register(Codex{}) }

// Codex detects OpenAI Codex CLI sessions.
// Busy state is determined by the "esc to interrupt" indicator that Codex
// renders while working (e.g. "• Working (11s • esc to interrupt)").
type Codex struct{}

func (Codex) Command() string { return "codex" }

func (Codex) IsBusy(lines []string, _ int, _ *ProcessTable) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "esc to interrupt") {
			return true
		}
	}
	return false
}
