package config

import "os"

// Defaults for new PTY sessions. Agent TUIs render best at 120×36; shell sessions
// use the same grid so observe mode and sidebar dimensions stay aligned.
const (
	DefaultCols = 120
	DefaultRows = 36
	MinCols     = 2
	MinRows     = 2
	MaxCols     = 500
	MaxRows     = 500
)

// Session holds per-session creation options.
type Session struct {
	Shell   string
	Command string
	Args    []string
	Cols    int
	Rows    int
	CLIName string
}

// NormalizeDimensions clamps terminal size to supported bounds.
func NormalizeDimensions(cols, rows int) (int, int) {
	if cols < MinCols {
		cols = MinCols
	}
	if rows < MinRows {
		rows = MinRows
	}
	if cols > MaxCols {
		cols = MaxCols
	}
	if rows > MaxRows {
		rows = MaxRows
	}
	return cols, rows
}

// DefaultSession returns session defaults with normalized dimensions.
func DefaultSession() Session {
	cols, rows := NormalizeDimensions(DefaultCols, DefaultRows)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return Session{
		Shell: shell,
		Cols:  cols,
		Rows:  rows,
	}
}
