package pty

import (
	"strings"
	"testing"
)

func TestSessionEnvOverridesDumbTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("COLORTERM", "invalid")
	t.Setenv("PWD", "/old")

	env := sessionEnv("/workspace")
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "TERM=xterm-256color") {
		t.Fatalf("expected xterm-256color, got:\n%s", joined)
	}
	if !strings.Contains(joined, "COLORTERM=truecolor") {
		t.Fatalf("expected truecolor, got:\n%s", joined)
	}
	if !strings.Contains(joined, "PWD=/workspace") {
		t.Fatalf("expected workspace pwd, got:\n%s", joined)
	}
	if strings.Contains(joined, "TERM=dumb") {
		t.Fatal("dumb TERM should be replaced")
	}
	if strings.Contains(joined, "NO_COLOR=") {
		t.Fatal("NO_COLOR should be stripped when forcing color")
	}
	if !strings.Contains(joined, "FORCE_COLOR=3") {
		t.Fatalf("expected FORCE_COLOR=3, got:\n%s", joined)
	}
}

func TestSessionEnvStripsTmuxForClaudeColors(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-0")
	t.Setenv("TMUX_PANE", "%0")

	env := sessionEnv("/workspace")
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "TMUX=") {
		t.Fatalf("TMUX should be stripped for agent color detection:\n%s", joined)
	}
	if !strings.Contains(joined, "CLAUDE_CODE_TMUX_TRUECOLOR=1") {
		t.Fatalf("expected tmux truecolor override, got:\n%s", joined)
	}
}
