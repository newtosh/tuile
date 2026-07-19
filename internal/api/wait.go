package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/term"
)

type waitRequest struct {
	Contains  string `json:"contains,omitempty"`
	Since     uint64 `json:"since,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Tail      int    `json:"tail,omitempty"`
	Format    string `json:"format,omitempty"` // plain, text, compact, or json (default json)
}

type waitResponse struct {
	Version uint64 `json:"version"`
	Matched bool   `json:"matched"`
	Text    string `json:"text,omitempty"`
}

func (s *Server) handleSessionWait(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSessionAny(r, auth.ScopeAgentRead, auth.ScopeHumanView)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}

	sess, ok := s.sess.Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	var req waitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	timeout := 15 * time.Second
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
		if timeout > 60*time.Second {
			timeout = 60 * time.Second
		}
	}
	tailN := req.Tail
	if tailN <= 0 {
		tailN = 20
	}
	if tailN > 200 {
		tailN = 200
	}
	outFormat := parseWaitFormat(req, r.URL.Query().Get("format"))

	deadline := time.Now().Add(timeout)
	since := req.Since
	ch := sess.SubscribeOutput(32)
	defer sess.UnsubscribeOutput(ch)
	initialVersion := sess.ScreenVersion()

	for {
		cur, ok := s.sess.Get(id)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}
		version := cur.ScreenVersion()
		snap := cur.ScreenSnapshot(false)

		matched := false
		switch {
		case req.Contains != "":
			matched = term.ContainsText(snap, req.Contains)
		case since > 0:
			matched = version > since
		default:
			matched = version > initialVersion
		}

		if matched {
			text := term.JoinTailText(term.TailLines(snap, tailN))
			writeAgentWait(w, outFormat, version, text, true)
			return
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			text := term.JoinTailText(term.TailLines(snap, tailN))
			writeAgentWait(w, outFormat, version, text, false)
			return
		}

		select {
		case <-r.Context().Done():
			return
		case _, open := <-ch:
			if !open {
				writeJSONError(w, http.StatusGone, "session ended")
				return
			}
		case <-time.After(minDuration(remaining, 200*time.Millisecond)):
		}
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
