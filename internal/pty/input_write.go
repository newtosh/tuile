package pty

import (
	"io"
	"time"
)

const (
	BracketedPasteStart = "\x1b[200~"
	BracketedPasteEnd   = "\x1b[201~"
)

// InputStrategy selects how payload bytes are delivered to a PTY.
type InputStrategy int

const (
	// StrategyStandard writes payload bytes directly, then optional submit.
	StrategyStandard InputStrategy = iota
	// StrategyBracketedPaste wraps payload in xterm bracketed paste so Ink TUIs
	// (e.g. Cursor Agent) accept text without treating embedded newlines as soft breaks.
	StrategyBracketedPaste
)

// WriteOpts controls PTY delivery semantics.
type WriteOpts struct {
	Strategy    InputStrategy
	SubmitDelay time.Duration
}

// WritePreparedInput writes payload and optional submit as separate PTY writes.
func WritePreparedInput(w io.Writer, in PTYInput, opts WriteOpts) error {
	switch {
	case opts.Strategy == StrategyBracketedPaste && len(in.Payload) > 0:
		if _, err := io.WriteString(w, BracketedPasteStart); err != nil {
			return err
		}
		if _, err := w.Write(in.Payload); err != nil {
			return err
		}
		if _, err := io.WriteString(w, BracketedPasteEnd); err != nil {
			return err
		}
	case len(in.Payload) > 0:
		if _, err := w.Write(in.Payload); err != nil {
			return err
		}
	}
	if in.Submit {
		if opts.SubmitDelay > 0 {
			time.Sleep(opts.SubmitDelay)
		}
		_, err := w.Write([]byte{'\r'})
		return err
	}
	return nil
}
