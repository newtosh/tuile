//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/session"
)

func TestCodexSpawn(t *testing.T) {
	opts, err := cli.SessionForCLI(cli.Codex)
	if errors.Is(err, cli.ErrNotFound) {
		t.Skip("codex not on PATH")
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
	if !screenLooksLikeCodex(snap) {
		t.Fatalf("expected codex startup UI, got lines=%v", nonEmptyLines(snap))
	}
}
