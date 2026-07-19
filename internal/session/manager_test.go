package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func TestCreateSessionWorkspacePWD(t *testing.T) {
	if os.Getenv("SHELL") == "" {
		t.Skip("SHELL not set")
	}

	dir := t.TempDir()
	mgr := session.NewManager()

	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(sess.ID) })

	if sess.Workspace != dir {
		t.Fatalf("session workspace = %q, want %q", sess.Workspace, dir)
	}

	marker := "tuile-workspace-" + filepath.Base(dir)
	if err := os.WriteFile(filepath.Join(dir, ".tuile-marker"), []byte(marker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = sess.PTY.File.Write([]byte("cat .tuile-marker\n"))
	if err != nil {
		t.Fatalf("write to pty: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var out string
	for time.Now().Before(deadline) {
		out = sess.Emulator.String()
		if strings.Contains(out, marker) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected marker %q in screen, got %q", marker, out)
}

func TestResizeUpdatesPTYAndEmulator(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager()

	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(sess.ID) })

	if err := mgr.ResizeAgent(sess.ID, 100, 40); err != nil {
		t.Fatalf("resize: %v", err)
	}

	cols, rows := sess.PTY.Winsize()
	if cols != 100 || rows != 40 {
		t.Fatalf("pty winsize = %dx%d, want 100x40", cols, rows)
	}
	if sess.Emulator.Cols() != 100 || sess.Emulator.Rows() != 40 {
		t.Fatalf("emulator size = %dx%d, want 100x40", sess.Emulator.Cols(), sess.Emulator.Rows())
	}
}

func TestResizeClampsSmallDimensions(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager()

	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(sess.ID) })

	if err := mgr.ResizeAgent(sess.ID, 1, 1); err != nil {
		t.Fatalf("resize: %v", err)
	}

	cols, rows := sess.PTY.Winsize()
	if cols < config.MinCols || rows < config.MinRows {
		t.Fatalf("expected clamped size >= %dx%d, got %dx%d", config.MinCols, config.MinRows, cols, rows)
	}
}

func TestInvalidWorkspaceNoSessionLeak(t *testing.T) {
	mgr := session.NewManager()
	_, err := mgr.Create("/nonexistent/workspace/path", config.DefaultSession())
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}
