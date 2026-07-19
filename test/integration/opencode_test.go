//go:build integration

package integration_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/session"
	"github.com/newtosh/tuile/internal/term"
)

func screenLooksLikeOpencode(snap term.ScreenSnapshot) bool {
	for _, line := range snap.Lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		if strings.Contains(lower, "opencode") {
			return true
		}
		if strings.Contains(trim, "█") || strings.Contains(trim, "▀") {
			return true
		}
		if strings.Contains(trim, "❯") || strings.HasPrefix(trim, ">") {
			return true
		}
	}
	return screenHasFrameChars(snap)
}

func TestOpencodeSpawn(t *testing.T) {
	opts, err := cli.SessionForCLI(cli.OpencodeCLI)
	if errors.Is(err, cli.ErrNotFound) {
		t.Skip("opencode not on PATH")
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
	if !screenLooksLikeOpencode(snap) {
		t.Fatalf("expected opencode startup UI, got lines=%v", nonEmptyLines(snap))
	}
}
