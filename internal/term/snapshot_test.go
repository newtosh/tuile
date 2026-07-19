package term_test

import (
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/term"
)

func TestSnapshotIncludesScrollRegion(t *testing.T) {
	e := term.New(80, 24)
	t.Cleanup(e.Close)

	e.Write([]byte("\x1b[2;10r")) // scroll region rows 2-10

	snap := e.Snapshot()
	if snap.Scroll == nil {
		t.Fatal("expected scroll region metadata")
	}
	if snap.Scroll.Top != 1 || snap.Scroll.Bottom != 9 {
		t.Fatalf("scroll = %+v, want top=1 bottom=9", snap.Scroll)
	}
}

func TestSnapshotCellsIncludeColorAndAttrs(t *testing.T) {
	e := term.New(40, 5)
	t.Cleanup(e.Close)

	e.Write([]byte("\x1b[1;31mB\x1b[0m\r\n"))

	snap := e.SnapshotWith(term.SnapshotOptions{IncludeCells: true})
	if len(snap.Grid) == 0 || len(snap.Grid[0].Cells) == 0 {
		t.Fatalf("expected grid cells, got %+v", snap.Grid)
	}
	cell := snap.Grid[0].Cells[0]
	if cell.Ch != "B" {
		t.Fatalf("cell ch = %q, want B", cell.Ch)
	}
	if !strings.HasPrefix(cell.Fg, "p") && !strings.HasPrefix(cell.Fg, "#") {
		t.Fatalf("expected fg color, got %q", cell.Fg)
	}
	if len(cell.Attr) == 0 || cell.Attr[0] != "bold" {
		t.Fatalf("expected bold attr, got %v", cell.Attr)
	}
}

func TestDiffSnapshotsChangedLines(t *testing.T) {
	before := term.ScreenSnapshot{
		Rows:  2,
		Lines: []string{"hello", "world"},
	}
	after := term.ScreenSnapshot{
		Rows:  2,
		Lines: []string{"hello", "tuile"},
		Cursor: struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{X: 3, Y: 1},
	}

	diff := term.DiffSnapshots(before, after)
	if len(diff.ChangedLines) != 1 || diff.ChangedLines[0].Y != 1 || diff.ChangedLines[0].Text != "tuile" {
		t.Fatalf("diff lines = %+v", diff.ChangedLines)
	}
	if diff.Cursor == nil || diff.Cursor.Y != 1 {
		t.Fatalf("cursor diff = %+v", diff.Cursor)
	}
}
