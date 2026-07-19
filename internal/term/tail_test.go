package term_test

import (
	"testing"

	"github.com/newtosh/tuile/internal/term"
)

func TestTailLinesNonEmpty(t *testing.T) {
	snap := term.ScreenSnapshot{
		Lines: []string{"", "alpha", "", "beta", "gamma"},
	}
	tail := term.TailLines(snap, 2)
	if len(tail) != 2 {
		t.Fatalf("tail len = %d, want 2", len(tail))
	}
	if tail[0].Text != "beta" || tail[1].Text != "gamma" {
		t.Fatalf("tail = %+v", tail)
	}
}

func TestContainsText(t *testing.T) {
	snap := term.ScreenSnapshot{Lines: []string{"hello", "world"}}
	if !term.ContainsText(snap, "wor") {
		t.Fatal("expected contains")
	}
	if term.ContainsText(snap, "missing") {
		t.Fatal("expected miss")
	}
}

func TestRegionLines(t *testing.T) {
	snap := term.ScreenSnapshot{Lines: []string{"a", "b", "c", "d"}}
	region := term.RegionLines(snap, 1, 2)
	if len(region) != 2 || region[0].Text != "b" || region[1].Text != "c" {
		t.Fatalf("region = %+v", region)
	}
}
