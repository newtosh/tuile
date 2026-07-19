//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/newtosh/tuile/testkit"
)

func TestViewerPrunesStaleAckState(t *testing.T) {
	srv := testkit.NewServer(t)
	keep := srv.NewSession(t, t.TempDir())
	removeA := srv.NewSession(t, t.TempDir())
	removeB := srv.NewSession(t, t.TempDir())

	ackSeed, err := json.Marshal(map[string]string{
		keep.ID:    "2026-01-01T00:00:00.000Z",
		removeA.ID: "2026-01-01T00:00:00.000Z",
		removeB.ID: "2026-01-01T00:00:00.000Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	viewURL := srv.URL + "/view"
	boot := string(srv.Boot)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewURL),
		chromedp.Evaluate(fmt.Sprintf(`localStorage.setItem('tuile_bootstrap', %q)`, boot), nil),
		chromedp.Evaluate(fmt.Sprintf(`localStorage.setItem('tuile_session_ack', %q)`, string(ackSeed)), nil),
		chromedp.Reload(),
		chromedp.WaitVisible("#session-list", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	srv.DeleteSession(t, removeA.ID)
	srv.DeleteSession(t, removeB.ID)

	deadline := time.Now().Add(8 * time.Second)
	var ackKeys []string
	for time.Now().Before(deadline) {
		var keysJSON string
		if err := chromedp.Run(ctx, chromedp.Evaluate(
			`JSON.stringify(Object.keys(JSON.parse(localStorage.getItem('tuile_session_ack') || '{}')))`,
			&keysJSON,
		)); err != nil {
			t.Skipf("browser automation unavailable: %v", err)
		}
		if err := json.Unmarshal([]byte(keysJSON), &ackKeys); err != nil {
			t.Fatalf("parse ack keys: %v", err)
		}
		if len(ackKeys) == 1 && ackKeys[0] == keep.ID {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("expected ack map to prune to [%s], got %v", keep.ID, ackKeys)
}
