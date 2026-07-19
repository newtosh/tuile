package tuileclient_test

import (
	"net/http/httptest"
	"testing"

	"github.com/newtosh/tuile/internal/api"
	"github.com/newtosh/tuile/internal/auth"
	"github.com/newtosh/tuile/internal/config"
	"github.com/newtosh/tuile/internal/session"
	"github.com/newtosh/tuile/internal/tuileclient"
)

func TestClientCreateAndRead(t *testing.T) {
	boot, err := auth.NewBootstrapSecret()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultServer()
	cfg.AllowedOrigins = []string{"http://127.0.0.1"}
	srv := api.NewServer(cfg, session.NewManager(), auth.NewStore(), boot)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	c := tuileclient.New(ts.URL, string(boot))
	created, err := c.CreateSession(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if created.SessionID == "" || created.Token == "" {
		t.Fatalf("create response: %+v", created)
	}

	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d", len(sessions))
	}

	if err := c.SendInput(created.SessionID, created.Token, "echo hi\n", false); err != nil {
		t.Fatal(err)
	}

	out, err := c.WaitWithTimeout(created.SessionID, created.Token, tuileclient.WaitRequest{
		Contains:  "hi",
		TimeoutMS: 3000,
		Tail:      5,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Matched {
		t.Fatalf("wait not matched: %+v", out)
	}
}
