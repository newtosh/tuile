package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newtosh/tuile/internal/api"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func newTestServer(t *testing.T, origins []string) (*api.Server, auth.BootstrapSecret) {
	t.Helper()
	boot, err := auth.NewBootstrapSecret()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultServer()
	cfg.AllowedOrigins = origins
	srv := api.NewServer(cfg, session.NewManager(), auth.NewStore(), boot)
	return srv, boot
}

func TestCreateSessionRequiresBootstrap(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	body := bytes.NewBufferString(`{"workspace":"."}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestCreateAndAccessSessionWithToken(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()

	createBody, _ := json.Marshal(map[string]string{"workspace": dir})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+string(boot))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		SessionID string `json:"session_id"`
		Token     string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+created.SessionID, nil)
	getReq.Header.Set("Authorization", "Bearer "+created.Token)
	getReq.SetPathValue("id", created.SessionID)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestListAndAttachSessions(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()

	listReq := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusUnauthorized {
		t.Fatalf("list without bootstrap = %d, want 401", listRec.Code)
	}

	createBody, _ := json.Marshal(map[string]string{"workspace": dir})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+string(boot))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	listReq = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer "+string(boot))
	listRec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listed struct {
		Sessions []struct {
			SessionID                string `json:"session_id"`
			Workspace                string `json:"workspace"`
			CreatedAt                string `json:"created_at"`
			LastMeaningfulActivityAt string `json:"last_meaningful_activity_at"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Sessions) != 1 || listed.Sessions[0].SessionID != created.SessionID {
		t.Fatalf("listed = %+v, want one session %s", listed.Sessions, created.SessionID)
	}
	if listed.Sessions[0].CreatedAt == "" || listed.Sessions[0].LastMeaningfulActivityAt == "" {
		t.Fatalf("expected activity timestamps in list response: %+v", listed.Sessions[0])
	}

	attachReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+created.SessionID+"/attach", nil)
	attachReq.Header.Set("Authorization", "Bearer "+string(boot))
	attachReq.SetPathValue("id", created.SessionID)
	attachRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(attachRec, attachReq)
	if attachRec.Code != http.StatusOK {
		t.Fatalf("attach status = %d body=%s", attachRec.Code, attachRec.Body.String())
	}
	var attached struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(attachRec.Body.Bytes(), &attached); err != nil {
		t.Fatal(err)
	}
	if attached.Token == "" {
		t.Fatal("expected viewer token")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+created.SessionID, nil)
	getReq.Header.Set("Authorization", "Bearer "+attached.Token)
	getReq.SetPathValue("id", created.SessionID)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("viewer token should grant session metadata, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var viewed struct {
		SessionID                string `json:"session_id"`
		Workspace                string `json:"workspace"`
		CreatedAt                string `json:"created_at"`
		LastMeaningfulActivityAt string `json:"last_meaningful_activity_at"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.SessionID != created.SessionID || viewed.Workspace == "" {
		t.Fatalf("viewer metadata = %+v, want session %s", viewed, created.SessionID)
	}
	if viewed.CreatedAt == "" || viewed.LastMeaningfulActivityAt == "" {
		t.Fatalf("expected activity timestamps in viewer metadata: %+v", viewed)
	}

	screenReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+created.SessionID+"/screen", nil)
	screenReq.Header.Set("Authorization", "Bearer "+attached.Token)
	screenReq.SetPathValue("id", created.SessionID)
	screenRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(screenRec, screenReq)
	if screenRec.Code != http.StatusOK {
		t.Fatalf("viewer token screen = %d, want 200 body=%s", screenRec.Code, screenRec.Body.String())
	}

	inputBody, _ := json.Marshal(map[string]string{"input": "echo bootstrap-input-ok\n"})
	inputReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+created.SessionID+"/input", bytes.NewReader(inputBody))
	inputReq.Header.Set("Authorization", "Bearer "+string(boot))
	inputReq.Header.Set("Content-Type", "application/json")
	inputReq.SetPathValue("id", created.SessionID)
	inputRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inputRec, inputReq)
	if inputRec.Code != http.StatusOK {
		t.Fatalf("bootstrap input = %d, want 200 body=%s", inputRec.Code, inputRec.Body.String())
	}
}

func TestDeleteAndPruneSessions(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()

	create := func() string {
		t.Helper()
		createBody, _ := json.Marshal(map[string]string{"workspace": dir})
		req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewReader(createBody))
		req.Header.Set("Authorization", "Bearer "+string(boot))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
		}
		var created struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		return created.SessionID
	}

	keep := create()
	drop := create()
	extra := create()

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/sessions/"+drop, nil)
	delReq.Header.Set("Authorization", "Bearer "+string(boot))
	delReq.SetPathValue("id", drop)
	delRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", delRec.Code, delRec.Body.String())
	}

	pruneBody, _ := json.Marshal(map[string]any{"except": []string{keep}})
	pruneReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/prune", bytes.NewReader(pruneBody))
	pruneReq.Header.Set("Authorization", "Bearer "+string(boot))
	pruneReq.Header.Set("Content-Type", "application/json")
	pruneRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pruneRec, pruneReq)
	if pruneRec.Code != http.StatusOK {
		t.Fatalf("prune status = %d body=%s", pruneRec.Code, pruneRec.Body.String())
	}
	var pruned struct {
		Closed []string `json:"closed"`
	}
	if err := json.Unmarshal(pruneRec.Body.Bytes(), &pruned); err != nil {
		t.Fatal(err)
	}
	if len(pruned.Closed) != 1 || pruned.Closed[0] != extra {
		t.Fatalf("closed = %v, want [%s]", pruned.Closed, extra)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer "+string(boot))
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	var listed struct {
		Sessions []struct {
			SessionID string `json:"session_id"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Sessions) != 1 || listed.Sessions[0].SessionID != keep {
		t.Fatalf("listed = %+v, want only %s", listed.Sessions, keep)
	}
}

func TestWSRejectsDisallowedOriginBeforeUpgrade(t *testing.T) {
	srv, boot := newTestServer(t, []string{"https://app.example"})
	dir := t.TempDir()

	createBody, _ := json.Marshal(map[string]string{"workspace": dir})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+string(boot))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var created struct {
		SessionID string `json:"session_id"`
		Token     string `json:"token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	wsReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+created.SessionID+"/ws?token="+created.Token, nil)
	wsReq.Header.Set("Origin", "https://evil.example")
	wsReq.SetPathValue("id", created.SessionID)
	wsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wsRec, wsReq)

	if wsRec.Code != http.StatusForbidden {
		t.Fatalf("AE2: expected 403 before WS upgrade, got %d body=%s", wsRec.Code, wsRec.Body.String())
	}
}

func TestWSRejectsMissingScope(t *testing.T) {
	srv, boot := newTestServer(t, []string{"https://app.example"})
	dir := t.TempDir()
	mgr := session.NewManager()
	tokens := auth.NewStore()
	cfg := config.DefaultServer()
	cfg.AllowedOrigins = []string{"https://app.example"}
	srv = api.NewServer(cfg, mgr, tokens, boot)

	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	// Mint read-only human view without write - actually need human:view for WS
	token, err := tokens.Mint(sess.ID, []auth.Scope{auth.ScopeAgentRead}, cfg.TokenTTL)
	if err != nil {
		t.Fatal(err)
	}

	wsReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sess.ID+"/ws?token="+token, nil)
	wsReq.Header.Set("Origin", "https://app.example")
	wsReq.SetPathValue("id", sess.ID)
	wsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wsRec, wsReq)
	if wsRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing human:view, got %d", wsRec.Code)
	}
}
