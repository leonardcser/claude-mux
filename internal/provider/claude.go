package provider

func init() { Register(Claude{}) }

// Claude detects Claude Code sessions.
// Busy state is determined by the presence of a caffeinate process in the
// shell's process tree (Claude Code spawns caffeinate while working).
type Claude struct{}

func (Claude) Command() string { return "claude" }

func (Claude) IsBusy(_ []string, shellPID int, pt *ProcessTable) bool {
	return pt.HasGrandchild(shellPID, "caffeinate")
}
