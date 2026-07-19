package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Session is one Tuile PTY session created via the HTTP API.
type Session struct {
	server *Server
	ID     string
	Token  string
}

// NewSession creates a shell session in workspace (use t.TempDir() for isolation).
func (s *Server) NewSession(t *testing.T, workspace string) *Session {
	t.Helper()
	body, err := json.Marshal(map[string]string{"workspace": workspace})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, s.URL+"/v1/sessions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(s.Boot))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create session status = %d body=%s", resp.StatusCode, b)
	}
	var created struct {
		SessionID string `json:"session_id"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	return &Session{
		server: s,
		ID:     created.SessionID,
		Token:  created.Token,
	}
}

// DeleteSession closes a session via DELETE /v1/sessions/{id}.
func (s *Server) DeleteSession(t *testing.T, id string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, s.URL+"/v1/sessions/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(s.Boot))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete session status = %d body=%s", resp.StatusCode, b)
	}
}

// WaitForShell blocks until the session PTY shows an interactive shell prompt.
func (sess *Session) WaitForShell(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if looksLikeShellPrompt(sess.PlainScreen(t, 15)) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatal("timed out waiting for shell prompt")
}

func looksLikeShellPrompt(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimRight(line, " \t")
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "❯") {
			return true
		}
		switch trim[len(trim)-1] {
		case '$', '#', '%', '>':
			return true
		}
	}
	return false
}

// EmitMarker writes marker to a file in workspace and cats it.
// Unlike echo/printf, the shell echo line does not contain marker text, so WaitContains
// cannot match prematurely on typed command text.
func (sess *Session) EmitMarker(t *testing.T, workspace, marker string) {
	t.Helper()
	path := filepath.Join(workspace, ".tuile-test-marker")
	if err := os.WriteFile(path, []byte(marker+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sess.WaitForShell(t)
	sess.Input(t, "cat .tuile-test-marker\n")
	sess.WaitContainsTimeout(t, marker, 15*time.Second)
}

// Input writes PTY input (agent token auth).
func (sess *Session) Input(t *testing.T, input string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"input": input})
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/input", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("input status = %d body=%s", resp.StatusCode, b)
	}
}

// WaitContains blocks until the session screen tail contains marker (POST /wait).
func (sess *Session) WaitContains(t *testing.T, marker string) {
	t.Helper()
	sess.waitContains(t, marker, 8*time.Second)
}

// WaitContainsTimeout is WaitContains with a custom timeout.
func (sess *Session) WaitContainsTimeout(t *testing.T, marker string, timeout time.Duration) {
	t.Helper()
	sess.waitContains(t, marker, timeout)
}

func (sess *Session) waitContains(t *testing.T, marker string, timeout time.Duration) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"contains":   marker,
		"timeout_ms": int(timeout.Milliseconds()),
		"tail":       40,
		"format":     "compact",
	})
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/wait", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("wait status = %d body=%s", resp.StatusCode, b)
	}
	var out struct {
		Matched bool   `json:"m"`
		Text    string `json:"t"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.Matched {
		t.Fatalf("screen never contained %q (tail=%q)", marker, out.Text)
	}
}

// PlainScreen returns the plain-text tail of the session screen.
func (sess *Session) PlainScreen(t *testing.T, tail int) string {
	t.Helper()
	url := fmt.Sprintf("%s/v1/sessions/%s/screen?format=plain&tail=%d", sess.server.URL, sess.ID, tail)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("screen status = %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Resize sets agent PTY dimensions.
func (sess *Session) Resize(t *testing.T, cols, rows int) {
	t.Helper()
	body, _ := json.Marshal(map[string]int{"cols": cols, "rows": rows})
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/resize", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resize status = %d", resp.StatusCode)
	}
}

// Takeover grants human PTY control.
func (sess *Session) Takeover(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/takeover", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("takeover status = %d", resp.StatusCode)
	}
}

// HumanResize sets PTY size while human controls the session.
func (sess *Session) HumanResize(t *testing.T, cols, rows int) {
	t.Helper()
	body, _ := json.Marshal(map[string]int{"cols": cols, "rows": rows})
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/human/resize", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("human resize status = %d", resp.StatusCode)
	}
}

// ScreenGrid returns cols and rows from the JSON screen endpoint.
func (sess *Session) ScreenGrid(t *testing.T) (cols, rows int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, sess.server.URL+"/v1/sessions/"+sess.ID+"/screen", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var screen struct {
		Screen struct {
			Cols int `json:"cols"`
			Rows int `json:"rows"`
		} `json:"screen"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&screen); err != nil {
		t.Fatal(err)
	}
	return screen.Screen.Cols, screen.Screen.Rows
}

// ViewURL returns the browser viewer URL for this session.
func (sess *Session) ViewURL() string {
	return fmt.Sprintf("%s/view?session=%s&token=%s", sess.server.URL, sess.ID, sess.Token)
}

// PostInputRaw sends a pre-marshaled JSON body to /input (for advanced tests).
func (sess *Session) PostInputRaw(t *testing.T, body []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, sess.server.URL+"/v1/sessions/"+sess.ID+"/input", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("input status = %d", resp.StatusCode)
	}
}

// GetWithToken performs GET with the session token (for isolation tests).
func (sess *Session) GetWithToken(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, sess.server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// ServerURL returns the test server base URL.
func (sess *Session) ServerURL() string {
	return sess.server.URL
}

// SessionID returns the session identifier.
func (sess *Session) SessionID() string {
	return sess.ID
}
