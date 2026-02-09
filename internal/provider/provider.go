package provider

import (
	"strconv"
	"strings"
)

// ProcessTable holds a snapshot of the system process tree.
type ProcessTable struct {
	Children map[int][]int  // ppid -> child pids
	Comm     map[int]string // pid -> command basename
	Args     map[int]string // pid -> full command line args
}

// HasGrandchild returns true if any grandchild of pid matches the given command name.
func (pt *ProcessTable) HasGrandchild(pid int, name string) bool {
	for _, childPID := range pt.Children[pid] {
		for _, grandchildPID := range pt.Children[childPID] {
			comm := pt.Comm[grandchildPID]
			if comm == name || strings.HasSuffix(comm, "/"+name) {
				return true
			}
		}
	}
	return false
}

// Provider defines how to detect an AI coding agent in tmux.
type Provider interface {
	// Command returns the binary name that appears as tmux pane_current_command.
	Command() string
	// IsBusy reports whether the agent is actively working.
	IsBusy(lines []string, shellPID int, pt *ProcessTable) bool
}

var registry = map[string]Provider{}

// Register adds a provider to the global registry.
func Register(p Provider) {
	registry[p.Command()] = p
}

// Get returns the provider for the given command, or nil.
func Get(cmd string) Provider {
	return registry[cmd]
}

// IsAgent returns true if the command matches a registered provider.
func IsAgent(cmd string) bool {
	_, ok := registry[cmd]
	return ok
}

// Resolve returns the provider command name for a tmux pane. It first checks
// the direct command, then falls back to inspecting children of the shell
// process via the process table (handles cases like gemini running as "node").
func Resolve(cmd string, shellPID int, pt *ProcessTable) string {
	if _, ok := registry[cmd]; ok {
		return cmd
	}
	// Check if any child of the shell is running a registered agent.
	// First check comm, then check args for script-based tools (e.g.
	// gemini runs as "node /opt/homebrew/bin/gemini").
	for _, childPID := range pt.Children[shellPID] {
		comm := pt.Comm[childPID]
		base := comm
		if idx := strings.LastIndex(comm, "/"); idx >= 0 {
			base = comm[idx+1:]
		}
		if _, ok := registry[base]; ok {
			return base
		}
		// Check each arg token for a registered command basename.
		for arg := range strings.SplitSeq(pt.Args[childPID], " ") {
			if idx := strings.LastIndex(arg, "/"); idx >= 0 {
				arg = arg[idx+1:]
			}
			if _, ok := registry[arg]; ok {
				return arg
			}
		}
	}
	return ""
}

// ParseProcessTable builds a ProcessTable from raw `ps -eo pid,ppid,comm,args` output.
func ParseProcessTable(out string) ProcessTable {
	pt := ProcessTable{
		Children: make(map[int][]int),
		Comm:     make(map[int]string),
		Args:     make(map[int]string),
	}
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		pt.Children[ppid] = append(pt.Children[ppid], pid)
		pt.Comm[pid] = fields[2]
		if len(fields) > 3 {
			pt.Args[pid] = strings.Join(fields[3:], " ")
		}
	}
	return pt
}
