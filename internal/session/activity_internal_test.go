package session

import (
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/term"
)

func TestNotePTYOutputUpdatesActivity(t *testing.T) {
	e := term.New(40, 10)
	sess := &Session{
		Emulator:                 e,
		LastMeaningfulActivityAt: time.Now().Add(-time.Hour),
	}
	before := sess.LastMeaningfulActivityAt

	if _, err := e.Write([]byte("activity line\n")); err != nil {
		t.Fatal(err)
	}
	sess.notePTYOutput()

	if !sess.LastMeaningfulActivityAt.After(before) {
		t.Fatalf("activity time not bumped: before=%v after=%v", before, sess.LastMeaningfulActivityAt)
	}

	stamp := sess.LastMeaningfulActivityAt
	if _, err := e.Write([]byte("\x1b[A")); err != nil {
		t.Fatal(err)
	}
	sess.notePTYOutput()
	if !sess.LastMeaningfulActivityAt.Equal(stamp) {
		t.Fatal("cursor-only output should not bump activity")
	}
}
