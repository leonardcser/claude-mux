package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leo/agent-mux/internal/tui"
)

func main() {
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "error: agent-mux must be run inside tmux")
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
