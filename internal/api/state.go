package api

import (
	"net/http"
	"strconv"

	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/term"
)

type sessionStateResponse struct {
	Version uint64 `json:"version"`
	Cols    int    `json:"cols"`
	Rows    int    `json:"rows"`
	Cursor  struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"cursor"`
	Text string `json:"text,omitempty"`
}

type sessionStateCompact struct {
	V  uint64 `json:"v"`
	C  int    `json:"c"`
	R  int    `json:"r"`
	CX int    `json:"cx"`
	CY int    `json:"cy"`
	T  string `json:"t,omitempty"`
}

func (s *Server) handleSessionState(w http.ResponseWriter, r *http.Request) {
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

	tailN, hasTail := parseTailQuery(r)
	if !hasTail {
		tailN = 20
	}
	format := outputFormat(r.URL.Query().Get("format"))
	if format == "json" {
		format = "text"
	}

	version := sess.ScreenVersion()
	snap := sess.ScreenSnapshot(false)
	text := term.JoinTailText(term.TailLines(snap, tailN))
	cols, rows := sess.PTY.Winsize()

	switch format {
	case "plain":
		writePlainText(w, version, text, nil)
	case "compact":
		writeJSON(w, http.StatusOK, sessionStateCompact{
			V:  version,
			C:  cols,
			R:  rows,
			CX: snap.Cursor.X,
			CY: snap.Cursor.Y,
			T:  text,
		})
	default:
		writeJSON(w, http.StatusOK, sessionStateResponse{
			Version: version,
			Cols:    cols,
			Rows:    rows,
			Cursor:  snap.Cursor,
			Text:    text,
		})
	}
}

type resizeResponse struct {
	Version uint64              `json:"version"`
	Cols    int                 `json:"cols"`
	Rows    int                 `json:"rows"`
	Screen  term.ScreenSnapshot `json:"screen,omitempty"`
}

type resizeCompact struct {
	V uint64 `json:"v"`
	C int    `json:"c"`
	R int    `json:"r"`
}

func writeResizeResponse(w http.ResponseWriter, format string, version uint64, cols, rows int, snap term.ScreenSnapshot) {
	switch format {
	case "plain":
		w.Header().Set(headerTuileVersion, strconv.FormatUint(version, 10))
		w.WriteHeader(http.StatusOK)
	case "compact":
		writeJSON(w, http.StatusOK, resizeCompact{V: version, C: cols, R: rows})
	case "text":
		writeJSON(w, http.StatusOK, resizeResponse{
			Version: version,
			Cols:    cols,
			Rows:    rows,
		})
	default:
		writeJSON(w, http.StatusOK, resizeResponse{
			Version: version,
			Cols:    cols,
			Rows:    rows,
			Screen:  snap,
		})
	}
}

func parseResizeFormat(r *http.Request) string {
	if f := r.URL.Query().Get("format"); f == "full" {
		return "full"
	}
	if f := r.URL.Query().Get("format"); f != "" {
		f = outputFormat(f)
		if f == "json" {
			return "full"
		}
		return f
	}
	return "compact"
}
