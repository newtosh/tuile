//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func TestCLISessionUsesWorkspace(t *testing.T) {
	_, err := cli.SessionForCLI(cli.Claude)
	if errors.Is(err, cli.ErrNotFound) {
		t.Skip("claude not on PATH")
	}
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	mgr := session.NewManager()
	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()
	if sess.Workspace != dir {
		t.Fatalf("workspace = %q, want %q", sess.Workspace, dir)
	}
}
