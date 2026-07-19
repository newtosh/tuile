package session_test

import (
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func TestListSortedByCreatedAtDesc(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager()

	first, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Close(first.ID)
		_ = mgr.Close(second.ID)
	})

	list := mgr.List()
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}
	if list[0].SessionID != second.ID {
		t.Fatalf("newest session first: got %s, want %s", list[0].SessionID, second.ID)
	}
	if list[0].CreatedAt.IsZero() || list[0].LastMeaningfulActivityAt.IsZero() {
		t.Fatal("expected timestamp fields on list items")
	}
}
