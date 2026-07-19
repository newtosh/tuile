package term

import (
	"fmt"
	"sync"

	xterm "github.com/gitpod-io/xterm-go"
)

// Emulator wraps xterm-go for Tuile's shared PTY parsing layer.
type Emulator struct {
	mu   sync.RWMutex
	term *xterm.Terminal
}

// New creates a terminal emulator with the given dimensions.
func New(cols, rows int) *Emulator {
	return &Emulator{
		term: xterm.New(xterm.WithCols(cols), xterm.WithRows(rows), xterm.WithScrollback(1000)),
	}
}

// Write feeds PTY output bytes into the emulator.
func (e *Emulator) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.term == nil {
		return len(p), nil
	}
	return e.term.Write(p)
}

// Resize updates terminal dimensions and reflows buffer state.
func (e *Emulator) Resize(cols, rows int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.term == nil {
		return
	}
	e.term.Resize(cols, rows)
}

// Cols returns terminal width in cells.
func (e *Emulator) Cols() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.Cols()
}

// Rows returns terminal height in cells.
func (e *Emulator) Rows() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.Rows()
}

// Cursor returns zero-based column and row.
func (e *Emulator) Cursor() (x, y int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.CursorX(), e.term.CursorY()
}

// Line returns visible text for row y (without trailing spaces trimmed by caller).
func (e *Emulator) Line(y int) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.GetLine(y)
}

// String returns the full visible screen as plain text lines.
func (e *Emulator) String() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.String()
}

// AltBufferActive reports whether the alternate screen is active.
func (e *Emulator) AltBufferActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term.IsAltBufferActive()
}

// Snapshot captures structured screen state for agent reads.
func (e *Emulator) Snapshot() ScreenSnapshot {
	return e.SnapshotWith(SnapshotOptions{})
}

// SnapshotWith captures structured screen state with optional per-cell detail.
func (e *Emulator) SnapshotWith(opts SnapshotOptions) ScreenSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.term == nil {
		return ScreenSnapshot{}
	}
	return snapshotFromTerminal(e.term, opts)
}

// ReplayANSI serializes the buffer as escape sequences for browser replay (U6).
func (e *Emulator) ReplayANSI() []byte {
	e.mu.RLock()
	defer e.mu.RUnlock()
	sa := xterm.NewSerializeAddon(e.term)
	return sa.Serialize(&xterm.SerializeOptions{ExcludeModes: true})
}

// Close releases emulator resources.
func (e *Emulator) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.term != nil {
		e.term.Dispose()
		e.term = nil
	}
}

// ParseError wraps invalid emulator input for callers.
type ParseError struct {
	Detail string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("terminal parse: %s", e.Detail)
}
