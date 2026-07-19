package term_test

import (
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/term"
)

func TestColoredTUIPanel(t *testing.T) {
	e := term.New(80, 24)
	t.Cleanup(e.Close)

	// Ratatui-style bordered panel with title (simplified ANSI).
	e.Write([]byte("\x1b[2J\x1b[H"))
	e.Write([]byte("\x1b[38;5;45m┌Title─────────────────────────────────────────────────────────────────────────┐\x1b[0m\r\n"))
	e.Write([]byte("\x1b[38;5;45m│\x1b[0m Body line                                                                  \x1b[38;5;45m│\x1b[0m\r\n"))
	e.Write([]byte("\x1b[38;5;45m└──────────────────────────────────────────────────────────────────────────────┘\x1b[0m\r\n"))

	snap := e.Snapshot()
	if snap.Lines[0] == "" {
		t.Fatal("expected non-empty first line after TUI frame write")
	}
	if !strings.Contains(snap.Lines[0], "Title") {
		t.Fatalf("expected title in line 0, got %q", snap.Lines[0])
	}
}

func TestResizeReflow(t *testing.T) {
	e := term.New(80, 24)
	t.Cleanup(e.Close)

	e.Write([]byte("abcdefghijklmnopqrstuvwxyz\r\n"))
	e.Resize(40, 24)

	cx, cy := e.Cursor()
	snap := e.Snapshot()
	if cy < 0 || cy >= snap.Rows {
		t.Fatalf("cursor y out of bounds after resize: %d", cy)
	}
	if cx < 0 || cx >= snap.Cols {
		t.Fatalf("cursor x out of bounds after resize: %d", cx)
	}
	// AE5-shaped: after resize, structured read reflects new geometry.
	if snap.Cols != 40 {
		t.Fatalf("expected cols 40 after resize, got %d", snap.Cols)
	}
}

func TestAltScreenSwitch(t *testing.T) {
	e := term.New(80, 24)
	t.Cleanup(e.Close)

	e.Write([]byte("\x1b[?1049h")) // alternate screen on
	if !e.AltBufferActive() {
		t.Fatal("expected alt buffer active after DECSET 1049")
	}
	e.Write([]byte("alt-content\r\n"))
	if !strings.Contains(e.String(), "alt-content") {
		t.Fatalf("expected alt content on screen, got %q", e.String())
	}

	e.Write([]byte("\x1b[?1049l")) // alternate screen off
	if e.AltBufferActive() {
		t.Fatal("expected normal buffer after DECRESET 1049")
	}
}

func TestMalformedUTF8DoesNotPanic(t *testing.T) {
	e := term.New(80, 24)
	t.Cleanup(e.Close)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emulator panicked on invalid UTF-8: %v", r)
		}
	}()

	_, _ = e.Write([]byte{0xff, 0xfe, 'x'})
	snap := e.Snapshot()
	if snap.Rows != 24 {
		t.Fatalf("expected emulator still functional, rows=%d", snap.Rows)
	}
}
