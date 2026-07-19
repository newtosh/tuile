package session

import (
	"testing"

	"github.com/newtosh/tuile/internal/term"
)

func TestScreenDiffSinceMatchesCachedVersion(t *testing.T) {
	h := newScreenHistory()
	h.record(1, term.ScreenSnapshot{Lines: []string{"a", "b"}})

	diff, ok := h.DiffSince(1, term.ScreenSnapshot{Lines: []string{"a", "c"}}, 2)
	if !ok {
		t.Fatal("expected diff")
	}
	if len(diff.ChangedLines) != 1 || diff.ChangedLines[0].Text != "c" {
		t.Fatalf("diff = %+v", diff.ChangedLines)
	}
}

func TestScreenDiffSinceMissesUnknownVersion(t *testing.T) {
	h := newScreenHistory()
	h.record(1, term.ScreenSnapshot{Lines: []string{"a"}})

	if _, ok := h.DiffSince(0, term.ScreenSnapshot{Lines: []string{"b"}}, 2); ok {
		t.Fatal("expected no diff for unknown since")
	}
}
