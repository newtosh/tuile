package api

import (
	"io/fs"
	"net/http"
	"strings"

	tuileweb "github.com/newtosh/tuile/web"
)

func (s *Server) registerStaticRoutes() {
	assets, _ := fs.Sub(tuileweb.FS, ".")
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	s.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/assets/favicon.ico", http.StatusPermanentRedirect)
	})
	s.mux.HandleFunc("GET /view", s.handleView)
	s.mux.HandleFunc("GET /view/", s.handleView)
	s.mux.HandleFunc("GET /", s.handleRootRedirect)
}

func (s *Server) handleRootRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.RawQuery
	target := "/view"
	if q != "" {
		target += "?" + q
	}
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/view" && r.URL.Path != "/view/" {
		http.NotFound(w, r)
		return
	}
	data, err := tuileweb.FS.ReadFile("index.html")
	if err != nil {
		http.Error(w, "viewer unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// DefaultDevOrigins returns browser Origin values for local viewer dev (R10 dev ergonomics).
func DefaultDevOrigins(listen string) []string {
	host := listen
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := listen
	if i := strings.LastIndex(listen, ":"); i >= 0 {
		port = listen[i+1:]
	}
	origins := []string{
		"http://127.0.0.1:" + port,
		"http://localhost:" + port,
	}
	if host != "127.0.0.1" && host != "localhost" {
		origins = append(origins, "http://"+listen)
	}
	return origins
}
