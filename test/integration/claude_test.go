//go:build integration

package integration_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/session"
	"github.com/newtosh/tuile/internal/term"
)

func waitForNonEmptyScreen(t *testing.T, sess *session.Session) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snap := sess.Emulator.Snapshot()
		for _, line := range snap.Lines {
			if strings.TrimSpace(line) != "" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timed out waiting for CLI TUI output")
}

func nonEmptyLines(snap term.ScreenSnapshot) []string {
	var out []string
	for _, line := range snap.Lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

func screenHasFrameChars(snap term.ScreenSnapshot) bool {
	for _, line := range snap.Lines {
		if strings.ContainsAny(line, "┌┐└┘│─┌") {
			return true
		}
	}
	return false
}

func screenLooksLikeInteractivePrompt(snap term.ScreenSnapshot) bool {
	for _, line := range snap.Lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "❯") || strings.HasPrefix(trim, ">") {
			return true
		}
	}
	return len(nonEmptyLines(snap)) > 0
}

func screenLooksLikeCodex(snap term.ScreenSnapshot) bool {
	if screenHasFrameChars(snap) {
		return true
	}
	for _, line := range snap.Lines {
		trim := strings.TrimSpace(line)
		if strings.Contains(line, "Codex") || strings.HasPrefix(trim, ">") {
			return true
		}
	}
	return false
}

func TestClaudeSpawn(t *testing.T) {
	opts, err := cli.SessionForCLI(cli.Claude)
	if errors.Is(err, cli.ErrNotFound) {
		t.Skip("claude not on PATH")
	}
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	mgr := session.NewManager()
	sess, err := mgr.Create(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	waitForNonEmptyScreen(t, sess)
	snap := sess.Emulator.Snapshot()
	if !screenLooksLikeInteractivePrompt(snap) {
		t.Fatalf("expected claude interactive prompt region, got lines=%v", nonEmptyLines(snap))
	}
}
