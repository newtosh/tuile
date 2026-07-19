package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/coder/websocket"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/session"
)

func (s *Server) registerConcurrencyRoutes() {
	s.mux.HandleFunc("POST /v1/sessions/{id}/takeover", s.handleSessionTakeover)
	s.mux.HandleFunc("POST /v1/sessions/{id}/release", s.handleSessionRelease)
	s.mux.HandleFunc("POST /v1/sessions/{id}/human/resize", s.handleSessionHumanResize)
}

func (s *Server) handleSessionTakeover(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSession(r, auth.ScopeHumanControl)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}
	if err := s.sess.Takeover(id); err != nil {
		if err == session.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"controller": string(session.ControllerHuman)})
}

func (s *Server) handleSessionRelease(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSession(r, auth.ScopeHumanControl)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}
	if err := s.sess.Release(id); err != nil {
		if err == session.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"controller": string(session.ControllerAgent)})
}

func (s *Server) handleSessionHumanResize(w http.ResponseWriter, r *http.Request) {
	claims, err := s.authorizeSession(r, auth.ScopeHumanControl)
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

	if err := s.sess.ResizeHuman(id, req.Cols, req.Rows); err != nil {
		writeConcurrencyError(w, err)
		return
	}

	sess, ok := s.sess.Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, screenResponse{
		Version: sess.ScreenVersion(),
		Screen:  sess.Emulator.Snapshot(),
	})
}

func (s *Server) serveSessionWS(w http.ResponseWriter, r *http.Request, sess *session.Session, claims auth.Claims) {
	opts := &websocket.AcceptOptions{
		OriginPatterns: s.cfg.AllowedOrigins,
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	ctx := r.Context()

	// Send faithful PTY replay (truecolor) before live tail.
	if replay := sess.ReplayForBrowser(); len(replay) > 0 {
		if err := conn.Write(ctx, websocket.MessageBinary, replay); err != nil {
			return
		}
	}

	out := sess.SubscribeOutput(128)
	defer sess.UnsubscribeOutput(out)

	errCh := make(chan error, 2)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case chunk, ok := <-out:
				if !ok {
					errCh <- nil
					return
				}
				if err := conn.Write(ctx, websocket.MessageBinary, chunk); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	go func() {
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				errCh <- err
				return
			}
			if typ != websocket.MessageBinary && typ != websocket.MessageText {
				continue
			}
			if !auth.HasScope(claims, auth.ScopeHumanControl) {
				continue
			}
			if err := s.sess.WriteHumanInput(sess.ID, data, false, false); err != nil {
				if errors.Is(err, session.ErrObserveOnly) {
					continue
				}
				errCh <- err
				return
			}
		}
	}()

	<-errCh
}

func writeConcurrencyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, session.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "session not found")
	case errors.Is(err, session.ErrHumanControls):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "human controls session",
			"code":  "human_controls",
		})
	case errors.Is(err, session.ErrObserveOnly):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "human is observe-only until takeover",
			"code":  "observe_only",
		})
	default:
		writeJSONError(w, http.StatusBadRequest, err.Error())
	}
}
