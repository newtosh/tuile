package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentInputRejectedDuringHumanTakeover(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	takeoverReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/takeover", nil)
	takeoverReq.Header.Set("Authorization", "Bearer "+token)
	takeoverReq.SetPathValue("id", id)
	takeoverRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(takeoverRec, takeoverReq)
	if takeoverRec.Code != http.StatusOK {
		t.Fatalf("takeover status = %d body=%s", takeoverRec.Code, takeoverRec.Body.String())
	}

	inputBody, _ := json.Marshal(map[string]string{"input": "x"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	if inRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", inRec.Code, inRec.Body.String())
	}
}

func TestAgentResizeRejectedDuringHumanTakeover(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	takeoverReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/takeover", nil)
	takeoverReq.Header.Set("Authorization", "Bearer "+token)
	takeoverReq.SetPathValue("id", id)
	takeoverRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(takeoverRec, takeoverReq)

	body, _ := json.Marshal(map[string]int{"cols": 90, "rows": 25})
	resizeReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/resize", bytes.NewReader(body))
	resizeReq.Header.Set("Authorization", "Bearer "+token)
	resizeReq.SetPathValue("id", id)
	resizeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resizeRec, resizeReq)
	if resizeRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resizeRec.Code)
	}
}

func TestReleaseRestoresAgentResizeAuthority(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	agentResize, _ := json.Marshal(map[string]int{"cols": 88, "rows": 28})
	ar := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/resize", bytes.NewReader(agentResize))
	ar.Header.Set("Authorization", "Bearer "+token)
	ar.SetPathValue("id", id)
	arec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(arec, ar)
	if arec.Code != http.StatusOK {
		t.Fatalf("agent resize: %d", arec.Code)
	}

	takeover := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/takeover", nil)
	takeover.Header.Set("Authorization", "Bearer "+token)
	takeover.SetPathValue("id", id)
	trec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(trec, takeover)

	humanResize, _ := json.Marshal(map[string]int{"cols": 50, "rows": 15})
	hr := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/human/resize", bytes.NewReader(humanResize))
	hr.Header.Set("Authorization", "Bearer "+token)
	hr.SetPathValue("id", id)
	hrec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(hrec, hr)
	if hrec.Code != http.StatusOK {
		t.Fatalf("human resize: %d body=%s", hrec.Code, hrec.Body.String())
	}

	release := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/release", nil)
	release.Header.Set("Authorization", "Bearer "+token)
	release.SetPathValue("id", id)
	rrec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rrec, release)
	if rrec.Code != http.StatusOK {
		t.Fatalf("release: %d", rrec.Code)
	}

	screenReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+id+"/screen", nil)
	screenReq.Header.Set("Authorization", "Bearer "+token)
	screenReq.SetPathValue("id", id)
	screenRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(screenRec, screenReq)
	if screenRec.Code != http.StatusOK {
		t.Fatalf("screen: %d", screenRec.Code)
	}
	var screen struct {
		Screen struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		} `json:"screen"`
	}
	if err := json.Unmarshal(screenRec.Body.Bytes(), &screen); err != nil {
		t.Fatal(err)
	}
	if screen.Screen.Cols != 88 || screen.Screen.Rows != 28 {
		t.Fatalf("after release grid = %dx%d, want 88x28", screen.Screen.Cols, screen.Screen.Rows)
	}
}
