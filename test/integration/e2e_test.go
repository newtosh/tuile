//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/newtosh/tuile/testkit"
)

func TestE2EAgentLoopF1(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)

	marker := "tuile-e2e-agent-ok"
	sess.EmitMarker(t, dir, marker)

	screenReq, _ := http.NewRequest(http.MethodGet, sess.ServerURL()+"/v1/sessions/"+sess.SessionID()+"/screen", nil)
	screenReq.Header.Set("Authorization", "Bearer "+sess.Token)
	screenResp, err := http.DefaultClient.Do(screenReq)
	if err != nil {
		t.Fatal(err)
	}
	defer screenResp.Body.Close()
	if screenResp.StatusCode != http.StatusOK {
		t.Fatalf("screen status = %d", screenResp.StatusCode)
	}

	resizeBody, _ := json.Marshal(map[string]int{"cols": 72, "rows": 22})
	resizeReq, _ := http.NewRequest(http.MethodPost, sess.ServerURL()+"/v1/sessions/"+sess.SessionID()+"/resize?format=compact", bytes.NewReader(resizeBody))
	resizeReq.Header.Set("Authorization", "Bearer "+sess.Token)
	resizeReq.Header.Set("Content-Type", "application/json")
	resizeResp, err := http.DefaultClient.Do(resizeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resizeResp.Body.Close()
	if resizeResp.StatusCode != http.StatusOK {
		t.Fatalf("resize status = %d", resizeResp.StatusCode)
	}
	var resized struct {
		C int `json:"c"`
		R int `json:"r"`
	}
	if err := json.NewDecoder(resizeResp.Body).Decode(&resized); err != nil {
		t.Fatal(err)
	}
	if resized.C != 72 || resized.R != 22 {
		t.Fatalf("resize grid = %dx%d, want 72x22 (AE4/AE5)", resized.C, resized.R)
	}
}

func TestE2EHumanObserveTakeoverF2(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)

	marker := "tuile-e2e-human"
	sess.EmitMarker(t, dir, marker)
	sess.AssertTerminalContains(t, marker)

	sess.Takeover(t)

	conflictBody, _ := json.Marshal(map[string]string{"input": "agent-blocked\n"})
	conflictReq, _ := http.NewRequest(http.MethodPost, sess.ServerURL()+"/v1/sessions/"+sess.SessionID()+"/input", bytes.NewReader(conflictBody))
	conflictReq.Header.Set("Authorization", "Bearer "+sess.Token)
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictResp, err := http.DefaultClient.Do(conflictReq)
	if err != nil {
		t.Fatal(err)
	}
	conflictResp.Body.Close()
	if conflictResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected agent 409 during human control, got %d", conflictResp.StatusCode)
	}
}

func TestE2ESessionTokenIsolation(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sessA := srv.NewSession(t, dir)
	sessB := srv.NewSession(t, dir)

	resp := sessA.GetWithToken(t, "/v1/sessions/"+sessB.SessionID()+"/screen")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("token A accessing session B: status = %d, want 403", resp.StatusCode)
	}

	wsReq, _ := http.NewRequest(http.MethodGet, sessA.ServerURL()+"/v1/sessions/"+sessA.SessionID()+"/ws?token="+sessA.Token, nil)
	wsReq.Header.Set("Origin", "https://evil.example")
	wsResp, err := http.DefaultClient.Do(wsReq)
	if err != nil {
		t.Fatal(err)
	}
	wsResp.Body.Close()
	if wsResp.StatusCode != http.StatusForbidden {
		t.Fatalf("AE2: bad origin status = %d, want 403", wsResp.StatusCode)
	}
}

func TestE2ENonZeroExitVisibleInScreen(t *testing.T) {
	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, t.TempDir())

	sess.PostInputRaw(t, mustJSON(map[string]string{"input": "/bin/sh -c 'false; printf TUILE_EXIT_1\\n'\n"}))
	sess.WaitContains(t, "TUILE_EXIT_1")

	sess.Input(t, "echo tuile-exit-check\n")
	sess.WaitContains(t, "tuile-exit-check")
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
