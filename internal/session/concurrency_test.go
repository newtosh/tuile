package session

import (
	"testing"

	"github.com/newtosh/tuile/internal/config"
)

func TestConcurrencyAgentDefault(t *testing.T) {
	mgr := NewManager()
	dir := t.TempDir()
	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	if sess.Access.Controller() != ControllerAgent {
		t.Fatalf("controller = %q, want agent", sess.Access.Controller())
	}
	if err := mgr.WriteAgentInput(sess.ID, []byte("x"), false, false); err != nil {
		t.Fatalf("agent write: %v", err)
	}
	if err := mgr.WriteHumanInput(sess.ID, []byte("y"), false, false); err != ErrObserveOnly {
		t.Fatalf("human write = %v, want observe-only", err)
	}
}

func TestConcurrencyTakeoverAndRelease(t *testing.T) {
	mgr := NewManager()
	dir := t.TempDir()
	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	if err := mgr.Takeover(sess.ID); err != nil {
		t.Fatal(err)
	}
	if err := mgr.WriteAgentInput(sess.ID, []byte("x"), false, false); err != ErrHumanControls {
		t.Fatalf("agent write = %v, want human controls", err)
	}
	if err := mgr.WriteHumanInput(sess.ID, []byte("y"), false, false); err != nil {
		t.Fatalf("human write: %v", err)
	}

	if err := mgr.Release(sess.ID); err != nil {
		t.Fatal(err)
	}
	if err := mgr.WriteHumanInput(sess.ID, []byte("z"), false, false); err != ErrObserveOnly {
		t.Fatalf("human write after release = %v", err)
	}
	if err := mgr.WriteAgentInput(sess.ID, []byte("a"), false, false); err != nil {
		t.Fatalf("agent write after release: %v", err)
	}
}

func TestConcurrencyResizeAuthority(t *testing.T) {
	mgr := NewManager()
	dir := t.TempDir()
	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	if err := mgr.ResizeAgent(sess.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	cols, rows := sess.PTY.Winsize()
	if cols != 100 || rows != 30 {
		t.Fatalf("size = %dx%d", cols, rows)
	}

	if err := mgr.Takeover(sess.ID); err != nil {
		t.Fatal(err)
	}
	if err := mgr.ResizeAgent(sess.ID, 80, 24); err != ErrHumanControls {
		t.Fatalf("agent resize during takeover = %v", err)
	}
	if err := mgr.ResizeHuman(sess.ID, 60, 20); err != nil {
		t.Fatalf("human resize: %v", err)
	}
	cols, rows = sess.PTY.Winsize()
	if cols != 60 || rows != 20 {
		t.Fatalf("human size = %dx%d", cols, rows)
	}

	if err := mgr.Release(sess.ID); err != nil {
		t.Fatal(err)
	}
	cols, rows = sess.PTY.Winsize()
	if cols != 100 || rows != 30 {
		t.Fatalf("after release expected agent size 100x30, got %dx%d", cols, rows)
	}
}
