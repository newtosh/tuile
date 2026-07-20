//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/testkit"
)

func TestSessionExportPNG(t *testing.T) {
	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, t.TempDir())

	input := map[string]string{"input": "printf tuile-export-test\n"}
	body, _ := json.Marshal(input)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions/"+sess.ID+"/input", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("input status %d", resp.StatusCode)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		screenReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/sessions/"+sess.ID+"/screen", nil)
		screenReq.Header.Set("Authorization", "Bearer "+sess.Token)
		screenResp, err := http.DefaultClient.Do(screenReq)
		if err == nil && screenResp.StatusCode == http.StatusOK {
			var screenBody struct {
				Screen struct {
					Lines []string `json:"lines"`
				} `json:"screen"`
			}
			_ = json.NewDecoder(screenResp.Body).Decode(&screenBody)
			screenResp.Body.Close()
			for _, line := range screenBody.Screen.Lines {
				if len(line) > 0 {
					goto ready
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("screen never populated")
ready:

	opts := export.DefaultOptions()
	opts.BackgroundPreset = "ink"
	payload, _ := json.Marshal(opts)
	expReq, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions/"+sess.ID+"/export", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	expReq.Header.Set("Authorization", "Bearer "+sess.Token)
	expReq.Header.Set("Content-Type", "application/json")
	expResp, err := http.DefaultClient.Do(expReq)
	if err != nil {
		t.Fatal(err)
	}
	defer expResp.Body.Close()
	if expResp.StatusCode != http.StatusOK {
		t.Fatalf("export status %d", expResp.StatusCode)
	}
	if ct := expResp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type %q", ct)
	}
}
