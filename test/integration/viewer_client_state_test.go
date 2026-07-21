//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/newtosh/tuile/testkit"
)

const sessionListPollInterval = 400 * time.Millisecond

func openViewerWithSessions(t *testing.T, srv *testkit.Server, sessionCount int) context.Context {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < sessionCount; i++ {
		ws := filepath.Join(root, fmt.Sprintf("ws-%d", i))
		if err := os.MkdirAll(ws, 0o755); err != nil {
			t.Fatal(err)
		}
		srv.NewSession(t, ws)
	}

	ctx, cancel := testkit.BrowserContext(t)
	t.Cleanup(cancel)

	viewURL := srv.URL + "/view"
	boot := string(srv.Boot)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewURL),
		chromedp.Evaluate(fmt.Sprintf(`localStorage.setItem('tuile_bootstrap', %q)`, boot), nil),
		chromedp.Reload(),
		chromedp.WaitVisible("#session-list", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}
	return ctx
}

func waitForSessionRowCount(t *testing.T, ctx context.Context, want int) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := chromedp.Run(ctx, chromedp.Evaluate(
			`document.querySelectorAll('#session-list .session-row').length`,
			&count,
		)); err != nil {
			t.Skipf("browser automation unavailable: %v", err)
		}
		if count == want {
			return
		}
		time.Sleep(sessionListPollInterval)
	}
	var count int
	_ = chromedp.Run(ctx, chromedp.Evaluate(
		`document.querySelectorAll('#session-list .session-row').length`,
		&count,
	))
	t.Fatalf("expected %d session rows, got %d", want, count)
}

func TestViewerDirectLinkShowsConnectedSessionWithoutBootstrap(t *testing.T) {
	srv := testkit.NewServer(t)
	sess := srv.NewSession(t, t.TempDir())

	ctx, cancel := testkit.BrowserContext(t)
	defer cancel()

	viewURL := fmt.Sprintf("%s/view?session=%s&token=%s", srv.URL, sess.ID, sess.Token)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(viewURL),
		chromedp.Evaluate(`localStorage.removeItem('tuile_bootstrap')`, nil),
		chromedp.Reload(),
		chromedp.WaitVisible("#session-list", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	waitForSessionRowCount(t, ctx, 1)
}

func TestViewerSessionListRendersAllSessions(t *testing.T) {
	srv := testkit.NewServer(t)
	const want = 5
	ctx := openViewerWithSessions(t, srv, want)
	waitForSessionRowCount(t, ctx, want)
}

func TestViewerSessionListIsScrollContainer(t *testing.T) {
	srv := testkit.NewServer(t)
	const want = 5
	ctx := openViewerWithSessions(t, srv, want)
	waitForSessionRowCount(t, ctx, want)

	var metrics struct {
		OverflowY    string  `json:"overflowY"`
		FlexGrow     string  `json:"flexGrow"`
		MinHeight    string  `json:"minHeight"`
		ScrollHeight float64 `json:"scrollHeight"`
		ClientHeight float64 `json:"clientHeight"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => {
		const list = document.getElementById('session-list');
		const s = getComputedStyle(list);
		return {
			overflowY: s.overflowY,
			flexGrow: s.flexGrow,
			minHeight: s.minHeight,
			scrollHeight: list.scrollHeight,
			clientHeight: list.clientHeight,
		};
	})()`, &metrics)); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	if metrics.OverflowY != "auto" {
		t.Fatalf("overflowY = %q, want auto", metrics.OverflowY)
	}
	if metrics.FlexGrow != "1" {
		t.Fatalf("flexGrow = %q, want 1", metrics.FlexGrow)
	}
	if metrics.MinHeight != "0px" {
		t.Fatalf("minHeight = %q, want 0px", metrics.MinHeight)
	}
	if metrics.ScrollHeight <= metrics.ClientHeight {
		t.Fatalf("scrollHeight (%v) should exceed clientHeight (%v) for %d sessions", metrics.ScrollHeight, metrics.ClientHeight, want)
	}
}

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
