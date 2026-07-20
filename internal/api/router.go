package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

// Server exposes HTTP and WebSocket endpoints for Tuile.
type Server struct {
	cfg     config.Server
	sess    *session.Manager
	tokens  *auth.Store
	boot    auth.BootstrapSecret
	mux     *http.ServeMux
}

// NewServer wires routes and dependencies.
func NewServer(cfg config.Server, mgr *session.Manager, tokens *auth.Store, boot auth.BootstrapSecret) *Server {
	s := &Server{
		cfg:    cfg,
		sess:   mgr,
		tokens: tokens,
		boot:   boot,
		mux:    http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/sessions", s.handleListSessions)
	s.mux.HandleFunc("POST /v1/sessions", s.handleCreateSession)
	s.mux.HandleFunc("POST /v1/sessions/prune", s.handlePruneSessions)
	s.mux.HandleFunc("DELETE /v1/sessions/{id}", s.handleDeleteSession)
	s.mux.HandleFunc("POST /v1/sessions/{id}/attach", s.handleAttachSession)
	s.mux.HandleFunc("GET /v1/sessions/{id}", s.handleGetSession)
	s.mux.HandleFunc("GET /v1/sessions/{id}/ws", s.handleSessionWS)
	s.registerHeadlessRoutes()
	s.registerExportRoutes()
	s.registerStaticRoutes()
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type createSessionRequest struct {
	Workspace string `json:"workspace"`
	CLI       string `json:"cli,omitempty"`
	Prompt    string `json:"prompt,omitempty"`
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	Token     string `json:"token"`
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeBootstrap(r) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap authorization required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": s.sess.List()})
}

type pruneSessionsRequest struct {
	Except []string `json:"except"`
}

func (s *Server) handlePruneSessions(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeBootstrap(r) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap authorization required")
		return
	}

	var req pruneSessionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	keep := make(map[string]struct{}, len(req.Except))
	for _, id := range req.Except {
		if id != "" {
			keep[id] = struct{}{}
		}
	}

	closed, err := s.pruneSessions(keep)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"closed": closed})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeBootstrap(r) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap authorization required")
		return
	}

	id := r.PathValue("id")
	if err := s.CloseSession(id); err != nil {
		if errors.Is(err, session.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pruneSessions(keep map[string]struct{}) ([]string, error) {
	closed, err := s.sess.CloseExcept(keep)
	if err != nil {
		return nil, err
	}
	for _, id := range closed {
		s.tokens.RevokeSession(id)
	}
	return closed, nil
}

type attachSessionResponse struct {
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	Token     string `json:"token"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

func (s *Server) handleAttachSession(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeBootstrap(r) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap authorization required")
		return
	}

	id := r.PathValue("id")
	if _, ok := s.sess.Get(id); !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	token, err := s.tokens.Mint(id, auth.HumanScopes, s.cfg.TokenTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "token mint failed")
		return
	}

	sess, _ := s.sess.Get(id)
	cols, rows := sess.PTY.Winsize()
	writeJSON(w, http.StatusOK, attachSessionResponse{
		SessionID: sess.ID,
		Workspace: sess.Workspace,
		Token:     token,
		Cols:      cols,
		Rows:      rows,
	})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeBootstrap(r) {
		writeJSONError(w, http.StatusUnauthorized, "bootstrap authorization required")
		return
	}

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	sessOpts := config.DefaultSession()
	if req.CLI != "" {
		var err error
		sessOpts, err = cli.SessionForCLI(req.CLI)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Prompt != "" {
		sessOpts.Args = append(sessOpts.Args, req.Prompt)
	}

	sess, err := s.sess.Create(req.Workspace, sessOpts)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := s.tokens.Mint(sess.ID, auth.SessionScopes, s.cfg.TokenTTL)
	if err != nil {
		_ = s.sess.Close(sess.ID)
		writeJSONError(w, http.StatusInternalServerError, "token mint failed")
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{
		SessionID: sess.ID,
		Workspace: sess.Workspace,
		Token:     token,
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
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

	cols, rows := sess.PTY.Winsize()
	writeJSON(w, http.StatusOK, session.SessionInfo{
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

func (s *Server) handleSessionWS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !OriginAllowed(origin, s.cfg.AllowedOrigins) {
		// AE2: reject before WS accept / PTY data.
		writeJSONError(w, http.StatusForbidden, "origin not allowed")
		return
	}

	token := bearerToken(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	claims, err := s.tokens.Validate(token)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if !auth.HasScope(claims, auth.ScopeHumanView) {
		writeJSONError(w, http.StatusForbidden, "insufficient scope")
		return
	}

	id := r.PathValue("id")
	if claims.SessionID != id {
		writeJSONError(w, http.StatusForbidden, "token not valid for this session")
		return
	}
	if _, ok := s.sess.Get(id); !ok {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	sess, _ := s.sess.Get(id)
	s.serveSessionWS(w, r, sess, claims)
}

func (s *Server) authorizeBootstrap(r *http.Request) bool {
	got := bearerToken(r)
	if got == "" {
		return false
	}
	return s.boot.ConstantTimeEqual(got)
}

func (s *Server) authorizeSession(r *http.Request, need auth.Scope) (auth.Claims, error) {
	token := bearerToken(r)
	if token == "" {
		return auth.Claims{}, auth.ErrUnauthorized
	}
	claims, err := s.tokens.Validate(token)
	if err != nil {
		return auth.Claims{}, err
	}
	if !auth.HasScope(claims, need) {
		return auth.Claims{}, auth.ErrForbidden
	}
	return claims, nil
}

func (s *Server) authorizeSessionAny(r *http.Request, needs ...auth.Scope) (auth.Claims, error) {
	token := bearerToken(r)
	if token == "" {
		return auth.Claims{}, auth.ErrUnauthorized
	}
	claims, err := s.tokens.Validate(token)
	if err != nil {
		return auth.Claims{}, err
	}
	for _, need := range needs {
		if auth.HasScope(claims, need) {
			return claims, nil
		}
	}
	return auth.Claims{}, auth.ErrForbidden
}

func (s *Server) authorizeAgentInput(r *http.Request, sessionID string) bool {
	if s.authorizeBootstrap(r) {
		_, ok := s.sess.Get(sessionID)
		return ok
	}
	claims, err := s.authorizeSession(r, auth.ScopeAgentWrite)
	if err != nil {
		return false
	}
	return claims.SessionID == sessionID
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, err.Error())
	default:
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// CloseSession revokes tokens and stops the PTY session.
func (s *Server) CloseSession(id string) error {
	s.tokens.RevokeSession(id)
	return s.sess.Close(id)
}

// ListenAndServe starts the HTTP server with optional TLS (KTD4).
func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:    s.cfg.Listen,
		Handler: s.Handler(),
	}
	if s.cfg.TLSCert != "" && s.cfg.TLSKey != "" {
		return srv.ListenAndServeTLS(s.cfg.TLSCert, s.cfg.TLSKey)
	}
	return srv.ListenAndServe()
}
