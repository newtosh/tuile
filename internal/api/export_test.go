package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/export"
)

func TestSessionExportPNG(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	inputBody, _ := json.Marshal(map[string]string{"input": "printf export-ok\n"})
	inReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/input", bytes.NewReader(inputBody))
	inReq.Header.Set("Authorization", "Bearer "+token)
	inReq.SetPathValue("id", id)
	inRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(inRec, inReq)
	if inRec.Code != http.StatusOK {
		t.Fatalf("input status = %d", inRec.Code)
	}
	waitForScreenContains(t, srv.Handler(), id, token, "export-ok")

	opts := export.DefaultOptions()
	body, _ := json.Marshal(opts)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/export", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Fatalf("content-type = %q", ct)
	}
	if len(rec.Body.Bytes()) < 8 {
		t.Fatal("empty png body")
	}
}

func TestSessionExportUnauthorized(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, _ := createSessionViaAPI(t, srv, boot, dir)
	opts := export.DefaultOptions()
	body, _ := json.Marshal(opts)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/export", bytes.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", rec.Code)
	}
}

func TestSessionExportMultipartBackground(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	transparent := export.DefaultOptions()
	transparent.BackgroundMode = export.BackgroundTransparent
	transparent.ChromePreset = export.ChromeOSWireframe
	transparentBody, _ := json.Marshal(transparent)
	transparentReq := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/export", bytes.NewReader(transparentBody))
	transparentReq.Header.Set("Authorization", "Bearer "+token)
	transparentReq.Header.Set("Content-Type", "application/json")
	transparentReq.SetPathValue("id", id)
	transparentRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(transparentRec, transparentReq)
	if transparentRec.Code != http.StatusOK {
		t.Fatalf("transparent export status = %d body=%s", transparentRec.Code, transparentRec.Body.String())
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chrome_preset", export.ChromeOSWireframe)
	_ = w.WriteField("background_mode", export.BackgroundCustom)
	_ = w.WriteField("format", export.FormatPNG)
	_ = w.WriteField("scale", "1")
	part, _ := w.CreateFormFile("background_image", "bg.png")
	_, _ = io.Copy(part, bytes.NewReader(customBackgroundPNG(t)))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/export", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d body=%s", rec.Code, rec.Body.String())
	}
	if bytes.Equal(transparentRec.Body.Bytes(), rec.Body.Bytes()) {
		t.Fatal("custom background export identical to transparent export")
	}
}

func TestSessionExportSVG(t *testing.T) {
	srv, boot := newTestServer(t, nil)
	dir := t.TempDir()
	id, token := createSessionViaAPI(t, srv, boot, dir)

	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.BackgroundMode = export.BackgroundTransparent
	body, _ := json.Marshal(opts)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+id+"/export", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "<svg") {
		t.Fatalf("unexpected svg body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `viewBox="0 0 `) {
		t.Fatal("missing viewBox")
	}
	if strings.Contains(rec.Body.String(), `href="data:image/jpeg;base64,`) {
		t.Fatal("svg export must be vector, not raster wrapper")
	}
	if strings.Contains(rec.Body.String(), `transform="scale(`) {
		t.Fatal("svg export must not use nested scale transforms")
	}
}

func customBackgroundPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	fill := color.RGBA{R: 220, G: 40, B: 120, A: 255}
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
