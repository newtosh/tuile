package session

import (
	"errors"
	"sync"
)

// Controller identifies who may write to the PTY and resize it (R14, KTD5).
type Controller string

const (
	ControllerAgent Controller = "agent"
	ControllerHuman Controller = "human"
)

// Concurrency errors for API mapping.
var (
	ErrHumanControls = errors.New("human controls session")
	ErrObserveOnly   = errors.New("human is observe-only until takeover")
	ErrAgentControls = errors.New("agent controls session")
)

// AccessGate tracks concurrent agent/human access for one session.
type AccessGate struct {
	mu            sync.Mutex
	controller    Controller
	lastAgentCols int
	lastAgentRows int
	hasAgentSize  bool
}

func newAccessGate() *AccessGate {
	return &AccessGate{controller: ControllerAgent}
}

// Controller returns the active PTY controller.
func (g *AccessGate) Controller() Controller {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.controller
}

// Takeover grants human PTY control.
func (g *AccessGate) Takeover() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.controller = ControllerHuman
}

// Release returns control to the agent and reports whether agent size should be restored.
func (g *AccessGate) Release() (cols, rows int, restore bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.controller = ControllerAgent
	if g.hasAgentSize {
		return g.lastAgentCols, g.lastAgentRows, true
	}
	return 0, 0, false
}

// AgentWrite checks whether agent input is allowed.
func (g *AccessGate) AgentWrite() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.controller == ControllerHuman {
		return ErrHumanControls
	}
	return nil
}

// HumanWrite checks whether human input is allowed.
func (g *AccessGate) HumanWrite() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.controller != ControllerHuman {
		return ErrObserveOnly
	}
	return nil
}

// AgentResize records agent-requested dimensions and checks authority.
func (g *AccessGate) AgentResize(cols, rows int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.controller == ControllerHuman {
		return ErrHumanControls
	}
	g.lastAgentCols = cols
	g.lastAgentRows = rows
	g.hasAgentSize = true
	return nil
}

// HumanResize checks human resize authority.
func (g *AccessGate) HumanResize() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.controller != ControllerHuman {
		return ErrObserveOnly
	}
	return nil
}
