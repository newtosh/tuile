package pty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
	"github.com/newtosh/tuile/internal/config"
)

// Master wraps a PTY file descriptor and child process.
type Master struct {
	File    *os.File
	Cmd     *exec.Cmd
	Cols    int
	Rows    int
	WorkDir string
}

// Start launches shell in a new PTY with the given workspace and dimensions.
func Start(workspace string, sess config.Session) (*Master, error) {
	workspace, err := resolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}

	cols, rows := config.NormalizeDimensions(sess.Cols, sess.Rows)

	var cmd *exec.Cmd
	if sess.Command != "" {
		cmd = exec.Command(sess.Command, sess.Args...)
	} else {
		shell := sess.Shell
		if shell == "" {
			shell = config.DefaultSession().Shell
		}
		cmd = exec.Command(shell)
	}
	cmd.Dir = workspace
	cmd.Env = sessionEnv(workspace)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	return &Master{
		File:    ptmx,
		Cmd:     cmd,
		Cols:    cols,
		Rows:    rows,
		WorkDir: workspace,
	}, nil
}

// Resize updates PTY winsize (R2).
func (m *Master) Resize(cols, rows int) error {
	cols, rows = config.NormalizeDimensions(cols, rows)
	if err := pty.Setsize(m.File, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
		return fmt.Errorf("set pty size: %w", err)
	}
	m.Cols = cols
	m.Rows = rows
	return nil
}

// Winsize returns current cols and rows tracked on the master.
func (m *Master) Winsize() (cols, rows int) {
	return m.Cols, m.Rows
}

// Close shuts down the PTY and reaps the child when possible.
func (m *Master) Close() error {
	if m.File != nil {
		_ = m.File.Close()
	}
	if m.Cmd != nil && m.Cmd.Process != nil {
		_ = m.Cmd.Process.Kill()
		_, _ = m.Cmd.Process.Wait()
	}
	return nil
}

func resolveWorkspace(workspace string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace path is required")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workspace not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace is not a directory: %s", abs)
	}
	return abs, nil
}

// sessionEnv builds PTY environment with a real terminal type (not dumb).
func sessionEnv(workspace string) []string {
	const (
		termVar       = "TERM=xterm-256color"
		colorVar      = "COLORTERM=truecolor"
		forceColorVar = "FORCE_COLOR=3"
		tmuxTruecolor = "CLAUDE_CODE_TMUX_TRUECOLOR=1"
	)
	out := make([]string, 0, len(os.Environ())+5)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "TERM=") ||
			strings.HasPrefix(kv, "COLORTERM=") ||
			strings.HasPrefix(kv, "PWD=") ||
			strings.HasPrefix(kv, "FORCE_COLOR=") ||
			strings.HasPrefix(kv, "NO_COLOR=") ||
			strings.HasPrefix(kv, "TMUX=") ||
			strings.HasPrefix(kv, "TMUX_PANE=") ||
			strings.HasPrefix(kv, "CLAUDE_CODE_TMUX_TRUECOLOR=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, termVar, colorVar, forceColorVar, tmuxTruecolor, "PWD="+workspace)
	return out
}
