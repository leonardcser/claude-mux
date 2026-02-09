package provider

import "strings"

func init() { Register(OpenCode{}) }

// OpenCode detects Open Code sessions.
// Busy state is determined by the "esc interrupt" indicator that Open Code
// renders at the bottom of the pane while processing.
type OpenCode struct{}

func (OpenCode) Command() string { return "opencode" }

func (OpenCode) IsBusy(lines []string, _ int, _ *ProcessTable) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "esc interrupt") {
			return true
		}
	}
	return false
}
