package session

import (
	"sync"

	"github.com/newtosh/tuile/internal/term"
)

const screenHistoryMax = 32

type screenHistory struct {
	mu      sync.Mutex
	byVer   map[uint64]term.ScreenSnapshot
	order   []uint64
}

func newScreenHistory() *screenHistory {
	return &screenHistory{byVer: make(map[uint64]term.ScreenSnapshot)}
}

func (h *screenHistory) record(version uint64, snap term.ScreenSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.byVer[version]; exists {
		return
	}
	h.byVer[version] = snap
	h.order = append(h.order, version)
	if len(h.order) > screenHistoryMax {
		old := h.order[0]
		h.order = h.order[1:]
		delete(h.byVer, old)
	}
}

// DiffSince returns a line diff when since matches a cached screen version.
func (h *screenHistory) DiffSince(since uint64, current term.ScreenSnapshot, currentVersion uint64) (term.ScreenDiff, bool) {
	h.mu.Lock()
	before, ok := h.byVer[since]
	h.mu.Unlock()
	if !ok || currentVersion <= since {
		return term.ScreenDiff{}, false
	}
	return term.DiffSnapshots(before, current), true
}
