package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/pty"
	"github.com/newtosh/tuile/internal/term"
)

// Session is one workspace-bound PTY with a shared emulator (R1).
type Session struct {
	ID                       string
	Workspace                string
	CLIName                  string
	CreatedAt                time.Time
	LastMeaningfulActivityAt time.Time
	PTY                      *pty.Master
	Emulator                 *term.Emulator
	Access                   *AccessGate
	screenVersion            uint64
	screenHist               *screenHistory

	activityMu      sync.Mutex
	tailFingerprint string

	outMu   sync.Mutex
	outSubs map[chan []byte]struct{}

	rawMu  sync.Mutex
	rawLog []byte
}

// Manager owns active sessions keyed by ID.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager creates an empty session manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Create starts a new session in workspace with optional dimension overrides.
func (m *Manager) Create(workspace string, opts config.Session) (*Session, error) {
	def := config.DefaultSession()
	if opts.Shell == "" {
		opts.Shell = def.Shell
	}
	if opts.Cols == 0 {
		opts.Cols = def.Cols
	}
	if opts.Rows == 0 {
		opts.Rows = def.Rows
	}

	master, err := pty.Start(workspace, opts)
	if err != nil {
		return nil, err
	}

	id, err := newSessionID()
	if err != nil {
		_ = master.Close()
		return nil, err
	}

	now := time.Now()
	sess := &Session{
		ID:                       id,
		Workspace:                master.WorkDir,
		CLIName:                  opts.CLIName,
		CreatedAt:                now,
		LastMeaningfulActivityAt: now,
		PTY:                      master,
		Emulator:                 term.New(master.Cols, master.Rows),
		Access:                   newAccessGate(),
		screenHist:               newScreenHistory(),
		outSubs:                  make(map[chan []byte]struct{}),
	}
	sess.tailFingerprint = term.TailFingerprint(sess.Emulator.Snapshot(), tailFingerprintLines)
	sess.screenHist.record(0, sess.Emulator.Snapshot())

	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	go pumpPTYToEmulator(sess)

	return sess, nil
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// SessionInfo is a read-only summary for discovery APIs.
type SessionInfo struct {
	SessionID                string    `json:"session_id"`
	Workspace                string    `json:"workspace"`
	CLI                      string    `json:"cli,omitempty"`
	Cols                     int       `json:"cols"`
	Rows                     int       `json:"rows"`
	Controller               string    `json:"controller"`
	CreatedAt                time.Time `json:"created_at"`
	LastMeaningfulActivityAt time.Time `json:"last_meaningful_activity_at"`
}

// List returns summaries of all active sessions sorted newest-first by creation time.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	items := make([]*Session, 0, len(m.sessions))
	for _, sess := range m.sessions {
		items = append(items, sess)
	}
	m.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	out := make([]SessionInfo, 0, len(items))
	for _, sess := range items {
		cols, rows := sess.PTY.Winsize()
		out = append(out, SessionInfo{
			SessionID:                sess.ID,
			Workspace:                sess.Workspace,
			CLI:                      sess.CLIName,
			Cols:                     cols,
			Rows:                     rows,
			Controller:               string(sess.Access.Controller()),
			CreatedAt:                sess.CreatedAt,
			LastMeaningfulActivityAt: sess.LastMeaningfulActivityAt,
		})
	}
	return out
}

// ResizeAgent updates PTY size when the agent controls the session (R2, R14).
func (m *Manager) ResizeAgent(id string, cols, rows int) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	if err := sess.Access.AgentResize(cols, rows); err != nil {
		return err
	}
	return m.applyResize(sess, cols, rows)
}

// ResizeHuman updates PTY size when the human controls the session (R14).
func (m *Manager) ResizeHuman(id string, cols, rows int) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	if err := sess.Access.HumanResize(); err != nil {
		return err
	}
	return m.applyResize(sess, cols, rows)
}

func (m *Manager) applyResize(sess *Session, cols, rows int) error {
	if err := sess.PTY.Resize(cols, rows); err != nil {
		return err
	}
	sess.Emulator.Resize(sess.PTY.Cols, sess.PTY.Rows)
	return nil
}

// Takeover grants human PTY control.
func (m *Manager) Takeover(id string) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	sess.Access.Takeover()
	return nil
}

// Release returns PTY control to the agent, restoring last agent size when set (KTD5).
func (m *Manager) Release(id string) error {
	sess, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	cols, rows, restore := sess.Access.Release()
	if restore {
		return m.applyResize(sess, cols, rows)
	}
	return nil
}

// CloseExcept terminates all sessions whose IDs are not in keep.
func (m *Manager) CloseExcept(keep map[string]struct{}) ([]string, error) {
	m.mu.RLock()
	toClose := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		if _, ok := keep[id]; ok {
			continue
		}
		toClose = append(toClose, id)
	}
	m.mu.RUnlock()

	closed := make([]string, 0, len(toClose))
	for _, id := range toClose {
		if err := m.Close(id); err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return closed, err
		}
		closed = append(closed, id)
	}
	return closed, nil
}

// Close terminates a session and removes it from the manager.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if !ok {
		return ErrNotFound
	}
	sess.closeOutputSubs()
	// Close the PTY first so pumpPTYToEmulator exits before the emulator is disposed.
	ptyErr := sess.PTY.Close()
	sess.Emulator.Close()
	return ptyErr
}

// ErrNotFound indicates an unknown session ID.
var ErrNotFound = errors.New("session not found")

func newSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("session id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func pumpPTYToEmulator(sess *Session) {
	buf := make([]byte, 32*1024)
	for {
		n, err := sess.PTY.File.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			sess.appendRawLog(chunk)
			sess.screenHist.record(sess.ScreenVersion(), sess.Emulator.Snapshot())
			_, _ = sess.Emulator.Write(chunk)
			sess.bumpScreenVersion()
			sess.notePTYOutput()
			sess.broadcastOutput(chunk)
		}
		if err != nil {
			return
		}
	}
}
