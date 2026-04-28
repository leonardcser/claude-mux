package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/leo/agent-mux/internal/agent"
)

// dw returns the display width of s, accounting for ANSI escapes and wide
// runes. Use this instead of len() whenever doing column math on rendered
// content; len() returns bytes and silently drifts on multibyte runes.
func dw(s string) int { return lipgloss.Width(s) }

// ItemKind distinguishes workspace headers from pane entries.
type ItemKind int

const (
	KindWorkspace ItemKind = iota
	KindPane
	KindSectionHeader
	KindProjectGroup
)

// TreeItem is one visible row in the flattened tree.
type TreeItem struct {
	Kind        ItemKind
	PaneID      string // stable tmux pane id (KindPane) or first pane id in workspace (KindWorkspace)
	HeaderTitle string // for KindSectionHeader
	InGroup     bool   // KindPane: pane lives under a multi-worktree project header
}

// NextPane returns the index of the next KindPane item after from, wrapping around if none.
func NextPane(items []TreeItem, from int) int {
	for i := from + 1; i < len(items); i++ {
		if items[i].Kind == KindPane {
			return i
		}
	}
	if len(items) > 0 {
		for i := range from {
			if items[i].Kind == KindPane {
				return i
			}
		}
	}
	return from
}

// PrevPane returns the index of the previous KindPane item before from, wrapping around if none.
func PrevPane(items []TreeItem, from int) int {
	for i := from - 1; i >= 0; i-- {
		if items[i].Kind == KindPane {
			return i
		}
	}
	if len(items) > 0 {
		for i := len(items) - 1; i > from; i-- {
			if items[i].Kind == KindPane {
				return i
			}
		}
	}
	return from
}

// NearestPane returns the closest KindPane to the given index without wrapping.
func NearestPane(items []TreeItem, from int) int {
	if len(items) == 0 {
		return 0
	}
	if from >= len(items) {
		from = len(items) - 1
	}
	if from < 0 {
		from = 0
	}
	if items[from].Kind == KindPane {
		return from
	}

	sectionStart, sectionEnd := sectionBounds(items, from)

	// Find closest pane within the current section (by distance).
	best := -1
	for i := sectionStart; i < sectionEnd; i++ {
		if items[i].Kind != KindPane {
			continue
		}
		dist := i - from
		bestDist := best - from
		if dist < 0 {
			dist = -dist
		}
		if bestDist < 0 {
			bestDist = -bestDist
		}
		if best < 0 || dist < bestDist {
			best = i
		}
	}
	if best >= 0 {
		return best
	}

	// Fall back to other sections.
	for i := from - 1; i >= 0; i-- {
		if items[i].Kind == KindPane {
			return i
		}
	}
	for i := from + 1; i < len(items); i++ {
		if items[i].Kind == KindPane {
			return i
		}
	}
	return 0
}

// sectionBounds returns the start (inclusive) and end (exclusive) indices of
// the section containing the given index.
func sectionBounds(items []TreeItem, idx int) (int, int) {
	start := 0
	for i := idx - 1; i >= 0; i-- {
		if items[i].Kind == KindSectionHeader && items[i].HeaderTitle != "" {
			start = i
			break
		}
	}
	end := len(items)
	for i := idx + 1; i < len(items); i++ {
		if items[i].Kind == KindSectionHeader && items[i].HeaderTitle != "" {
			end = i
			break
		}
	}
	return start, end
}

// LastPane returns the index of the last KindPane item, or 0 if none.
func LastPane(items []TreeItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Kind == KindPane {
			return i
		}
	}
	return 0
}

// FirstPane returns the index of the first KindPane item, or -1 if none.
func FirstPane(items []TreeItem) int {
	for i, it := range items {
		if it.Kind == KindPane {
			return i
		}
	}
	return -1
}

// renderTreeItem renders a single row.
func (m Model) renderTreeItem(item TreeItem, selected bool, width int) string {
	if item.Kind == KindSectionHeader {
		if item.HeaderTitle == "" {
			return ""
		}
		label := " " + item.HeaderTitle + " "
		lineLen := max(width-dw(label)-1, 0)
		return stashedSectionStyle.Render("─" + label + strings.Repeat("─", lineLen))
	}

	p := m.panes[item.PaneID]
	if p == nil {
		return ""
	}

	switch item.Kind {
	case KindWorkspace:
		return renderWorkspaceHeader(p, width)
	case KindProjectGroup:
		return renderProjectGroupHeader(p, width)
	case KindPane:
		return renderPaneRow(p, selected, width, item.InGroup)
	}
	return ""
}

