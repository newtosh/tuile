package testkit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const defaultBrowserTimeout = 30 * time.Second

// BrowserContext returns a chromedp context with timeout; caller must cancel.
func BrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := chromedp.NewContext(context.Background())
	ctx, timeout := context.WithTimeout(ctx, defaultBrowserTimeout)
	return ctx, func() {
		timeout()
		cancel()
	}
}

// TerminalText opens the session view URL, waits for xterm, and returns visible terminal text.
// Skips the test when Chrome/chromedp is unavailable.
func (sess *Session) TerminalText(t *testing.T) string {
	t.Helper()
	ctx, cancel := BrowserContext(t)
	defer cancel()

	if err := chromedp.Run(ctx,
		chromedp.Navigate(sess.ViewURL()),
		chromedp.WaitVisible(".xterm-rows", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		var termText string
		err := chromedp.Run(ctx,
			chromedp.Sleep(400*time.Millisecond),
			chromedp.Text(".xterm-rows", &termText, chromedp.ByQuery),
		)
		if err != nil {
			t.Skipf("browser automation unavailable: %v", err)
		}
		if strings.TrimSpace(termText) != "" {
			return termText
		}
	}
	t.Fatal("timed out waiting for terminal text in browser viewer")
	return ""
}

// AssertTerminalContains polls the viewer until marker appears or times out.
func (sess *Session) AssertTerminalContains(t *testing.T, marker string) {
	t.Helper()
	ctx, cancel := BrowserContext(t)
	defer cancel()

	if err := chromedp.Run(ctx,
		chromedp.Navigate(sess.ViewURL()),
		chromedp.WaitVisible(".xterm-rows", chromedp.ByQuery),
	); err != nil {
		t.Skipf("browser automation unavailable: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var termText string
		err := chromedp.Run(ctx,
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Text(".xterm-rows", &termText, chromedp.ByQuery),
		)
		if err != nil {
			t.Skipf("browser automation unavailable: %v", err)
		}
		if strings.Contains(termText, marker) {
			return
		}
	}
	var last string
	_ = chromedp.Run(ctx, chromedp.Text(".xterm-rows", &last, chromedp.ByQuery))
	t.Fatalf("expected terminal to contain %q, got %q", marker, last)
}
