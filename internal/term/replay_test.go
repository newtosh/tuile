package term

import (
	"bytes"
	"testing"
)

func TestReplayANSIPreservesColor(t *testing.T) {
	e := New(80, 24)
	defer e.Close()

	_, _ = e.Write([]byte("\x1b[31m\x1b[44mcolor\x1b[0m plain"))
	replay := e.ReplayANSI()
	if !bytes.Contains(replay, []byte("\x1b[")) {
		t.Fatalf("expected ANSI SGR in replay, got %q", replay)
	}

	e2 := New(80, 24)
	defer e2.Close()
	_, _ = e2.Write(replay)
	if e2.String() != e.String() {
		t.Fatalf("replay round-trip mismatch:\nwant %q\ngot  %q", e.String(), e2.String())
	}
}
