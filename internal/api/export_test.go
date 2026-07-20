package api_test

import (
	"bytes"
	"encoding/json"
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

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chrome_preset", export.ChromeOSWireframe)
	_ = w.WriteField("background_mode", export.BackgroundCustom)
	_ = w.WriteField("format", export.FormatPNG)
	_ = w.WriteField("scale", "1")
	part, _ := w.CreateFormFile("background_image", "bg.png")
	_, _ = io.Copy(part, bytes.NewReader([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}))
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
}
