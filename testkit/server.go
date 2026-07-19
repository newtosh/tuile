package testkit

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newtosh/tuile/internal/api"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
)

// Server is an in-process Tuile HTTP server for tests.
type Server struct {
	URL  string
	Boot auth.BootstrapSecret

	httptest *httptest.Server
}

// NewServer starts Tuile on an ephemeral 127.0.0.1 port and registers cleanup on t.
func NewServer(t *testing.T) *Server {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	port := addr[strings.LastIndex(addr, ":")+1:]

	boot, err := auth.NewBootstrapSecret()
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultServer()
	cfg.AllowedOrigins = []string{
		"http://127.0.0.1:" + port,
		"http://localhost:" + port,
	}
	srv := api.NewServer(cfg, session.NewManager(), auth.NewStore(), boot)
	ts := httptest.NewUnstartedServer(srv.Handler())
	ts.Listener = ln
	ts.Start()
	t.Cleanup(ts.Close)

	return &Server{
		URL:      ts.URL,
		Boot:     boot,
		httptest: ts,
	}
}

// HealthOK reports whether GET /health returns 200.
func (s *Server) HealthOK(t *testing.T) bool {
	t.Helper()
	resp, err := http.Get(s.URL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
