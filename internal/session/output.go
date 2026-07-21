package session

const rawReplayMax = 512 * 1024

// alignRawReplayStart drops bytes left by ring-buffer truncation mid-escape
// (e.g. "32;2H" without the leading ESC) so browser replay starts cleanly.
func alignRawReplayStart(p []byte) []byte {
	if len(p) == 0 || p[0] == '\x1b' || p[0] == '\n' {
		return p
	}
	if i := csiTailEnd(p); i > 0 {
		return p[i:]
	}
	return p
}

func csiTailEnd(p []byte) int {
	i := 0
	for i < len(p) && (p[i] >= '0' && p[i] <= '9' || p[i] == ';') {
		i++
	}
	if i > 0 && i < len(p) && p[i] >= 0x40 && p[i] <= 0x7e {
		return i + 1
	}
	return 0
}

func trimRawLog(p []byte) []byte {
	if len(p) <= rawReplayMax {
		return p
	}
	return p[len(p)-rawReplayMax:]
}

// appendRawLog retains recent PTY bytes for faithful browser color replay.
func (s *Session) appendRawLog(p []byte) {
	if len(p) == 0 {
		return
	}
	s.rawMu.Lock()
	defer s.rawMu.Unlock()
	s.rawLog = append(s.rawLog, p...)
	if len(s.rawLog) > rawReplayMax {
		s.rawLog = trimRawLog(s.rawLog)
	}
}

// ReplayRaw returns a copy of recent raw PTY output for browser replay.
func (s *Session) ReplayRaw() []byte {
	s.rawMu.Lock()
	defer s.rawMu.Unlock()
	return alignRawReplayStart(append([]byte(nil), s.rawLog...))
}

// ReplayForBrowser prefers raw PTY bytes (preserves truecolor) over emulator serialize.
func (s *Session) ReplayForBrowser() []byte {
	if raw := s.ReplayRaw(); len(raw) > 0 {
		return raw
	}
	return s.Emulator.ReplayANSI()
}

// SubscribeOutput registers for raw PTY output broadcasts (browser WS).
func (s *Session) SubscribeOutput(buf int) chan []byte {
	if buf <= 0 {
		buf = 64
	}
	ch := make(chan []byte, buf)
	s.outMu.Lock()
	if s.outSubs == nil {
		s.outSubs = make(map[chan []byte]struct{})
	}
	s.outSubs[ch] = struct{}{}
	s.outMu.Unlock()
	return ch
}

// UnsubscribeOutput removes a PTY output subscriber.
func (s *Session) UnsubscribeOutput(ch chan []byte) {
	s.outMu.Lock()
	delete(s.outSubs, ch)
	s.outMu.Unlock()
}

func (s *Session) outputSubscriberCount() int {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	return len(s.outSubs)
}

func (s *Session) broadcastOutput(p []byte) {
	s.outMu.Lock()
	subs := make([]chan []byte, 0, len(s.outSubs))
	for ch := range s.outSubs {
		subs = append(subs, ch)
	}
	s.outMu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- p:
		default:
		}
	}
}

func (s *Session) closeOutputSubs() {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	for ch := range s.outSubs {
		close(ch)
	}
	s.outSubs = nil
}
