package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/api"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

func createSessionViaAPI(t *testing.T, srv *api.Server, boot auth.BootstrapSecret, workspace string) (sessionID, token string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"workspace": workspace})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewReader(body))
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
	return created.SessionID, created.Token
}

func waitForScreenContains(t *testing.T, handler http.Handler, id, token, needle string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), needle) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("screen never contained %q", needle)
}

func TestHeadlessInputAndScreen(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf tuile-headless-ok\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	if inRec.Code != http.StatusOK {
		t.Fatalf("input status = %d body=%s", inRec.Code, inRec.Body.String())
	}

	waitForScreenContains(t, srv.Handler(), id, token, "tuile-headless-ok")
}

func TestHeadlessScreenRequiresReadScope(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	mgr := session.NewManager()
	tokens := auth.NewStore()
	cfg := config.DefaultServer()
	srv = api.NewServer(cfg, mgr, tokens, boot)

	sess, err := mgr.Create(dir, config.DefaultSession())
	if err != nil {
		t.Fatal(err)
	}
	writeOnly, err := tokens.Mint(sess.ID, []auth.Scope{auth.ScopeAgentWrite}, cfg.TokenTTL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sess.ID+"/screen", nil)
	req.Header.Set("Authorization", "Bearer "+writeOnly)
	req.SetPathValue("id", sess.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestHeadlessResizeUpdatesGrid(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	body, _ := json.Marshal(map[string]int{"cols": 40, "rows": 10})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/resize", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resize status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		C int `json:"c"`
		R int `json:"r"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.C != 40 || resp.R != 10 {
		t.Fatalf("grid = %dx%d, want 40x10", resp.C, resp.R)
	}
}

func TestHeadlessScreenSinceNotModified(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getReq.SetPathValue("id", id)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("screen status = %d", getRec.Code)
	}

	var first struct {
		Version uint64 `json:"version"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}

	sinceReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?since="+strconv.FormatUint(first.Version, 10), nil)
	sinceReq.Header.Set("Authorization", "Bearer "+token)
	sinceReq.SetPathValue("id", id)
	sinceRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(sinceRec, sinceReq)
	if sinceRec.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", sinceRec.Code)
	}
}

func TestHeadlessScreenSinceReturnsDiff(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	baseReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen", nil)
	baseReq.Header.Set("Authorization", "Bearer "+token)
	baseReq.SetPathValue("id", id)
	baseRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(baseRec, baseReq)
	var base struct {
		Version uint64 `json:"version"`
	}
	if err := json.Unmarshal(baseRec.Body.Bytes(), &base); err != nil {
		t.Fatal(err)
	}

	inputBody, _ := json.Marshal(map[string]string{"input": "printf tuile-diff-ok\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "tuile-diff-ok")

	// Re-read version after output settles so since matches cached history.
	settleReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen", nil)
	settleReq.Header.Set("Authorization", "Bearer "+token)
	settleReq.SetPathValue("id", id)
	settleRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(settleRec, settleReq)
	var settled struct {
		Version uint64 `json:"version"`
	}
	if err := json.Unmarshal(settleRec.Body.Bytes(), &settled); err != nil {
		t.Fatal(err)
	}

	inputBody2, _ := json.Marshal(map[string]string{"input": "printf tuile-diff-2\n"})
	inReq2 := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody2))
	inReq2.Header.Set("Authorization", "Bearer "+token)
	inReq2.SetPathValue("id", id)
	inRec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec2, inReq2)
	waitForScreenContains(t, srv.Handler(), id, token, "tuile-diff-2")

	diffReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?since="+strconv.FormatUint(settled.Version, 10), nil)
	diffReq.Header.Set("Authorization", "Bearer "+token)
	diffReq.SetPathValue("id", id)
	diffRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(diffRec, diffReq)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("diff status = %d body=%s", diffRec.Code, diffRec.Body.String())
	}
	var resp struct {
		Version uint64 `json:"version"`
		Since   uint64 `json:"since"`
		Diff    struct {
			ChangedLines []struct {
				Y    int    `json:"y"`
				Text string `json:"text"`
			} `json:"changed_lines"`
		} `json:"diff"`
		Screen struct {
			Lines []string `json:"lines"`
		} `json:"screen"`
	}
	if err := json.Unmarshal(diffRec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Since != settled.Version {
		t.Fatalf("since = %d, want %d", resp.Since, settled.Version)
	}
	if len(resp.Diff.ChangedLines) == 0 {
		t.Fatalf("expected changed lines in diff, body=%s", diffRec.Body.String())
	}
	found := false
	for _, line := range resp.Diff.ChangedLines {
		if strings.Contains(line.Text, "tuile-diff-2") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diff missing marker: %+v", resp.Diff.ChangedLines)
	}
}

func TestHeadlessScreenDetailCells(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf '\\033[1;31mB\\033[0m\\n'\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "B")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?detail=cells", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Screen struct {
			Grid []struct {
				Cells []struct {
					Ch   string   `json:"ch"`
					Attr []string `json:"attr"`
				} `json:"cells"`
			} `json:"grid"`
		} `json:"screen"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Screen.Grid) == 0 || len(resp.Screen.Grid[0].Cells) == 0 {
		t.Fatalf("expected grid cells, body=%s", rec.Body.String())
	}
}

func TestHeadlessScreenTailCompact(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf line-one\\nprintf line-two\\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "line-two")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?format=text&tail=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Text    string `json:"text"`
		Version uint64 `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "line-two") {
		t.Fatalf("text = %q", resp.Text)
	}
	if resp.Version == 0 {
		t.Fatal("expected version")
	}
}

func TestHeadlessScreenPlainFormat(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf plain-ok\\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "plain-ok")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?format=plain&tail=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("content-type = %q", ct)
	}
	if rec.Header().Get("X-Tuile-Version") == "" {
		t.Fatal("missing version header")
	}
	if !strings.Contains(rec.Body.String(), "plain-ok") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHeadlessScreenCompactFormat(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf compact-ok\\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "compact-ok")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen?format=compact&tail=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	var resp struct {
		V uint64 `json:"v"`
		T string `json:"t"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.T, "compact-ok") {
		t.Fatalf("t = %q", resp.T)
	}
	if resp.V == 0 {
		t.Fatal("expected version")
	}
}

func TestHeadlessWaitForText(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	go func() {
		time.Sleep(100 * time.Millisecond)
		body, _ := json.Marshal(map[string]string{"input": "printf wait-marker\\n"})
		req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
	}()

	body, _ := json.Marshal(map[string]any{
		"contains":   "wait-marker",
		"timeout_ms": 3000,
		"tail":       5,
		"format":     "compact",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/wait", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("wait status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Matched bool   `json:"m"`
		Text    string `json:"t"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Matched {
		t.Fatalf("expected match, body=%s", rec.Body.String())
	}
	if !strings.Contains(resp.Text, "wait-marker") {
		t.Fatalf("text = %q", resp.Text)
	}
}
