//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
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

func TestSessionExportCustomBackgroundMultipart(t *testing.T) {
	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, t.TempDir())

	transparent := export.DefaultOptions()
	transparent.BackgroundMode = export.BackgroundTransparent
	transparent.ChromePreset = export.ChromeOSWireframe
	transparentPayload, _ := json.Marshal(transparent)
	transparentReq, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions/"+sess.ID+"/export", bytes.NewReader(transparentPayload))
	if err != nil {
		t.Fatal(err)
	}
	transparentReq.Header.Set("Authorization", "Bearer "+sess.Token)
	transparentReq.Header.Set("Content-Type", "application/json")
	transparentResp, err := http.DefaultClient.Do(transparentReq)
	if err != nil {
		t.Fatal(err)
	}
	transparentBody, _ := io.ReadAll(transparentResp.Body)
	transparentResp.Body.Close()
	if transparentResp.StatusCode != http.StatusOK {
		t.Fatalf("transparent export status %d", transparentResp.StatusCode)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("chrome_preset", export.ChromeOSWireframe)
	_ = w.WriteField("background_mode", export.BackgroundCustom)
	_ = w.WriteField("format", export.FormatPNG)
	_ = w.WriteField("scale", "1")
	part, err := w.CreateFormFile("background_image", "bg.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewReader(customExportBackgroundPNG(t))); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	customReq, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions/"+sess.ID+"/export", &body)
	if err != nil {
		t.Fatal(err)
	}
	customReq.Header.Set("Authorization", "Bearer "+sess.Token)
	customReq.Header.Set("Content-Type", w.FormDataContentType())
	customResp, err := http.DefaultClient.Do(customReq)
	if err != nil {
		t.Fatal(err)
	}
	defer customResp.Body.Close()
	if customResp.StatusCode != http.StatusOK {
		t.Fatalf("custom export status %d", customResp.StatusCode)
	}
	customBody, err := io.ReadAll(customResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(transparentBody, customBody) {
		t.Fatal("custom background export identical to transparent export")
	}
}

func customExportBackgroundPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	fill := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
