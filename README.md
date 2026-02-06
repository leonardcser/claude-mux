# claude-mux

A TUI for multiplexing Claude Code sessions in tmux.

Lists all active Claude panes grouped by workspace, with a live preview panel
showing each session's output. Select a session and press enter to jump to it.

## Requirements

- Go 1.25+
- tmux (must be run inside a tmux session)

## Install

```
go install
```

## Usage

From inside tmux:

```
claude-mux
```

### tmux binding

Add to your `~/.tmux.conf` to open claude-mux with `prefix + j`:

```tmux
bind j run-shell "tmux neww claude-mux"
```

### Keys

| Key         | Action            |
| ----------- | ----------------- |
| `j` / `k`   | Navigate up/down  |
| `enter`     | Switch to session |
| `dd`        | Kill session      |
| `q` / `esc` | Quit              |
