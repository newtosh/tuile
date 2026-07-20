package session

import (
	"testing"

	"github.com/newtosh/tuile/internal/config"
)

func TestWritePTYResponseObserveOnly(t *testing.T) {
	mgr := NewManager()
	dir := t.TempDir()
	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close(sess.ID) }()

	resp := []byte("\x1b]11;rgb:0a/0a/0a\x1b\\")
	if err := mgr.WritePTYResponse(sess.ID, resp); err != nil {
		t.Fatalf("WritePTYResponse: %v", err)
	}
	if err := mgr.WriteHumanInput(sess.ID, []byte("y"), false, false); err != ErrObserveOnly {
		t.Fatalf("human write = %v, want observe-only", err)
	}
}
