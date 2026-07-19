package session

import "testing"

func TestReplayRawPreservesPTYBytes(t *testing.T) {
	s := &Session{}
	s.appendRawLog([]byte("\x1b[38;2;255;153;51mcolor\x1b[0m"))
	raw := s.ReplayRaw()
	if string(raw) != "\x1b[38;2;255;153;51mcolor\x1b[0m" {
		t.Fatalf("raw = %q", raw)
	}
}

func TestReplayRawRingBuffer(t *testing.T) {
	s := &Session{}
	big := make([]byte, rawReplayMax+100)
	for i := range big {
		big[i] = byte(i % 256)
	}
	s.appendRawLog(big)
	raw := s.ReplayRaw()
	if len(raw) != rawReplayMax {
		t.Fatalf("len = %d, want %d", len(raw), rawReplayMax)
	}
	if raw[0] != big[100] {
		t.Fatalf("ring trim mismatch")
	}
}

func TestAlignRawReplayStartDropsTruncatedEscape(t *testing.T) {
	raw := alignRawReplayStart([]byte("32;2H\x1b[0m\x1b[2Jhello"))
	if string(raw) != "\x1b[0m\x1b[2Jhello" {
		t.Fatalf("aligned = %q", raw)
	}
}

func TestReplayRawAlignsLeadingFragment(t *testing.T) {
	s := &Session{}
	s.rawLog = []byte("32;2H\x1b[0mcolor\x1b[0m")
	raw := s.ReplayRaw()
	if string(raw) != "\x1b[0mcolor\x1b[0m" {
		t.Fatalf("raw = %q", raw)
	}
}
