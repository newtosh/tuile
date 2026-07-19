package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionStateCompact(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf state-ok\\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	waitForScreenContains(t, srv.Handler(), id, token, "state-ok")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/state?format=compact&tail=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		V uint64 `json:"v"`
		C int    `json:"c"`
		R int    `json:"r"`
		T string `json:"t"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.V == 0 || resp.C == 0 || resp.R == 0 {
		t.Fatalf("unexpected meta: %+v", resp)
	}
	if !strings.Contains(resp.T, "state-ok") {
		t.Fatalf("t = %q", resp.T)
	}
}

func TestResizeCompactDefault(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	body, _ := json.Marshal(map[string]int{"cols": 80, "rows": 30})
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/resize", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		V uint64 `json:"v"`
		C int    `json:"c"`
		R int    `json:"r"`
		Screen any `json:"screen"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.C != 80 || resp.R != 30 {
		t.Fatalf("dims = %dx%d", resp.C, resp.R)
	}
	if resp.Screen != nil {
		t.Fatal("expected no screen in compact resize response")
	}
}
