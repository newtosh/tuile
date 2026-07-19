//go:build integration

package integration_test

import (
	"testing"

	"github.com/newtosh/tuile/testkit"
)

func TestBrowserViewerShowsPTYOutput(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)

	marker := "tuile-browser-marker"
	sess.EmitMarker(t, dir, marker)
	sess.AssertTerminalContains(t, marker)
}

func TestBrowserResizePropagatesWhenControlling(t *testing.T) {
	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, t.TempDir())

	sess.Takeover(t)
	sess.HumanResize(t, 55, 18)

	cols, rows := sess.ScreenGrid(t)
	if cols != 55 || rows != 18 {
		t.Fatalf("PTY grid = %dx%d, want 55x18 (AE3)", cols, rows)
	}
}
