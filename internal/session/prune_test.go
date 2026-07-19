package session_test

import (
	"testing"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func TestCloseExceptKeepsListedSessions(t *testing.T) {
	mgr := session.NewManager()
	dir := t.TempDir()

	a, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	c, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}

	closed, err := mgr.CloseExcept(map[string]struct{}{a.ID: {}, c.ID: {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0] != b.ID {
		t.Fatalf("closed = %v, want [%s]", closed, b.ID)
	}
	if _, ok := mgr.Get(a.ID); !ok {
		t.Fatal("expected session a to remain")
	}
	if _, ok := mgr.Get(c.ID); !ok {
		t.Fatal("expected session c to remain")
	}
	if _, ok := mgr.Get(b.ID); ok {
		t.Fatal("expected session b to be removed")
	}
}