func renderProjectGroupHeader(p *agent.Pane, width int) string {
	name := p.ProjectShort
	if name == "" {
		name = p.ShortPath
	}
	branch := p.ProjectBranch
	if branch != "" && p.ProjectDirty {
		branch += "*"
	}

	avail := width - 2
	if branch != "" {
		needed := dw(name) + 1 + dw(branch)
		if needed > avail {
			branchAvail := avail - dw(name) - 1
			if branchAvail >= 4 {
				branch = truncate(branch, branchAvail)
			} else {
				branch = ""
			}
		}
		if branch == "" {
			name = truncate(name, avail)
		}
	} else {
		name = truncate(name, avail)
	}

	text := " " + name
	if branch != "" {
		pad := max(width-dw(text)-dw(branch)-1, 0)
		text += strings.Repeat(" ", pad)
		return workspaceStyle.Render(text) + branchStyle.Render(branch) + branchStyle.Render(" ")
	}
	text += strings.Repeat(" ", max(width-dw(text), 0))
	return workspaceStyle.Render(text)
}

func renderWorkspaceHeader(p *agent.Pane, width int) string {
	avail := width - 2
	name := p.ShortPath
	branch := p.GitBranch
	if branch != "" && p.GitDirty {
		branch += "*"
	}

	if branch != "" {
		needed := dw(name) + 1 + dw(branch)
		if needed > avail {
			branchAvail := avail - dw(name) - 1
			if branchAvail >= 4 {
				branch = truncate(branch, branchAvail)
			} else {
				branch = ""
			}
		}
		if branch == "" {
			name = truncate(name, avail)
		}
	} else {
		name = truncate(name, avail)
	}

	text := " " + name
	if branch != "" {
		pad := max(width-dw(text)-dw(branch)-1, 0)
		text += strings.Repeat(" ", pad)
		return workspaceStyle.Render(text) + branchStyle.Render(branch) + branchStyle.Render(" ")
	}
	text += strings.Repeat(" ", max(width-dw(text), 0))
	return workspaceStyle.Render(text)
}

func renderPaneRow(p *agent.Pane, selected bool, width int, inGroup bool) string {
	var winLabel string
	if p.WindowName != "" {
		winLabel = fmt.Sprintf("%s:%s", p.Window, p.WindowName)
	} else {
		winLabel = fmt.Sprintf("%s:%s", p.Session, p.Window)
	}

	// Worktree label: dim, only for actual worktrees (Path != ProjectRoot).
	worktree := ""
	if inGroup && p.ShortPath != "" && p.Path != p.ProjectRoot {
		worktree = p.ShortPath
	}

	// Timer column has a fixed width so the right edge stays aligned across
	// busy rows (no timer) and idle rows. formatElapsed uses a single unit,
	// max " 999s "-ish; 5 cols covers the common case.
	const elapsedSlotW = 5
	elapsedRendered := strings.Repeat(" ", elapsedSlotW)
	if !p.LastActive.IsZero() && p.Status != agent.StatusBusy {
		v := " " + formatElapsed(time.Since(p.LastActive)) + " "
		if dw(v) > elapsedSlotW {
			v = truncate(v, elapsedSlotW)
		}
		elapsedRendered = strings.Repeat(" ", elapsedSlotW-dw(v)) + v
	}

	prefix := "   "
	middleAvail := width - dw(prefix) - 2 - elapsedSlotW // 2 = icon cell + leading space

	// window:idx is always shown in full; truncate as a last resort if
	// somehow wider than the available middle.
	if dw(winLabel) > middleAvail {
		winLabel = truncate(winLabel, middleAvail)
	}
	remaining := middleAvail - dw(winLabel)

	// Worktree name takes whatever space is left after the window label, but
	// reserves room for a 2-space separator. If too tight, drop it.
	worktreeRendered := ""
	if worktree != "" && remaining >= 4 {
		sepW := 2
		avail := remaining - sepW
		if dw(worktree) > avail {
			worktree = truncate(worktree, avail)
		}
		worktreeRendered = strings.Repeat(" ", sepW) + worktree
	}
	gap := max(remaining-dw(worktreeRendered), 0)

	icons := normalIcons
	if selected {
		icons = selectedIcons
	} else if p.Stashed {
		icons = stashedIcons
	}

	var icon string
	switch p.Status {
	case agent.StatusBusy:
		icon = icons.busy
	case agent.StatusNeedsAttention, agent.StatusUnread:
		icon = icons.attention
	default:
		icon = icons.idle
	}

	if selected {
		body := " " + winLabel + worktreeRendered + strings.Repeat(" ", gap) + elapsedRendered
		return selectedStyle.Render(prefix) + icon + selectedStyle.Render(body)
	}

	line := icons.text.Render(prefix) + icon + icons.text.Render(" "+winLabel)
	if worktreeRendered != "" {
		line += icons.dim.Render(worktreeRendered)
	}
	line += icons.dim.Render(strings.Repeat(" ", gap) + elapsedRendered)
	return line
}

// truncate shortens s to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "…"
}

// formatElapsed returns a compact duration string using a single unit.
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
}

// VisibleSlice returns the start index for scrolling the tree view.
func VisibleSlice(total, cursor, height int) int {
	if total <= height {
		return 0
	}
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	if start+height > total {
		start = total - height
	}
	return start
}
