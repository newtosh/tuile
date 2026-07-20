package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/api"
)

func TestRootRedirectsToView(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/?session=x&token=y", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/view?session=x&token=y" {
		t.Fatalf("location = %q", loc)
	}
}

func TestViewPageServed(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/view?session=x&token=y", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Tuile") {
		t.Fatal("expected viewer HTML")
	}
}

func TestAssetsServed(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "connectWS") {
		t.Fatal("expected app.js content")
	}
}

func TestBundledFontServed(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/assets/fonts/JetBrainsMonoNerdFont-Regular.woff2", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("font status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "font") && ct != "application/octet-stream" {
		t.Fatalf("unexpected content-type %q", ct)
	}
}

func TestFaviconServed(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	for _, path := range []string{"/assets/favicon.svg", "/assets/favicon.ico", "/assets/favicon.png"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("favicon.ico redirect status = %d", rec.Code)
	}
}

func TestDefaultDevOrigins(t *testing.T) {
	origins := api.DefaultDevOrigins("127.0.0.1:7710")
	if len(origins) < 2 {
		t.Fatalf("origins = %v", origins)
	}
	found := false
	for _, o := range origins {
		if o == "http://127.0.0.1:7710" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing loopback origin in %v", origins)
	}
}
