package provider

import "strings"

func init() { Register(Gemini{}) }

// Gemini detects Google Gemini CLI sessions.
// Gemini runs as "node" in tmux, so it's resolved via the process table.
// Busy state is determined by the "esc to cancel" indicator that Gemini
// renders while working (e.g. "⠹ Investigating the Project (esc to cancel, 8s)").
type Gemini struct{}

func (Gemini) Command() string { return "gemini" }

func (Gemini) IsBusy(lines []string, _ int, _ *ProcessTable) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "esc to cancel") {
			return true
		}
	}
	return false
}
