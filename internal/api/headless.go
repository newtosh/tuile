package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/term"
)

func (s *Server) registerHeadlessRoutes() {
	s.mux.HandleFunc("POST /v1/sessions/{id}/input", s.handleSessionInput)
	s.mux.HandleFunc("GET /v1/sessions/{id}/screen", s.handleSessionScreen)
	s.mux.HandleFunc("GET /v1/sessions/{id}/state", s.handleSessionState)
	s.mux.HandleFunc("POST /v1/sessions/{id}/wait", s.handleSessionWait)
	s.mux.HandleFunc("GET /v1/sessions/{id}/stream", s.handleSessionStream)
	s.mux.HandleFunc("POST /v1/sessions/{id}/resize", s.handleSessionResize)
	s.registerConcurrencyRoutes()
}

type inputRequest struct {
	Input  string `json:"input"`
	Raw    bool   `json:"raw,omitempty"`
	Submit bool   `json:"submit,omitempty"`
}

func (s *Server) handleSessionInput(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.authorizeAgentInput(r, id) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap or agent authorization required")
		return
	}

	var req inputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Input == "" && !req.Submit {
		writeJSONError(w, http.StatusBadRequest, "input required")
		return
	}

	if err := s.sess.WriteAgentInput(id, []byte(req.Input), req.Raw, req.Submit); err != nil {
		writeConcurrencyError(w, err)
		return
	}

	sess, ok := s.sess.Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	writeInputAck(w, sess.ScreenVersion())
}

type screenResponse struct {
	Version   uint64              `json:"version"`
	Since     uint64              `json:"since,omitempty"`
	Screen    term.ScreenSnapshot `json:"screen,omitempty"`
	Diff      *term.ScreenDiff    `json:"diff,omitempty"`
	Text      string              `json:"text,omitempty"`
	Tail      []term.TailLine     `json:"tail,omitempty"`
	ReplayB64 string              `json:"replay_b64,omitempty"`
}

func parseTailQuery(r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("tail")
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	if n > 200 {
		n = 200
	}
	return n, true
}

func parseRegionQuery(r *http.Request) (y1, y2 int, ok bool) {
	raw := r.URL.Query().Get("region")
	if raw == "" {
		return 0, 0, false
	}
	sep := ":"
	if strings.Contains(raw, "-") && !strings.Contains(raw, ":") {
		sep = "-"
	}
	parts := strings.SplitN(raw, sep, 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	y1, err1 := strconv.Atoi(parts[0])
	y2, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return y1, y2, true
}

func applyCompactScreen(resp *screenResponse, snap term.ScreenSnapshot, format string, tailN int, hasTail bool, region bool, y1, y2 int) {
	switch format {
	case "text", "plain", "compact":
		resp.Text = screenText(snap, format, tailN, hasTail, region, y1, y2)
		return
	}
	meta := term.ScreenSnapshot{
		Cols:   snap.Cols,
		Rows:   snap.Rows,
		Cursor: snap.Cursor,
		Alt:    snap.Alt,
		Scroll: snap.Scroll,
	}
	switch {
	case hasTail:
		resp.Tail = term.TailLines(snap, tailN)
		resp.Screen = meta
	case region:
		resp.Tail = term.RegionLines(snap, y1, y2)
		resp.Screen = meta
	default:
		resp.Screen = snap
	}
}

func (s *Server) handleSessionScreen(w http.ResponseWriter, r *http.Request) {
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

	version := sess.ScreenVersion()
	includeCells := r.URL.Query().Get("detail") == "cells"
	format := outputFormat(r.URL.Query().Get("format"))
	tailN, hasTail := parseTailQuery(r)
	if (format == "text" || format == "plain" || format == "compact") && !hasTail && r.URL.Query().Get("region") == "" {
		hasTail = true
		tailN = 20
	}
	y1, y2, hasRegion := parseRegionQuery(r)

	sinceRaw := r.URL.Query().Get("since")
	if sinceRaw != "" {
		prev, err := strconv.ParseUint(sinceRaw, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid since version")
			return
		}
		if prev >= version {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if format == "plain" || format == "text" || format == "compact" {
			snap := sess.ScreenSnapshot(false)
			if !hasTail {
				hasTail = true
				tailN = 20
			}
			writeAgentScreen(w, format, version, screenText(snap, format, tailN, hasTail, hasRegion, y1, y2))
			return
		}
		if diff, ok := sess.ScreenDiffSince(prev); ok && (len(diff.ChangedLines) > 0 || diff.Cursor != nil || diff.Scroll != nil) {
			writeJSON(w, http.StatusOK, screenResponse{
				Version: version,
				Since:   prev,
				Diff:    &diff,
			})
			return
		}
	}

	snap := sess.ScreenSnapshot(includeCells)
	if format == "plain" || format == "text" || format == "compact" {
		writeAgentScreen(w, format, version, screenText(snap, format, tailN, hasTail, hasRegion, y1, y2))
		return
	}

	resp := screenResponse{Version: version}
	applyCompactScreen(&resp, snap, format, tailN, hasTail, hasRegion, y1, y2)
	if r.URL.Query().Get("replay") == "1" {
		resp.ReplayB64 = base64.StdEncoding.EncodeToString(sess.ReplayForBrowser())
	}
	writeJSON(w, http.StatusOK, resp)
}

type resizeRequest struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

func (s *Server) handleSessionResize(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSession(r, auth.ScopeAgentWrite)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}

	var req resizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Cols == 0 || req.Rows == 0 {
		writeJSONError(w, http.StatusBadRequest, "cols and rows required")
		return
	}

	if err := s.sess.ResizeAgent(id, req.Cols, req.Rows); err != nil {
		writeConcurrencyError(w, err)
		return
	}

	sess, ok := s.sess.Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	cols, rows := sess.PTY.Winsize()
	writeResizeResponse(w, parseResizeFormat(r), sess.ScreenVersion(), cols, rows, sess.ScreenSnapshot(false))
}

func (s *Server) handleSessionStream(w http.ResponseWriter, r *http.Request) {
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := sess.SubscribeOutput(64)
	defer sess.UnsubscribeOutput(ch)

	fmt.Fprintf(w, "event: version\ndata: %d\n\n", sess.ScreenVersion())
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case chunk, open := <-ch:
			if !open {
				return
			}
			payload, _ := json.Marshal(map[string]any{
				"version": sess.ScreenVersion(),
				"b64":     base64.StdEncoding.EncodeToString(chunk),
			})
			fmt.Fprintf(w, "event: output\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}
