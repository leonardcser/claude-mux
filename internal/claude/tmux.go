package claude

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ListClaudePanes returns all tmux panes currently running claude.
func ListClaudePanes() ([]ClaudePane, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}:#{window_index}.#{pane_index}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_pid}").Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}

	history := LastActiveByProject()

	var panes []ClaudePane
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) < 4 {
			continue
		}
		target, cmd, path, pidStr := fields[0], fields[1], fields[2], fields[3]
		if cmd != "claude" {
			continue
		}
		pid, _ := strconv.Atoi(pidStr)
		session, window, pane := parseTarget(target)
		panes = append(panes, ClaudePane{
			Target:     target,
			Session:    session,
			Window:     window,
			Pane:       pane,
			Path:       path,
			PID:        pid,
			Status:     detectStatus(pid, target),
			LastActive: history[path],
		})
	}
	return panes, nil
}

// detectStatus determines whether Claude needs attention, is busy, or is idle.
func detectStatus(shellPID int, target string) PaneStatus {
	if needsAttention(target) {
		return StatusNeedsAttention
	}
	if isClaudeBusy(shellPID) {
		return StatusBusy
	}
	return StatusIdle
}

// isClaudeBusy checks if Claude is actively working by looking for caffeinate
// child processes. Claude Code spawns caffeinate while processing (thinking,
// streaming, running tools) and kills it when idle.
func isClaudeBusy(shellPID int) bool {
	// Find the claude process (direct child of the shell)
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(shellPID)).Output()
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		childPID := strings.TrimSpace(line)
		if childPID == "" {
			continue
		}
		// Check if this child has a caffeinate subprocess
		if err := exec.Command("pgrep", "-P", childPID, "caffeinate").Run(); err == nil {
			return true
		}
	}
	return false
}

// needsAttention checks if Claude is waiting for user interaction.
func needsAttention(target string) bool {
	out, err := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-S", "-15").Output()
	if err != nil {
		return false
	}
	content := string(out)
	for _, pattern := range []string{
		// Tool permission prompts
		"Do you want to proceed?",
		"Do you want to allow",
		"Allow once",
		"press Enter to approve",
		// Question / selection prompts
		"Enter to select",
		"Type something",
		"Esc to cancel",
		// Waiting for user response
		"I'll wait for your",
		"waiting for your response",
		"Let me know when",
		"Please let me know",
		"What would you like",
		"How would you like",
		"Should I proceed",
		"Would you like me to",
		"please provide",
		"please specify",
		"I need more information",
		"Could you clarify",
		"awaiting your",
		"ready when you are",
		"let me know if you'd like",
		"Feel free to ask",
		"Is there anything else",
		"What else can I help",
	} {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// CapturePane captures the visible content of a tmux pane.
func CapturePane(target string, lines int) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", target, "-e", "-p", "-S",
		fmt.Sprintf("-%d", lines)).Output()
	if err != nil {
		return "", fmt.Errorf("capture-pane %s: %w", target, err)
	}
	return string(out), nil
}

// SwitchToPane switches the tmux client to the given pane.
func SwitchToPane(target string) error {
	session, window, _ := parseTarget(target)
	sessionWindow := session + ":" + window
	if err := exec.Command("tmux", "switch-client", "-t", sessionWindow).Run(); err != nil {
		return fmt.Errorf("switch-client: %w", err)
	}
	if err := exec.Command("tmux", "select-pane", "-t", target).Run(); err != nil {
		return fmt.Errorf("select-pane: %w", err)
	}
	return nil
}

// KillPane kills a tmux pane. If it's the only pane in the window, kills the window instead.
func KillPane(target string) error {
	session, window, _ := parseTarget(target)
	sessionWindow := session + ":" + window

	out, err := exec.Command("tmux", "list-panes", "-t", sessionWindow).Output()
	if err != nil {
		return fmt.Errorf("list-panes: %w", err)
	}
	paneCount := len(strings.Split(strings.TrimSpace(string(out)), "\n"))

	if paneCount <= 1 {
		return exec.Command("tmux", "kill-window", "-t", sessionWindow).Run()
	}
	return exec.Command("tmux", "kill-pane", "-t", target).Run()
}

// parseTarget splits "foo:2.1" into session="foo", window="2", pane="1".
func parseTarget(s string) (session, window, pane string) {
	colonIdx := strings.LastIndex(s, ":")
	if colonIdx < 0 {
		return s, "", ""
	}
	session = s[:colonIdx]
	rest := s[colonIdx+1:]
	dotIdx := strings.LastIndex(rest, ".")
	if dotIdx < 0 {
		return session, rest, ""
	}
	return session, rest[:dotIdx], rest[dotIdx+1:]
}
