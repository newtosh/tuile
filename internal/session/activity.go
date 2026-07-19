package session

import (
	"time"

	"github.com/newtosh/tuile/internal/term"
)

const tailFingerprintLines = 5

func (s *Session) notePTYOutput() {
	fp := term.TailFingerprint(s.Emulator.Snapshot(), tailFingerprintLines)
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	if fp == s.tailFingerprint {
		return
	}
	s.tailFingerprint = fp
	s.LastMeaningfulActivityAt = time.Now()
}
