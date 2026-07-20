package session

import (
	"sync/atomic"
	"time"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/pty"
	"github.com/newtosh/tuile/internal/term"
)

// WriteAgentInput sends agent bytes when the agent controls the session.
func (m *Manager) WriteAgentInput(id string, data []byte, raw bool, submit bool) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	in := pty.PreparePTYInput(data, raw, submit)
	if len(in.Payload) == 0 && !in.Submit {
		return nil
	}
	if err := sess.Access.AgentWrite(); err != nil {
		return err
	}
	return pty.WritePreparedInput(sess.PTY.File, in, writeOptsForSession(sess))
}

// WritePTYResponse forwards auto-generated terminal replies to the PTY.
// These bypass observe-only gating so apps like Neovim can query colors in observe mode.
func (m *Manager) WritePTYResponse(id string, data []byte) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	if len(data) == 0 {
		return nil
	}
	return pty.WritePreparedInput(sess.PTY.File, pty.PTYInput{Payload: data}, writeOptsForSession(sess))
}

// WriteHumanInput sends human bytes when the human has taken over.
func (m *Manager) WriteHumanInput(id string, data []byte, raw bool, submit bool) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	in := pty.PreparePTYInput(data, raw, submit)
	if len(in.Payload) == 0 && !in.Submit {
		return nil
	}
	if err := sess.Access.HumanWrite(); err != nil {
		return err
	}
	return pty.WritePreparedInput(sess.PTY.File, in, writeOptsForSession(sess))
}

func writeOptsForSession(sess *Session) pty.WriteOpts {
	opts := pty.WriteOpts{Strategy: cli.InputStrategy(sess.CLIName)}
	if sess.CLIName == cli.CursorCLI || sess.CLIName == cli.CopilotCLI {
		opts.SubmitDelay = 20 * time.Millisecond
	}
	return opts
}

// ScreenVersion returns a monotonic counter bumped on PTY output.
func (s *Session) ScreenVersion() uint64 {
	return atomic.LoadUint64(&s.screenVersion)
}

func (s *Session) bumpScreenVersion() {
	atomic.AddUint64(&s.screenVersion, 1)
}

// ScreenDiffSince returns an incremental diff when the client's since version is cached.
func (s *Session) ScreenDiffSince(since uint64) (term.ScreenDiff, bool) {
	current := s.Emulator.Snapshot()
	return s.screenHist.DiffSince(since, current, s.ScreenVersion())
}

// ScreenSnapshot captures the current structured screen, optionally with per-cell detail.
func (s *Session) ScreenSnapshot(includeCells bool) term.ScreenSnapshot {
	return s.Emulator.SnapshotWith(term.SnapshotOptions{IncludeCells: includeCells})
}
