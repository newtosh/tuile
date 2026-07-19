package testkit_test

import (
	"testing"

	"github.com/newtosh/tuile/testkit"
)

func TestServerHealthAndSessionRoundTrip(t *testing.T) {
	srv := testkit.NewServer(t)
	if !srv.HealthOK(t) {
		t.Fatal("expected /health OK")
	}
	sess := srv.NewSession(t, t.TempDir())
	sess.Input(t, "printf tuile-testkit-ok\n")
	sess.WaitContains(t, "tuile-testkit-ok")
}
